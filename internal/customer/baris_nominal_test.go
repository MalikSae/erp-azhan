package customer

import (
	"context"
	"encoding/json"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Rincian booking akun (SPEK-LAYANAN-TAMBAHAN-BOOKING-2026-11-27): layanan dan diskon dikirim sebagai [] bila
// kosong, dan berisi nama + nominal bila ada.
func TestDaftarBarisNominalLayananDanDiskon(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 300, Blocked: true})

	kosong, err := daftarBarisNominal(ctx, tx, sqlLayananBooking, b.ID)
	if err != nil {
		t.Fatal(err)
	}
	if js, _ := json.Marshal(kosong); string(js) != "[]" {
		t.Fatalf("layanan kosong = %s, mau []", js)
	}

	testdb.Exec(t, tx, `INSERT INTO booking_layanan (booking_id, nama, nominal) VALUES (?, 'Kereta cepat', 3000000)`, b.ID)
	testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'Promo', 100)`, b.ID)
	layanan, err := daftarBarisNominal(ctx, tx, sqlLayananBooking, b.ID)
	if err != nil || len(layanan) != 1 || layanan[0].Nama != "Kereta cepat" || layanan[0].Nominal != 3_000_000 {
		t.Fatalf("layanan = %+v err=%v", layanan, err)
	}
	diskon, err := daftarBarisNominal(ctx, tx, sqlDiskonBooking, b.ID)
	if err != nil || len(diskon) != 1 || diskon[0].Nama != "Promo" {
		t.Fatalf("diskon = %+v err=%v", diskon, err)
	}
}
