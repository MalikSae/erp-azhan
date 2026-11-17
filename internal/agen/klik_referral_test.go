package agen

import (
	"strings"
	"testing"
)

func TestPengunjungValid(t *testing.T) {
	cases := map[string]bool{
		"3f1c2b9e-1111-4222-8333-944455556666": true,
		strings.Repeat("a", 16):                true,  // batas bawah
		strings.Repeat("a", 64):                true,  // batas atas
		strings.Repeat("a", 15):                false, // kurang dari 16
		strings.Repeat("a", 65):                false, // lebih dari 64
		"":                                     false,
		"palsu dengan spasi 1234567890":        false,
		"<script>alert(1)</script>xxxxxxxx":    false,
	}
	for s, want := range cases {
		if got := pengunjungValid(s); got != want {
			t.Errorf("pengunjungValid(%q) = %v, mau %v", s, got, want)
		}
	}
}

func TestJatahKlikPerIP(t *testing.T) {
	h := NewHandler(nil)

	// Satu kode dari satu IP: klikPerIPPerKodeSehari klik tercatat per hari, sisanya ditolak.
	for i := 0; i < klikPerIPPerKodeSehari; i++ {
		if !h.ambilJatahKlik("10.0.0.1", 3, "ABC123") {
			t.Fatalf("klik ke-%d ditolak", i+1)
		}
	}
	if h.ambilJatahKlik("10.0.0.1", 3, "ABC123") {
		t.Error("klik di atas batas per kode tetap diizinkan")
	}
	// IP lain dan brand lain punya jatah sendiri.
	if !h.ambilJatahKlik("10.0.0.2", 3, "ABC123") {
		t.Error("IP lain ikut terbatas")
	}
	if !h.ambilJatahKlik("10.0.0.1", 2, "ABC123") {
		t.Error("brand lain ikut terbatas")
	}

	// Batas semua kode per jam: kode berbeda dari satu IP berhenti di klikPerIPPerJam.
	h = NewHandler(nil)
	diizinkan := 0
	for i := 0; i < klikPerIPPerJam+10; i++ {
		if h.ambilJatahKlik("10.0.0.9", 3, "KODE"+string(rune('A'+i%26))+string(rune('A'+i/26))) {
			diizinkan++
		}
	}
	if diizinkan != klikPerIPPerJam {
		t.Errorf("klik diizinkan = %d, mau %d", diizinkan, klikPerIPPerJam)
	}
}

// Buka ulang oleh pengunjung yang sama (atau kode tak dikenal) tidak menambah baris, jadi jatahnya dikembalikan dan
// tidak menghabiskan jatah orang lain di balik IP yang sama.
func TestJatahKlikDikembalikan(t *testing.T) {
	h := NewHandler(nil)
	for i := 0; i < klikPerIPPerKodeSehari*3; i++ {
		if !h.ambilJatahKlik("10.0.0.1", 3, "ABC123") {
			t.Fatalf("klik ke-%d ditolak padahal jatah selalu dikembalikan", i+1)
		}
		h.kembalikanJatahKlik("10.0.0.1", 3, "ABC123")
	}

	// Ditolak di jatah kode: jatah per jam ikut dikembalikan.
	h = NewHandler(nil)
	for i := 0; i < klikPerIPPerKodeSehari; i++ {
		h.ambilJatahKlik("10.0.0.1", 3, "ABC123")
	}
	for i := 0; i < klikPerIPPerJam; i++ {
		h.ambilJatahKlik("10.0.0.1", 3, "ABC123") // semua ditolak
	}
	sisa := 0
	for i := 0; i < klikPerIPPerJam; i++ {
		if h.ambilJatahKlik("10.0.0.1", 3, "LAIN"+string(rune('A'+i%26))+string(rune('A'+i/26))) {
			sisa++
		}
	}
	if want := klikPerIPPerJam - klikPerIPPerKodeSehari; sisa != want {
		t.Errorf("jatah per jam tersisa = %d, mau %d", sisa, want)
	}
}
