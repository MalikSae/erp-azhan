package schedule

import "testing"

// Code review Paket dan master, 11 November 2026: nominal paket harus bilangan bulat rupiah (ADR-0001).
func TestPesanNominalPaketBulat(t *testing.T) {
	bulat, pecahan, nol := 25_000_000.0, 25_000_000.6, 0.0
	if msg := pesanNominalPaketBulat(nominalPaket{"harga_quad", &bulat}, nominalPaket{"harga_infant", nil}, nominalPaket{"nominal_komisi_langsung", &nol}); msg != "" {
		t.Fatalf("nominal sah ditolak: %s", msg)
	}
	msg := pesanNominalPaketBulat(nominalPaket{"harga_quad", &bulat}, nominalPaket{"harga_triple", &pecahan})
	if msg != "harga_triple harus bilangan bulat rupiah" {
		t.Fatalf("pesan = %q", msg)
	}
}

func TestValidateDPMenolakPecahan(t *testing.T) {
	pecahan, bulat := 1_500_000.5, 1_500_000.0
	if err := validateDP(&pecahan, 25_000_000); err == nil {
		t.Fatal("minimal_dp pecahan harus ditolak")
	}
	if err := validateDP(&bulat, 25_000_000); err != nil {
		t.Fatalf("minimal_dp bulat ditolak: %v", err)
	}
	if err := validateDP(nil, 25_000_000); err != nil {
		t.Fatalf("minimal_dp kosong ditolak: %v", err)
	}
}

func TestAlasanUbahTanggalBatasPanjang(t *testing.T) {
	pas := make([]rune, maxAlasanTanggal)
	for i := range pas {
		pas[i] = 'a'
	}
	if err := validateMaxLen("alasan_ubah_tanggal", string(pas), maxAlasanTanggal); err != nil {
		t.Fatalf("tepat di batas ditolak: %v", err)
	}
	if err := validateMaxLen("alasan_ubah_tanggal", string(pas)+"a", maxAlasanTanggal); err == nil {
		t.Fatal("melebihi batas harus ditolak")
	}
}

func TestAlasanUbahKuota(t *testing.T) {
	if got := alasanUbahKuota(45, 50); got != "Ubah kuota seat lewat edit paket: 45 menjadi 50" {
		t.Fatalf("dapat %q", got)
	}
	if len([]rune(alasanUbahKuota(99999, 100000))) > maxAlasanSeat {
		t.Fatal("keterangan melebihi lebar kolom reason")
	}
}

// Paket hanya boleh dihapus bila draft dan belum pernah tayang; mengembalikan paket tayang ke draft tidak
// membukanya untuk dihapus.
func TestTolakHapusPaket(t *testing.T) {
	cases := []struct {
		status       string
		pernahTayang bool
		tolak        bool
	}{
		{StatusDraft, false, false},
		{StatusDraft, true, true},
		{StatusPublished, true, true},
		{StatusArchived, true, true},
		{StatusArchived, false, true},
	}
	for _, c := range cases {
		if got := tolakHapusPaket(c.status, c.pernahTayang) != ""; got != c.tolak {
			t.Errorf("status %s pernah tayang %v: tolak=%v, mau %v", c.status, c.pernahTayang, got, c.tolak)
		}
	}
}
