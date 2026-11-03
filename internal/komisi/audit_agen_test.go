package komisi_test

import (
	"context"
	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
	"testing"
)

// Dua probe pertama mendokumentasikan perilaku yang diaudit; PASS berarti temuan itu
// terkonfirmasi, bukan berarti perilakunya benar. Fixtures are rolled back by testdb.Tx.
func TestAuditAgenSkippedCommissionAppearsAfterReactivation(t *testing.T) {
	_, tx := testdb.Tx(t)
	schedule, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1000000), BerangkatTanggal: "2099-01-01"})
	agent := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Audit Agent", StatusAgen: "nonaktif"})
	pax := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Pax", Direkrut: agent})
	booking, _ := testdb.NewBookingPax(t, tx, schedule, "dp", 20000000, testdb.Pax{JamaahID: pax})
	lunas(t, tx, booking)
	var before, after int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM transaksi_komisi WHERE booking_id=?`, booking).Scan(&before); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `UPDATE customer_accounts SET status_agen='aktif' WHERE id=?`, agent)
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
	// Cabang repeat dimatikan di tahap awal; probe ini mendokumentasikan perilaku cabang itu bila dinyalakan lagi.
	nyalakanRepeatCashback(t)
	schedule, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1000000), BerangkatTanggal: "2099-01-01"})
	agent := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Audit Agent", StatusAgen: "aktif"})
	pax := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Audit Repeat", Direkrut: agent})
	old, _ := testdb.NewBookingPax(t, tx, schedule, "dp", 20000000, testdb.Pax{JamaahID: pax})
	lunas(t, tx, old)
	current, _ := testdb.NewBookingPax(t, tx, schedule, "dp", 20000000, testdb.Pax{JamaahID: pax})
	pemesan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Audit Pemesan"})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, pemesan, current)
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

// Komisi yang sudah tercatat tetap menjadi hak agen walau booking atau pax dibatalkan setelah lunas (K20, spek 5.1):
// tidak ada pembalikan komisi. Secara bisnis tidak ada refund penuh, jadi komisi tercatat tidak menjadi beban travel.
func TestKomisiTercatatTetapMenjadiSaldoTersediaSetelahBookingBatal(t *testing.T) {
	_, tx := testdb.Tx(t)
	schedule, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1000000), BerangkatTanggal: "2099-01-01"})
	agent := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Audit Agent", StatusAgen: "aktif"})
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
		t.Fatalf("saldo tersedia = %v, want 1000000: komisi yang sudah tercatat tidak boleh dibalik saat booking batal", saldo.Tersedia)
	}
	var baris int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM transaksi_komisi WHERE booking_id=?`, booking).Scan(&baris); err != nil || baris != 1 {
		t.Fatalf("baris komisi setelah batal = %d (err %v), want 1", baris, err)
	}
	t.Logf("komisi Rp%.0f tetap menjadi saldo tersedia setelah booking batal (disengaja)", saldo.Tersedia)
}
