package shared

import (
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
