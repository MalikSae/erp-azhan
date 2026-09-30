package komisi_test

import (
	"context"
	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
	"testing"
)

// Audit probes intentionally demonstrate current behavior; PASS confirms the
// finding, not correctness. Fixtures are rolled back by testdb.Tx.
func TestAuditAgenSkippedCommissionAppearsAfterReactivation(t *testing.T) {
	_, tx := testdb.Tx(t)
	schedule, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1000000), BerangkatTanggal: "2099-01-01"})
	agent := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Agent", StatusAgen: "nonaktif"})
	pax := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Pax", Direkrut: agent})
	booking, _ := testdb.NewBookingPax(t, tx, schedule, "dp", 20000000, testdb.Pax{JamaahID: pax})
	lunas(t, tx, booking)
	var before, after int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM transaksi_komisi WHERE booking_id=?`, booking).Scan(&before); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE jamaah SET status_agen='aktif' WHERE id=?`, agent)
	testdb.Exec(t, tx, `UPDATE bookings SET status='dp' WHERE id=?`, booking)
	lunas(t, tx, booking)
	if err := tx.QueryRow(`SELECT COUNT(*) FROM transaksi_komisi WHERE booking_id=?`, booking).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != 0 || after != 1 {
		t.Fatalf("probe changed: before=%d after=%d", before, after)
	}
	t.Logf("CONFIRMED: commission rows before reactivation=%d; after reactivation and re-settlement=%d", before, after)
}

func TestAuditAgenRepeatChangesAfterHistoricalPaxCancellation(t *testing.T) {
	_, tx := testdb.Tx(t)
	schedule, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1000000), BerangkatTanggal: "2099-01-01"})
	agent := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Agent", StatusAgen: "aktif"})
	pax := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Repeat", Direkrut: agent})
	old, _ := testdb.NewBookingPax(t, tx, schedule, "dp", 20000000, testdb.Pax{JamaahID: pax})
	lunas(t, tx, old)
	current, _ := testdb.NewBookingPax(t, tx, schedule, "dp", 20000000, testdb.Pax{JamaahID: pax})
	lunas(t, tx, current)
	var before, after float64
	if err := tx.QueryRow(`SELECT SUM(nominal) FROM transaksi_komisi WHERE booking_id=?`, current).Scan(&before); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='batal' WHERE booking_id=?`, old)
	testdb.Exec(t, tx, `UPDATE bookings SET status='dp' WHERE id=?`, current)
	lunas(t, tx, current)
	if err := tx.QueryRow(`SELECT SUM(nominal) FROM transaksi_komisi WHERE booking_id=?`, current).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if before != 1000000 || after != 2000000 {
		t.Fatalf("probe changed: before=%v after=%v", before, after)
	}
	t.Logf("CONFIRMED: same booking ledger grows from Rp%.0f to Rp%.0f after historical pax cancellation", before, after)
}

func TestAuditAgenCancelledSourceStillBecomesWithdrawable(t *testing.T) {
	_, tx := testdb.Tx(t)
	schedule, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1000000), BerangkatTanggal: "2099-01-01"})
	agent := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Agent", StatusAgen: "aktif"})
	pax := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Cancel", Direkrut: agent})
	booking, _ := testdb.NewBookingPax(t, tx, schedule, "dp", 20000000, testdb.Pax{JamaahID: pax})
	lunas(t, tx, booking)
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, booking)
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='batal' WHERE booking_id=?`, booking)
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2000-01-01' WHERE id=?`, schedule)
	saldo, err := komisi.HitungSaldo(context.Background(), tx, agent)
	if err != nil {
		t.Fatal(err)
	}
	if saldo.Tersedia != 1000000 {
		t.Fatalf("probe changed: %v", saldo.Tersedia)
	}
	t.Logf("CONFIRMED BUSINESS RISK: cancelled booking/pax still yields available balance Rp%.0f", saldo.Tersedia)
}
