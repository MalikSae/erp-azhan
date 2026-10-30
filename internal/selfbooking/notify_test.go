package selfbooking

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"erp-azhan/api/internal/mailer"
)

type mailTiruan struct {
	mu   sync.Mutex
	msgs []mailer.Message
	err  error
}

func (m *mailTiruan) Send(_ context.Context, msg mailer.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.msgs = append(m.msgs, msg)
	return m.err
}

func (m *mailTiruan) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.msgs)
}

func handlerEmail(t *testing.T, m mailer.Sender, replayed bool) *Handler {
	t.Helper()
	t.Setenv("SITE_URL_TEMPLATE", "https://{domain}")
	h := newTestHandler(t, "", 0)
	h.mail = m
	h.loadInfo = func(context.Context, int64, int64) (*mailInfo, error) {
		return &mailInfo{BrandName: "Hana Tours", Domain: "hana.test", Paket: "Umroh Reguler", Berangkat: time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC)}, nil
	}
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		return &BookingResponse{Replayed: replayed, Status: "success", Booking: BookingSummary{
			BookingID: 42, BookingCode: "HNA-0042", TotalHarga: 30000000, MinimalDP: 5000000,
			SeatHoldExpiresAt: "2098-12-31T17:00:00Z",
		}, BankAccounts: []BankAccountInfo{{BankName: "BCA", AccountNumber: "123", AccountHolder: "PT Hana"}}}, nil
	}
	return h
}

const bodyCheckout = `{"schedule_id":10,"pic":{"nama_lengkap":"Uji","jenis_kelamin":"L","room_type":"Quad","nik":"3201010101010001","no_hp":"081234567890"},"request_key":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","terms_version":"` + TermsVersion + `","terms_accepted":true,"expected_total":30000000,"expected_dp":5000000}`

func TestEmailBookingDikirimHanyaUntukBookingBaru(t *testing.T) {
	m := &mailTiruan{}
	h := handlerEmail(t, m, false)
	if rec := postBook(h, bodyCheckout, akunHandlerUji); rec.Code != http.StatusOK {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	h.mailWG.Wait()
	if m.count() != 1 {
		t.Fatalf("email terkirim %d kali, want 1", m.count())
	}
	msg := m.msgs[0]
	if msg.To != "uji@example.com" {
		t.Fatalf("tujuan: %q", msg.To)
	}
	for _, want := range []string{"HNA-0042", "https://hana.test/pesanan/42", "Umroh Reguler", "1 Januari 2099", "Rp 30.000.000", "DP minimal Rp 5.000.000", "WIB", "BCA 123 a.n. PT Hana"} {
		if !strings.Contains(msg.Text, want) {
			t.Fatalf("email tanpa %q:\n%s", want, msg.Text)
		}
	}
}

func TestEmailBookingTidakDikirimUntukJawabanRetryIdempoten(t *testing.T) {
	m := &mailTiruan{}
	h := handlerEmail(t, m, true)
	if rec := postBook(h, bodyCheckout, akunHandlerUji); rec.Code != http.StatusOK {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	h.mailWG.Wait()
	if m.count() != 0 {
		t.Fatalf("retry idempoten mengirim email %d kali", m.count())
	}
}

func TestKegagalanKirimEmailTidakMembatalkanBooking(t *testing.T) {
	m := &mailTiruan{err: errors.New("smtp down")}
	h := handlerEmail(t, m, false)
	rec := postBook(h, bodyCheckout, akunHandlerUji)
	h.mailWG.Wait()
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "HNA-0042") || strings.Contains(rec.Body.String(), "smtp") {
		t.Fatalf("booking terpengaruh kegagalan email: %d %s", rec.Code, rec.Body.String())
	}
}

func TestEmailBookingTidakDikirimBilaBookingGagal(t *testing.T) {
	m := &mailTiruan{}
	h := handlerEmail(t, m, false)
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) { return nil, ErrSeatHabis }
	if rec := postBook(h, bodyCheckout, akunHandlerUji); rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d", rec.Code)
	}
	h.mailWG.Wait()
	if m.count() != 0 {
		t.Fatal("email terkirim untuk booking gagal")
	}
}

func TestBuildBookingEmailBayarPenuhDanTanpaDomain(t *testing.T) {
	t.Setenv("SITE_URL_TEMPLATE", "")
	resp := &BookingResponse{Booking: BookingSummary{BookingID: 7, BookingCode: "K-7", TotalHarga: 1000000, MinimalDP: 1000000, SeatHoldExpiresAt: "2099-01-01T03:30:00Z"}}
	msg, ok := buildBookingEmail("a@b.com", "Budi", &mailInfo{BrandName: "Hana", Domain: "hana.id", Paket: "P"}, resp)
	if !ok || !strings.Contains(msg.Text, "Bayar penuh Rp 1.000.000") || !strings.Contains(msg.Text, "https://hana.id/pesanan/7") || !strings.Contains(msg.Text, "1 Januari 2099 10:30 WIB") {
		t.Fatalf("ok=%v\n%s", ok, msg.Text)
	}
	if _, ok := buildBookingEmail("a@b.com", "Budi", &mailInfo{BrandName: "Hana"}, resp); ok {
		t.Fatal("tanpa domain harus dilewati")
	}
	if _, ok := buildBookingEmail(" ", "Budi", &mailInfo{Domain: "x"}, resp); ok {
		t.Fatal("tanpa alamat harus dilewati")
	}
}
