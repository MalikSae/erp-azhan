package brand

import "testing"

// Code review Paket dan master, 11 November 2026: nominal brand harus bilangan bulat rupiah (ADR-0001).
func TestSemuaRupiahBulat(t *testing.T) {
	if !semuaRupiahBulat() || !semuaRupiahBulat(0, 500_000, 100_000) {
		t.Fatal("nominal bulat harus lolos")
	}
	if semuaRupiahBulat(500_000, 100_000.5) {
		t.Fatal("nominal pecahan harus ditolak")
	}
}
