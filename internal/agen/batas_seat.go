package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Batas hold seat agen (keputusan 12 November 2026). Booking dari portal agen menahan seat 24 jam tanpa
// pembayaran; tanpa batas, satu agen bisa menahan ratusan seat dengan NIK asal sampai jadwal tampak penuh.
// Yang dihitung: seat jamaah dewasa (bookings.seat_count) dari booking buatan agen di portal
// (dibuat_oleh_agen_akun_id) yang masih berstatus 'baru' dan hold seat-nya belum lewat. Booking yang sudah DP
// atau lunas tidak dihitung. Booking pelanggan downline dan booking pribadi agen memakai batas akun biasa.
const (
	BatasSeatAgenPerJadwalBawaan = 20
	BatasSeatAgenTotalBawaan     = 45
	// Batas khusus per agen yang diisi admin tidak boleh melebihi angka ini (mencegah salah ketik membuka
	// lagi celah menghabiskan seat). Kuota seat jadwal tetap menjadi batas akhir.
	MaksBatasSeatAgenPerJadwal = 200
	MaksBatasSeatAgenTotal     = 500
)

var (
	ErrBatasSeatTidakValid = fmt.Errorf("batas seat harus antara 1 dan %d per jadwal, 1 dan %d total, dan batas per jadwal tidak boleh melebihi batas total", MaksBatasSeatAgenPerJadwal, MaksBatasSeatAgenTotal)
	ErrBatasSeatSama       = errors.New("batas seat tidak berubah")
	ErrAlasanBatasWajib    = errors.New("alasan perubahan wajib diisi (5 sampai 500 karakter)")
)

// BatasSeatAgen: batas yang berlaku untuk satu agen beserta seat yang sedang ditahan booking buatannya.
type BatasSeatAgen struct {
	PerJadwal int `json:"per_jadwal"`
	Total     int `json:"total"`
	// Batas khusus dari admin; nil = memakai batas bawaan.
	KhususPerJadwal *int `json:"khusus_per_jadwal"`
	KhususTotal     *int `json:"khusus_total"`
	// Seat yang sedang ditahan di semua jadwal.
	SeatDitahan int `json:"seat_ditahan"`
	// Riwayat perubahan batas khusus (hanya untuk admin; kosong di portal agen).
	Riwayat []BatasSeatLog `json:"riwayat,omitempty"`
}

type BatasSeatLog struct {
	PerJadwalLama *int      `json:"per_jadwal_lama"`
	PerJadwalBaru *int      `json:"per_jadwal_baru"`
	TotalLama     *int      `json:"total_lama"`
	TotalBaru     *int      `json:"total_baru"`
	Alasan        string    `json:"alasan"`
	DiubahOleh    *string   `json:"diubah_oleh"`
	DiubahAt      time.Time `json:"diubah_at"`
}

func intPtr(v sql.NullInt64) *int {
	if !v.Valid {
		return nil
	}
	n := int(v.Int64)
	return &n
}

func berlaku(khusus *int, bawaan int) int {
	if khusus != nil {
		return *khusus
	}
	return bawaan
}

// KunciBatasSeatAgen mengunci baris akun agen (FOR UPDATE) dan mengembalikan batas yang berlaku. Dipanggil
// di transaksi booking agen sebelum menghitung seat, sehingga booking paralel milik agen yang sama antre.
func KunciBatasSeatAgen(ctx context.Context, tx *sql.Tx, akunID int64) (perJadwal, total int, err error) {
	var pj, tot sql.NullInt64
	err = tx.QueryRowContext(ctx, `SELECT batas_seat_agen_per_jadwal, batas_seat_agen_total FROM customer_accounts WHERE id = ? FOR UPDATE`, akunID).Scan(&pj, &tot)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, ErrNotFound
	}
	if err != nil {
		return 0, 0, fmt.Errorf("agen.KunciBatasSeatAgen: %w", err)
	}
	return berlaku(intPtr(pj), BatasSeatAgenPerJadwalBawaan), berlaku(intPtr(tot), BatasSeatAgenTotalBawaan), nil
}

