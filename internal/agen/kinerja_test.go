package agen

import (
	"context"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// Dashboard (A3), riwayat (A6/B3), dan detail (B2a) membaca ledger yang
// sama: komisi booking yang belum berangkat tertahan, cashback jadi kredit.
func TestKinerjaAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	adam := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Adam", StatusAgen: "aktif"})
	nuh := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Nuh", StatusAgen: "aktif", Direkrut: adam, Upline: adam})
	umar := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Umar", Direkrut: nuh})

	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: umar})
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))

	d, err := getDashboardAgen(ctx, tx, nuh)
	must(t, "dashboard", err)
	if d.Tertahan != 1_000_000 || d.Tersedia != 0 || d.JumlahClosing != 1 || d.JumlahDownline != 0 {
		t.Fatalf("dashboard Nuh = %+v", d)
	}
	if len(d.KomisiTerbaru) != 1 || d.KomisiTerbaru[0].Ketersediaan != "tertahan" || d.KomisiTerbaru[0].Jenis != "langsung" {
		t.Fatalf("komisi terbaru Nuh = %+v", d.KomisiTerbaru)
	}

	// Berangkat sudah lewat: komisi jadi tersedia.
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2000-01-01' WHERE id=?`, sched)
	items, err := listKomisi(ctx, tx, KomisiFilter{PenerimaID: adam})
	must(t, "riwayat Adam", err)
	if len(items) != 1 || items[0].Jenis != "pembinaan" || items[0].Ketersediaan != "tersedia" || items[0].SumberNama != "Umar (uji)" {
		t.Fatalf("riwayat Adam = %+v", items)
	}

	// Detail agen: brand scope + jaringan.
	det, err := getDetailAgen(ctx, tx, adam, &brand)
	must(t, "detail", err)
	if det.JumlahDownline != 1 || det.JumlahClosing != 1 || det.Total.Pembinaan != 300_000 || det.Saldo.Tersedia != 300_000 {
		t.Fatalf("detail Adam = %+v", det)
	}
	lain := brand + 999_999
	if _, err := getDetailAgen(ctx, tx, adam, &lain); err != ErrNotFound {
		t.Fatalf("detail brand lain: %v, want ErrNotFound", err)
	}
	if _, err := getDetailAgen(ctx, tx, umar, nil); err != ErrNotFound {
		t.Fatalf("detail jamaah bukan agen: %v, want ErrNotFound", err)
	}

	// Dashboard hanya untuk agen aktif (A4: nonaktif tertutup).
	testdb.Exec(t, tx, `UPDATE jamaah SET status_agen='nonaktif' WHERE id=?`, nuh)
	if _, err := getDashboardAgen(ctx, tx, nuh); err != ErrBukanAgenAktif {
		t.Fatalf("dashboard nonaktif: %v", err)
	}

	// Daftar agen B2: aktif & nonaktif, scope brand.
	list, err := listAgen(ctx, tx, &brand, "(uji)")
	must(t, "list agen", err)
	found := map[int64]string{}
	for _, a := range list {
		found[a.JamaahID] = a.StatusAgen
	}
	if found[adam] != "aktif" || found[nuh] != "nonaktif" || found[umar] != "" {
		t.Fatalf("daftar agen = %v", found)
	}
}
