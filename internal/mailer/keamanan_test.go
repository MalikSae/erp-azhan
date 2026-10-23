package mailer

import (
	"bytes"
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"testing"
)

const payloadNama = "Budi,\n\nAkun Anda akan diblokir: https://evil.example\n\n"

// Butir 6: nilai yang disisipkan ke email tidak bisa membentuk baris/paragraf baru.
func TestTemplateMenormalkanNilaiSisipan(t *testing.T) {
	cek := func(nama, text string) {
		t.Helper()
		baris := strings.Split(text, "\n")
		if baris[0] != "Halo Budi, Akun Anda akan diblokir: https://evil.example," {
			t.Fatalf("%s: baris sapaan = %q", nama, baris[0])
		}
		for _, l := range baris[1:] {
			if strings.Contains(l, "evil.example") {
				t.Fatalf("%s: payload membentuk baris sendiri: %q", nama, l)
			}
		}
	}
	_, v := VerifyEmail("Hana", payloadNama, "https://hana.test/v?token=a")
	cek("verifikasi", v)
	_, r := ResetPassword("Hana", payloadNama, "https://hana.test/r?token=a")
	cek("reset", r)
	subj, o := OrderCreated("Hana", payloadNama, "K-1\r\nBcc: x@y", "Paket\nBaru", "-", "-", "-", "-", "https://hana.test/b/1", []BankInfo{{Bank: "BCA\n", Nomor: "1", AtasNama: "PT\nHana"}})
	cek("booking", o)
	if strings.ContainsAny(subj, "\r\n") || !strings.Contains(o, "BCA 1 a.n. PT Hana") {
		t.Fatalf("subjek/rekening tidak dinormalkan: %q\n%s", subj, o)
	}
	if got := inline(strings.Repeat("a", 500), maxInlineNama); len([]rune(got)) != maxInlineNama {
		t.Fatalf("panjang tidak dibatasi: %d", len(got))
	}
}

// Butir 7h: produksi tanpa SMTP tidak memakai LogSender dan tidak menulis token ke log.
func TestProduksiTanpaSMTPTidakMenulisToken(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	t.Setenv("APP_ENV", "production")
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	s := NewFromEnv()
	if _, ok := s.(LogSender); ok {
		t.Fatal("produksi tanpa SMTP tidak boleh memakai LogSender")
	}
	msg := Message{To: "a@b.co", Subject: "Atur ulang", Text: "https://x/reset?token=TOKEN-RAHASIA"}
	if err := s.Send(context.Background(), msg); !errors.Is(err, ErrMailerDisabled) {
		t.Fatalf("err = %v", err)
	}
	if err := (LogSender{}).Send(context.Background(), msg); !errors.Is(err, ErrMailerDisabled) {
		t.Fatalf("LogSender di produksi: err = %v", err)
	}
	if strings.Contains(buf.String(), "TOKEN-RAHASIA") {
		t.Fatalf("token tertulis ke log: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "SMTP_HOST KOSONG DI PRODUKSI") {
		t.Fatalf("peringatan keras tidak muncul: %s", buf.String())
	}
}
