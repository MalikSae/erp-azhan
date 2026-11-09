package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// Kunci (kaitan) jamaah ke akun agen dan pengikatan akun ke agen (docs/spek/SPEK-KEAGENAN-AKUN-2026-11-03.md bagian 4).
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
		`SELECT id FROM customer_accounts WHERE brand_id = ? AND kode_referral = ? AND status_agen = 'aktif'`, brandID, kode).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("agen: resolve kode referral: %w", err)
	}
	return id, nil
}

// ValidasiAgenAktif memastikan agenAkunID adalah agen aktif di brand ini (Jalur 3).
func ValidasiAgenAktif(ctx context.Context, q querier, brandID, agenAkunID int64) error {
	var n int
	if err := q.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM customer_accounts WHERE id = ? AND brand_id = ? AND status_agen = 'aktif'`, agenAkunID, brandID).Scan(&n); err != nil {
		return fmt.Errorf("agen: validasi agen: %w", err)
	}
	if n == 0 {
		return ErrAgenTidakValid
	}
	return nil
}

// IkatJamaah mengikat jamaah ke akun agen bila masih 'belum_ditentukan'
// (first-set wins, kunci seumur hidup). sumber: salah satu konstanta Sumber*.
func IkatJamaah(ctx context.Context, tx *sql.Tx, jamaahID, agenAkunID int64, sumber string) error {
	if jamaahID <= 0 || agenAkunID <= 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE jamaah SET direkrut_oleh_akun_id = ?, kaitan_status = 'terikat_agen', kaitan_sumber = ?
		WHERE id = ? AND kaitan_status = 'belum_ditentukan'`, agenAkunID, sumber, jamaahID)
	if err != nil {
		return fmt.Errorf("agen: ikat jamaah: %w", err)
	}
	return nil
}

// AgenPengikatAkun mengembalikan id akun agen yang mengikat akun ini saat akun
// dibuat (customer_accounts.direkrut_oleh_akun_id), asalkan agen itu masih
// aktif. 0 bila akun tidak punya pengikat atau pengikatnya tidak lagi aktif.
func AgenPengikatAkun(ctx context.Context, q querier, akunID int64) (int64, error) {
	var id int64
	err := q.QueryRowContext(ctx, `
		SELECT a.id FROM customer_accounts c JOIN customer_accounts a ON a.id = c.direkrut_oleh_akun_id
		WHERE c.id = ? AND a.status_agen = 'aktif'`, akunID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("agen: baca pengikat akun: %w", err)
	}
	return id, nil
}

// IkatAkunDariReferral mengikat akun yang baru dibuat ke agen pemilik kode
// referral (cookie azhan_ref, klik terakhir) dan mengembalikan id agennya.
// Kode kosong, tidak dikenal, atau milik agen nonaktif diabaikan diam-diam (0).
// Pengikatan hanya terjadi di sini: setelah akun ada, cookie tidak dibaca lagi.
func IkatAkunDariReferral(ctx context.Context, q querier, brandID int64, kode string) (int64, error) {
	return ResolveKodeReferral(ctx, q, brandID, kode)
}

// IkatAkunDariJamaah menyamakan pengikat akun dengan pemilik kunci jamaah milik akun itu sendiri: akun pemesan yang ikut berangkat
// (jamaahID = dirinya) dan belum punya pengikat menjadi downline agen aktif yang lebih dulu mengunci jamaah itu (first-set wins,
// docs/spek/SPEK-KEAGENAN-AKUN-2026-11-03.md). Akun yang sudah punya pengikat tidak diubah; kunci jamaah juga tidak pernah diubah.
func IkatAkunDariJamaah(ctx context.Context, tx *sql.Tx, akunID, jamaahID int64) error {
	if akunID <= 0 || jamaahID <= 0 {
		return nil
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE customer_accounts c
		JOIN jamaah j ON j.id = ? AND j.kaitan_status = 'terikat_agen'
		JOIN customer_accounts a ON a.id = j.direkrut_oleh_akun_id AND a.brand_id = c.brand_id AND a.status_agen = 'aktif' AND a.id <> c.id
		SET c.direkrut_oleh_akun_id = a.id
		WHERE c.id = ? AND c.direkrut_oleh_akun_id IS NULL`, jamaahID, akunID)
	if err != nil {
		return fmt.Errorf("agen: ikat akun dari jamaah: %w", err)
	}
	return nil
}

// IkatRombongan mengikat semua jamaah yang masih 'belum_ditentukan' ke akun
// agen (auto-bind multi-pax). Jamaah yang sudah terikat tidak diubah. Prioritas
// penentu agen ditetapkan pemanggil: portal agen memakai agen pembuat booking,
// web publik memakai agen pengikat akun pemesan (AgenPengikatAkun), cookie
// tidak dibaca saat booking. agenAkunID <= 0: tidak ada yang diikat.
func IkatRombongan(ctx context.Context, tx *sql.Tx, jamaahIDs []int64, agenAkunID int64, sumber string) error {
	for _, id := range jamaahIDs {
		if err := IkatJamaah(ctx, tx, id, agenAkunID, sumber); err != nil {
			return err
		}
	}
	return nil
}

// IkatRombonganBooking menjalankan pengikatan untuk booking yang sudah
// tersimpan (booking admin & finalisasi draft, tanpa akun pemesan): semua pax
// yang belum terikat ikut agen pemilik kunci PIC.
func IkatRombonganBooking(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	var direkrut sql.NullInt64
	err := tx.QueryRowContext(ctx, `
		SELECT j.direkrut_oleh_akun_id FROM bookings b JOIN jamaah j ON j.id = b.pic_jamaah_id
		WHERE b.id = ?`, bookingID).Scan(&direkrut)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !direkrut.Valid) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("agen: baca agen PIC: %w", err)
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
	return IkatRombongan(ctx, tx, ids, direkrut.Int64, SumberIkutPIC)
}

// TetapkanKaitanJalur3 menyimpan pilihan eksplisit Admin untuk jamaah baru
// (3.5): 'agen' dengan agenAkunID, atau 'tanpa_agen'.
func TetapkanKaitanJalur3(ctx context.Context, tx *sql.Tx, jamaahID int64, mode string, agenAkunID int64) error {
	switch mode {
	case "agen":
		_, err := tx.ExecContext(ctx,
			`UPDATE jamaah SET direkrut_oleh_akun_id = ?, kaitan_status = 'terikat_agen', kaitan_sumber = 'jalur3' WHERE id = ?`, agenAkunID, jamaahID)
		return err
	case "tanpa_agen":
		_, err := tx.ExecContext(ctx,
			`UPDATE jamaah SET direkrut_oleh_akun_id = NULL, kaitan_status = 'tanpa_agen', kaitan_sumber = 'jalur3' WHERE id = ?`, jamaahID)
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
		SELECT id, nama_lengkap, kode_referral, no_hp FROM customer_accounts
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
