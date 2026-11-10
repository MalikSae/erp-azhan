package komisi_test

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/payment"
	"erp-azhan/api/internal/testdb"
)

// ledgerBatal: baris komisi yang sudah dibatalkan, format "penerima:jenis:nominal:sebab".
func ledgerBatal(t *testing.T, tx *sql.Tx, bookingID int64, nama map[int64]string) []string {
	t.Helper()
	rows, err := tx.Query(`SELECT akun_penerima_id, jenis, nominal, COALESCE(sebab_batal,'') FROM transaksi_komisi
		WHERE booking_id=? AND dibatalkan_at IS NOT NULL`, bookingID)
	if err != nil {
		t.Fatalf("baca ledger batal: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id int64
		var jenis, sebab string
		var nominal float64
		if err := rows.Scan(&id, &jenis, &nominal, &sebab); err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprintf("%s:%s:%.0f:%s", nama[id], jenis, nominal, sebab))
	}
	sort.Strings(out)
	return out
}

func sinkron(t *testing.T, ctx context.Context, tx *sql.Tx, bookingID int64) {
	t.Helper()
	if err := payment.SyncBookingStatusTx(ctx, tx, bookingID); err != nil {
		t.Fatalf("SyncBookingStatusTx: %v", err)
	}
}

// T-01, K25: komisi mengikuti lunas terakhir. Turun dari lunas membatalkan komisi; lunas lagi mencatat
// baris baru dengan penerima saat itu, tanpa komisi ganda untuk pax yang sama.
func TestKomisiBatalSaatTurunDariLunasLaluDihitungUlang(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	adam := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Adam", StatusAgen: "aktif"})
	baqir := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Baqir", StatusAgen: "aktif"})
	nuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nuh", StatusAgen: "aktif", Upline: adam})
	hud := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Hud", Direkrut: nuh})
	b, _ := testdb.NewBookingPax(t, tx, sched, "baru", 1000, testdb.Pax{JamaahID: hud})
	nama := map[int64]string{adam: "Adam", baqir: "Baqir", nuh: "Nuh"}
	if err := komisi.SnapshotNominal(ctx, tx, b); err != nil {
		t.Fatal(err)
	}

	// Lunas pertama lewat jalur produksi: pembayaran terkonfirmasi lalu sinkron status.
	bayar := testdb.AddPayment(t, tx, b, 1000, "confirmed")
	sinkron(t, ctx, tx, b)
	if got := testdb.Status(t, tx, b); got != "lunas" {
		t.Fatalf("status = %s, want lunas", got)
	}
	expect(t, "lunas pertama", ledger(t, tx, b, nama), "Nuh:langsung:1000000", "Adam:pembinaan:300000")

	// Upline Nuh diganti, lalu pembayaran di-void: booking turun dari lunas, komisinya batal.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET upline_akun_id=? WHERE id=?`, baqir, nuh)
	testdb.Exec(t, tx, `UPDATE payments SET status='void' WHERE id=?`, bayar)
	sinkron(t, komisi.DenganSebabBatal(ctx, komisi.SebabVoid), tx, b)
	if got := testdb.Status(t, tx, b); got == "lunas" {
		t.Fatalf("status masih lunas setelah void")
	}
	expect(t, "setelah void: tidak ada komisi aktif", ledger(t, tx, b, nama))
	expect(t, "setelah void: baris batal", ledgerBatal(t, tx, b, nama),
		"Nuh:langsung:1000000:void_pembayaran", "Adam:pembinaan:300000:void_pembayaran")

	// Pembayaran dicatat ulang: lunas lagi, dihitung dengan upline yang berlaku sekarang.
	testdb.AddPayment(t, tx, b, 1000, "confirmed")
	sinkron(t, ctx, tx, b)
	expect(t, "lunas kedua", ledger(t, tx, b, nama), "Nuh:langsung:1000000", "Baqir:pembinaan:300000")
	if got := len(ledgerBatal(t, tx, b, nama)); got != 2 {
		t.Fatalf("baris batal = %d, want tetap 2", got)
	}

	// Pemanggilan ulang tanpa turun dari lunas tidak menambah baris, walau penerima berubah lagi.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET upline_akun_id=? WHERE id=?`, adam, nuh)
	if err := komisi.ProcessBookingLunas(ctx, tx, b); err != nil {
		t.Fatal(err)
	}
	expect(t, "panggilan ulang", ledger(t, tx, b, nama), "Nuh:langsung:1000000", "Baqir:pembinaan:300000")
}

