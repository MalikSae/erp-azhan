package schedule

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

var ErrConflict = errors.New("perubahan paket ditolak")

func remainingAfterCapacityChange(total, remaining, nextTotal int) (int, error) {
	allocated := total - remaining
	if remaining < 0 || remaining > total || nextTotal <= 0 || nextTotal < allocated {
		return 0, fmt.Errorf("%w: kapasitas tidak boleh kurang dari seat teralokasi", ErrConflict)
	}
	return nextTotal - allocated, nil
}

func validateDP(dp *float64, prices ...float64) error {
	if dp == nil {
		return nil
	}
	if math.IsNaN(*dp) || math.IsInf(*dp, 0) || *dp < 0 {
		return errors.New("minimal_dp harus berupa angka tidak negatif")
	}
	for _, price := range prices {
		if *dp > price {
			return errors.New("minimal_dp tidak boleh melebihi harga kamar termurah")
		}
	}
	return nil
}

// Batas kolom schedules (migrations/001_init.sql dan 026): nilai yang lebih panjang
// ditolak MySQL sebagai error 1406 dan sampai ke klien sebagai 500.
const (
	maxKodePenerbangan = 50
	maxTransitBandara  = 255
	maxAlasanSeat      = 500 // schedule_seat_adjustments.reason
)

// validateJam menerima kosong, "HH:MM", atau "HH:MM:SS" (bentuk keluaran kolom TIME).
func validateJam(label, jam string) error {
	jam = strings.TrimSpace(jam)
	if jam == "" {
		return nil
	}
	if _, err := time.Parse("15:04", jam); err == nil {
		return nil
	}
	if _, err := time.Parse("15:04:05", jam); err == nil {
		return nil
	}
	return fmt.Errorf("%s tidak valid, gunakan format JJ:MM", label)
}

func validateMaxLen(label, value string, max int) error {
	if len([]rune(value)) > max {
		return fmt.Errorf("%s maksimal %d karakter", label, max)
	}
	return nil
}

// normalizeKota menyamakan ejaan kota hotel: "Mekkah" dan "Makkah" dianggap sama.
func normalizeKota(kota string) string {
	k := strings.ToLower(strings.TrimSpace(kota))
	if k == "mekkah" {
		return "makkah"
	}
	return k
}

// seatSisaForCreate: paket baru selalu mulai penuh. Semua kursi terjual lewat booking
// di ERP, jadi sisa tidak boleh diisi lebih kecil dari total saat paket dibuat
// (perubahan kemudian hanya lewat booking atau penyesuaian beralasan).
func seatSisaForCreate(total int, requested *int) (int, error) {
	if requested != nil && *requested != total {
		return 0, errors.New("seat_sisa tidak bisa diisi saat membuat paket: seat berkurang otomatis lewat booking")
	}
	return total, nil
}
