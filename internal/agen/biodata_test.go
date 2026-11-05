package agen

import (
	"testing"
	"time"
)

func TestValidasiBiodataAgen(t *testing.T) {
	sekarang := time.Date(2026, 11, 5, 12, 0, 0, 0, time.UTC)
	ok := AjukanRequest{NIK: "3201010101010001", Domisili: "Kota Bandung", JenisKelamin: "L", TanggalLahir: "1990-05-12", SetujuSyaratKetentuan: true}
	if msg := validasiBiodataAgen(&ok, sekarang); msg != "" {
		t.Fatalf("biodata sah ditolak: %s", msg)
	}

	for nama, c := range map[string]struct {
		ubah func(r *AjukanRequest)
		want string
	}{
		"jenis kelamin kosong":    {func(r *AjukanRequest) { r.JenisKelamin = "" }, "jenis kelamin"},
		"jenis kelamin lain":      {func(r *AjukanRequest) { r.JenisKelamin = "X" }, "jenis kelamin"},
		"tanggal lahir kosong":    {func(r *AjukanRequest) { r.TanggalLahir = "" }, "tanggal lahir"},
		"format tanggal salah":    {func(r *AjukanRequest) { r.TanggalLahir = "12-05-1990" }, "tanggal lahir"},
		"tanggal di masa depan":   {func(r *AjukanRequest) { r.TanggalLahir = "2030-01-01" }, "tanggal lahir"},
		"sebelum 1900":            {func(r *AjukanRequest) { r.TanggalLahir = "1899-12-31" }, "tanggal lahir"},
		"belum 17 tahun":          {func(r *AjukanRequest) { r.TanggalLahir = "2009-11-06" }, "17 tahun"},
		"tepat 17 tahun hari ini": {nil, ""},
	} {
		r := ok
		if c.ubah != nil {
			c.ubah(&r)
		} else {
			r.TanggalLahir = "2009-11-05"
		}
		msg := validasiBiodataAgen(&r, sekarang)
		if c.want == "" {
			if msg != "" {
				t.Errorf("%s: ditolak (%s), seharusnya lolos", nama, msg)
			}
			continue
		}
		if msg == "" || !contains(msg, c.want) {
			t.Errorf("%s: msg = %q, want memuat %q", nama, msg, c.want)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
