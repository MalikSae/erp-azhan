package agen

import (
	"strings"
	"time"
)

// usiaMinimalAgen: pendaftar agen minimal berusia 17 tahun (batas usia memiliki KTP).
const usiaMinimalAgen = 17

// validasiBiodataAgen memeriksa jenis kelamin dan tanggal lahir pengajuan agen; mengembalikan pesan galat, atau "" bila sah.
// Nilainya dirapikan di tempat (trim, huruf besar untuk jenis kelamin).
func validasiBiodataAgen(req *AjukanRequest, sekarang time.Time) string {
	req.JenisKelamin = strings.ToUpper(strings.TrimSpace(req.JenisKelamin))
	req.TanggalLahir = strings.TrimSpace(req.TanggalLahir)
	if req.JenisKelamin != "L" && req.JenisKelamin != "P" {
		return "jenis kelamin wajib dipilih"
	}
	if req.TanggalLahir == "" {
		return "tanggal lahir wajib diisi"
	}
	lahir, err := time.Parse("2006-01-02", req.TanggalLahir)
	hariIni := time.Date(sekarang.Year(), sekarang.Month(), sekarang.Day(), 0, 0, 0, 0, time.UTC)
	if err != nil || lahir.After(hariIni) || lahir.Year() < 1900 {
		return "tanggal lahir tidak valid"
	}
	if lahir.AddDate(usiaMinimalAgen, 0, 0).After(hariIni) {
		return "pendaftar agen minimal berusia 17 tahun"
	}
	return ""
}
