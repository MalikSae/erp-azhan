package shared

import (
	"testing"
	"time"
)

func TestRupiahBulat(t *testing.T) {
	for v, mau := range map[float64]bool{0: true, 1: true, 500_000: true, 25_000_000: true, 0.5: false, 1000.01: false, -3: true} {
		if got := RupiahBulat(v); got != mau {
			t.Errorf("RupiahBulat(%v) = %v, mau %v", v, got, mau)
		}
	}
}

func TestTanggalTidakDiMasaDepan(t *testing.T) {
	wib := time.FixedZone("WIB", 7*60*60)
	hariIni := time.Now().In(wib)
	cases := map[string]bool{
		hariIni.Format("2006-01-02"):                   true,
		" " + hariIni.Format("2006-01-02") + " ":       true,
		hariIni.AddDate(0, 0, -1).Format("2006-01-02"): true,
		hariIni.AddDate(0, 0, 1).Format("2006-01-02"):  false,
		"2000-01-01": true,
		"":           false,
		"01-02-2026": false,
		"2026-02-30": false,
	}
	for s, mau := range cases {
		if got := TanggalTidakDiMasaDepan(s); got != mau {
			t.Errorf("TanggalTidakDiMasaDepan(%q) = %v, mau %v", s, got, mau)
		}
	}
}
