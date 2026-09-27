package booking

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

func TestRecalculateTotalTransitions(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()

	calls := map[int64]int{}
	orig := komisi.OnBookingLunas
	komisi.OnBookingLunas = func(ctx context.Context, tx *sql.Tx, bookingID int64) error {
		calls[bookingID]++
		return nil
	}
	t.Cleanup(func() { komisi.OnBookingLunas = orig })

	recalc := func(bookingID int64) {
		t.Helper()
		if err := repo.recalculateTotalTx(ctx, tx, bookingID); err != nil {
			t.Fatalf("recalculateTotalTx: %v", err)
		}
	}

	// dp: tagihan 1000, dibayar 800.
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 1000, MinimalDP: 300, Blocked: true})
	testdb.AddPayment(t, tx, b.ID, 800, "confirmed")

	// Diskon 200 -> total 800 = dibayar -> lunas, hook terpanggil.
	testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'Diskon uji', 200)`, b.ID)
	recalc(b.ID)
	if got := testdb.Status(t, tx, b.ID); got != "lunas" {
		t.Fatalf("setelah diskon: status=%s, want lunas", got)
	}
	if calls[b.ID] != 1 {
		t.Fatalf("setelah diskon: hook terpanggil %d kali, want 1", calls[b.ID])
	}

	// Diskon dihapus -> total 1000 > dibayar -> mundur ke dp, hook tidak terpanggil.
	testdb.Exec(t, tx, `DELETE FROM booking_discounts WHERE booking_id=?`, b.ID)
	recalc(b.ID)
	if got := testdb.Status(t, tx, b.ID); got != "dp" {
		t.Fatalf("diskon dihapus: status=%s, want dp", got)
	}
	if calls[b.ID] != 1 {
		t.Fatalf("diskon dihapus: hook terpanggil %d kali, want 1", calls[b.ID])
	}

	// Booking 'baru' tidak disentuh rekalkulasi status.
	b2 := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 500, MinimalDP: 300})
	testdb.AddPayment(t, tx, b2.ID, 500, "confirmed")
	recalc(b2.ID)
	if got := testdb.Status(t, tx, b2.ID); got != "baru" {
		t.Fatalf("booking baru: status=%s, want baru", got)
	}
	if calls[b2.ID] != 0 {
		t.Fatalf("booking baru: hook terpanggil %d kali", calls[b2.ID])
	}
}
