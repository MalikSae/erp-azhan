package shared

import (
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

// Header proxy (CF-Connecting-IP, X-Forwarded-For, X-Real-IP) bisa diisi bebas
// oleh client. Header tersebut hanya dipercaya jika koneksi TCP datang dari
// reverse proxy tepercaya: loopback, ditambah daftar TRUSTED_PROXIES
// (IP atau CIDR, dipisah koma) — contoh: IP container Caddy/Nginx di Docker.

var (
	trustedProxiesOnce sync.Once
	trustedProxies     []*net.IPNet
)

func loadTrustedProxies() []*net.IPNet {
	trustedProxiesOnce.Do(func() {
		trustedProxies = parseTrustedProxies(os.Getenv("TRUSTED_PROXIES"))
	})
	return trustedProxies
}

// parseTrustedProxies mengubah daftar IP/CIDR menjadi network. Entri tidak
// valid diabaikan. Loopback selalu disertakan.
func parseTrustedProxies(raw string) []*net.IPNet {
	nets := []*net.IPNet{
		{IP: net.IPv4(127, 0, 0, 0), Mask: net.CIDRMask(8, 32)},
		{IP: net.IPv6loopback, Mask: net.CIDRMask(128, 128)},
	}
	for _, entry := range strings.Split(raw, ",") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if !strings.Contains(entry, "/") {
			if ip := net.ParseIP(entry); ip != nil {
				bits := 128
				if ip.To4() != nil {
					bits = 32
				}
				nets = append(nets, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
			}
			continue
		}
		if _, n, err := net.ParseCIDR(entry); err == nil {
			nets = append(nets, n)
		}
	}
	return nets
}

func isTrusted(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ClientIP mengembalikan IP client untuk rate limit dan audit log.
func ClientIP(r *http.Request) string {
	return clientIP(r, loadTrustedProxies())
}

func clientIP(r *http.Request, trusted []*net.IPNet) string {
	remote, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		remote = r.RemoteAddr
	}
	remoteIP := net.ParseIP(strings.TrimSpace(remote))
	if remoteIP == nil {
		return remote
	}
	if !isTrusted(remoteIP, trusted) {
		return remoteIP.String()
	}

	// Only the canonical proxy chain is trusted. CF headers arriving through
	// a general reverse proxy can originate from an arbitrary client.

	// X-Forwarded-For dibaca dari kanan: entri paling kanan ditambahkan oleh
	// proxy terdekat. Lewati proxy tepercaya; entri pertama yang tidak tepercaya
	// adalah client. Entri paling kiri bisa dipalsukan, jadi tidak dipakai.
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for i := len(parts) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(parts[i]))
			if ip == nil {
				break
			}
			if !isTrusted(ip, trusted) {
				return ip.String()
			}
		}
	}

	if ip := net.ParseIP(strings.TrimSpace(r.Header.Get("X-Real-IP"))); ip != nil {
		return ip.String()
	}

	return remoteIP.String()
}
