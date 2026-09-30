package shared

import (
	"net/url"
	"strings"
)

// ValidMediaURL menerima URL media yang aman ditampilkan: kosong, path unggahan
// lokal (/uploads/… atau /api/admin/media/…), atau URL absolut http(s) dengan host.
// Skema lain (javascript:, data:, dll.) dan teks bebas ditolak (MP-08).
func ValidMediaURL(raw string) bool {
	s := strings.TrimSpace(raw)
	if s == "" {
		return true
	}
	if strings.HasPrefix(s, "/uploads/") || strings.HasPrefix(s, "/api/admin/media/") {
		return !strings.Contains(s, "..")
	}
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}
