package payment

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Aturan status booking dari pembayaran: satu tabel untuk baru, dp, dan lunas.
func TestStatusBookingDari(t *testing.T) {
	cases := []struct {
		nama                     string
		total, terbayar, dpWajib float64
		mau                      string
	}{
		{"belum bayar", 30_000_000, 0, 5_000_000, bookingBaru},
		{"bayar di bawah DP", 30_000_000, 4_999_999, 5_000_000, bookingBaru},
		{"tepat DP", 30_000_000, 5_000_000, 5_000_000, bookingDP},
		{"DP nol, bayar sebagian", 30_000_000, 1, 0, bookingDP},
		{"DP nol, belum bayar", 30_000_000, 0, 0, bookingBaru},
		{"kurang serupiah dari lunas", 30_000_000, 29_999_999, 5_000_000, bookingDP},
		{"tepat lunas", 30_000_000, 30_000_000, 5_000_000, bookingLunas},
		{"total nol tidak pernah lunas", 0, 0, 0, bookingBaru},
		{"total nol dengan pembayaran", 0, 1000, 0, bookingDP},
	}
	for _, c := range cases {
		if got := statusBookingDari(c.total, c.terbayar, c.dpWajib); got != c.mau {
			t.Errorf("%s: dapat %s, mau %s", c.nama, got, c.mau)
		}
	}
}

// Pesan bisnis sampai ke pengguna tanpa awalan teknis, walau error-nya dibungkus berlapis.
func TestHandleRepoErrorPesanBisnisTanpaAwalan(t *testing.T) {
	for _, bisnis := range []error{ErrSeatUnavailable, ErrOverpayment, ErrBookingClosed, ErrNotVoidable, ErrVoidBelowRefund} {
		rec := httptest.NewRecorder()
		handleRepoError(rec, fmt.Errorf("payment.Create sync booking: %w", fmt.Errorf("lapis dalam: %w", bisnis)))
		var body map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if rec.Code != http.StatusConflict || body["error"] != bisnis.Error() {
			t.Errorf("%v: status %d, pesan %q", bisnis, rec.Code, body["error"])
		}
	}

	rec := httptest.NewRecorder()
	handleRepoError(rec, fmt.Errorf("payment.GetByID: %w", ErrNotFound))
	if rec.Code != http.StatusNotFound {
		t.Errorf("ErrNotFound terbungkus: status %d, mau 404", rec.Code)
	}
}
