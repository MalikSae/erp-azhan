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

// ErrSayaSendiriBukanAnda: pemesan agen yang ikut berangkat memakai NIK selain NIK profil agennya.
var ErrSayaSendiriBukanAnda = errors.New("NIK tidak sesuai profil agen")

// ErrNIKAgenBelumAda: pemesan agen ikut berangkat tetapi belum punya NIK tercatat.
var ErrNIKAgenBelumAda = errors.New("Lengkapi NIK di profil agen.")

// ErrInputTidakValid: input yang hanya berlaku untuk agen aktif (jamaah_id) dikirim akun biasa.
var ErrInputTidakValid = errors.New("data tidak valid")

// resolvePax adalah satu pintu pencocokan NIK untuk PIC dan anggota semua pemesan. NIK sama dan nama sama
// memakai ulang jamaah lama, NIK sama dengan nama berbeda ditolak generik, NIK baru membuat jamaah baru.
// Bayi tanpa NIK dicocokkan lewat nama dan tanggal lahir.
//
// jamaahID (Jamaah Saya) hanya untuk pemesan agen aktif dan hanya jalan pintas: jamaah itu tetap harus cocok
// dengan NIK dan nama yang dikirim, jadi id yang ditebak tidak membuka apa pun. Akun biasa: ErrInputTidakValid.
func resolvePax(ctx context.Context, tx *sql.Tx, brandID int64, p pemesan, jamaahID *int64, a AnggotaInput) (int64, error) {
	if jamaahID == nil || *jamaahID <= 0 {
		return resolveAnggotaPublik(ctx, tx, brandID, p.id, a)
	}
	if !p.agen {
		return 0, ErrInputTidakValid
	}
	var nik, nama sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT nik, nama_lengkap FROM jamaah WHERE id = ? AND brand_id = ? FOR UPDATE`,
		*jamaahID, brandID).Scan(&nik, &nama)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrJamaahBukanMilik
	}
	if err != nil {
		return 0, fmt.Errorf("cek jamaah: %w", err)
	}
	// Jamaah ber-NIK: NIK harus sama. Jamaah tanpa NIK (bayi): cukup nama yang sama.
	if (nik.String != "" && nikBersih(a.NIK) != nik.String) || normalizeName(a.NamaLengkap) != normalizeName(nama.String) {
		return 0, ErrJamaahBukanMilik
	}
	return *jamaahID, nil
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
