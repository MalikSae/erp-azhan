package shared

import "testing"

// Audit keamanan 2026-11-26 (Q7): APP_ENV kosong ditolak saat startup, bukan diam-diam mode pengembangan.
func TestValidateAppEnv(t *testing.T) {
	for _, v := range []string{"", "   "} {
		t.Setenv("APP_ENV", v)
		if ValidateAppEnv() == nil {
			t.Fatalf("APP_ENV=%q harus ditolak", v)
		}
	}
	for _, v := range []string{"development", "production", " Production "} {
		t.Setenv("APP_ENV", v)
		if err := ValidateAppEnv(); err != nil {
			t.Fatalf("APP_ENV=%q: %v", v, err)
		}
	}
}
