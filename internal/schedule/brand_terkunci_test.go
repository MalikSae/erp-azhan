package schedule

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Brand paket dikunci sejak dibuat, termasuk paket tanpa booking: manasik, banner, dan data
// brand lama lain bisa merujuknya.
func TestBrandPaketTerkunci(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	sched = testdb.CloneSchedule(t, tx, sched) // paket baru tanpa booking
	var brandLain int64
	if err := tx.QueryRow(`SELECT id FROM brands WHERE id <> ? ORDER BY id LIMIT 1`, brand).Scan(&brandLain); err != nil {
		t.Skipf("butuh dua brand: %v", err)
	}
	var total int
	if err := tx.QueryRow(`SELECT seat_total FROM schedules WHERE id=?`, sched).Scan(&total); err != nil {
		t.Fatal(err)
	}
	inp := ScheduleInput{SeatTotal: total, BerangkatTanggal: "2099-01-01"}
	if err := prepareScheduleUpdate(ctx, tx, sched, &inp, nil, brandLain); !errors.Is(err, ErrConflict) {
		t.Fatalf("ganti brand paket tanpa booking: err=%v, mau ErrConflict", err)
	}
	if err := prepareScheduleUpdate(ctx, tx, sched, &inp, nil, brand); err != nil {
		t.Fatalf("brand sama: %v", err)
	}
}
