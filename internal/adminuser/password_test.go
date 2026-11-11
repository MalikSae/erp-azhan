package adminuser

import (
	"errors"
	"strings"
	"testing"
)

// Code review auth dan izin, 11 November 2026: password di atas 72 byte ditolak sebagai kesalahan isian,
// bukan berakhir sebagai galat 500 dari bcrypt.
func TestPesanPanjangPassword(t *testing.T) {
	cases := map[string]bool{
		"":                      true,
		"1234567":               true,
		"12345678":              false,
		strings.Repeat("a", 72): false,
		strings.Repeat("a", 73): true,
		strings.Repeat("é", 36): false, // 72 byte
		strings.Repeat("é", 37): true,  // 74 byte
	}
	for pw, tolak := range cases {
		if got := pesanPanjangPassword(pw, "Password") != ""; got != tolak {
			t.Errorf("panjang %d byte: tolak=%v, mau %v", len(pw), got, tolak)
		}
	}
	if msg := pesanPanjangPassword("x", "Password baru"); !strings.HasPrefix(msg, "Password baru") {
		t.Errorf("pesan harus menyebut label isian: %q", msg)
	}
}

// Reset password menaikkan perm_version supaya access token lama ditolak, bukan hidup sampai kedaluwarsa.
func TestIntegrasiResetPasswordMenaikkanPermVersion(t *testing.T) {
	f := setup(t)
	id := f.create(pengguna("gotest.au.pw@local.test", nil, []string{"direktur"}))
	versi := func() int {
		return f.count(`SELECT perm_version FROM admin_users WHERE id = ?`, id)
	}
	awal := versi()
	if err := f.repo.ResetPassword(f.ctx, id, "hash-baru"); err != nil {
		t.Fatal(err)
	}
	if got := versi(); got != awal+1 {
		t.Fatalf("perm_version = %d, mau %d", got, awal+1)
	}
	hash, err := f.repo.GetPasswordHash(f.ctx, id)
	if err != nil || hash != "hash-baru" {
		t.Fatalf("hash = %q, %v", hash, err)
	}
	// Hash yang sama tetap menaikkan versi (reset ulang harus tetap mematikan sesi).
	if err := f.repo.ResetPassword(f.ctx, id, "hash-baru"); err != nil {
		t.Fatal(err)
	}
	if got := versi(); got != awal+2 {
		t.Fatalf("perm_version setelah reset ulang = %d, mau %d", got, awal+2)
	}
	if err := f.repo.ResetPassword(f.ctx, id+9_999_999, "x"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("pengguna tidak ada: err = %v, mau ErrNotFound", err)
	}
}
