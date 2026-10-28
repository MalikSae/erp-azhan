package customer

import "testing"

func TestKelengkapanDataJamaah(t *testing.T) {
	d := dataJamaah{PaxType: "reguler", TempatLahir: "Depok", TanggalLahir: "1990-01-01", NoHP: "0812"}
	d.hitungKelengkapan()
	if d.Total != 10 || d.Terisi != 3 || len(d.Kurang) != 7 {
		t.Fatalf("reguler: terisi %d dari %d, kurang %v", d.Terisi, d.Total, d.Kurang)
	}
	inf := dataJamaah{PaxType: "infant"}
	inf.hitungKelengkapan()
	if inf.Total != 8 {
		t.Fatalf("infant tanpa kontak darurat: total %d, want 8", inf.Total)
	}
}

func TestNIKTersamar(t *testing.T) {
	if got := nikTersamar("3171010101900001"); got != "••••••••••••0001" {
		t.Fatalf("got %q", got)
	}
	if nikTersamar("12") != "" || nikTersamar("") != "" {
		t.Fatal("NIK pendek tidak boleh ditampilkan")
	}
}

func TestRamahPesanData(t *testing.T) {
	cases := map[string]string{
		"tanggal_lahir tidak boleh di masa depan":                         "Tanggal lahir tidak boleh di masa depan",
		"paspor_berlaku_sampai tidak boleh sebelum tanggal_paspor_keluar": "Masa berlaku paspor tidak boleh sebelum Tanggal paspor dikeluarkan",
		"no_hp harus berisi 10–15 digit angka":                            "Nomor WhatsApp harus berisi 10–15 digit angka",
		"no HP kontak darurat harus berisi 10–15 digit angka":             "Nomor telepon kontak darurat harus berisi 10–15 digit angka",
	}
	for in, want := range cases {
		if got := ramahPesanData(in); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestPeriksaFormatData(t *testing.T) {
	ok := simpanDataJamaahRequest{NamaAyahKandung: "Budi O'Brien-Lee", EmergencyNama: "Siti Aminah, S.Pd.", EmergencyHubungan: "Istri", NoPaspor: "X1234567", NoAsuransiBPJS: "0001234567"}
	if msg := periksaFormatData(&ok); msg != "" {
		t.Fatalf("data sah ditolak: %s", msg)
	}
	if msg := periksaFormatData(&simpanDataJamaahRequest{}); msg != "" {
		t.Fatalf("isian kosong harus lolos: %s", msg)
	}
	buruk := []simpanDataJamaahRequest{
		{EmergencyNama: "00a99s7a7"},
		{NamaAyahKandung: "Budi123"},
		{EmergencyHubungan: "1"},
		{NoPaspor: "X12-34"},
		{NoPaspor: "AB"},
		{NoAsuransiBPJS: "12 34"},
	}
	for i, b := range buruk {
		if periksaFormatData(&b) == "" {
			t.Errorf("kasus %d lolos padahal salah: %+v", i, b)
		}
	}
}

func TestHitungDokumenLengkap(t *testing.T) {
	p := progressPax{PaxType: "reguler"}
	p.hitungDokumenLengkap(map[string]bool{"paspor": true, "ktp": true})
	if p.DokumenLengkap || p.DokumenSetuju != 2 || p.DokumenWajib != 3 {
		t.Fatalf("belum lengkap: %+v", p)
	}
	p.hitungDokumenLengkap(map[string]bool{"paspor": true, "ktp": true, "pas_foto": true, "kk": true})
	if !p.DokumenLengkap || p.DokumenSetuju != 3 {
		t.Fatalf("lengkap (dokumen opsional tidak dihitung): %+v", p)
	}
	inf := progressPax{PaxType: "infant"}
	inf.hitungDokumenLengkap(map[string]bool{"paspor": true, "akte_lahir": true, "pas_foto": true})
	if !inf.DokumenLengkap {
		t.Fatalf("infant lengkap: %+v", inf)
	}
	kosong := progressPax{PaxType: "reguler"}
	kosong.hitungDokumenLengkap(nil)
	if kosong.DokumenLengkap {
		t.Fatal("tanpa dokumen tidak boleh lengkap")
	}
}
