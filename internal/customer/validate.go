package customer

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

var nonDigit = regexp.MustCompile(`\D`)

// ValidEmail menerima alamat polos (tanpa nama tampilan) sepanjang maksimal 190 karakter.
func ValidEmail(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > 190 {
		return false
	}
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s && strings.Contains(s[strings.LastIndex(s, "@"):], ".")
}

// ValidPhone: 10–15 digit setelah karakter pemisah dibuang.
func ValidPhone(s string) bool {
	if !regexp.MustCompile(`^\+?[0-9 ()./-]+$`).MatchString(strings.TrimSpace(s)) {
		return false
	}
	n := len(nonDigit.ReplaceAllString(s, ""))
	return n >= 10 && n <= 15
}

// ValidPassword: minimal 8 karakter dan maksimal 72 byte (batas bcrypt).
func ValidPassword(s string) (ok bool, msg string) {
	if utf8.RuneCountInString(s) < 8 {
		return false, "password minimal 8 karakter"
	}
	if len(s) > 72 {
		return false, "password maksimal 72 byte"
	}
	return true, ""
}

func ValidName(s string) bool {
	n := utf8.RuneCountInString(strings.TrimSpace(s))
	return n >= 2 && n <= 150
}
