package shared

import (
	"errors"
	"os"
	"strings"
)

// AppEnv mengembalikan APP_ENV dalam huruf kecil (mis. "production", "development"). Kosong
// berarti tidak diatur; perlakuan aman (fail closed) tidak boleh bergantung pada nilai kosong.
func AppEnv() string {
	return strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
}

// IsProduction true bila APP_ENV=production.
func IsProduction() bool { return AppEnv() == "production" }

// ValidateAppEnv dijalankan saat startup API: APP_ENV wajib diisi. Tanpa nilai, server diperlakukan sebagai
// pengembangan (origin CORS *.test, isi email reset ditulis ke log), sehingga server publik yang lupa mengisinya
// diam-diam berjalan dalam mode dev (audit keamanan 2026-11-26). Lokal: APP_ENV=development.
func ValidateAppEnv() error {
	if AppEnv() == "" {
		return errors.New("APP_ENV wajib diisi (production di server publik, development di lokal)")
	}
	return nil
}
