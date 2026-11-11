package selfbooking

import "testing"

func TestPemesanIkut(t *testing.T) {
	kasus := []struct {
		nama     string
		modeAgen bool
		req      BookingRequest
		mau      bool
	}{
		{"publik, pemesan ikut berangkat", false, BookingRequest{}, true},
		{"publik, memesankan orang lain", false, BookingRequest{PemesanTidakIkut: true}, false},
		{"agen, jamaah pertama saya sendiri", true, BookingRequest{PIC: PICInput{SayaSendiri: true}}, true},
		{"agen, memesankan jamaah", true, BookingRequest{}, false},
		{"agen mengabaikan pemesan_tidak_ikut", true, BookingRequest{PemesanTidakIkut: true, PIC: PICInput{SayaSendiri: true}}, true},
	}
	for _, k := range kasus {
		if dapat := pemesanIkut(k.modeAgen, k.req); dapat != k.mau {
			t.Errorf("%s: dapat %v, mau %v", k.nama, dapat, k.mau)
		}
	}
}
