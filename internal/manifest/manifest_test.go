package manifest

import (
	"slices"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestTanggalManifest(t *testing.T) {
	for in, want := range map[string]string{
		"2026-09-14":          "14-SEP-2026",
		"1968-02-29":          "29-FEB-1968",
		"2026-12-01T00:00:00": "01-DEC-2026",
		"":                    "",
		"14/09/2026":          "",
	} {
		if got := tanggalManifest(in); got != want {
			t.Errorf("tanggalManifest(%q) = %q, mau %q", in, got, want)
		}
	}
}

func TestJenisTiketDariUsiaSaatBerangkat(t *testing.T) {
	cases := []struct {
		lahir, tipe, want string
	}{
		{"2024-09-15", "infant", "INFANT"}, // 1 tahun 364 hari
		{"2024-09-14", "reguler", "CHILD"}, // tepat 2 tahun
		{"2014-09-15", "reguler", "CHILD"}, // 11 tahun
		{"2014-09-14", "reguler", "ADULT"}, // tepat 12 tahun
		{"", "infant", "INFANT"},
		{"", "reguler", "ADULT"},
	}
	for _, c := range cases {
		p := Pax{TanggalLahir: c.lahir, TipePax: c.tipe, NoPaspor: "X1", PasporHabis: "2030-01-01", JenisKelamin: "L"}
		periksaPax(&p, "2026-09-14", "2026-09-22")
		if p.JenisTiket != c.want {
			t.Errorf("lahir %q tipe %s: %s, mau %s", c.lahir, c.tipe, p.JenisTiket, c.want)
		}
	}
}

func TestPeriksaPax(t *testing.T) {
	lengkap := Pax{NoPaspor: "X9457805", PasporTerbit: "2026-06-18", PasporHabis: "2036-06-18", JenisKelamin: "L", TanggalLahir: "1968-02-29"}
	periksaPax(&lengkap, "2026-09-14", "2026-09-22")
	if len(lengkap.Masalah) != 0 {
		t.Fatalf("pax lengkap tidak boleh bermasalah: %v", lengkap.Masalah)
	}
	if lengkap.Usia == nil || *lengkap.Usia != 58 {
		t.Errorf("usia saat berangkat = %v, mau 58", lengkap.Usia)
	}

	cases := map[string]struct {
		p    Pax
		want string
	}{
		"paspor kosong":         {Pax{PasporHabis: "2036-01-01", JenisKelamin: "P", TanggalLahir: "1990-01-01"}, "Nomor paspor kosong"},
		"masa berlaku kosong":   {Pax{NoPaspor: "X1", JenisKelamin: "P", TanggalLahir: "1990-01-01"}, "Masa berlaku paspor kosong"},
		"terbit sama dgn habis": {Pax{NoPaspor: "X1", PasporTerbit: "2026-08-03", PasporHabis: "2026-08-03", JenisKelamin: "P", TanggalLahir: "1960-08-05"}, "Tanggal paspor tidak sesuai"},
		"habis < 6 bln pulang":  {Pax{NoPaspor: "X1", PasporHabis: "2027-03-21", JenisKelamin: "P", TanggalLahir: "1990-01-01"}, "Paspor habis kurang dari 6 bulan"},
		"jenis kelamin kosong":  {Pax{NoPaspor: "X1", PasporHabis: "2036-01-01", TanggalLahir: "1990-01-01"}, "Jenis kelamin kosong"},
		"tanggal lahir kosong":  {Pax{NoPaspor: "X1", PasporHabis: "2036-01-01", JenisKelamin: "L"}, "Tanggal lahir kosong"},
	}
	for nama, c := range cases {
		p := c.p
		periksaPax(&p, "2026-09-14", "2026-09-22")
		if !slices.Contains(p.Masalah, c.want) {
			t.Errorf("%s: masalah %v, mau memuat %q", nama, p.Masalah, c.want)
		}
	}

	// Tepat 6 bulan setelah tanggal pulang masih boleh.
	batas := Pax{NoPaspor: "X1", PasporHabis: "2027-03-22", JenisKelamin: "L", TanggalLahir: "1990-01-01"}
	periksaPax(&batas, "2026-09-14", "2026-09-22")
	if len(batas.Masalah) != 0 {
		t.Errorf("paspor habis tepat 6 bulan setelah pulang ditandai: %v", batas.Masalah)
	}
}

func TestLamaHariDanNama(t *testing.T) {
	if got := lamaHari("2026-09-14", "2026-09-22"); got != 9 {
		t.Errorf("lamaHari = %d, mau 9", got)
	}
	if got := namaPaspor(Pax{Nama: "Sunarti  Joyoatmo"}); got != "SUNARTI JOYOATMO" {
		t.Errorf("fallback nama lengkap = %q", got)
	}
	if got := namaPaspor(Pax{Nama: "Sunarti", NamaPaspor: "SUNARTI JOYOATMO"}); got != "SUNARTI JOYOATMO" {
		t.Errorf("nama paspor = %q", got)
	}
	p := Paket{Nama: `Fantastic *5 Combine / "SV"`, BerangkatTanggal: "2026-09-14"}
	if got := NamaFile(p); got != "14 SEP 2026 FANTASTIC *5 COMBINE SV.xlsx" {
		t.Errorf("NamaFile = %q", got)
	}
	if got := judulManifest(Paket{Nama: "Fantastic *5 Combine", BerangkatTanggal: "2026-09-14"}); got != "MANIFEST GROUP 14 SEPTEMBER 2026 FANTASTIC *5 COMBINE" {
		t.Errorf("judul = %q", got)
	}
}

func TestRapikanInput(t *testing.T) {
	in := InputInfo{PNR: " 97bc5w ", DetailPenerbangan: "SV 827 14SEP 1 CGKJED 0040 0610\r\n\r\n  SV 826  22SEP 2 JEDCGK 0840 2245  "}
	if err := in.rapikan(); err != nil {
		t.Fatal(err)
	}
	if in.PNR != "97BC5W" {
		t.Errorf("PNR = %q", in.PNR)
	}
	if want := "SV 827 14SEP 1 CGKJED 0040 0610\nSV 826 22SEP 2 JEDCGK 0840 2245"; in.DetailPenerbangan != want {
		t.Errorf("detail = %q", in.DetailPenerbangan)
	}
	panjang := InputInfo{PNR: strings.Repeat("A", maksPNR+1)}
	if err := panjang.rapikan(); err == nil {
		t.Error("PNR terlalu panjang harus ditolak")
	}
}

func TestBuatFile(t *testing.T) {
	m := &Manifest{
		Paket: Paket{Nama: "Fantastic *5 Combine", BerangkatTanggal: "2026-09-14", PulangTanggal: "2026-09-22",
			Maskapai: "Saudia Airlines", KodeMaskapai: "SV", PNR: "97BC5W",
			DetailPenerbangan: "SV 827 14SEP 1 CGKJED 0040 0610\nSV 826 22SEP 2 JEDCGK 0840 2245"},
		Pax: []Pax{
			{Nama: "Sutarmo Suwardi", JenisKelamin: "L", TempatLahir: "Purworejo", TanggalLahir: "1968-02-29", NIK: "3603192902680002",
				NoPaspor: "x9457805", PasporTerbit: "2026-06-18", PasporHabis: "2036-06-18", KantorPaspor: "Tangerang", TipePax: "reguler"},
			{Nama: "Rini", NamaPaspor: "RINI SULISTYANI", JenisKelamin: "P", TanggalLahir: "1971-09-10", TipePax: "reguler"},
		},
	}
	m.LamaHari = lamaHari(m.Paket.BerangkatTanggal, m.Paket.PulangTanggal)
	for i := range m.Pax {
		periksaPax(&m.Pax[i], m.Paket.BerangkatTanggal, m.Paket.PulangTanggal)
	}
	f, err := BuatFile(m)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	if got, want := f.GetSheetList(), []string{"BLOK SEAT SAUDIA", "MANIFEST", "VISA", "ASURANSI"}; !slices.Equal(got, want) {
		t.Fatalf("sheet = %v, mau %v", got, want)
	}
	cek := func(sheet, cell, want string) {
		t.Helper()
		got, err := f.GetCellValue(sheet, cell)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s!%s = %q, mau %q", sheet, cell, got, want)
		}
	}
	cek("MANIFEST", "A1", "MANIFEST GROUP 14 SEPTEMBER 2026 FANTASTIC *5 COMBINE")
	cek("MANIFEST", "A3", "Flight Carrier: SAUDIA AIRLINES (SV)")
	cek("MANIFEST", "A4", "Date of Departure: 14-SEP-2026")
	cek("MANIFEST", "A5", "Pax Number: 2 PAX")
	cek("MANIFEST", "A6", "Package: 09 DAYS")
	cek("VISA", "A4", "Date of Departure: 14-SEP-2026")
	// Blok kanan: cari kolom PNR di baris 3, baris penerbangan di kolom yang sama.
	kanan := ""
	for c := 2; c <= 11; c++ {
		if v, _ := f.GetCellValue("MANIFEST", sel(c, 3)); v == "PNR Code: 97BC5W" {
			kanan, _ = excelize.ColumnNumberToName(c)
		}
	}
	if kanan == "" {
		t.Fatal("PNR Code tidak ditemukan di baris 3")
	}
	cek("MANIFEST", kanan+"4", "Flight Detail: SV 827 14SEP 1 CGKJED 0040 0610")
	cek("MANIFEST", kanan+"5", "Flight Detail: SV 826 22SEP 2 JEDCGK 0840 2245")
	cek("MANIFEST", "A8", "NO")
	cek("MANIFEST", "B9", "SUTARMO SUWARDI")
	cek("MANIFEST", "E9", "29-FEB-1968")
	cek("MANIFEST", "F9", "X9457805")
	cek("MANIFEST", "K9", "ADULT")
	cek("MANIFEST", "B10", "RINI SULISTYANI")
	cek("VISA", "B9", "X9457805")
	cek("VISA", "C9", "SUTARMO SUWARDI")
	cek("VISA", "E9", "")
	cek("ASURANSI", "D9", "58")
	cek("ASURANSI", "E9", "3603192902680002")
	cek("BLOK SEAT SAUDIA", "K8", "KETERANGAN")
}
