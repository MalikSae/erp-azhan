package airport

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateAirportInput(t *testing.T) {
	for _, k := range []struct{ nama, kode, kota, mau string }{
		{strings.Repeat("a", maksNama+1), "JED", "Jeddah", "Nama bandara maksimal 150 karakter."},
		{"King Abdulaziz", "JEDDA", "Jeddah", "Kode bandara maksimal 4 karakter."},
		{"King Abdulaziz", "JED", strings.Repeat("k", maksKota+1), "Kota bandara maksimal 100 karakter."},
	} {
		w := httptest.NewRecorder()
		_, _, _, ok := validateAirportInput(w, k.nama, k.kode, k.kota)
		var body map[string]string
		_ = json.Unmarshal(w.Body.Bytes(), &body)
		if ok || body["error"] != k.mau {
			t.Errorf("ok=%v pesan=%q, mau %q", ok, body["error"], k.mau)
		}
	}
}
