package selfbooking

import (
	"errors"
	"testing"
)

// Keputusan 2026-11-26: paling banyak satu infant per jamaah dewasa (PIC dihitung sebagai dewasa).
func TestCekJumlahInfant(t *testing.T) {
	infant := AnggotaInput{PaxType: "infant"}
	dewasa := AnggotaInput{PaxType: "reguler"}
	cases := []struct {
		nama    string
		anggota []AnggotaInput
		gagal   bool
	}{
		{"tanpa infant", nil, false},
		{"PIC + 1 infant", []AnggotaInput{infant}, false},
		{"PIC + 2 infant", []AnggotaInput{infant, infant}, true},
		{"PIC + 1 dewasa + 2 infant", []AnggotaInput{dewasa, infant, infant}, false},
		{"PIC + 1 dewasa + 3 infant", []AnggotaInput{dewasa, infant, infant, infant}, true},
	}
	for _, c := range cases {
		err := cekJumlahInfant(BookingRequest{Anggota: c.anggota})
		if c.gagal != errors.Is(err, ErrInfantMelebihiDewasa) {
			t.Errorf("%s: err = %v, gagal mau %v", c.nama, err, c.gagal)
		}
	}
}
