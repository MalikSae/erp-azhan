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

// DevCaptchaBypass true hanya bila pengembang secara eksplisit meminta captcha dilewati:
// APP_ENV=development DAN ALLOW_DEV_CAPTCHA=true. Selain itu captcha tanpa secret DITOLAK.
func DevCaptchaBypass() bool {
	return AppEnv() == "development" && strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_DEV_CAPTCHA")), "true")
}
