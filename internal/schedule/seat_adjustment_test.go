package schedule

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// SM-03: penyesuaian kursi manual tercatat di riwayat beserta pelakunya, dan
// penyesuaian yang ditolak tidak meninggalkan catatan.
func TestUpdateSeatTxRecordsAdjustmentWithActor(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, _ := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	var actor int64
	if err := tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	// testdb.Schedule menambah kuota 100; setel sisa ke batas atas yang sah
	// (kapasitas dikurangi kursi yang dipegang booking) agar uji bebas dari data lain.
	var current int
	if err := tx.QueryRow(`SELECT seat_total - COALESCE((SELECT SUM(seat_count) FROM bookings WHERE schedule_id=? AND is_seat_blocked=1 AND status<>'batal'),0) FROM schedules WHERE id=?`, sched, sched).Scan(&current); err != nil {
		t.Fatal(err)
	}
	if current < 1 {
		t.Skip("paket fixture tidak punya kursi bebas")
	}
	testdb.Exec(t, tx, `UPDATE schedules SET seat_sisa=? WHERE id=?`, current, sched)
	target := current - 1

	rows := func() int {
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM schedule_seat_adjustments WHERE schedule_id=?`, sched).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	// Alasan kosong dan nilai sebelumnya basi: ditolak, tanpa catatan.
	if _, _, err := updateSeatTx(ctx, tx, sched, target, &current, nil, "  ", actor); !errors.Is(err, ErrConflict) {
		t.Fatalf("alasan kosong: err=%v, want ErrConflict", err)
	}
	stale := current + 5
	if _, _, err := updateSeatTx(ctx, tx, sched, target, &stale, nil, "koreksi", actor); !errors.Is(err, ErrConflict) {
		t.Fatalf("nilai sebelumnya basi: err=%v, want ErrConflict", err)
	}
	if n := rows(); n != 0 {
		t.Fatalf("penyesuaian ditolak meninggalkan %d catatan", n)
	}

	if _, before, err := updateSeatTx(ctx, tx, sched, target, &current, nil, "  koreksi kursi terjual offline  ", actor); err != nil || before != current {
		t.Fatalf("penyesuaian sah: before=%d err=%v", before, err)
	}
	var gotActor int64
	var seatBefore, seatAfter int
	var reason string
	if err := tx.QueryRow(`SELECT admin_user_id, seat_before, seat_after, reason FROM schedule_seat_adjustments WHERE schedule_id=?`, sched).Scan(&gotActor, &seatBefore, &seatAfter, &reason); err != nil {
		t.Fatal(err)
	}
	if gotActor != actor || seatBefore != current || seatAfter != target || reason != "koreksi kursi terjual offline" {
		t.Fatalf("catatan: actor=%d before=%d after=%d reason=%q", gotActor, seatBefore, seatAfter, reason)
	}
	var sisa int
	if err := tx.QueryRow(`SELECT seat_sisa FROM schedules WHERE id=?`, sched).Scan(&sisa); err != nil || sisa != target {
		t.Fatalf("seat_sisa=%d err=%v, want %d", sisa, err, target)
	}
}

// SM-02: ringkasan kursi untuk halaman rekonsiliasi (hanya baca).
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
	if o.SeatTotal != total || o.SeatSisa != sisa || o.SeatDialokasikan != allocated ||
		o.SeatDiLuarBooking != total-sisa-allocated || o.SeatSisaMaks != total-allocated {
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
