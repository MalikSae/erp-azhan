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

	pertama, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Pemilik NIK", JenisKelamin: "L", NIK: &nik})
	if err != nil {
		t.Fatal(err)
	}
	var tersimpan sql.NullString
	if err := tx.QueryRow(`SELECT nik FROM jamaah WHERE id=?`, pertama).Scan(&tersimpan); err != nil || tersimpan.String != nik {
		t.Fatalf("NIK tersimpan = %q (err %v), want %s", tersimpan.String, err, nik)
	}

	ulang, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "  pemilik   nik ", JenisKelamin: "L", NIK: &nik})
	if err != nil || ulang != pertama {
		t.Fatalf("nama sama harus memakai ulang jamaah %d, dapat %d (err %v)", pertama, ulang, err)
	}

	_, err = resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Orang Lain", JenisKelamin: "L", NIK: &nik, Nomor: 2})
	if !errors.Is(err, ErrNIKAtasNamaLain) {
		t.Fatalf("nama beda: err=%v, want ErrNIKAtasNamaLain", err)
	}
	// Pesan generik: menyebut nomor jamaah di form, tidak membuka nama terdaftar maupun keberadaan NIK.
	if got := err.Error(); !strings.Contains(got, "Jamaah 2") || strings.Contains(got, "Pemilik NIK") || strings.Contains(strings.ToLower(got), "terdaftar") {
		t.Fatalf("pesan = %q: harus menyebut Jamaah 2 tanpa membuka keberadaan data jamaah lain", got)
	}

	// Nomor WA bukan kunci identitas (spek 4.3): NIK berbeda dengan nomor yang sama menghasilkan jamaah berbeda.
	hp := nomorUji(990)
	nikBaru := "3174010101018888"
	nikLain := "3174010101017777"
	satu, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Berbagi Nomor Satu", NoHP: &hp, JenisKelamin: "P", NIK: &nikBaru})
	if err != nil {
		t.Fatal(err)
	}
	dua, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Berbagi Nomor Dua", NoHP: &hp, JenisKelamin: "L", NIK: &nikLain})
	if err != nil || dua == satu {
		t.Fatalf("dua NIK dengan nomor sama: id1=%d id2=%d err=%v, want dua jamaah", satu, dua, err)
	}
}

// Regresi audit keamanan 2026-11-12: NIK yang belum terdaftar tidak boleh mengklaim jamaah lama lewat
// nama + tanggal lahir. Dulu NIK acak + nama + tanggal lahir korban mengembalikan jamaah korban, lalu
// jamaah itu terikat ke agen penyerang dan NIK-nya bisa diisi NIK palsu.
func TestResolveAnggotaPublikNIKBaruTidakMengklaimJamaahLama(t *testing.T) {
	tx, _, brandID, _ := setupAgenBooking(t)
	ctx := context.Background()
	lahir := "1980-05-17"
	nikKorban := "3174010101016666"
	nikPenyerang := "3174010101015555"

	// Korban ber-NIK: NIK lain dengan nama + tanggal lahir sama menghasilkan jamaah baru.
	korban, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Korban Ber NIK", JenisKelamin: "L", NIK: &nikKorban, TanggalLahir: &lahir})
	if err != nil {
		t.Fatal(err)
	}
	klaim, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "korban ber nik", JenisKelamin: "L", NIK: &nikPenyerang, TanggalLahir: &lahir})
	if err != nil {
		t.Fatal(err)
	}
	if klaim == korban {
		t.Fatalf("NIK baru + nama + tanggal lahir korban mengembalikan jamaah korban %d", korban)
	}
	// Tanpa NIK pun jamaah ber-NIK tidak boleh cocok lewat nama + tanggal lahir.
	tanpaNIK, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Korban Ber NIK", JenisKelamin: "L", TanggalLahir: &lahir})
	if err != nil {
		t.Fatal(err)
	}
	if tanpaNIK == korban {
		t.Fatalf("pax tanpa NIK cocok ke jamaah ber-NIK %d", korban)
	}

	// Korban tanpa NIK (mis. impor CRM): NIK baru tidak mengklaim dan tidak mengisi NIK korban.
	lahir2 := "1975-02-03"
	korban2, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Korban Tanpa NIK", JenisKelamin: "P", TanggalLahir: &lahir2})
	if err != nil {
		t.Fatal(err)
	}
	nikPenyerang2 := "3174010101014444"
	klaim2, err := resolveAnggotaPublik(ctx, tx, brandID, 0, AnggotaInput{PaxType: "reguler", NamaLengkap: "Korban Tanpa NIK", JenisKelamin: "P", NIK: &nikPenyerang2, TanggalLahir: &lahir2})
	if err != nil {
		t.Fatal(err)
	}
	if klaim2 == korban2 {
		t.Fatalf("NIK baru mengklaim jamaah tanpa NIK %d", korban2)
	}
	var nik sql.NullString
	if err := tx.QueryRow(`SELECT nik FROM jamaah WHERE id=?`, korban2).Scan(&nik); err != nil || nik.String != "" {
		t.Fatalf("NIK korban tanpa NIK berubah menjadi %q (err %v)", nik.String, err)
	}
}
