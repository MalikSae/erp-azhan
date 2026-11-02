package surat

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func dataUji(jenis string) Data {
	return Data{
		Jenis:    jenis,
		Nomor:    FormatNomor(100808, "HAW", jenis, time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC)),
		TerbitAt: time.Date(2026, time.November, 2, 0, 0, 0, 0, time.UTC),
		Pengaturan: Pengaturan{
			NamaPerusahaan: "PT. Hana Asia Wisata", KodeSurat: "HAW", PenandatanganNama: "Fikry Shahab", PenandatanganJabatan: "Direktur Utama",
		},
		Merek: Merek{
			Nama: "Hana Tours & Travel", Alamat: "Jl. Margonda Raya No.274, Kemiri Muka, Kecamatan Beji, Kota Depok, Jawa Barat 16424",
			Kota: "Kota Depok", WhatsApp: "+6281211829993", Email: "cs@hanatours.id", PPIU: "No. 484/2020", PIHK: "No. 395/2021",
		},
		Jamaah:    Jamaah{Nama: "Syahid Malik", TempatLahir: "Jakarta", TanggalLahir: time.Date(1990, time.January, 12, 0, 0, 0, 0, time.UTC), Kota: "Kota Depok"},
		Berangkat: time.Date(2026, time.December, 19, 0, 0, 0, 0, time.UTC),
		Pulang:    time.Date(2026, time.December, 28, 0, 0, 0, 0, time.UTC),
	}
}

func TestRenderMenghasilkanPDF(t *testing.T) {
	for _, jenis := range []string{JenisKeteranganUmroh, JenisRekomPaspor} {
		out, err := Render(dataUji(jenis))
		if err != nil {
			t.Fatalf("%s: %v", jenis, err)
		}
		if !bytes.HasPrefix(out, []byte("%PDF-")) || len(out) < 1500 {
			t.Fatalf("%s: bukan PDF yang wajar (%d byte)", jenis, len(out))
		}
		if n := bytes.Count(out, []byte("/Type /Page\n")); n != 1 {
			t.Fatalf("%s: harus satu halaman, dapat %d", jenis, n)
		}
		// Contoh untuk ditinjau mata: ERP_SURAT_CONTOH=C:\folder menulis kedua PDF ke sana.
		if dir := os.Getenv("ERP_SURAT_CONTOH"); dir != "" {
			if err := os.WriteFile(dir+string(os.PathSeparator)+NamaBerkas(jenis, "Contoh")+"", out, 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestRenderMenolakDataTidakLengkap(t *testing.T) {
	d := dataUji(JenisKeteranganUmroh)
	d.Jamaah.TempatLahir = ""
	if _, err := Render(d); err == nil {
		t.Fatal("harus gagal tanpa tempat lahir")
	}
	d = dataUji(JenisKeteranganUmroh)
	d.Pengaturan.PenandatanganNama = ""
	if _, err := Render(d); err != ErrPengaturanBelumLengkap {
		t.Fatalf("harus ErrPengaturanBelumLengkap, dapat %v", err)
	}
	if _, err := Render(Data{Jenis: "lain"}); err == nil {
		t.Fatal("jenis tidak dikenal harus gagal")
	}
}
