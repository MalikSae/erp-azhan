package booking

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// JB-05: pesan bisnis tetap sampai ke pengguna, error internal tidak membocorkan SQL.
func TestHandleRepoErrorTidakMembocorkanSQL(t *testing.T) {
	cases := []struct {
		nama     string
		err      error
		kode     int
		harus    string
		dilarang string
	}{
		{"bisnis", bizErr("Jamaah %s sudah terdaftar pada booking %s di jadwal ini", "A", "TP1234"), http.StatusBadRequest, "sudah terdaftar pada booking TP1234", ""},
		{"booking batal", ErrBookingBatal, http.StatusConflict, "sudah dibatalkan", ""},
		{"sql mentah", fmt.Errorf("booking.ListPax: %w", errors.New("Error 1054 (42S22): Unknown column 'bp.x'")), http.StatusInternalServerError, "kesalahan internal", "Error 1054"},
		{"overflow", fmt.Errorf("booking.AddAddon insert: %w", errors.New("Error 1264 (22003): Out of range value")), http.StatusInternalServerError, "kesalahan internal", "Error 1264"},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		handleRepoError(rec, c.err)
		body := rec.Body.String()
		if rec.Code != c.kode || !strings.Contains(body, c.harus) || (c.dilarang != "" && strings.Contains(body, c.dilarang)) {
			t.Errorf("%s: kode=%d body=%s, want %d berisi %q tanpa %q", c.nama, rec.Code, body, c.kode, c.harus, c.dilarang)
		}
	}
}

// JB-07: diskon/perubahan tidak boleh membuat tagihan di bawah pembayaran terkonfirmasi.
func TestEnsureTotalCoversPaidTx(t *testing.T) {
	db, tx := testdb.Tx(t)
	ctx := context.Background()
	repo := NewRepository(db)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 1000, MinimalDP: 300, Blocked: true})
	testdb.AddPayment(t, tx, b.ID, 600, "confirmed")

	testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'pas', 400)`, b.ID)
	if err := repo.recalculateTotalTx(ctx, tx, b.ID); err != nil {
		t.Fatal(err)
	}
	if err := ensureTotalCoversPaidTx(ctx, tx, b.ID); err != nil {
		t.Fatalf("total 600 = dibayar 600: %v", err)
	}

	testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'lebih', 1)`, b.ID)
	if err := repo.recalculateTotalTx(ctx, tx, b.ID); err != nil {
		t.Fatal(err)
	}
	var bisnis *BusinessError
	if err := ensureTotalCoversPaidTx(ctx, tx, b.ID); !errors.As(err, &bisnis) {
		t.Fatalf("total 599 < dibayar 600: err=%v, want BusinessError", err)
	}
}

// JB-06: booking batal tidak boleh diubah tagihannya.
func TestLockNotBatalTx(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	aktif := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 1000, Blocked: true})
	if err := lockNotBatalTx(ctx, tx, aktif.ID); err != nil {
		t.Fatalf("booking dp: %v", err)
	}
	batal := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "batal", TotalHarga: 1000})
	if err := lockNotBatalTx(ctx, tx, batal.ID); !errors.Is(err, ErrBookingBatal) {
		t.Fatalf("booking batal: err=%v, want ErrBookingBatal", err)
	}
	if err := lockNotBatalTx(ctx, tx, -1); !errors.Is(err, ErrNotFound) {
		t.Fatalf("booking tidak ada: err=%v, want ErrNotFound", err)
	}
}
