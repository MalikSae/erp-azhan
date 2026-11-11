package customer

import (
	"strings"
	"testing"

	"erp-azhan/api/internal/payment"
)

// T-03: pembayaran void tampil ke pelanggan sebagai ditolak, tanpa alasan internal admin.
func TestPembayaranUntukPelanggan(t *testing.T) {
	alasan := "salah booking, seharusnya ASAB12"
	pelaku := int64(7)
	out := pembayaranUntukPelanggan([]payment.Payment{
		{ID: 1, Status: "confirmed"},
		{ID: 2, Status: "void", VoidReason: &alasan, VoidedBy: &pelaku},
	})
	if len(out) != 2 || out[0].Status != "confirmed" || out[0].RejectionReason != nil {
		t.Fatalf("pembayaran biasa berubah: %+v", out)
	}
	v := out[1]
	if v.Status != "rejected" || v.RejectionReason == nil || v.VoidReason != nil || v.VoidedBy != nil {
		t.Fatalf("void = %+v", v)
	}
	catatan, penerima := "titipan dari kantor cabang", "Staf Kasir"
	bersih := pembayaranUntukPelanggan([]payment.Payment{{ID: 3, Status: payment.StatusConfirmed, Notes: &catatan, ReceivedBy: &penerima, VerifiedBy: &pelaku}})[0]
	if bersih.Notes != nil || bersih.ReceivedBy != nil || bersih.VerifiedBy != nil {
		t.Fatalf("kolom kerja admin ikut terkirim: %+v", bersih)
	}
	if got := pembayaranUntukPelanggan(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil harus menjadi daftar kosong: %#v", got)
	}
}

// Code review endpoint publik, 11 November 2026: isian pengirim dipangkas dan dibatasi selebar kolom.
func TestRapikanPengirim(t *testing.T) {
	nama, bank := "  Budi Santoso ", "   "
	req := payment.CreatePaymentRequest{SenderName: &nama, SenderBank: &bank}
	if msg := rapikanPengirim(&req); msg != "" || *req.SenderName != "Budi Santoso" || req.SenderBank != nil {
		t.Fatalf("pesan %q, nama %q, bank %v", msg, *req.SenderName, req.SenderBank)
	}
	panjang := strings.Repeat("a", maksNamaPengirim+1)
	if msg := rapikanPengirim(&payment.CreatePaymentRequest{SenderName: &panjang}); msg == "" {
		t.Fatal("nama pengirim melebihi batas harus ditolak")
	}
	pas, bankPanjang := strings.Repeat("é", maksNamaPengirim), strings.Repeat("b", maksBankPengirim+1)
	if msg := rapikanPengirim(&payment.CreatePaymentRequest{SenderName: &pas}); msg != "" {
		t.Fatalf("nama tepat di batas ditolak: %s", msg)
	}
	if msg := rapikanPengirim(&payment.CreatePaymentRequest{SenderName: &pas, SenderBank: &bankPanjang}); msg == "" {
		t.Fatal("bank pengirim melebihi batas harus ditolak")
	}
}

// Konfirmasi yang masih menunggu verifikasi ikut dihitung terhadap sisa tagihan.
func TestPesanMelebihiMenunggu(t *testing.T) {
	cases := []struct {
		jumlah, menunggu, sisa float64
		tolak                  bool
	}{
		{5_000_000, 0, 5_000_000, false},
		{5_000_000, 0, 4_000_000, false}, // tanpa yang menunggu, batas sisa diperiksa di tempat lain
		{2_000_000, 3_000_000, 5_000_000, false},
		{2_000_001, 3_000_000, 5_000_000, true},
		{1, 5_000_000, 5_000_000, true},
	}
	for _, c := range cases {
		if got := pesanMelebihiMenunggu(c.jumlah, c.menunggu, c.sisa) != ""; got != c.tolak {
			t.Errorf("jumlah %v menunggu %v sisa %v: tolak=%v, mau %v", c.jumlah, c.menunggu, c.sisa, got, c.tolak)
		}
	}
}

func TestNIKTersamarUntukTravelers(t *testing.T) {
	if got := nikTersamar("3201234567890123"); got != strings.Repeat("•", 12)+"0123" {
		t.Fatalf("dapat %q", got)
	}
	if got := nikTersamar(""); got != "" {
		t.Fatalf("NIK kosong harus kosong, dapat %q", got)
	}
}

// Keputusan 11 November 2026: konfirmasi portal wajib menutup kekurangan DP sekaligus.
func TestPesanKurangDP(t *testing.T) {
	cases := []struct {
		nama                                 string
		jumlah, minimalDP, terbayar, menunggu float64
		tolak                                bool
	}{
		{"belum bayar, di bawah DP", 2_000_000, 5_000_000, 0, 0, true},
		{"belum bayar, tepat DP", 5_000_000, 5_000_000, 0, 0, false},
		{"belum bayar, di atas DP", 7_000_000, 5_000_000, 0, 0, false},
		{"cicilan admin 2 juta, kurang 3 juta, kirim 1 juta", 1_000_000, 5_000_000, 2_000_000, 0, true},
		{"cicilan admin 2 juta, kirim tepat kekurangan", 3_000_000, 5_000_000, 2_000_000, 0, false},
		{"DP sudah tercapai, nominal bebas", 1, 5_000_000, 5_000_000, 0, false},
		{"DP tertutup konfirmasi yang menunggu", 1, 5_000_000, 0, 5_000_000, false},
		{"menunggu belum menutup DP", 1, 5_000_000, 0, 4_000_000, true},
		{"minimal DP nol", 1, 0, 0, 0, false},
	}
	for _, c := range cases {
		if got := pesanKurangDP(c.jumlah, c.minimalDP, c.terbayar, c.menunggu) != ""; got != c.tolak {
			t.Errorf("%s: tolak=%v, mau %v", c.nama, got, c.tolak)
		}
	}
	if msg := pesanKurangDP(1, 5_000_000, 2_000_000, 0); !strings.Contains(msg, "Rp3.000.000") {
		t.Errorf("pesan harus menyebut kekurangan: %q", msg)
	}
}

func TestRupiahTeks(t *testing.T) {
	for v, mau := range map[float64]string{0: "Rp0", 999: "Rp999", 1000: "Rp1.000", 5_000_000: "Rp5.000.000", 116_200_000: "Rp116.200.000"} {
		if got := rupiahTeks(v); got != mau {
			t.Errorf("rupiahTeks(%v) = %q, mau %q", v, got, mau)
		}
	}
}
