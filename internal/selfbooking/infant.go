package selfbooking

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrTanggalLahirTidakValid = errors.New("tanggal lahir tidak valid (format YYYY-MM-DD, tidak boleh di masa depan)")
	ErrUsiaInfant             = errors.New("usia infant harus di bawah 2 tahun saat keberangkatan")
)

// tanggalLahirOrNil mengembalikan tanggal lahir untuk disimpan, atau nil bila kosong.
func tanggalLahirOrNil(s *string) any {
	if s == nil || strings.TrimSpace(*s) == "" {
		return nil
	}
	return strings.TrimSpace(*s)
}

// validasiTanggalLahirAnggota: tanggal lahir yang diisi harus valid dan tidak
// di masa depan; infant harus berusia di bawah 2 tahun pada tanggal berangkat
// (harga infant hanya berlaku untuk bayi).
func validasiTanggalLahirAnggota(anggota []AnggotaInput, berangkat, hariIni time.Time) error {
	for _, a := range anggota {
		if a.TanggalLahir == nil || strings.TrimSpace(*a.TanggalLahir) == "" {
			if a.PaxType == "infant" {
				return ErrTanggalLahirTidakValid
			}
			continue
		}
		lahir, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(*a.TanggalLahir), hariIni.Location())
		if err != nil || lahir.After(hariIni) {
			return ErrTanggalLahirTidakValid
		}
		if a.PaxType == "infant" && !berangkat.Before(lahir.AddDate(2, 0, 0)) {
			return ErrUsiaInfant
		}
	}
	return nil
}