// SeatDitahanAgen menghitung seat yang ditahan booking buatan agen: total di semua jadwal dan di scheduleID
// (0 = tidak menghitung per jadwal).
func SeatDitahanAgen(ctx context.Context, q querier, akunID, scheduleID int64) (total, diJadwal int, err error) {
	err = q.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(seat_count),0), COALESCE(SUM(CASE WHEN schedule_id=? THEN seat_count ELSE 0 END),0)
		FROM bookings
		WHERE dibuat_oleh_agen_akun_id=? AND status='baru' AND is_seat_blocked=TRUE
		  AND (seat_hold_expires_at IS NULL OR seat_hold_expires_at > ?)`,
		scheduleID, akunID, time.Now()).Scan(&total, &diJadwal)
	if err != nil {
		return 0, 0, fmt.Errorf("agen.SeatDitahanAgen: %w", err)
	}
	return total, diJadwal, nil
}

// getBatasSeatAgen membaca batas yang berlaku dan seat ditahan. brandID non-nil membatasi ke brand admin.
func getBatasSeatAgen(ctx context.Context, q querier, akunID int64, brandID *int64, denganRiwayat bool) (*BatasSeatAgen, error) {
	var pj, tot sql.NullInt64
	var status string
	var jBrand int64
	err := q.QueryRowContext(ctx, `SELECT status_agen, brand_id, batas_seat_agen_per_jadwal, batas_seat_agen_total FROM customer_accounts WHERE id = ?`, akunID).
		Scan(&status, &jBrand, &pj, &tot)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (!sudahAgen(status) || (brandID != nil && *brandID != jBrand))) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("agen.getBatasSeatAgen: %w", err)
	}
	b := BatasSeatAgen{KhususPerJadwal: intPtr(pj), KhususTotal: intPtr(tot)}
	b.PerJadwal = berlaku(b.KhususPerJadwal, BatasSeatAgenPerJadwalBawaan)
	b.Total = berlaku(b.KhususTotal, BatasSeatAgenTotalBawaan)
	if b.SeatDitahan, _, err = SeatDitahanAgen(ctx, q, akunID, 0); err != nil {
		return nil, err
	}
	if !denganRiwayat {
		return &b, nil
	}
	rows, err := q.QueryContext(ctx, `
		SELECT l.per_jadwal_lama, l.per_jadwal_baru, l.total_lama, l.total_baru, l.alasan, a.display_name, l.diubah_at
		FROM agen_batas_seat_log l LEFT JOIN admin_users a ON a.id = l.diubah_oleh
		WHERE l.akun_id = ? ORDER BY l.diubah_at DESC, l.id DESC LIMIT 20`, akunID)
	if err != nil {
		return nil, fmt.Errorf("agen.getBatasSeatAgen riwayat: %w", err)
	}
	defer rows.Close()
	b.Riwayat = []BatasSeatLog{}
	for rows.Next() {
		var l BatasSeatLog
		var pjl, pjb, tl, tb sql.NullInt64
		var oleh sql.NullString
		if err := rows.Scan(&pjl, &pjb, &tl, &tb, &l.Alasan, &oleh, &l.DiubahAt); err != nil {
			return nil, fmt.Errorf("agen.getBatasSeatAgen scan: %w", err)
		}
		l.PerJadwalLama, l.PerJadwalBaru, l.TotalLama, l.TotalBaru = intPtr(pjl), intPtr(pjb), intPtr(tl), intPtr(tb)
		if oleh.Valid {
			l.DiubahOleh = &oleh.String
		}
		b.Riwayat = append(b.Riwayat, l)
	}
	return &b, rows.Err()
}

// UbahBatasSeatRequest: nil = kembali ke batas bawaan untuk kolom itu.
type UbahBatasSeatRequest struct {
	PerJadwal *int   `json:"per_jadwal"`
	Total     *int   `json:"total"`
	Alasan    string `json:"alasan"`
}

func validasiBatasSeat(req UbahBatasSeatRequest) error {
	if req.PerJadwal != nil && (*req.PerJadwal < 1 || *req.PerJadwal > MaksBatasSeatAgenPerJadwal) {
		return ErrBatasSeatTidakValid
	}
	if req.Total != nil && (*req.Total < 1 || *req.Total > MaksBatasSeatAgenTotal) {
		return ErrBatasSeatTidakValid
	}
	if berlaku(req.PerJadwal, BatasSeatAgenPerJadwalBawaan) > berlaku(req.Total, BatasSeatAgenTotalBawaan) {
		return ErrBatasSeatTidakValid
	}
	if n := utf8.RuneCountInString(strings.TrimSpace(req.Alasan)); n < 5 || n > 500 {
		return ErrAlasanBatasWajib
	}
	return nil
}

func samaPtr(a, b *int) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func nullInt(p *int) any {
	if p == nil {
		return nil
	}
	return *p
}

// ubahBatasSeatTx menyimpan batas khusus agen dan mencatat perubahannya. brandID non-nil = admin brand.
func ubahBatasSeatTx(ctx context.Context, tx *sql.Tx, akunID int64, brandID *int64, adminID int64, req UbahBatasSeatRequest) error {
	if err := validasiBatasSeat(req); err != nil {
		return err
	}
	var pj, tot sql.NullInt64
	var status string
	var jBrand int64
	err := tx.QueryRowContext(ctx, `SELECT status_agen, brand_id, batas_seat_agen_per_jadwal, batas_seat_agen_total FROM customer_accounts WHERE id = ? FOR UPDATE`, akunID).
		Scan(&status, &jBrand, &pj, &tot)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && (!sudahAgen(status) || (brandID != nil && *brandID != jBrand))) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("agen.UbahBatasSeat baca: %w", err)
	}
	lamaPJ, lamaTot := intPtr(pj), intPtr(tot)
	if samaPtr(lamaPJ, req.PerJadwal) && samaPtr(lamaTot, req.Total) {
		return ErrBatasSeatSama
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_accounts SET batas_seat_agen_per_jadwal = ?, batas_seat_agen_total = ? WHERE id = ?`,
		nullInt(req.PerJadwal), nullInt(req.Total), akunID); err != nil {
		return fmt.Errorf("agen.UbahBatasSeat update: %w", err)
	}
	var pelaku any
	if adminID > 0 {
		pelaku = adminID
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO agen_batas_seat_log (akun_id, per_jadwal_lama, per_jadwal_baru, total_lama, total_baru, alasan, diubah_oleh)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		akunID, nullInt(lamaPJ), nullInt(req.PerJadwal), nullInt(lamaTot), nullInt(req.Total), strings.TrimSpace(req.Alasan), pelaku); err != nil {
		return fmt.Errorf("agen.UbahBatasSeat log: %w", err)
	}
	return nil
}

func (r *Repository) GetBatasSeatAgen(ctx context.Context, akunID int64, brandID *int64) (*BatasSeatAgen, error) {
	return getBatasSeatAgen(ctx, r.db, akunID, brandID, true)
}

func (r *Repository) UbahBatasSeatAgen(ctx context.Context, akunID int64, brandID *int64, adminID int64, req UbahBatasSeatRequest) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return ubahBatasSeatTx(ctx, tx, akunID, brandID, adminID, req) })
}
