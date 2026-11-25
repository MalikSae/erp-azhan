package hotel

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func pesanGalat(w *httptest.ResponseRecorder) string {
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	return body["error"]
}

func TestValidateHotelInput(t *testing.T) {
	for _, k := range []struct{ nama, kota, mau string }{
		{" ", "Makkah", "Nama wajib diisi."},
		{strings.Repeat("a", maksNama+1), "Makkah", "Nama maksimal 255 karakter."},
		{"Hilton", " ", "Kota wajib diisi."},
		{"Hilton", strings.Repeat("k", maksKota+1), "Kota maksimal 100 karakter."},
	} {
		w := httptest.NewRecorder()
		if _, _, _, _, ok := validateHotelInput(w, k.nama, k.kota, nil, nil); ok || pesanGalat(w) != k.mau {
			t.Errorf("%q/%q: ok=%v pesan=%q, mau %q", k.nama, k.kota, ok, pesanGalat(w), k.mau)
		}
	}
	w := httptest.NewRecorder()
	if name, city, _, _, ok := validateHotelInput(w, " hilton ", " Makkah ", nil, nil); !ok || name != "HILTON" || city != "Makkah" {
		t.Errorf("input sah: %q %q ok=%v", name, city, ok)
	}
}

func TestValidateMedia(t *testing.T) {
	panjang := "https://x.test/" + strings.Repeat("a", maksMediaURL)
	w := httptest.NewRecorder()
	if _, _, ok := validateMedia(w, &panjang, nil); ok || pesanGalat(w) != "URL foto atau video maksimal 500 karakter." {
		t.Errorf("foto terlalu panjang: ok=%v pesan=%q", ok, pesanGalat(w))
	}
	kosong := " "
	w = httptest.NewRecorder()
	if p, v, ok := validateMedia(w, &kosong, &kosong); !ok || p != nil || v != nil {
		t.Errorf("media kosong harus jadi nil: %v %v ok=%v", p, v, ok)
	}
}
