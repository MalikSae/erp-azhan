package main

import (
	"strings"
	"testing"
)

// Butir 7i: origin pengembangan (localhost, *.test, dengan credentials) hanya di luar produksi.
func TestCORSOriginsPengembanganHanyaNonProduksi(t *testing.T) {
	prod := corsOrigins(true, " https://admin.azhan.id , ,https://cms.azhan.id")
	if strings.Join(prod, ",") != "https://admin.azhan.id,https://cms.azhan.id" {
		t.Fatalf("produksi: %v", prod)
	}
	for _, o := range prod {
		if strings.Contains(o, "localhost") || strings.HasSuffix(o, ".test") {
			t.Fatalf("origin dev bocor ke produksi: %s", o)
		}
	}
	dev := strings.Join(corsOrigins(false, ""), ",")
	if !strings.Contains(dev, "http://localhost:5173") || !strings.Contains(dev, "https://*.test") {
		t.Fatalf("dev: %s", dev)
	}
}

// Audit keamanan 2026-11-26 (Q7): produksi tanpa TRUSTED_PROXIES diberi peringatan saat start.
func TestPeringatanTrustedProxies(t *testing.T) {
	if peringatanTrustedProxies(true, "  ") == "" {
		t.Fatal("produksi tanpa TRUSTED_PROXIES harus diberi peringatan")
	}
	if p := peringatanTrustedProxies(true, "172.18.0.0/16"); p != "" {
		t.Fatalf("produksi dengan TRUSTED_PROXIES: %q", p)
	}
	if p := peringatanTrustedProxies(false, ""); p != "" {
		t.Fatalf("non-produksi: %q", p)
	}
}
