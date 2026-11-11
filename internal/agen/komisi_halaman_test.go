package agen

import (
	"context"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// S-12: daftar komisi admin dipaginasi di server dan ringkasannya menghitung seluruh hasil filter.
// K25: baris batal tidak ikut hitungan, hanya tampil bila diminta.
func TestDaftarKomisiPaginasiRingkasanDanBatal(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	adam := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Adam", StatusAgen: "aktif"})
	nuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nuh", StatusAgen: "aktif", Direkrut: adam, Upline: adam})
	a := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahA", Direkrut: nuh})
	c := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahC", Direkrut: nuh})

	lunasi := func(jamaah int64) int64 {
		t.Helper()
		b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaah})
		must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
		testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
		must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))
		return b
	}
	lunasi(a)
	b2 := lunasi(c)

	f := KomisiFilter{BrandID: &brand, PenerimaID: nuh}
	r, err := ringkasanKomisi(ctx, tx, f)
	must(t, "ringkasan", err)
	if r.Jumlah != 2 || r.Total != 2_000_000 || r.Matang != 0 || r.BelumMatang != 2_000_000 {
		t.Fatalf("ringkasan awal = %+v", r)
	}
	// Halaman berisi satu baris, ringkasan tetap menghitung semuanya.
	f.Limit, f.Offset = 1, 1
	items, err := listKomisi(ctx, tx, f)
	must(t, "halaman 2", err)
	if len(items) != 1 {
		t.Fatalf("halaman 2 = %d baris, want 1", len(items))
	}
	if r2, _ := ringkasanKomisi(ctx, tx, f); r2 != r {
		t.Fatalf("ringkasan berubah oleh paginasi: %+v", r2)
	}

	// Komisi booking kedua batal: keluar dari daftar bawaan, total, saldo, dan daftar agen.
	if _, err := komisi.BatalkanKomisiBookingTx(komisi.DenganSebabBatal(ctx, komisi.SebabVoid), tx, b2); err != nil {
		t.Fatal(err)
	}
	f.Limit, f.Offset = 0, 0
	r, err = ringkasanKomisi(ctx, tx, f)
	must(t, "ringkasan setelah batal", err)
	if r.Jumlah != 1 || r.Total != 1_000_000 {
		t.Fatalf("ringkasan setelah batal = %+v", r)
	}
	f.Status = "batal"
	items, err = listKomisi(ctx, tx, f)
	must(t, "daftar batal", err)
	if len(items) != 1 || items[0].Ketersediaan != "batal" || items[0].SebabBatal == nil || *items[0].SebabBatal != komisi.SebabVoid || items[0].DibatalkanAt == nil {
		t.Fatalf("daftar batal = %+v", items)
	}
	f.Status = "semua"
	if rs, _ := ringkasanKomisi(ctx, tx, f); rs.Jumlah != 2 || rs.Total != 1_000_000 {
		t.Fatalf("ringkasan semua: jumlah baris 2 dan total hanya yang aktif, dapat %+v", rs)
	}
	// Status tertahan dan tersedia: jadwal belum berangkat, jadi komisi yang tersisa tertahan.
	f.Status = "tertahan"
	items, err = listKomisi(ctx, tx, f)
	must(t, "daftar tertahan", err)
	if len(items) != 1 || items[0].Ketersediaan != "tertahan" {
		t.Fatalf("daftar tertahan = %+v", items)
	}
	f.Status = "tersedia"
	items, err = listKomisi(ctx, tx, f)
	must(t, "daftar tersedia", err)
	if rt, _ := ringkasanKomisi(ctx, tx, f); len(items) != 0 || rt.Jumlah != 0 {
		t.Fatalf("daftar tersedia harus kosong: %d baris, ringkasan %+v", len(items), rt)
	}

	tot, err := totalPerJenis(ctx, tx, nuh)
	must(t, "total per jenis", err)
	if tot.Langsung != 1_000_000 {
		t.Fatalf("total langsung Nuh = %.0f, want 1000000", tot.Langsung)
	}
	// Portal agen tidak menerima baris batal walau memintanya.
	portal, err := listKomisi(ctx, tx, KomisiFilter{PenerimaID: nuh})
	must(t, "portal", err)
	if len(portal) != 1 {
		t.Fatalf("daftar bawaan = %d baris, want 1", len(portal))
	}
	// Tanggal tersedia = berangkat + hari tahan.
	if got := portal[0].TersediaMulai.Sub(portal[0].BerangkatTanggal).Hours(); got != float64(24*komisi.HariTahanSetelahBerangkat) {
		t.Fatalf("tersedia_mulai selisih %.0f jam dari berangkat", got)
	}
}
