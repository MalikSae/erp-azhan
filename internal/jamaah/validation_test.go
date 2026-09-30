package jamaah

import (
	"strings"
	"testing"
)

// JB-05/JB-09: validasi format data jamaah sebelum disimpan.
func TestValidateJamaahInput(t *testing.T) {
	s := func(v string) *string { return &v }
	base := CreateJamaahRequest{
		NamaLengkap: "Uji", NIK: s("3171010101800001"), TanggalLahir: s("1980-01-01"),
		Email: s("jamaah@contoh.id"), NoHP: s("0812-3456-7890"),
		TanggalPasporKeluar: s("2024-01-10"), PasporBerlakuSampai: s("2034-01-10"),
	}
	cases := []struct {
		nama  string
		ubah  func(r *CreateJamaahRequest)
		pesan string
	}{
		{"valid", func(r *CreateJamaahRequest) {}, ""},
		{"paspor kedaluwarsa diterima", func(r *CreateJamaahRequest) { r.TanggalPasporKeluar = s("2015-01-10"); r.PasporBerlakuSampai = s("2020-01-10") }, ""},
		{"field kosong dilewati", func(r *CreateJamaahRequest) { *r = CreateJamaahRequest{NamaLengkap: "X", NIK: s(""), Email: s("")} }, ""},
		{"format tanggal", func(r *CreateJamaahRequest) { r.TanggalLahir = s("31-12-1990") }, "tanggal_lahir harus berformat"},
		{"lahir masa depan", func(r *CreateJamaahRequest) { r.TanggalLahir = s("2999-01-01") }, "tanggal_lahir tidak boleh di masa depan"},
		{"berlaku sebelum keluar", func(r *CreateJamaahRequest) { r.PasporBerlakuSampai = s("2020-01-01") }, "paspor_berlaku_sampai tidak boleh sebelum"},
		{"NIK pendek", func(r *CreateJamaahRequest) { r.NIK = s("123") }, "NIK harus 16 digit"},
		{"NIK huruf", func(r *CreateJamaahRequest) { r.NIK = s("31710101018000AB") }, "NIK harus 16 digit"},
		{"email", func(r *CreateJamaahRequest) { r.Email = s("bukan-email") }, "format email"},
		{"hp huruf", func(r *CreateJamaahRequest) { r.NoHP = s("abc") }, "no_hp harus"},
		{"hp pendek", func(r *CreateJamaahRequest) { r.NoHP = s("08123") }, "no_hp harus"},
	}
	for _, c := range cases {
		req := base
		c.ubah(&req)
		got := validateJamaahInput(&req)
		if (c.pesan == "" && got != "") || (c.pesan != "" && !strings.Contains(got, c.pesan)) {
			t.Errorf("%s: got %q, want %q", c.nama, got, c.pesan)
		}
	}
}
