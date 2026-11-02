package surat

import (
	"testing"
	"time"
)

func TestFormatNomor(t *testing.T) {
	terbit := time.Date(2026, time.November, 2, 10, 0, 0, 0, time.UTC)
	if got := FormatNomor(100808, "haw", JenisKeteranganUmroh, terbit); got != "100808/HAW/SKU/XI/2026" {
		t.Fatalf("nomor SKU: %q", got)
	}
	if got := FormatNomor(100809, " HAW ", JenisRekomPaspor, terbit); got != "100809/HAW/SRP/XI/2026" {
		t.Fatalf("nomor SRP: %q", got)
	}
}

func TestTanggalIndonesia(t *testing.T) {
	if got := TanggalIndonesia(time.Date(2026, time.September, 11, 0, 0, 0, 0, time.UTC)); got != "11 September 2026" {
		t.Fatalf("tanggal: %q", got)
	}
}

func TestKotaSurat(t *testing.T) {
	cases := map[string]string{"Kota Depok": "Depok", "Kabupaten Bogor": "Bogor", "Jakarta Timur": "Jakarta Timur", "": "", "kota depok": "depok"}
	for in, want := range cases {
		if got := KotaSurat(in); got != want {
			t.Errorf("KotaSurat(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNamaBerkas(t *testing.T) {
	if got := NamaBerkas(JenisKeteranganUmroh, "Syahid  Malik, S.Kom"); got != "Surat-Keterangan-Umroh-Syahid-Malik-SKom.pdf" {
		t.Fatalf("nama berkas: %q", got)
	}
	if got := NamaBerkas(JenisRekomPaspor, "###"); got != "Surat-Rekomendasi-Paspor-jamaah.pdf" {
		t.Fatalf("nama berkas kosong: %q", got)
	}
}

func TestValidasiPengaturan(t *testing.T) {
	p := Pengaturan{KodeSurat: " haw ", PenandatanganNama: " Fikry Shahab ", KopURL: "/uploads/brand-logos/abc-1.png"}
	if err := p.Validasi(); err != nil || p.KodeSurat != "HAW" || p.PenandatanganNama != "Fikry Shahab" {
		t.Fatalf("valid: %v %+v", err, p)
	}
	for _, buruk := range []Pengaturan{
		{KodeSurat: "HAW/1"},
		{KopURL: "/uploads/dokumen-jamaah/x.png"},
		{StempelURL: "https://contoh.com/x.png"},
		{TandaTanganURL: "/uploads/brand-logos/../x.png"},
	} {
		if err := buruk.Validasi(); err == nil {
			t.Errorf("harus ditolak: %+v", buruk)
		}
	}
}
