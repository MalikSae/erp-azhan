package surat

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
	_ "golang.org/x/image/webp"

	"erp-azhan/api/internal/shared"
)

// Merek adalah identitas brand di kop teks dan footer izin.
type Merek struct {
	Nama     string
	Alamat   string
	Kota     string
	Telepon  string
	Email    string
	LogoURL  string
	PPIU     string
	PIHK     string
	WhatsApp string
}

// Jamaah adalah data jamaah yang dicetak di surat.
type Jamaah struct {
	Nama         string
	TempatLahir  string
	TanggalLahir time.Time
	Kota         string
}

// Data adalah semua isian satu surat.
type Data struct {
	Jenis      string
	Nomor      string
	TerbitAt   time.Time
	Pengaturan Pengaturan
	Merek      Merek
	Jamaah     Jamaah
	Berangkat  time.Time
	Pulang     time.Time
}

const (
	margin      = 20.0
	lebarIsi    = 210 - 2*margin
	tinggiBaris = 5.0
)

// bacaGambar membaca gambar unggahan brand dari disk (hanya /uploads/brand-logos/<nama>) dan mengubahnya ke PNG supaya WebP, JPG, dan GIF
// bisa dipakai di PDF. Mengembalikan nil bila url kosong atau berkas hilang (gambar opsional; surat tetap dibuat tanpanya).
func bacaGambar(url string) []byte {
	url = strings.TrimSpace(url)
	if !strings.HasPrefix(url, "/uploads/brand-logos/") {
		return nil
	}
	nama := path.Base(url)
	if nama == "." || nama == "/" || strings.ContainsAny(nama, "\\/") {
		return nil
	}
	data, err := os.ReadFile(filepath.Join("uploads", "brand-logos", nama))
	if err != nil {
		return nil
	}
	// Cek dimensi dari header dulu: gambar raksasa (bom dekompresi) dilewati, tidak didekode.
	if err := shared.CekDimensiGambar(bytes.NewReader(data), shared.MaxLogoPixels); err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil
	}
	return buf.Bytes()
}

type berkasPDF struct {
	pdf *fpdf.Fpdf
	tr  func(string) string
}

func (b *berkasPDF) gambar(nama string, data []byte) (w, h float64, ok bool) {
	if len(data) == 0 {
		return 0, 0, false
	}
	info := b.pdf.RegisterImageOptionsReader(nama, fpdf.ImageOptions{ImageType: "PNG"}, bytes.NewReader(data))
	if info == nil || b.pdf.Error() != nil {
		return 0, 0, false
	}
	w, h = info.Extent()
	return w, h, w > 0 && h > 0
}

func (b *berkasPDF) teks(gaya string, ukuran float64) {
	b.pdf.SetFont("Times", gaya, ukuran)
}

func (b *berkasPDF) baris(s string) {
	b.pdf.CellFormat(lebarIsi, tinggiBaris, b.tr(s), "", 1, "L", false, 0, "")
}

func (b *berkasPDF) paragraf(s string) {
	b.pdf.MultiCell(lebarIsi, tinggiBaris, b.tr(s), "", "J", false)
}

// isian menulis "Label    : nilai" dengan titik dua sejajar.
func (b *berkasPDF) isian(label, nilai string) {
	b.pdf.CellFormat(42, tinggiBaris, b.tr(label), "", 0, "L", false, 0, "")
	b.pdf.CellFormat(4, tinggiBaris, ":", "", 0, "L", false, 0, "")
	b.pdf.MultiCell(lebarIsi-46, tinggiBaris, b.tr(nilai), "", "L", false)
}

