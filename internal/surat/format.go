// Package surat membuat surat jamaah (Surat Keterangan Umroh, Surat Rekomendasi Paspor) dalam PDF: pengaturan per brand, nomor surat
// berurutan yang disimpan, dan penyusunan halaman.
package surat

import (
	"fmt"
	"strings"
	"time"
)

const (
	JenisKeteranganUmroh = "keterangan_umroh"
	JenisRekomPaspor     = "rekom_paspor"
)

var bulanIndonesia = [...]string{"Januari", "Februari", "Maret", "April", "Mei", "Juni", "Juli", "Agustus", "September", "Oktober", "November", "Desember"}

var bulanRomawi = [...]string{"I", "II", "III", "IV", "V", "VI", "VII", "VIII", "IX", "X", "XI", "XII"}

// TanggalIndonesia: 2026-09-11 menjadi "11 September 2026".
func TanggalIndonesia(t time.Time) string {
	return fmt.Sprintf("%d %s %d", t.Day(), bulanIndonesia[t.Month()-1], t.Year())
}

// KodeJenis adalah singkatan jenis surat di nomor surat; kosong bila jenis tidak dikenal.
func KodeJenis(jenis string) string {
	switch jenis {
	case JenisKeteranganUmroh:
		return "SKU"
	case JenisRekomPaspor:
		return "SRP"
	}
	return ""
}

// JudulJenis adalah judul surat.
func JudulJenis(jenis string) string {
	switch jenis {
	case JenisKeteranganUmroh:
		return "Surat Keterangan Umroh"
	case JenisRekomPaspor:
		return "Surat Rekomendasi Paspor"
	}
	return ""
}

// LabelJenis adalah nama surat yang dilihat jamaah di daftar surat dan nama berkas ("Surat Cuti"); judul di kepala surat tetap JudulJenis.
func LabelJenis(jenis string) string {
	if jenis == JenisKeteranganUmroh {
		return "Surat Cuti"
	}
	return JudulJenis(jenis)
}

// FormatNomor menyusun nomor surat: urut/KODE/JENIS/BULAN-ROMAWI/TAHUN, mis. 100808/HAW/SKU/XI/2026.
func FormatNomor(urut uint32, kode, jenis string, terbit time.Time) string {
	return fmt.Sprintf("%d/%s/%s/%s/%d", urut, strings.ToUpper(strings.TrimSpace(kode)), KodeJenis(jenis), bulanRomawi[terbit.Month()-1], terbit.Year())
}

// KotaSurat membuang awalan administratif dari kota brand: "Kota Depok" menjadi "Depok".
func KotaSurat(city string) string {
	c := strings.TrimSpace(city)
	for _, awalan := range []string{"Kota Administrasi ", "Kota ", "Kabupaten ", "Kab. ", "Kab "} {
		if len(c) > len(awalan) && strings.EqualFold(c[:len(awalan)], awalan) {
			return strings.TrimSpace(c[len(awalan):])
		}
	}
	return c
}

// NamaBerkas menyusun nama berkas unduhan: Surat-Keterangan-Umroh-NAMA-JAMAAH.pdf (hanya huruf, angka, dan tanda hubung).
func NamaBerkas(jenis, namaJamaah string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(namaJamaah) {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == ' ' || r == '-' || r == '_':
			if s := b.String(); s != "" && !strings.HasSuffix(s, "-") {
				b.WriteByte('-')
			}
		}
	}
	nama := strings.Trim(b.String(), "-")
	if nama == "" {
		nama = "jamaah"
	}
	return strings.ReplaceAll(LabelJenis(jenis), " ", "-") + "-" + nama + ".pdf"
}
