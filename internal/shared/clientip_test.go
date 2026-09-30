package shared

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP(t *testing.T) {
	trusted := parseTrustedProxies("172.18.0.0/16, 10.0.0.5, bukan-ip")

	tests := []struct {
		name    string
		remote  string
		headers map[string]string
		want    string
	}{
		{
			name:    "client langsung: header palsu diabaikan",
			remote:  "198.51.100.8:4321",
			headers: map[string]string{"X-Forwarded-For": "1.2.3.4", "CF-Connecting-IP": "5.6.7.8", "True-Client-IP": "9.9.9.9", "X-Real-IP": "7.7.7.7"},
			want:    "198.51.100.8",
		},
		{
			name:    "proxy loopback: CF-Connecting-IP palsu diabaikan",
			remote:  "127.0.0.1:4321",
			headers: map[string]string{"CF-Connecting-IP": "203.0.113.10", "X-Forwarded-For": "198.51.100.20"},
			want:    "198.51.100.20",
		},
		{
			name:    "proxy tepercaya: XFF dibaca dari kanan, entri kiri palsu diabaikan",
			remote:  "172.18.0.3:4321",
			headers: map[string]string{"X-Forwarded-For": "6.6.6.6, 203.0.113.50"},
			want:    "203.0.113.50",
		},
		{
			name:    "proxy berantai: lewati semua proxy tepercaya",
			remote:  "127.0.0.1:4321",
			headers: map[string]string{"X-Forwarded-For": "6.6.6.6, 203.0.113.51, 10.0.0.5, 172.18.0.9"},
			want:    "203.0.113.51",
		},
		{
			name:    "proxy tepercaya tanpa XFF: X-Real-IP",
			remote:  "10.0.0.5:4321",
			headers: map[string]string{"X-Real-IP": "203.0.113.60"},
			want:    "203.0.113.60",
		},
		{
			name:   "proxy tepercaya tanpa header: pakai remote",
			remote: "[::1]:4321",
			want:   "::1",
		},
		{
			name:    "IP di luar daftar dianggap client",
			remote:  "172.19.0.3:4321",
			headers: map[string]string{"X-Forwarded-For": "203.0.113.70"},
			want:    "172.19.0.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", nil)
			req.RemoteAddr = tt.remote
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if got := clientIP(req, trusted); got != tt.want {
				t.Fatalf("clientIP() = %q, want %q", got, tt.want)
			}
		})
	}
}
