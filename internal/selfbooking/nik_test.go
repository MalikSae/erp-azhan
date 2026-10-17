package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

func TestValidateCheckoutNIK(t *testing.T) {
	total, dp := 1000.0, 100.0
	dasar := func() BookingRequest {
		return BookingRequest{TermsAccepted: true, TermsVersion: TermsVersion, RequestKey: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			ExpectedTotal: &total, ExpectedDP: &dp,
			PIC: PICInput{NamaLengkap: "Test PIC", NoHP: "081234567890", JenisKelamin: "L", NIK: nikp("3174010101010001")}}
	}
	if msg := validateCheckout(ptrReq(dasar()), false); msg != "" {
		t.Fatalf("NIK 16 digit harus lolos: %q", msg)
	}
	r := dasar()
	r.PIC.NIK = nikp("12345")
	if msg := validateCheckout(&r, false); msg == "" {
		t.Fatal("NIK pendek harus ditolak")
	}
	r = dasar()
	r.PIC.NIK = nil
	if msg := validateCheckout(&r, false); msg == "" {
		t.Fatal("NIK kosong untuk jamaah 1 harus ditolak")
	}
	r = dasar()
	r.Anggota = []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Anggota Dua", JenisKelamin: "P"}}
	if msg := validateCheckout(&r, false); msg == "" {
		t.Fatal("NIK kosong untuk anggota reguler harus ditolak")
	}
	r.Anggota = []AnggotaInput{{PaxType: "infant", NamaLengkap: "Bayi", JenisKelamin: "P"}}
	if msg := validateCheckout(&r, false); msg != "" {
		t.Fatalf("NIK infant opsional: %q", msg)
	}
	r.Anggota = []AnggotaInput{{PaxType: "infant", NamaLengkap: "Bayi", JenisKelamin: "P", NIK: nikp("abc")}}
	if msg := validateCheckout(&r, false); msg == "" {
		t.Fatal("NIK infant yang diisi harus 16 digit")
	}
}

func ptrReq(r BookingRequest) *BookingRequest { return &r }

// NIK sudah terdaftar: nama sama dipakai ulang, nama beda ditolak, NIK jamaah lama yang kosong diisi.
func TestResolveAnggotaPublikNIK(t *testing.T) {
	tx, _, brandID, _ := setupAgenBooking(t)
	ctx := context.Background()
	nik := "3174010101019999"

	pertama, err := resolveAnggotaPublik(ctx, tx, brandID, AnggotaInput{PaxType: "reguler", NamaLengkap: "Pemilik NIK", JenisKelamin: "L", NIK: &nik})
	if err != nil {
		t.Fatal(err)
	}
	var tersimpan sql.NullString
	if err := tx.QueryRow(`SELECT nik FROM jamaah WHERE id=?`, pertama).Scan(&tersimpan); err != nil || tersimpan.String != nik {
		t.Fatalf("NIK tersimpan = %q (err %v), want %s", tersimpan.String, err, nik)
	}

	ulang, err := resolveAnggotaPublik(ctx, tx, brandID, AnggotaInput{PaxType: "reguler", NamaLengkap: "  pemilik   nik ", JenisKelamin: "L", NIK: &nik})
	if err != nil || ulang != pertama {
		t.Fatalf("nama sama harus memakai ulang jamaah %d, dapat %d (err %v)", pertama, ulang, err)
	}

	_, err = resolveAnggotaPublik(ctx, tx, brandID, AnggotaInput{PaxType: "reguler", NamaLengkap: "Orang Lain", JenisKelamin: "L", NIK: &nik, Nomor: 2})
	if !errors.Is(err, ErrNIKAtasNamaLain) {
		t.Fatalf("nama beda: err=%v, want ErrNIKAtasNamaLain", err)
	}
	if got := err.Error(); !strings.HasPrefix(got, "NIK Jamaah 2 sudah terdaftar.") || strings.Contains(got, "Pemilik NIK") {
		t.Fatalf("pesan = %q: harus menyebut Jamaah 2 dan tidak membuka nama yang terdaftar", got)
	}

	// Jamaah lama (dikenali lewat nomor) tanpa NIK: NIK diisikan, tidak menimpa NIK yang ada.
	hp := nomorUji(990)
	lama, err := resolveAnggotaPublik(ctx, tx, brandID, AnggotaInput{PaxType: "reguler", NamaLengkap: "Jamaah Lama", NoHP: &hp, JenisKelamin: "P"})
	if err != nil {
		t.Fatal(err)
	}
	nikBaru := "3174010101018888"
	if _, err := resolveAnggotaPublik(ctx, tx, brandID, AnggotaInput{PaxType: "reguler", NamaLengkap: "Jamaah Lama", NoHP: &hp, JenisKelamin: "P", NIK: &nikBaru}); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(`SELECT nik FROM jamaah WHERE id=?`, lama).Scan(&tersimpan); err != nil || tersimpan.String != nikBaru {
		t.Fatalf("NIK jamaah lama = %q (err %v), want %s", tersimpan.String, err, nikBaru)
	}
	nikLain := "3174010101017777"
	if _, err := resolveAnggotaPublik(ctx, tx, brandID, AnggotaInput{PaxType: "reguler", NamaLengkap: "Jamaah Lama", NoHP: &hp, JenisKelamin: "P", NIK: &nikLain}); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(`SELECT nik FROM jamaah WHERE id=?`, lama).Scan(&tersimpan); err != nil || tersimpan.String != nikBaru {
		t.Fatalf("NIK yang sudah ada tidak boleh ditimpa: %q", tersimpan.String)
	}
}
