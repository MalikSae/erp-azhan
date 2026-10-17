// Package mailer mengirim email transaksional (verifikasi email, reset password,
// pesanan dibuat). Tanpa SMTP_HOST, email hanya ditulis ke log sehingga alur
// akun bisa dicoba di dev tanpa server email.
package mailer

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"mime"
	"net"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type Message struct {
	To      string
	Subject string
	Text    string
}

type Sender interface {
	Send(ctx context.Context, m Message) error
}

// NewFromEnv memilih SMTP bila SMTP_HOST terisi, selain itu pengirim log.
//
//	SMTP_HOST, SMTP_PORT (default 587; 465 = TLS langsung), SMTP_USER, SMTP_PASSWORD, SMTP_FROM
func NewFromEnv() Sender {
	host := strings.TrimSpace(os.Getenv("SMTP_HOST"))
	if host == "" {
		log.Println("[INFO] mailer: SMTP_HOST kosong, email hanya ditulis ke log")
		return LogSender{}
	}
	port := strings.TrimSpace(os.Getenv("SMTP_PORT"))
	if port == "" {
		port = "587"
	}
	from := strings.TrimSpace(os.Getenv("SMTP_FROM"))
	if from == "" {
		from = strings.TrimSpace(os.Getenv("SMTP_USER"))
	}
	return &SMTPSender{
		Host: host, Port: port,
		User: strings.TrimSpace(os.Getenv("SMTP_USER")), Password: os.Getenv("SMTP_PASSWORD"),
		From: from,
	}
}

// LogSender menulis email ke log (dev). Isi email memuat token sekali pakai,
// jadi jangan dipakai di produksi.
type LogSender struct{}

func (LogSender) Send(_ context.Context, m Message) error {
	log.Printf("[MAIL-DEV] to=%s subject=%q\n%s", clean(m.To), clean(m.Subject), m.Text)
	return nil
}

type SMTPSender struct {
	Host, Port, User, Password, From string
}

// clean membuang CR/LF agar nilai tidak bisa menyisipkan header tambahan.
func clean(s string) string {
	return strings.TrimSpace(strings.NewReplacer("\r", "", "\n", "").Replace(s))
}

func (s *SMTPSender) Send(ctx context.Context, m Message) error {
	to, from := clean(m.To), clean(s.From)
	if to == "" || from == "" {
		return fmt.Errorf("mailer: alamat pengirim atau tujuan kosong")
	}
	body := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + mime.QEncoding.Encode("utf-8", clean(m.Subject)),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=utf-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		strings.ReplaceAll(m.Text, "\r\n", "\n"),
	}, "\r\n")

	addr := net.JoinHostPort(s.Host, s.Port)
	var auth smtp.Auth
	if s.User != "" {
		auth = smtp.PlainAuth("", s.User, s.Password, s.Host)
	}
	done := make(chan error, 1)
	go func() {
		if s.Port == "465" {
			conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.Host})
			if err != nil {
				done <- err
				return
			}
			c, err := smtp.NewClient(conn, s.Host)
			if err != nil {
				done <- err
				return
			}
			defer c.Close()
			if auth != nil {
				if err = c.Auth(auth); err != nil {
					done <- err
					return
				}
			}
			if err = c.Mail(from); err != nil {
				done <- err
				return
			}
			if err = c.Rcpt(to); err != nil {
				done <- err
				return
			}
			w, err := c.Data()
			if err != nil {
				done <- err
				return
			}
			if _, err = w.Write([]byte(body)); err != nil {
				done <- err
				return
			}
			done <- w.Close()
			return
		}
		done <- smtp.SendMail(addr, auth, from, []string{to}, []byte(body))
	}()
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(20 * time.Second):
		return fmt.Errorf("mailer: timeout mengirim email")
	}
}
