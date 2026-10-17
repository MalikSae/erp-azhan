package customer

import (
	"strings"
	"testing"
)

func TestValidEmail(t *testing.T) {
	good := []string{"a@b.co", "budi.santoso+tag@mail.example.com"}
	bad := []string{"", "a@b", "Budi <a@b.co>", "a@b.co, c@d.co", "no-at.example.com", strings.Repeat("a", 200) + "@b.co"}
	for _, e := range good {
		if !ValidEmail(e) {
			t.Errorf("%q seharusnya valid", e)
		}
	}
	for _, e := range bad {
		if ValidEmail(e) {
			t.Errorf("%q seharusnya tidak valid", e)
		}
	}
}

func TestValidPhone(t *testing.T) {
	if !ValidPhone("0812-3456-7890") || !ValidPhone("+6281234567890") {
		t.Error("nomor lazim harus valid")
	}
	if ValidPhone("12345") || ValidPhone("abc0812345678") || ValidPhone("08123456789012345678") {
		t.Error("nomor tidak lazim harus ditolak")
	}
}

func TestValidPassword(t *testing.T) {
	if ok, _ := ValidPassword("1234567"); ok {
		t.Error("7 karakter harus ditolak")
	}
	if ok, _ := ValidPassword("12345678"); !ok {
		t.Error("8 karakter harus diterima")
	}
	if ok, _ := ValidPassword(strings.Repeat("a", 73)); ok {
		t.Error("73 byte harus ditolak (batas bcrypt 72)")
	}
}

func TestNormalizeEmail(t *testing.T) {
	if NormalizeEmail("  Budi@Mail.COM ") != "budi@mail.com" {
		t.Error("email harus huruf kecil dan tanpa spasi tepi")
	}
}

func TestSiteLinkUsesConfiguredTemplate(t *testing.T) {
	t.Setenv("SITE_URL_TEMPLATE", "http://{domain}:3100")
	got := siteLink("hana.azhan.test", "/verifikasi-email", map[string][]string{"token": {"abc"}})
	if got != "http://hana.azhan.test:3100/verifikasi-email?token=abc" {
		t.Fatalf("link salah: %s", got)
	}
}
