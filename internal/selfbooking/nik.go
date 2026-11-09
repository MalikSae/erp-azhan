package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"erp-azhan/api/internal/agen"
)

var nikPattern = regexp.MustCompile(`^\d{16}$`)

// ErrNIKAtasNamaLain: NIK sudah dimiliki jamaah lain di brand ini dengan nama berbeda.
var ErrNIKAtasNamaLain = errors.New("NIK sudah terdaftar dengan nama yang berbeda")

// ErrSayaSendiriBukanAnda: pilihan "Saya sendiri" dipakai untuk jamaah yang NIK-nya bukan NIK agen (K24).
var ErrSayaSendiriBukanAnda = errors.New("NIK jamaah ini tidak sama dengan NIK Anda sebagai agen, jadi tidak bisa dipilih sebagai Saya sendiri")

// ErrNIKAgenBelumAda: agen memilih "Saya sendiri" tetapi belum punya NIK tercatat (agen lama, sebelum NIK wajib).
var ErrNIKAgenBelumAda = errors.New("NIK Anda sebagai agen belum tercatat, jadi Saya sendiri belum bisa dipakai. Pilih Isi baru atau hubungi admin untuk melengkapinya")

// isiNIKSayaSendiri mengisi NIK jamaah pertama dari NIK agen bila memilih "Saya sendiri": NIK yang dikirim browser diabaikan,
// jadi agen tidak perlu mengetik ulang dan NIK utuhnya tidak pernah dikirim ke browser. Pemeriksaan pastikanSayaSendiri tetap berlaku.
func isiNIKSayaSendiri(ctx context.Context, q interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}, agenID int64, req *BookingRequest) error {
	if !req.PIC.SayaSendiri {
		return nil
	}
	var nik sql.NullString
	if err := q.QueryRowContext(ctx, `SELECT nik_agen FROM customer_accounts WHERE id = ?`, agenID).Scan(&nik); err != nil {
		return fmt.Errorf("selfbooking: baca NIK agen: %w", err)
	}
	if !nik.Valid || nik.String == "" {
		return ErrNIKAgenBelumAda
	}
	n := nik.String
	req.PIC.NIK = &n
	return nil
}

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

// ikatJamaahAgen mengunci jamaah hasil booking portal agen. Jamaah yang NIK-nya sama dengan NIK agen adalah diri agen itu sendiri:
// agen tidak memiliki dirinya, jadi pemilik kuncinya upline aktif agen (sama dengan penerima komisi langsungnya). Agen tanpa upline
// aktif: jamaah itu tidak dikunci ke siapa pun. Jamaah lain dikunci ke agen seperti biasa. First-set wins berlaku di IkatJamaah.
func ikatJamaahAgen(ctx context.Context, tx *sql.Tx, jamaahIDs []int64, agenID int64) error {
	for _, id := range jamaahIDs {
		var uplineAktif int64
		var diri bool
		if err := tx.QueryRowContext(ctx, `
			SELECT COALESCE(u.id, 0), (a.nik_agen IS NOT NULL AND a.nik_agen = j.nik)
			FROM customer_accounts a JOIN jamaah j ON j.id = ?
			LEFT JOIN customer_accounts u ON u.id = a.upline_akun_id AND u.status_agen = 'aktif'
			WHERE a.id = ?`, id, agenID).Scan(&uplineAktif, &diri); err != nil {
			return fmt.Errorf("selfbooking: baca diri agen: %w", err)
		}
		pemilik := agenID
		if diri {
			pemilik = uplineAktif
		}
		if err := agen.IkatJamaah(ctx, tx, id, pemilik, agen.SumberJalur1); err != nil {
			return err
		}
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
