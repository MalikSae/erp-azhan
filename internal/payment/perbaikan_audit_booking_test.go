package payment

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"

	"github.com/go-chi/chi/v5"
)

// Audit booking 2026-11-26.

// Bug 4: jumlah pembayaran admin wajib rupiah bulat dan dalam batas kolom; ditolak sebelum menyentuh database.
func TestPembayaranAdminJumlahRupiahBulat(t *testing.T) {
	h := NewHandler(nil)
	for _, c := range []struct{ jumlah, pesan string }{
		{"0.4", "bilangan bulat rupiah"},
		{"999999.6", "bilangan bulat rupiah"},
		{"1000000000000", "terlalu besar"},
	} {
		req := httptest.NewRequest(http.MethodPost, "/api/admin/bookings/1/payments", strings.NewReader(`{"jumlah":`+c.jumlah+`}`))
		rc := chi.NewRouteContext()
		rc.URLParams.Add("booking_id", "1")
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rc))
		w := httptest.NewRecorder()
		h.CreatePayment(w, req)
		if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), c.pesan) {
			t.Errorf("jumlah %s: %d %s, want 400 berisi %q", c.jumlah, w.Code, w.Body.String(), c.pesan)
		}
	}
}

// Bug 5: konfirmasi transfer portal yang masih menunggu ikut dihitung di dalam kunci booking, sehingga kiriman
// kedua yang melebihi sisa tagihan ditolak.
func TestPembayaranPortalMenungguIkutDihitung(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 100, Blocked: true})
	testdb.Exec(t, tx, `UPDATE bookings SET seat_hold_expires_at=DATE_ADD(UTC_TIMESTAMP(), INTERVAL 1 DAY) WHERE id=?`, b.ID)
	testdb.AddPayment(t, tx, b.ID, 800, "pending")

	_, err := createTx(ctx, tx, b.ID, &CreatePaymentRequest{Jumlah: 300, Source: SourcePortal}, nil)
	if !errors.Is(err, ErrOverpayment) {
		t.Fatalf("kiriman 300 dengan 800 menunggu dari tagihan 1000: err = %v, want ErrOverpayment", err)
	}
	if _, err := createTx(ctx, tx, b.ID, &CreatePaymentRequest{Jumlah: 200, Source: SourcePortal}, nil); err != nil {
		t.Fatalf("kiriman 200 (pas sisa): %v", err)
	}
}

// Bug 8: verifikasi yang kalah balapan dengan keputusan berbeda ditolak, bukan dijawab berhasil.
func TestVerifikasiSudahDiputuskanAdminLain(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 100, Blocked: true})
	p := testdb.AddPayment(t, tx, b.ID, 300, "confirmed")

	if err := updateStatusTx(ctx, tx, p, StatusRejected, nil, 1); !errors.Is(err, ErrSudahDiputuskan) {
		t.Fatalf("tolak pembayaran yang sudah dikonfirmasi: err = %v, want ErrSudahDiputuskan", err)
	}
	if err := updateStatusTx(ctx, tx, p, StatusConfirmed, nil, 1); err != nil {
		t.Fatalf("keputusan sama tetap idempoten: %v", err)
	}
}