// K20 (bagian yang tetap): pembatalan booking setelah lunas tidak membatalkan komisi.
func TestBookingBatalSetelahLunasKomisiTetap(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	nuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nuh", StatusAgen: "aktif"})
	hud := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Hud", Direkrut: nuh})
	b, _ := testdb.NewBookingPax(t, tx, sched, "baru", 1000, testdb.Pax{JamaahID: hud})
	nama := map[int64]string{nuh: "Nuh"}
	if err := komisi.SnapshotNominal(ctx, tx, b); err != nil {
		t.Fatal(err)
	}
	testdb.AddPayment(t, tx, b, 1000, "confirmed")
	sinkron(t, ctx, tx, b)
	expect(t, "lunas", ledger(t, tx, b, nama), "Nuh:langsung:1000000")

	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, b)
	sinkron(t, ctx, tx, b)
	expect(t, "setelah batal", ledger(t, tx, b, nama), "Nuh:langsung:1000000")
}

// K26: komisi tersedia 3 hari kalender setelah tanggal berangkat.
// K27: komisi yang sudah dicairkan lalu batal membuat saldo tersedia negatif.
func TestSaldoHPlus3DanNegatif(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	nuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nuh", StatusAgen: "aktif"})
	hud := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Hud", Direkrut: nuh})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: hud})
	lunas(t, tx, b)

	saldo := func(label string) komisi.Saldo {
		t.Helper()
		s, err := komisi.HitungSaldo(ctx, tx, nuh)
		if err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		return s
	}
	berangkat := func(hariLalu int) {
		testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal = DATE_SUB(CURRENT_DATE, INTERVAL ? DAY) WHERE id=?`, hariLalu, sched)
	}

	for _, c := range []struct {
		hariLalu           int
		tersedia, tertahan float64
	}{
		{0, 0, 1_000_000}, // hari keberangkatan
		{2, 0, 1_000_000}, // H+2
		{3, 1_000_000, 0}, // H+3
		{10, 1_000_000, 0},
	} {
		berangkat(c.hariLalu)
		if s := saldo("saldo"); s.Tersedia != c.tersedia || s.Tertahan != c.tertahan {
			t.Errorf("berangkat %d hari lalu: tersedia %.0f tertahan %.0f, want %.0f / %.0f", c.hariLalu, s.Tersedia, s.Tertahan, c.tersedia, c.tertahan)
		}
	}

	// Sudah dicairkan penuh, lalu komisinya batal: saldo tersedia minus sebesar yang sudah cair.
	testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama)
		VALUES (?, ?, 1000000, 'disetujui', 'Bank Uji', '123', 'Nuh')`, brand, nuh)
	if s := saldo("setelah cair"); s.Tersedia != 0 {
		t.Fatalf("setelah cair: tersedia %.0f, want 0", s.Tersedia)
	}
	if n, err := komisi.BatalkanKomisiBookingTx(ctx, tx, b); err != nil || n != 1 {
		t.Fatalf("batalkan: n=%d err=%v", n, err)
	}
	if s := saldo("setelah batal"); s.Tersedia != -1_000_000 || s.Tertahan != 0 {
		t.Fatalf("setelah batal: tersedia %.0f tertahan %.0f, want -1000000 / 0", s.Tersedia, s.Tertahan)
	}
}
