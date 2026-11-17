package booking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"
)

// errBelumJatuhTempo: guard pembatalan otomatis menemukan booking tidak lagi memenuhi syarat (sudah lunas, tenggat
// bergeser, atau pembayaran yang menunggu verifikasi sudah menutup tagihan). Bukan galat: booking dilewati.
var errBelumJatuhTempo = errors.New("booking tidak memenuhi syarat batal jatuh tempo")

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// kandidatJatuhTempo: booking dp yang hari tenggat lunasnya sudah lewat. Hari tenggat itu sendiri masih termasuk.
// Booking yang pernah lunas (pertama_lunas_at terisi) tidak dibatalkan otomatis: ia turun ke dp karena penyesuaian
// harga atau pembayaran yang dibatalkan admin, bukan karena pelanggan lalai melunasi; admin yang memutuskan.
func kandidatJatuhTempo(ctx context.Context, q querier, hari string) ([]int64, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT b.id FROM bookings b
		JOIN booking_checkout c ON c.booking_id = b.id
		WHERE b.status = 'dp' AND b.pertama_lunas_at IS NULL AND c.due_at IS NOT NULL AND DATE(c.due_at) < ?
		ORDER BY b.id`, hari)
	if err != nil {
		return nil, fmt.Errorf("booking.kandidatJatuhTempo: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// batalkanJatuhTempoTx membatalkan satu booking di transaksi tx (pemanggil yang commit). false = dilewati.
// Syarat diperiksa ulang setelah baris booking dikunci, jadi pembayaran atau verifikasi yang masuk di antara
// pencarian kandidat dan pembatalan tidak bisa terlewat.
func batalkanJatuhTempoTx(ctx context.Context, tx *sql.Tx, bookingID int64, hari string) (bool, error) {
	guard := func(ctx context.Context, tx *sql.Tx) error {
		var layak bool
		err := tx.QueryRowContext(ctx, `
			SELECT b.status = 'dp' AND b.pertama_lunas_at IS NULL AND DATE(c.due_at) < ?
			   AND COALESCE(b.total_harga, 0) >
			       COALESCE((SELECT SUM(p.jumlah) FROM payments p WHERE p.booking_id = b.id AND p.status IN ('confirmed','pending')), 0)
			FROM bookings b JOIN booking_checkout c ON c.booking_id = b.id
			WHERE b.id = ?`, hari, bookingID).Scan(&layak)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && !layak) {
			return errBelumJatuhTempo
		}
		return err
	}
	if err := updateBookingStatusTx(ctx, tx, bookingID, "batal", guard); err != nil {
		if errors.Is(err, errBelumJatuhTempo) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// BatalkanLewatBatasPelunasan membatalkan booking berstatus dp yang melewati tenggat lunas (booking_checkout.due_at,
// H-30 secara bawaan). Hari tenggat masih termasuk: booking dibatalkan mulai hari berikutnya. Pembatalan melepas
// seat seperti pembatalan admin; pembayaran yang sudah masuk tidak disentuh, urusan keuangannya ditangani manual.
//
// Booking dilewati bila pembayaran yang masih menunggu verifikasi, bila diterima, menutup tagihan: admin akan
// memverifikasinya (lunas) atau menolaknya (dibatalkan pada putaran berikutnya). Mengembalikan id yang dibatalkan.
func (r *Repository) BatalkanLewatBatasPelunasan(ctx context.Context, now time.Time) ([]int64, error) {
	hari := now.Format("2006-01-02")
	kandidat, err := kandidatJatuhTempo(ctx, r.db, hari)
	if err != nil {
		return nil, err
	}
	var dibatalkan []int64
	for _, id := range kandidat {
		ok, err := r.batalkanSatu(ctx, id, hari)
		if err != nil {
			log.Printf("[WARN] batal jatuh tempo booking %d: %v", id, err)
			continue
		}
		if ok {
			dibatalkan = append(dibatalkan, id)
		}
	}
	return dibatalkan, nil
}

func (r *Repository) batalkanSatu(ctx context.Context, bookingID int64, hari string) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	ok, err := batalkanJatuhTempoTx(ctx, tx, bookingID, hari)
	if err != nil || !ok {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
