package selfbooking

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/mail"
	"regexp"
	"strings"

	"erp-azhan/api/internal/identity"
	"golang.org/x/crypto/bcrypt"
)

const TermsVersion = "booking-2026-09-28"

func resolveInvoiceToken(ctx context.Context, q invoiceReader, token string, brandID int64) (string, error) {
	if len(token) != 64 || brandID <= 0 {
		return "", ErrNotFound
	}
	if _, err := hex.DecodeString(token); err != nil {
		return "", ErrNotFound
	}
	var code string
	err := q.QueryRowContext(ctx, `SELECT b.id_booking FROM booking_checkout c JOIN bookings b ON b.id=c.booking_id JOIN schedules s ON s.id=b.schedule_id WHERE c.invoice_token=? AND s.brand_id=?`, token, brandID).Scan(&code)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return code, err
}

var ErrQuoteChanged = errors.New("harga atau DP berubah; muat ulang paket dan periksa kembali ringkasan sebelum memesan")
var ErrRequestConflict = errors.New("kunci pemesanan telah digunakan untuk data berbeda; mulai pemesanan baru")
var numericPIN = regexp.MustCompile(`^[0-9]{6}$`)
var phonePattern = regexp.MustCompile(`^\+?[0-9 () .-]+$`)
var phoneNonDigits = regexp.MustCompile(`\D`)
var requestKeyPattern = regexp.MustCompile(`^[a-fA-F0-9-]{32,64}$`)

func validPhone(value string) bool {
	digits := phoneNonDigits.ReplaceAllString(value, "")
	return phonePattern.MatchString(value) && len(digits) >= 10 && len(digits) <= 15
}

func validateCheckout(req *BookingRequest, agent bool) string {
	if req.TermsVersion != TermsVersion || !req.TermsAccepted {
		return "baca dan setujui ketentuan pemesanan terbaru"
	}
	if !requestKeyPattern.MatchString(req.RequestKey) {
		return "kunci pemesanan tidak valid, muat ulang formulir"
	}
	if req.ExpectedTotal == nil || req.ExpectedDP == nil || *req.ExpectedTotal <= 0 || *req.ExpectedDP < 0 {
		return "ringkasan harga wajib dikonfirmasi"
	}
	if !(agent && req.PIC.JamaahID != nil) {
		if len(strings.TrimSpace(req.PIC.NamaLengkap)) < 2 || len(req.PIC.NamaLengkap) > 255 {
			return "nama lengkap harus 2–255 karakter"
		}
		if !validPhone(req.PIC.NoHP) {
			return "nomor WhatsApp harus 10–15 digit"
		}
		if req.PIC.JenisKelamin != "L" && req.PIC.JenisKelamin != "P" {
			return "jenis kelamin tidak valid"
		}
	}
	if req.PIC.Email != nil && *req.PIC.Email != "" {
		if _, err := mail.ParseAddress(*req.PIC.Email); err != nil || len(*req.PIC.Email) > 255 {
			return "email tidak valid"
		}
	}
	if req.PIC.PortalPIN != "" && !numericPIN.MatchString(req.PIC.PortalPIN) {
		return "PIN portal harus 6 digit angka"
	}
	for _, p := range req.Anggota {
		if !(agent && p.JamaahID != nil) && (len(strings.TrimSpace(p.NamaLengkap)) < 2 || len(p.NamaLengkap) > 255 || (p.JenisKelamin != "L" && p.JenisKelamin != "P")) {
			return "nama atau jenis kelamin anggota tidak valid"
		}
		if p.NoHP != nil && strings.TrimSpace(*p.NoHP) != "" && !validPhone(*p.NoHP) {
			return "nomor WhatsApp anggota harus 10–15 digit"
		}
	}
	return ""
}

func hashText(value string) string {
	v := sha256.Sum256([]byte(value))
	return hex.EncodeToString(v[:])
}
func requestFingerprint(req BookingRequest, ini inisiator) string {
	req.CaptchaToken = ""
	// Store only a digest; never persist plaintext PIN or submitted identity payload.
	data, _ := json.Marshal(struct {
		Request BookingRequest
		Actor   inisiatorFingerprint
	}{req, inisiatorFingerprint{ini.jamaahLogin, ini.agenID}})
	return hashText(string(data))
}

type inisiatorFingerprint struct {
	Jamaah int64
	Agent  int64
}

func newInvoiceToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	return hex.EncodeToString(b), err
}

func replayCheckout(ctx context.Context, tx *sql.Tx, req BookingRequest, ini inisiator) (*BookingResponse, error) {
	if req.RequestKey == "" {
		return nil, nil
	}
	var fingerprint string
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT request_hash,response_json FROM booking_checkout WHERE request_key_hash=?`, hashText(req.RequestKey)).Scan(&fingerprint, &raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if fingerprint != requestFingerprint(req, ini) {
		return nil, ErrRequestConflict
	}
	var stored struct {
		Response   BookingResponse
		IssueToken bool
	}
	if err = json.Unmarshal(raw, &stored); err != nil {
		return nil, err
	}
	if stored.IssueToken {
		// A retry must not restore portal access after the PIC changes their PIN.
		var currentPIN string
		if err = tx.QueryRowContext(ctx, `SELECT COALESCE(portal_pin_hash,'') FROM jamaah WHERE id=?`, stored.Response.Jamaah.ID).Scan(&currentPIN); err != nil {
			return nil, err
		}
		if ini.jamaahLogin != stored.Response.Jamaah.ID && bcrypt.CompareHashAndPassword([]byte(currentPIN), []byte(req.PIC.PortalPIN)) != nil {
			return nil, ErrInvalidPin
		}
		stored.Response.PortalToken, err = identity.GenerateAccountPortalToken(ctx, tx, stored.Response.Jamaah.ID)
	}
	stored.Response.Replayed = true
	return &stored.Response, err
}