func (b *berkasPDF) kop(d Data) {
	if w, h, ok := b.gambar("kop", bacaGambar(d.Pengaturan.KopURL)); ok {
		lebar := 210.0 - 20.0
		tinggi := lebar * h / w
		b.pdf.ImageOptions("kop", 10, 8, lebar, tinggi, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		b.pdf.SetY(8 + tinggi + 4)
		return
	}
	// Tanpa gambar kop: kop teks dari data brand (logo, nama, alamat, kontak).
	teksX := margin
	if w, h, ok := b.gambar("logo", bacaGambar(d.Merek.LogoURL)); ok {
		tinggi := 18.0
		lebar := tinggi * w / h
		if lebar > 40 {
			lebar = 40
			tinggi = lebar * h / w
		}
		b.pdf.ImageOptions("logo", margin, 12, lebar, tinggi, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
		teksX = margin + lebar + 4
	}
	nama := d.Pengaturan.NamaPerusahaan
	if nama == "" {
		nama = d.Merek.Nama
	}
	b.pdf.SetXY(teksX, 12)
	b.teks("B", 15)
	b.pdf.CellFormat(210-margin-teksX, 6.5, b.tr(strings.ToUpper(nama)), "", 1, "L", false, 0, "")
	b.teks("", 9)
	if d.Merek.Alamat != "" {
		b.pdf.SetX(teksX)
		b.pdf.MultiCell(210-margin-teksX, 4.2, b.tr(d.Merek.Alamat), "", "L", false)
	}
	kontak := []string{}
	if d.Merek.Telepon != "" {
		kontak = append(kontak, "Telp. "+d.Merek.Telepon)
	}
	if d.Merek.WhatsApp != "" {
		kontak = append(kontak, "WhatsApp "+d.Merek.WhatsApp)
	}
	if d.Merek.Email != "" {
		kontak = append(kontak, d.Merek.Email)
	}
	if len(kontak) > 0 {
		b.pdf.SetX(teksX)
		b.pdf.CellFormat(210-margin-teksX, 4.2, b.tr(strings.Join(kontak, "  |  ")), "", 1, "L", false, 0, "")
	}
	y := b.pdf.GetY()
	if y < 34 {
		y = 34
	}
	b.pdf.SetDrawColor(0, 0, 0)
	b.pdf.SetLineWidth(0.6)
	b.pdf.Line(margin, y+2, 210-margin, y+2)
	b.pdf.SetLineWidth(0.2)
	b.pdf.SetY(y + 8)
}

func (b *berkasPDF) pembuka(d Data, judul string) {
	b.teks("BU", 13)
	b.pdf.CellFormat(lebarIsi, 6.5, b.tr(strings.ToUpper(judul)), "", 1, "C", false, 0, "")
	b.teks("", 10.5)
	b.pdf.CellFormat(lebarIsi, tinggiBaris, b.tr("Nomor : "+d.Nomor), "", 1, "C", false, 0, "")
	b.pdf.Ln(5)
}

func (b *berkasPDF) penandatangan(d Data) {
	b.teks("", 10.5)
	b.paragraf("Assalamualaikum Warohmatullahi Wabarokatuh,")
	b.paragraf("Dengan hormat,")
	b.paragraf("Saya yang bertanda tangan dibawah ini,")
	b.isian("Nama", d.Pengaturan.PenandatanganNama)
	b.isian("Jabatan", d.Pengaturan.PenandatanganJabatan)
	b.pdf.Ln(3)
	b.paragraf("Bersama ini saya menerangkan dengan sesungguhnya bahwa :")
	b.isian("Nama", d.Jamaah.Nama)
	b.isian("Tempat/Tanggal Lahir", d.Jamaah.TempatLahir+", "+TanggalIndonesia(d.Jamaah.TanggalLahir))
	b.pdf.Ln(3)
}

func (b *berkasPDF) tandaTangan(d Data) {
	if b.pdf.GetY() > 215 {
		b.pdf.AddPage()
	}
	kota := KotaSurat(d.Merek.Kota)
	tanggal := TanggalIndonesia(d.TerbitAt)
	if kota != "" {
		tanggal = kota + ", " + tanggal
	}
	b.pdf.Ln(2)
	b.teks("", 10.5)
	b.baris(tanggal)
	b.baris("Hormat kami,")
	yTtd := b.pdf.GetY()
	if w, h, ok := b.gambar("stempel", bacaGambar(d.Pengaturan.StempelURL)); ok {
		tinggi := 30.0
		lebar := tinggi * w / h
		b.pdf.ImageOptions("stempel", margin-2, yTtd-1, lebar, tinggi, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}
	if w, h, ok := b.gambar("ttd", bacaGambar(d.Pengaturan.TandaTanganURL)); ok {
		tinggi := 24.0
		lebar := tinggi * w / h
		b.pdf.ImageOptions("ttd", margin+14, yTtd+1, lebar, tinggi, false, fpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}
	b.pdf.SetY(yTtd + 27)
	b.teks("BU", 10.5)
	b.baris(d.Pengaturan.PenandatanganNama)
	b.teks("", 10.5)
	b.baris(d.Pengaturan.PenandatanganJabatan)
}

func (b *berkasPDF) kaki(d Data) {
	izin := []string{}
	if d.Merek.PPIU != "" {
		izin = append(izin, "Izin PPIU "+d.Merek.PPIU)
	}
	if d.Merek.PIHK != "" {
		izin = append(izin, "Izin PIHK "+d.Merek.PIHK)
	}
	if len(izin) == 0 {
		return
	}
	b.pdf.SetAutoPageBreak(false, 0) // footer di dasar halaman, di bawah batas pemotongan halaman
	b.pdf.SetY(-18)
	b.pdf.SetDrawColor(120, 120, 120)
	b.pdf.Line(margin, b.pdf.GetY(), 210-margin, b.pdf.GetY())
	b.pdf.Ln(1.5)
	b.teks("", 8.5)
	b.pdf.CellFormat(lebarIsi, 4, b.tr(strings.Join(izin, "  |  ")), "", 1, "C", false, 0, "")
}

func periode(d Data) string {
	return TanggalIndonesia(d.Berangkat) + " s/d " + TanggalIndonesia(d.Pulang)
}

func (b *berkasPDF) isiKeteranganUmroh(d Data, perusahaan string) {
	b.paragraf("KEPADA YTH,\nBAPAK/IBU KEPALA KANTOR\nDI TEMPAT")
	b.pdf.Ln(3)
	b.penandatangan(d)
	b.paragraf(fmt.Sprintf("Yang bersangkutan adalah benar jamaah yang telah terdaftar dan akan melaksanakan ibadah umroh melalui %s untuk estimasi keberangkatan tanggal %s.", perusahaan, periode(d)))
	b.pdf.Ln(2)
	b.paragraf("Terkait dengan hal tersebut diatas maka mohon kiranya dapat diberikan izin guna menunaikan ibadah umroh.")
	b.pdf.Ln(2)
	b.paragraf("Demikian pernyataan penjamin ini kami buat dengan sesungguhnya agar dapat dipergunakan sebagaimana mestinya.")
	b.paragraf("Wassalamualaikum Warohmatullahi Wabarokatuh.")
}

func (b *berkasPDF) isiRekomPaspor(d Data) {
	tujuan := "KANTOR IMIGRASI"
	if kota := strings.TrimSpace(d.Jamaah.Kota); kota != "" {
		tujuan = "KEPALA KANTOR IMIGRASI " + strings.ToUpper(KotaSurat(kota))
	}
	b.paragraf("KEPADA YTH,\n" + tujuan + "\nDI TEMPAT")
	b.pdf.Ln(3)
	b.penandatangan(d)
	b.paragraf(fmt.Sprintf("Adalah benar yang bersangkutan telah mendaftar dan berniat melaksanakan ibadah Umroh melalui kami untuk keberangkatan tanggal %s. Berdasarkan SE DIRJEN Imigrasi No.IMI0342.GR.01.01 Tahun 2014 tentang penerbitan pengurusan paspor RI oleh PPIU tanggal 04 Maret 2014, kami yang menyatakan bahwa :", periode(d)))
	b.pdf.Ln(1)
	butir := []string{
		"Pemohon paspor RI adalah warga negara Indonesia ( WNI ) yang sebenarnya akan melakukan perjalanan ke Saudi Arabia dalam rangka ibadah Umroh.",
		"Menjamin bahwa nama diatas tidak akan melakukan pelanggaran peraturan Keimigrasian berupa penyalahgunaan Izin tinggal ( Overstay ) melebihi izin yang telah ditetapkan, memalsukan atau membuat palsu paspor yang diberikan kepadanya maupun bekerja secara ilegal.",
		"Apabila jamaah tersebut membatalkan atau berangkat Umroh diluar travel kami, maka hal-hal yang menyangkut dengan jamaah tersebut bukan menjadi tanggung jawab kami selaku penyelenggara Umroh.",
	}
	for i, s := range butir {
		b.pdf.SetX(margin + 4)
		b.pdf.CellFormat(7, tinggiBaris, fmt.Sprintf("%d.", i+1), "", 0, "L", false, 0, "")
		b.pdf.MultiCell(lebarIsi-11, tinggiBaris, b.tr(s), "", "J", false)
		b.pdf.SetX(margin)
	}
	b.pdf.Ln(2)
	b.paragraf("Demikian pernyataan penjamin ini kami buat dengan sesungguhnya agar dapat dipergunakan sebagaimana mestinya.")
	b.paragraf("Wassalamualaikum Warohmatullahi Wabarokatuh.")
}

// Render menyusun satu surat dan mengembalikan berkas PDF-nya.
func Render(d Data) ([]byte, error) {
	if JudulJenis(d.Jenis) == "" {
		return nil, fmt.Errorf("surat.Render: jenis %q tidak dikenal", d.Jenis)
	}
	if !d.Pengaturan.Lengkap() {
		return nil, ErrPengaturanBelumLengkap
	}
	if strings.TrimSpace(d.Jamaah.Nama) == "" || d.Jamaah.TanggalLahir.IsZero() || strings.TrimSpace(d.Jamaah.TempatLahir) == "" {
		return nil, errors.New("surat.Render: data jamaah belum lengkap")
	}
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(margin, 15, margin)
	pdf.SetAutoPageBreak(true, 22)
	pdf.SetTitle(JudulJenis(d.Jenis)+" "+d.Nomor, true)
	pdf.SetCreator(d.Merek.Nama, true)
	pdf.AddPage()
	b := &berkasPDF{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}
	perusahaan := d.Pengaturan.NamaPerusahaan
	if perusahaan == "" {
		perusahaan = d.Merek.Nama
	}
	b.kop(d)
	b.pembuka(d, JudulJenis(d.Jenis))
	switch d.Jenis {
	case JenisKeteranganUmroh:
		b.isiKeteranganUmroh(d, perusahaan)
	case JenisRekomPaspor:
		b.isiRekomPaspor(d)
	}
	b.tandaTangan(d)
	b.kaki(d)
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("surat.Render output: %w", err)
	}
	return buf.Bytes(), nil
}
