package payment

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// MP-02: DP per pax yang tersimpan saat booking dibuat tidak terpengaruh
// perubahan minimal_dp paket setelahnya.
func TestSyncMemakaiSnapshotDPBooking(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 20000000, MinimalDP: 5000000, Blocked: true})
	testdb.Exec(t, tx, `UPDATE bookings SET dp_per_pax=5000000 WHERE id=?`, b.ID)
	testdb.AddPayment(t, tx, b.ID, 5000000, "confirmed")

	// DP paket dinaikkan setelah booking dibuat.
	testdb.Exec(t, tx, `UPDATE schedules SET minimal_dp=8000000 WHERE id=?`, b.ScheduleID)
	if err := repo.syncBookingStatusTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	if got := testdb.Status(t, tx, b.ID); got != "dp" {
		t.Fatalf("snapshot DP 5 jt, dibayar 5 jt: status=%s, want dp", got)
	}

	// Tanpa snapshot (booking lama sebelum 070) tetap membaca nilai paket.
	testdb.Exec(t, tx, `UPDATE bookings SET dp_per_pax=NULL WHERE id=?`, b.ID)
	if err := repo.syncBookingStatusTx(context.Background(), tx, b.ID); err != nil {
		t.Fatal(err)
	}
	if got := testdb.Status(t, tx, b.ID); got != "baru" {
		t.Fatalf("tanpa snapshot, DP paket 8 jt: status=%s, want baru", got)
	}
}

// MP-02: backfill 070 membekukan DP efektif untuk booking non-draft tanpa snapshot.
func TestBackfillDPMigration(t *testing.T) {
	_, tx := testdb.Tx(t)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 300})
	testdb.Exec(t, tx, `UPDATE bookings SET dp_per_pax=NULL WHERE id=?`, b.ID)

	_, file, _, _ := runtime.Caller(0)
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "migrations", "070_booking_dp_per_pax.sql"))
	if err != nil {
		t.Fatal(err)
	}
	src := strings.ReplaceAll(string(raw), "\r\n", "\n")
	stmt := strings.TrimSpace(src[strings.Index(src, "UPDATE bookings b"):])
	testdb.Exec(t, tx, strings.TrimSuffix(stmt, ";"))

	var dp float64
	if err := tx.QueryRow(`SELECT dp_per_pax FROM bookings WHERE id=?`, b.ID).Scan(&dp); err != nil {
		t.Fatal(err)
	}
	if dp != 300 {
		t.Fatalf("dp_per_pax=%.0f, want 300 (minimal_dp paket)", dp)
	}
}
