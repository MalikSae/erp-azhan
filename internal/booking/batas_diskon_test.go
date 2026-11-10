package booking

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestMelebihiBatasDiskon(t *testing.T) {
	cases := []struct {
		harga, diskon float64
		want          bool
	}{
		{30_000_000, 3_000_000, false}, // tepat 10%
		{30_000_000, 3_000_001, true},
		{30_000_000, 0, false},
		{0, 1, true},
	}
	for _, c := range cases {
		if got := melebihiBatasDiskon(c.harga, c.diskon); got != c.want {
			t.Errorf("harga %.0f diskon %.0f: got %v, want %v", c.harga, c.diskon, got, c.want)
		}
	}
}

// T-02: total diskon di atas 10% harga pax ditolak tanpa izin diskon besar.
func TestCekBatasDiskonTx(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 30_000_000, MinimalDP: 5_000_000, Blocked: true})
	tambah := func(nominal float64) {
		testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'Uji', ?)`, b.ID, nominal)
	}
	tambah(2_000_000)
	if err := cekBatasDiskonTx(ctx, tx, b.ID); err != nil {
		t.Fatalf("diskon 2 jt dari 30 jt: %v", err)
	}
	tambah(1_000_000) // total tepat 10%
	if err := cekBatasDiskonTx(ctx, tx, b.ID); err != nil {
		t.Fatalf("total tepat 10%%: %v", err)
	}
	tambah(1) // total melewati 10%
	var biz *BusinessError
	if err := cekBatasDiskonTx(ctx, tx, b.ID); !errors.As(err, &biz) {
		t.Fatalf("total di atas 10%%: err = %v, want BusinessError", err)
	}
}
