package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"erp-azhan/api/internal/shared"
)

func nullableID(id int64) any {
	if id <= 0 {
		return nil
	}
	return id
}

// resolveAkunPIC menentukan jamaah PIC untuk booking atas nama akun pelanggan.
//
//  1. Akun sudah punya jamaah_id: pakai itu.
//  2. Nomor WhatsApp akun sudah dimiliki jamaah di brand ini: pakai ulang hanya
//     bila nama cocok; bila berbeda, tolak (nomor itu milik orang lain).
//  3. Belum ada: buat jamaah baru dari data akun.
//
// Nomor, email, dan identitas akun diambil dari tabel akun, bukan dari request,
// sehingga PIC tidak bisa diarahkan ke nomor orang lain lewat payload.
func resolveAkunPIC(ctx context.Context, tx *sql.Tx, brandID, accountID int64, pic PICInput) (int64, error) {
	var accBrand int64
	var email, noHP string
	var jamaahID sql.NullInt64
	err := tx.QueryRowContext(ctx,
		`SELECT brand_id, email, no_hp, jamaah_id FROM customer_accounts WHERE id=? FOR UPDATE`, accountID,
	).Scan(&accBrand, &email, &noHP, &jamaahID)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && accBrand != brandID) {
		return 0, ErrAkunTidakValid
	}
	if err != nil {
		return 0, fmt.Errorf("baca akun: %w", err)
	}
	if jamaahID.Valid {
		return jamaahID.Int64, nil
	}

	canonical, local := shared.PhoneVariants(noHP)
	var existingID int64
	var existingNama string
	err = tx.QueryRowContext(ctx, `
		SELECT id, nama_lengkap FROM jamaah
		WHERE brand_id = ? AND REGEXP_REPLACE(COALESCE(no_hp,''),'[^0-9]','') IN (?,?)
		ORDER BY id LIMIT 1 FOR UPDATE`, brandID, canonical, local).Scan(&existingID, &existingNama)

	switch {
	case errors.Is(err, sql.ErrNoRows):
		jk := pic.JenisKelamin
		em := email
		existingID, err = shared.ResolveJamaah(ctx, tx, brandID, shared.JamaahInput{
			NamaLengkap:  strings.TrimSpace(pic.NamaLengkap),
			NoHP:         noHP,
			Email:        &em,
			JenisKelamin: &jk,
		})
		if err != nil {
			return 0, fmt.Errorf("buat jamaah pemesan: %w", err)
		}
	case err != nil:
		return 0, fmt.Errorf("cari jamaah pemesan: %w", err)
	default:
		if normalizeName(pic.NamaLengkap) != normalizeName(existingNama) {
			return 0, ErrNomorAtasNamaLain
		}
	}

	if _, err = tx.ExecContext(ctx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, existingID, accountID); err != nil {
		return 0, fmt.Errorf("kaitkan akun ke jamaah: %w", err)
	}
	return existingID, nil
}

// simpanTravelers menyimpan anggota rombongan ke daftar jamaah tersimpan akun
// (tanpa duplikat menurut nama + tanggal lahir).
func simpanTravelers(ctx context.Context, tx *sql.Tx, accountID int64, anggota []AnggotaInput) error {
	for _, a := range anggota {
		nama := strings.TrimSpace(a.NamaLengkap)
		if nama == "" {
			continue
		}
		var dob any
		if a.TanggalLahir != nil && strings.TrimSpace(*a.TanggalLahir) != "" {
			dob = strings.TrimSpace(*a.TanggalLahir)
		}
		var nik any
		if n := nikBersih(a.NIK); n != "" {
			nik = n
		}
		var hp any
		if a.NoHP != nil && strings.TrimSpace(*a.NoHP) != "" {
			hp = strings.TrimSpace(*a.NoHP)
		}
		_, err := tx.ExecContext(ctx, `
			INSERT INTO customer_travelers (account_id, nama_lengkap, nama_normal, jenis_kelamin, no_hp, tanggal_lahir, nik)
			SELECT ?, ?, ?, ?, ?, ?, ? FROM DUAL
			WHERE NOT EXISTS (
				SELECT 1 FROM customer_travelers WHERE account_id=? AND nama_normal=? AND tanggal_lahir <=> ?
			)`,
			accountID, nama, normalizeName(nama), a.JenisKelamin, hp, dob, nik,
			accountID, normalizeName(nama), dob)
		if err != nil {
			return fmt.Errorf("simpan jamaah tersimpan: %w", err)
		}
		if nik != nil { // jamaah tersimpan lama yang belum punya NIK dilengkapi
			if _, err = tx.ExecContext(ctx, `UPDATE customer_travelers SET nik=? WHERE account_id=? AND nama_normal=? AND tanggal_lahir <=> ? AND nik IS NULL`, nik, accountID, normalizeName(nama), dob); err != nil {
				return fmt.Errorf("lengkapi nik jamaah tersimpan: %w", err)
			}
		}
	}
	return nil
}
