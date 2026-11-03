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

// ErrSayaSendiriBukanAnda: pilihan "Saya sendiri" dipakai untuk jamaah yang NIK-nya bukan NIK agen (K24).
var ErrSayaSendiriBukanAnda = errors.New("pilihan Saya sendiri hanya untuk data Anda sendiri")

// pastikanSayaSendiri menolak "Saya sendiri" bila NIK jamaah pertama berbeda dari NIK yang agen isi saat mendaftar.
// Tanpa pemeriksaan ini agen bisa mengetik NIK jamaah yang terkunci ke agen lain dan mengambil komisinya.
func pastikanSayaSendiri(ctx context.Context, tx *sql.Tx, agenID, jamaahID int64) error {
	var nikAgen, nikJamaah sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT nik_agen FROM customer_accounts WHERE id = ?`, agenID).Scan(&nikAgen); err != nil {
		return fmt.Errorf("selfbooking: baca NIK agen: %w", err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT nik FROM jamaah WHERE id = ?`, jamaahID).Scan(&nikJamaah); err != nil {
		return fmt.Errorf("selfbooking: baca NIK jamaah: %w", err)
	}
	if !nikAgen.Valid || nikAgen.String == "" || nikAgen.String != nikJamaah.String {
		return ErrSayaSendiriBukanAnda
	}
	return nil
}

// nikBedaError: NIK sudah terdaftar dengan nama berbeda. Pesannya generik: menyebut nomor jamaah di
// form tanpa membuka nama terdaftar maupun keberadaan NIK itu (bukan oracle data pihak lain), dan
// lolos errors.Is(err, ErrNIKAtasNamaLain). Percobaan yang gagal dibatasi per akun (handler.go).
type nikBedaError struct{ nomor int }

func (e *nikBedaError) Error() string {
	return "Data Jamaah " + strconv.Itoa(e.nomor) + " belum dapat diproses. Periksa kembali NIK dan nama sesuai KTP, atau hubungi kami via WhatsApp."
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
