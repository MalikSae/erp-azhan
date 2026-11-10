package komisi_test

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// snapshot membaca snapshot komisi booking setelah SnapshotNominal dijalankan.
func snapshot(t *testing.T, tx *sql.Tx, bookingID int64) (langsung, pembinaan sql.NullFloat64) {
	t.Helper()
	if err := komisi.SnapshotNominal(context.Background(), tx, bookingID); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if err := tx.QueryRow(`SELECT snapshot_komisi_langsung, snapshot_bonus_pembinaan FROM bookings WHERE id=?`, bookingID).
		Scan(&langsung, &pembinaan); err != nil {
		t.Fatalf("baca snapshot: %v", err)
	}
	return langsung, pembinaan
}

// Audit menu Paket T-01: nominal komisi jadwal yang kosong mengikuti bawaan brand; nilai jadwal (termasuk 0)
// selalu menang; bawaan brand 0 berarti brand belum ikut program.
func TestKomisiBawaanBrandDanOverrideJadwal(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X"})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})

	// Jadwal kosong, brand 0: tidak ikut program.
	l, p := snapshot(t, tx, b)
	if l.Valid || p.Valid {
		t.Fatalf("brand 0 dan jadwal kosong harus NULL, dapat %v %v", l, p)
	}

	// Jadwal kosong, brand diisi: ikut brand.
	testdb.Exec(t, tx, `UPDATE brands SET ujroh_closing=500000, ujroh_pembinaan=100000 WHERE id=?`, brand)
	l, p = snapshot(t, tx, b)
	if l.Float64 != 500_000 || p.Float64 != 100_000 {
		t.Fatalf("ikut brand: dapat %v %v", l, p)
	}

	// Override per kolom: komisi langsung jadwal menang, bonus pembinaan tetap ikut brand.
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=750000 WHERE id=?`, sched)
	l, p = snapshot(t, tx, b)
	if l.Float64 != 750_000 || p.Float64 != 100_000 {
		t.Fatalf("override langsung: dapat %v %v", l, p)
	}

	// Jadwal 0 = sengaja tanpa komisi walau brand punya bawaan: snapshot 0, bukan NULL, dan ledger kosong.
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=0, nominal_bonus_pembinaan=0 WHERE id=?`, sched)
	l, p = snapshot(t, tx, b)
	if !l.Valid || l.Float64 != 0 || !p.Valid || p.Float64 != 0 {
		t.Fatalf("jadwal 0: dapat %v %v", l, p)
	}
}

// Booking dari jadwal kosong mencatat komisi senilai bawaan brand saat lunas, dan perubahan bawaan brand
// setelah booking dibuat tidak mengubahnya (D9).
func TestKomisiBawaanBrandTercatatSaatLunas(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE brands SET ujroh_closing=500000, ujroh_pembinaan=100000 WHERE id=?`, brand)
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Upline", StatusAgen: "aktif"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "aktif", Upline: upline})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: agen})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	if err := komisi.SnapshotNominal(ctx, tx, b); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	testdb.Exec(t, tx, `UPDATE brands SET ujroh_closing=900000, ujroh_pembinaan=900000 WHERE id=?`, brand)
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	if err := komisi.ProcessBookingLunas(ctx, tx, b); err != nil {
		t.Fatalf("ProcessBookingLunas: %v", err)
	}
	expect(t, "bawaan brand", ledger(t, tx, b, map[int64]string{upline: "Upline", agen: "Agen"}),
		"Agen:langsung:500000", "Upline:pembinaan:100000")
}
