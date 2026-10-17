package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var nikPattern = regexp.MustCompile(`^\d{16}$`)

// ErrNIKAtasNamaLain: NIK sudah dimiliki jamaah lain di brand ini dengan nama berbeda.
var ErrNIKAtasNamaLain = errors.New("NIK sudah terdaftar dengan nama yang berbeda")

// nikBedaError: NIK sudah terdaftar dengan nama berbeda. Pesannya menyebut nomor jamaah di form,
// tanpa membuka nama yang terdaftar (privasi), dan lolos errors.Is(err, ErrNIKAtasNamaLain).
type nikBedaError struct{ nomor int }

func (e *nikBedaError) Error() string {
	return "NIK Jamaah " + strconv.Itoa(e.nomor) + " sudah terdaftar. Periksa NIK dan nama sesuai KTP, atau hubungi kami via WhatsApp."
}

func (e *nikBedaError) Is(target error) bool { return target == ErrNIKAtasNamaLain }

func errNIKBeda(nomor int) error { return &nikBedaError{nomor: nomor} }

func nikBersih(p *string) string {
	if p == nil {
		return ""
	}
	return strings.TrimSpace(*p)
}

func nikValid(nik string) bool { return nikPattern.MatchString(nik) }

// cariJamaahByNIK mencari jamaah di brand dengan NIK itu (kunci unik brand + nik).
func cariJamaahByNIK(ctx context.Context, tx *sql.Tx, brandID int64, nik string) (id int64, nama string, ada bool, err error) {
	err = tx.QueryRowContext(ctx, `SELECT id, nama_lengkap FROM jamaah WHERE brand_id=? AND nik=? LIMIT 1 FOR UPDATE`, brandID, nik).Scan(&id, &nama)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, fmt.Errorf("cari jamaah by nik: %w", err)
	}
	return id, nama, true, nil
}

// isiNIKBilaKosong mengisi NIK jamaah hanya bila belum ada; NIK yang sudah tersimpan tidak ditimpa.
func isiNIKBilaKosong(ctx context.Context, tx *sql.Tx, jamaahID int64, nik string) error {
	if _, err := tx.ExecContext(ctx, `UPDATE jamaah SET nik=? WHERE id=? AND (nik IS NULL OR nik='')`, nik, jamaahID); err != nil {
		return fmt.Errorf("isi nik jamaah: %w", err)
	}
	return nil
}

// pasangNIKPIC: NIK pemesan yang ikut berangkat. Bila NIK itu milik jamaah lain, ditolak; bila jamaah
// pemesan belum punya NIK, NIK diisikan.
func pasangNIKPIC(ctx context.Context, tx *sql.Tx, brandID, jamaahID int64, nik string) error {
	if nik == "" {
		return nil
	}
	id, _, ada, err := cariJamaahByNIK(ctx, tx, brandID, nik)
	if err != nil {
		return err
	}
	if ada && id != jamaahID {
		return errNIKBeda(1)
	}
	return isiNIKBilaKosong(ctx, tx, jamaahID, nik)
}
