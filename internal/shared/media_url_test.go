package shared

import "testing"

func TestValidMediaURL(t *testing.T) {
	for raw, want := range map[string]bool{
		"":                                    true,
		"/uploads/hotel-logos/fairmont.png":   true,
		"/api/admin/media/dokumen-jamaah/a.pdf": true,
		"https://www.youtube.com/watch?v=abc": true,
		"http://cdn.contoh.id/brosur.jpg":     true,
		"javascript:alert(1)":                 false,
		"data:text/html;base64,PHNjcmlwdD4=":  false,
		"bukan-url":                           false,
		"https://":                            false,
		"/uploads/../../.env":                 false,
		"/etc/passwd":                         false,
	} {
		if got := ValidMediaURL(raw); got != want {
			t.Errorf("ValidMediaURL(%q) = %v, want %v", raw, got, want)
		}
	}
}
