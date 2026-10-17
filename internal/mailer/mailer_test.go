package mailer

import (
	"context"
	"strings"
	"testing"
)

func TestCleanStripsHeaderInjection(t *testing.T) {
	got := clean("a@b.com\r\nBcc: x@y.com")
	if strings.ContainsAny(got, "\r\n") {
		t.Fatalf("clean masih memuat CR/LF: %q", got)
	}
}

func TestNewFromEnvWithoutHostIsLogSender(t *testing.T) {
	t.Setenv("SMTP_HOST", "")
	if _, ok := NewFromEnv().(LogSender); !ok {
		t.Fatal("tanpa SMTP_HOST harus memakai LogSender")
	}
}

func TestSMTPSenderRejectsEmptyRecipient(t *testing.T) {
	s := &SMTPSender{Host: "127.0.0.1", Port: "1", From: "noreply@example.com"}
	if err := s.Send(context.Background(), Message{To: "  ", Subject: "x", Text: "y"}); err == nil {
		t.Fatal("penerima kosong harus ditolak")
	}
}

func TestTemplatesContainLinks(t *testing.T) {
	_, text := ResetPassword("Hana", "Budi", "https://hana.test/reset?token=abc")
	if !strings.Contains(text, "https://hana.test/reset?token=abc") {
		t.Fatal("link reset tidak ada di isi email")
	}
}
