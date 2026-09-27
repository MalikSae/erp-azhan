package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Kaitan agen ke jamaah (agen-azhan.md 3.2–3.5, 7.1, 7.3).
// Aturan inti: first-set wins — hanya jamaah berstatus 'belum_ditentukan'
// yang boleh diikat; 'terikat_agen' dan 'tanpa_agen' tidak pernah diubah di
// sini (ganti hanya lewat C4 Admin Master).

var ErrAgenTidakValid = errors.New("agen tidak ditemukan atau tidak aktif di brand ini")

// Asal kaitan (kolom jamaah.kaitan_sumber, migrasi 064). Hanya SumberJalur3
// yang boleh diganti Admin Master lewat C4.
const (
	SumberJalur1   = "jalur1"
	SumberReferral = "jalur2"
	SumberJalur3   = "jalur3"
	SumberIkutPIC  = "ikut_pic"
)

// ResolveKodeReferral mengembalikan ID agen aktif pemilik kode di brand ini.
// Kode kosong, tidak dikenal, milik agen nonaktif, atau brand lain -> 0 tanpa
// error (diabaikan diam-diam, keputusan L8).
func ResolveKodeReferral(ctx context.Context, q querier, brandID int64, kode string) (int64, error) {
	kode = strings.ToUpper(strings.TrimSpace(kode))
	if kode == "" || len(kode) > 20 {
		return 0, nil
	}
	var id int64
	err := q.QueryRowContext(ctx,
		`SELECT id FROM jamaah WHERE brand_id = ? AND kode_referral = ? AND status_agen = 'aktif'`, brandID, kode).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("agen: resolve kode referral: %w", err)
	}
	return id, nil
}

// ValidasiAgenAktif memastikan agenID adalah agen aktif di brand ini (Jalur 3).
func ValidasiAgenAktif(ctx context.Context, q querier, brandID, agenID int64) error {
	var n int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM jamaah WHERE id = ? AND brand_id = ? AND status_agen = 'aktif'`, agenID, brandID).Scan(&n); err != nil {
		return fmt.Errorf("agen: validasi agen: %w", err)
	}
	if n == 0 {
		return ErrAgenTidakValid
	}
	return nil
}

// IkatJamaah mengikat jamaah ke agen bila masih 'belum_ditentukan'. Jamaah
// tidak pernah diikat ke dirinya sendiri. sumber: salah satu konstanta Sumber*.
func IkatJamaah(ctx context.Context, tx *sql.Tx, jamaahID, agenID int64, sumber string) error {
	if jamaahID <= 0 || agenID <= 0 || jamaahID == agenID {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE jamaah SET direkrut_oleh_jamaah_id = ?, kaitan_status = 'terikat_agen', kaitan_sumber = ?
		WHERE id = ? AND kaitan_status = 'belum_ditentukan'`, agenID, sumber, jamaahID)
	if err != nil {
		return fmt.Errorf("agen: ikat jamaah: %w", err)
	}
	return nil
}

// IkatRombongan menerapkan auto-bind multi-pax (3.4): agen booking adalah
// agen dari kode referral bila ada, selain itu agen milik PIC. Semua pax yang
// masih 'belum_ditentukan' diikat ke agen tersebut.
func IkatRombongan(ctx context.Context, tx *sql.Tx, picID int64, jamaahIDs []int64, agenReferral int64) error {
	agen := agenReferral
	sumber := SumberReferral
	if agen <= 0 {
		sumber = SumberIkutPIC
		var direkrut sql.NullInt64
		if err := tx.QueryRowContext(ctx, `SELECT direkrut_oleh_jamaah_id FROM jamaah WHERE id = ?`, picID).Scan(&direkrut); err != nil {
			return fmt.Errorf("agen: baca agen PIC: %w", err)
		}
		if !direkrut.Valid {
			return nil
		}
		agen = direkrut.Int64
	}
	if err := IkatJamaah(ctx, tx, picID, agen, sumber); err != nil {
		return err
	}
	for _, id := range jamaahIDs {
		if err := IkatJamaah(ctx, tx, id, agen, sumber); err != nil {
			return err
		}
	}
	return nil
}

// IkatRombonganBooking menjalankan IkatRombongan untuk booking yang sudah
// tersimpan (booking admin & finalisasi draft): ikut agen PIC.
func IkatRombonganBooking(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	var pic sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT pic_jamaah_id FROM bookings WHERE id = ?`, bookingID).Scan(&pic); err != nil {
		return fmt.Errorf("agen: baca PIC booking: %w", err)
	}
	if !pic.Valid {
		return nil
	}
	rows, err := tx.QueryContext(ctx, `SELECT jamaah_id FROM booking_pax WHERE booking_id = ? AND pax_status = 'aktif'`, bookingID)
	if err != nil {
		return fmt.Errorf("agen: baca pax booking: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	return IkatRombongan(ctx, tx, pic.Int64, ids, 0)
}

// TetapkanKaitanJalur3 menyimpan pilihan eksplisit Admin untuk jamaah baru
// (3.5): 'agen' dengan agenID, atau 'tanpa_agen'.
func TetapkanKaitanJalur3(ctx context.Context, tx *sql.Tx, jamaahID int64, mode string, agenID int64) error {
	switch mode {
	case "agen":
		_, err := tx.ExecContext(ctx,
			`UPDATE jamaah SET direkrut_oleh_jamaah_id = ?, kaitan_status = 'terikat_agen', kaitan_sumber = 'jalur3' WHERE id = ?`, agenID, jamaahID)
		return err
	case "tanpa_agen":
		_, err := tx.ExecContext(ctx,
			`UPDATE jamaah SET direkrut_oleh_jamaah_id = NULL, kaitan_status = 'tanpa_agen', kaitan_sumber = 'jalur3' WHERE id = ?`, jamaahID)
		return err
	}
	return fmt.Errorf("agen: mode kaitan tidak dikenal: %q", mode)
}

// AgenAktif adalah satu pilihan di picker "Kaitkan ke Agen" (screen B4/C3).
type AgenAktif struct {
	ID           int64   `json:"id"`
	NamaLengkap  string  `json:"nama_lengkap"`
	KodeReferral *string `json:"kode_referral"`
	NoHP         *string `json:"no_hp"`
}

// ListAgenAktif mencari agen aktif di satu brand (nama, no HP, atau kode).
func ListAgenAktif(ctx context.Context, q querier, brandID int64, cari string) ([]AgenAktif, error) {
	like := "%" + strings.TrimSpace(cari) + "%"
	rows, err := q.QueryContext(ctx, `
		SELECT id, nama_lengkap, kode_referral, no_hp FROM jamaah
		WHERE brand_id = ? AND status_agen = 'aktif'
		  AND (? = '%%' OR nama_lengkap LIKE ? OR no_hp LIKE ? OR kode_referral LIKE ?)
		ORDER BY nama_lengkap LIMIT 20`, brandID, like, like, like, like)
	if err != nil {
		return nil, fmt.Errorf("agen: list agen aktif: %w", err)
	}
	defer rows.Close()
	items := []AgenAktif{}
	for rows.Next() {
		var a AgenAktif
		if err := rows.Scan(&a.ID, &a.NamaLengkap, &a.KodeReferral, &a.NoHP); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}
