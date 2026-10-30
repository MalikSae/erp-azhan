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

func TestOrderCreatedMemuatKodeTautanDanIstilahBaku(t *testing.T) {
	subject, text := OrderCreated("Hana Tours", "Budi", "HNA-0001", "Umroh Reguler", "1 Januari 2099", "Rp 30.000.000", "DP minimal Rp 5.000.000", "2 Januari 2099 10:00 WIB",
		"https://hana.test/pesanan/12", []BankInfo{{Bank: "BCA", Nomor: "123", AtasNama: "PT Hana"}})
	for _, want := range []string{"HNA-0001", "https://hana.test/pesanan/12", "BCA 123 a.n. PT Hana", "Booking seat Anda di Hana Tours"} {
		if !strings.Contains(text, want) {
			t.Fatalf("isi email tanpa %q:\n%s", want, text)
		}
	}
	if !strings.Contains(subject, "berhasil dibuat") || !strings.Contains(subject, "HNA-0001") {
		t.Fatalf("subjek: %q", subject)
	}
	for _, dilarang := range []string{"pesanan", "pemesanan", "kursi", "reservasi", "jemaah"} {
		if strings.Contains(strings.ToLower(subject+strings.ReplaceAll(text, "/pesanan/", "/")), dilarang) {
			t.Fatalf("memuat kata terlarang %q", dilarang)
		}
	}
	_, tanpaBank := OrderCreated("Hana", "Budi", "K", "P", "-", "-", "-", "-", "https://x", nil)
	if strings.Contains(tanpaBank, "a.n.") || !strings.Contains(tanpaBank, "hubungi kami") {
		t.Fatalf("tanpa rekening harus memakai keterangan pengganti:\n%s", tanpaBank)
	}
}
