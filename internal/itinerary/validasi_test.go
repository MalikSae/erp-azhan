package itinerary

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateItineraryInput(t *testing.T) {
	hari := []DayRequest{{Title: "Jeddah - Makkah", Activities: []Activity{{Text: "Tiba"}}}}
	for _, k := range []struct {
		judul string
		days  []DayRequest
		mau   string
	}{
		{" ", hari, "Judul wajib diisi."},
		{strings.Repeat("j", maksTeks+1), hari, "Judul maksimal 255 karakter."},
		{"Umroh 9 Hari", nil, "Itinerary minimal 1 hari."},
		{"Umroh 9 Hari", []DayRequest{{Title: strings.Repeat("r", maksTeks+1), Activities: []Activity{{Text: "Tiba"}}}},
			"rute/lokasi hari ke-1 maksimal 255 karakter"},
	} {
		w := httptest.NewRecorder()
		ok := validateItineraryInput(w, k.judul, k.days)
		var body map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if ok || body["error"] != k.mau {
			t.Errorf("ok=%v pesan=%q, mau %q", ok, body["error"], k.mau)
		}
	}
	if !validateItineraryInput(httptest.NewRecorder(), "Umroh 9 Hari", hari) {
		t.Error("input sah ditolak")
	}
}

// Itinerary khusus paket memakai ValidateDays saja: tanpa hari tetap diterima (gate minimal 1 hari hanya master).
func TestValidateDaysKosongDiterima(t *testing.T) {
	if err := ValidateDays(nil); err != nil {
		t.Errorf("hari kosong: %v", err)
	}
}
