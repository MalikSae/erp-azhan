package booking

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// JB-04: jamaah yang aktif di booking lain pada jadwal yang sama ditolak;
// booking itu sendiri (finalisasi draft) dan booking batal tidak dihitung.
func TestEnsureJamaahFreeOnSchedule(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Duplikat"})
	y := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Bebas"})
	aktif, _ := testdb.NewBookingPax(t, tx, sched, "baru", 1000, testdb.Pax{JamaahID: x})

	err := ensureJamaahFreeOnSchedule(ctx, tx, sched, 0, []int64{y, x})
	if err == nil || !strings.Contains(err.Error(), "Duplikat (uji)") {
		t.Fatalf("jamaah aktif di booking lain: err=%v, want ditolak", err)
	}
	if err := ensureJamaahFreeOnSchedule(ctx, tx, sched, aktif, []int64{x}); err != nil {
		t.Fatalf("booking sendiri dikecualikan: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, aktif)
	if err := ensureJamaahFreeOnSchedule(ctx, tx, sched, 0, []int64{x}); err != nil {
		t.Fatalf("booking batal tidak dihitung: %v", err)
	}
}

// JB-02: migrasi 068 menyelaraskan seat_count dengan pax reguler aktif.
func TestBackfillSeatCountMigration(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	a := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "A"})
	b := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "B"})
	c := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "C"})
	id, _ := testdb.NewBookingPax(t, tx, sched, "baru", 3000,
		testdb.Pax{JamaahID: a}, testdb.Pax{JamaahID: b}, testdb.Pax{JamaahID: c, Batal: true})
	// Kondisi lama: booking admin tersimpan dengan default seat_count=1.
	testdb.Exec(t, tx, `UPDATE bookings SET seat_count=1 WHERE id=?`, id)

	_, file, _, _ := runtime.Caller(0)
	sqlBytes, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "migrations", "068_backfill_booking_seat_count.sql"))
	if err != nil {
		t.Fatal(err)
	}
	src := string(sqlBytes)
	stmt := strings.TrimSpace(src[strings.Index(src, "UPDATE bookings"):])
	testdb.Exec(t, tx, strings.TrimSuffix(stmt, ";"))

	var seatCount int
	if err := tx.QueryRow(`SELECT seat_count FROM bookings WHERE id=?`, id).Scan(&seatCount); err != nil {
		t.Fatal(err)
	}
	if seatCount != 2 {
		t.Fatalf("seat_count=%d, want 2 (pax reguler aktif)", seatCount)
	}
}
