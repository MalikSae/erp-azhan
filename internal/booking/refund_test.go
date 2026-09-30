package booking

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Validasi input refund ditolak sebelum menyentuh database.
func TestCreateRefundValidasiInput(t *testing.T) {
	h := &Handler{}
	router := chi.NewRouter()
	router.Post("/bookings/{id}/refunds", h.CreateRefund)
	cases := []struct {
		body  string
		pesan string
	}{
		{`{"jumlah":0,"tanggal":"2026-09-01"}`, "jumlah harus bilangan bulat"},
		{`{"jumlah":1500.5,"tanggal":"2026-09-01"}`, "jumlah harus bilangan bulat"},
		{`{"jumlah":1e15,"tanggal":"2026-09-01"}`, "jumlah terlalu besar"},
		{`{"jumlah":1000,"tanggal":"2999-01-01"}`, "tanggal pengembalian tidak valid"},
		{`{"jumlah":1000,"tanggal":"01-09-2026"}`, "tanggal pengembalian tidak valid"},
		{`bukan json`, "request body tidak valid"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bookings/1/refunds", strings.NewReader(c.body)))
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), c.pesan) {
			t.Errorf("%s: kode=%d body=%s, want 400 %q", c.body, rec.Code, rec.Body.String(), c.pesan)
		}
	}
}

func TestFormatRupiah(t *testing.T) {
	for in, want := range map[float64]string{0: "Rp 0", 999: "Rp 999", 1000: "Rp 1.000", 36000000: "Rp 36.000.000", 1234567: "Rp 1.234.567", -5000: "Rp -5.000"} {
		if got := formatRupiah(in); got != want {
			t.Errorf("formatRupiah(%.0f) = %q, want %q", in, got, want)
		}
	}
}

func TestTrimOptional(t *testing.T) {
	s := func(v string) *string { return &v }
	if trimOptional(s("   "), 10) != nil || trimOptional(nil, 10) != nil {
		t.Fatal("teks kosong harus menjadi nil")
	}
	if got := *trimOptional(s(" transfer BCA "), 50); got != "transfer BCA" {
		t.Fatalf("got %q", got)
	}
	if got := *trimOptional(s("abcdef"), 3); got != "abc" {
		t.Fatalf("pemotongan: got %q", got)
	}
}
