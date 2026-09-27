package booking

import (
	"context"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// B5 / D4: kredit cashback dipakai sebagai potongan booking berikutnya,
// dibatasi kredit dan sisa tagihan, dan kembali utuh bila diskonnya dihapus.
func TestPakaiKreditCashback(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	var adminID int64
	if err := tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	jamaah := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PenerimaCashback"})
	lain := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "BukanPax"})

	// Booking lama menghasilkan cashback 500rb untuk jamaah.
	lama, paxLama := testdb.NewBookingPax(t, tx, sched, "lunas", 0, testdb.Pax{JamaahID: jamaah})
	testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, jamaah_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal)
		VALUES (?,?,?,?,?,'cashback',500000)`, brand, jamaah, jamaah, lama, paxLama[0])

	// Booking baru: tagihan 10 juta, status dp.
	baru, paxBaru := testdb.NewBookingPax(t, tx, sched, "dp", 10_000_000, testdb.Pax{JamaahID: jamaah})
	testdb.Exec(t, tx, `UPDATE booking_pax SET harga_pax=10000000 WHERE id=?`, paxBaru[0])

	kredit := func() float64 {
		t.Helper()
		s, err := komisi.HitungSaldo(ctx, tx, jamaah)
		if err != nil {
			t.Fatal(err)
		}
		return s.KreditCashback
	}
	pakai := func(jamaahID int64, nominal float64) error {
		return pakaiCashbackTx(ctx, tx, repo, baru, &brand, adminID, PakaiCashbackRequest{JamaahID: jamaahID, Nominal: nominal})
	}

	info, err := kreditCashbackBooking(ctx, tx, baru, &brand)
	if err != nil || len(info.Pax) != 1 || info.Pax[0].KreditCashback != 500000 || !info.BisaDipakai {
		t.Fatalf("info kredit = %+v, %v", info, err)
	}
	lainBrand := brand + 999_999
	if _, err := kreditCashbackBooking(ctx, tx, baru, &lainBrand); err != ErrNotFound {
		t.Fatalf("brand lain: %v", err)
	}
	if err := pakai(jamaah, 600000); err != ErrCashbackMelebihiKredit {
		t.Fatalf("melebihi kredit: %v", err)
	}
	if err := pakai(lain, 100000); err != ErrCashbackBukanPax {
		t.Fatalf("bukan pax: %v", err)
	}
	if err := pakai(jamaah, 300000); err != nil {
		t.Fatalf("pakai 300rb: %v", err)
	}
	var total float64
	tx.QueryRow(`SELECT total_harga FROM bookings WHERE id=?`, baru).Scan(&total)
	if total != 9_700_000 || kredit() != 200000 {
		t.Fatalf("setelah pakai: total=%v kredit=%v", total, kredit())
	}

	// Hapus diskon lewat fitur diskon booking: kredit kembali (FK cascade 063).
	testdb.Exec(t, tx, `DELETE FROM booking_discounts WHERE booking_id=?`, baru)
	if k := kredit(); k != 500000 {
		t.Fatalf("setelah hapus diskon kredit=%v, want 500000", k)
	}

	// Batas sisa tagihan.
	testdb.AddPayment(t, tx, baru, 9_900_000, "confirmed")
	if err := pakai(jamaah, 200000); err != ErrCashbackMelebihiTagihan {
		t.Fatalf("melebihi tagihan: %v", err)
	}

	// Booking lunas tidak bisa dipotong lagi.
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, baru)
	if err := pakai(jamaah, 50000); err != ErrCashbackStatusBooking {
		t.Fatalf("booking lunas: %v", err)
	}
}
