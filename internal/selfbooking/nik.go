package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
)

var nikPattern = regexp.MustCompile(`^\d{16}$`)

// ErrNIKAtasNamaLain: NIK sudah dimiliki jamaah lain di brand ini dengan nama berbeda.
var ErrNIKAtasNamaLain = errors.New("NIK sudah terdaftar dengan nama yang berbeda")

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

// pasangNIKPIC: NIK pemesan yang ikut berangkat. Bila jamaah pemesan belum punya NIK, NIK diisikan.
// Bila NIK itu sudah milik jamaah lain, booking TIDAK ditolak dan NIK tidak dipasang: menolak akan
// memberi tahu bahwa NIK itu terdaftar (oracle), dan sejak K1 PIC akun selalu jamaah baru sehingga
// pelanggan lama yang NIK-nya tercatat di baris lama pasti bentrok. Bentrok dicatat di log audit
// (tanpa NIK) agar admin dapat menggabungkan jamaah secara manual.
func pasangNIKPIC(ctx context.Context, tx *sql.Tx, brandID, jamaahID int64, nik string) error {
	if nik == "" {
		return nil
	}
	id, _, ada, err := cariJamaahByNIK(ctx, tx, brandID, nik)
	if err != nil {
		return err
	}
	if ada && id != jamaahID {
		log.Printf("[AUDIT] selfbooking nik_pic_bentrok brand=%d jamaah_pic=%d jamaah_lama=%d (perlu tinjauan/penggabungan admin)", brandID, jamaahID, id)
		return nil
	}
	return isiNIKBilaKosong(ctx, tx, jamaahID, nik)
}
