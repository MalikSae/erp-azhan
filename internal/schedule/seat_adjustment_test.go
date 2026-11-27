package schedule

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Tidak ada booking di luar sistem: Samakan seat mengisi seat_sisa = seat_total − seat dipegang
// booking, tanpa angka bebas, dan menolak paket brand lain.
func TestSamakanSeatTx(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 20_000_000, Blocked: true})
	var total, dipegang int
	var brand int64
	if err := tx.QueryRow(`SELECT seat_total, brand_id FROM schedules WHERE id=?`, b.ScheduleID).Scan(&total, &brand); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(`SELECT COALESCE(SUM(seat_count),0) FROM bookings WHERE schedule_id=? AND is_seat_blocked=1 AND status<>'batal'`, b.ScheduleID).Scan(&dipegang); err != nil {
		t.Fatal(err)
	}
	if dipegang < 1 {
		t.Fatalf("fixture tidak memegang seat: %d", dipegang)
	}

	// Sisa seat dibuat tidak cocok (lebih kecil dan lebih besar), lalu disamakan.
	for _, rusak := range []int{0, total} {
		testdb.Exec(t, tx, `UPDATE schedules SET seat_sisa=? WHERE id=?`, rusak, b.ScheduleID)
		_, before, after, err := samakanSeatTx(ctx, tx, b.ScheduleID, &brand, 0)
		if err != nil {
			t.Fatalf("samakan dari %d: %v", rusak, err)
		}
		if before != rusak || after != total-dipegang {
			t.Fatalf("samakan dari %d: before=%d after=%d, want after=%d", rusak, before, after, total-dipegang)
		}
		var sisa int
		if err := tx.QueryRow(`SELECT seat_sisa FROM schedules WHERE id=?`, b.ScheduleID).Scan(&sisa); err != nil || sisa != total-dipegang {
			t.Fatalf("seat_sisa=%d err=%v, want %d", sisa, err, total-dipegang)
		}
	}

	// Setiap Samakan yang mengubah sisa tercatat di riwayat; menyamakan ulang tanpa selisih tidak.
	if _, _, _, err := samakanSeatTx(ctx, tx, b.ScheduleID, &brand, 0); err != nil {
		t.Fatal(err)
	}
	rows, err := tx.Query(`SELECT jenis, seat_before, seat_after FROM schedule_seat_adjustments WHERE schedule_id=? ORDER BY id`, b.ScheduleID)
	if err != nil {
		t.Fatal(err)
	}
	var riwayat []string
	for rows.Next() {
		var jenis string
		var before, after int
		if err := rows.Scan(&jenis, &before, &after); err != nil {
			t.Fatal(err)
		}
		riwayat = append(riwayat, fmt.Sprintf("%s %d→%d", jenis, before, after))
	}
	rows.Close()
	want := []string{fmt.Sprintf("samakan 0→%d", total-dipegang), fmt.Sprintf("samakan %d→%d", total, total-dipegang)}
	if fmt.Sprint(riwayat) != fmt.Sprint(want) {
		t.Fatalf("riwayat = %v, want %v", riwayat, want)
	}

	other := brand + 1000
	if _, _, _, err := samakanSeatTx(ctx, tx, b.ScheduleID, &other, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("brand lain: err=%v, want ErrNotFound", err)
	}
	if _, _, _, err := samakanSeatTx(ctx, tx, -1, nil, 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id tak ada: err=%v, want ErrNotFound", err)
	}
}

// Ringkasan seat: SeatCocok hanya bila seat_sisa = seat_total − seat dipegang booking.
func TestSeatOverviewReconcilesWithBookings(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	repo := NewRepository(db)

	var id, brandID int64
	var total, sisa, allocated int
	if err := db.QueryRow(`
		SELECT s.id, s.brand_id, s.seat_total, s.seat_sisa,
		       COALESCE((SELECT SUM(b.seat_count) FROM bookings b WHERE b.schedule_id=s.id AND b.is_seat_blocked=1 AND b.status<>'batal'),0)
		FROM schedules s ORDER BY s.id LIMIT 1`).Scan(&id, &brandID, &total, &sisa, &allocated); err != nil {
		t.Fatal(err)
	}

	o, err := repo.SeatOverview(ctx, id, nil)
	if err != nil {
		t.Fatal(err)
	}
	sesuai := total - allocated
	if sesuai < 0 {
		sesuai = 0
	}
	if o.SeatTotal != total || o.SeatSisa != sisa || o.SeatDialokasikan != allocated ||
		o.SeatSesuaiBooking != sesuai || o.SeatCocok != (sisa == sesuai) {
		t.Fatalf("ringkasan salah: %+v (total=%d sisa=%d alokasi=%d)", o, total, sisa, allocated)
	}
	if o.Adjustments == nil {
		t.Fatal("riwayat harus berupa array kosong, bukan null")
	}

	if _, err := repo.SeatOverview(ctx, id, &brandID); err != nil {
		t.Fatalf("brand pemilik: %v", err)
	}
	other := brandID + 1000
	if _, err := repo.SeatOverview(ctx, id, &other); !errors.Is(err, ErrNotFound) {
		t.Fatalf("brand lain: err=%v, want ErrNotFound", err)
	}
	if _, err := repo.SeatOverview(ctx, -1, nil); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id tak ada: err=%v, want ErrNotFound", err)
	}
}
