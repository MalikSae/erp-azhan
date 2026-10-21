package schedule

import "time"

// Status registrasi untuk web publik (SPEK-WEB-PUBLIK-CMS-2026-10-20.md §4.2). Diturunkan dari seat dan
// tanggal berangkat, tidak disimpan. Enum mengikuti spesifikasi SEO (RegistrationStatus).
const (
	RegOpen       = "OPEN"
	RegAlmostFull = "ALMOST_FULL"
	RegClosed     = "CLOSED"
	RegSoldOut    = "SOLD_OUT"
)

// Ambang ALMOST_FULL: sisa seat <= max(almostFullMinSeat, almostFullPercent% dari kuota).
// Konstanta dulu; pengaturan per brand menyusul bila diperlukan.
const (
	almostFullMinSeat = 5
	almostFullPercent = 20
)

// registrationStatus menurunkan status registrasi. batas adalah tanggal berangkat paling awal yang masih
// dijual (batasPublikWIB); berangkat sebelum batas berarti pendaftaran online ditutup.
func registrationStatus(seatTotal, seatSisa int, berangkat, batas string) string {
	switch {
	case seatSisa <= 0:
		return RegSoldOut
	case berangkat < batas:
		return RegClosed
	}
	ambang := (seatTotal*almostFullPercent + 99) / 100
	if ambang < almostFullMinSeat {
		ambang = almostFullMinSeat
	}
	if seatSisa <= ambang {
		return RegAlmostFull
	}
	return RegOpen
}

// RegistrationStatus menurunkan status registrasi untuk tanggal hari ini (WIB).
func RegistrationStatus(seatTotal, seatSisa int, berangkat string) string {
	return registrationStatus(seatTotal, seatSisa, berangkat, batasPublikWIB())
}

// durationDays menghitung durasi paket dalam hari, termasuk hari berangkat dan hari pulang
// (sama dengan durasiHari di azhan-booking). 0 bila tanggal tidak valid atau pulang sebelum berangkat.
func durationDays(berangkat, pulang string) int {
	b, err1 := time.Parse("2006-01-02", berangkat)
	p, err2 := time.Parse("2006-01-02", pulang)
	if err1 != nil || err2 != nil || p.Before(b) {
		return 0
	}
	return int(p.Sub(b).Hours()/24+0.5) + 1
}
