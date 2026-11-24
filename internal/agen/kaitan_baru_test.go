package agen

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/payment"
	"erp-azhan/api/internal/testdb"
)

// Keputusan 2026-11-24: hanya jamaah baru yang terikat otomatis. Jamaah tanpa agen yang sudah ada hanya dikaitkan
// admin; agen yang membawanya dicatat sebagai klaim di booking; pengaitan admin berlaku ke depan untuk komisi.

func klaimOf(t *testing.T, tx *sql.Tx, bookingID int64) int64 {
	t.Helper()
	var id sql.NullInt64
	must(t, "baca klaim", tx.QueryRow(`SELECT agen_klaim_akun_id FROM bookings WHERE id=?`, bookingID).Scan(&id))
	return id.Int64
}

// (a) Jamaah baru terikat otomatis; jamaah tanpa agen yang sudah ikut booking lain tidak terikat, dan agen yang
// membawanya dicatat sebagai klaim booking.
func TestJamaahBaruTerikatOtomatisJamaahLamaJadiKlaim(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})
	y := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenY", StatusAgen: "aktif"})

	// Jamaah lama: pernah ikut booking lain tanpa agen.
	lama := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Lama"})
	testdb.NewBookingPax(t, tx, sched, "lunas", 1000, testdb.Pax{JamaahID: lama})

	pic := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PIC", Direkrut: x})
	baru := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Baru"})
	b, _ := testdb.NewBookingPax(t, tx, sched, "baru", 3000,
		testdb.Pax{JamaahID: pic}, testdb.Pax{JamaahID: baru}, testdb.Pax{JamaahID: lama})
	must(t, "ikat booking", IkatRombonganBooking(ctx, tx, b))

	// Pax booking ini sendiri tidak dihitung sebagai booking lain.
	if st, by := kaitanOf(t, tx, baru); st != "terikat_agen" || by != x {
		t.Fatalf("jamaah baru = %s/%d, want terikat_agen/%d", st, by, x)
	}
	if st, by := kaitanOf(t, tx, lama); st != "tanpa_agen" || by != 0 {
		t.Fatalf("jamaah lama = %s/%d, want tetap tanpa_agen", st, by)
	}
	if k := klaimOf(t, tx, b); k != x {
		t.Fatalf("klaim booking = %d, want agen X %d", k, x)
	}
	// Klaim pertama tidak ditimpa.
	must(t, "klaim kedua", CatatKlaimAgen(ctx, tx, b, y))
	if k := klaimOf(t, tx, b); k != x {
		t.Fatalf("klaim tertimpa = %d, want tetap X %d", k, x)
	}

	// Semua pax terikat: tidak ada klaim.
	pic2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PIC2", Direkrut: x})
	baru2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Baru2"})
	b2, _ := testdb.NewBookingPax(t, tx, sched, "baru", 2000, testdb.Pax{JamaahID: pic2}, testdb.Pax{JamaahID: baru2})
	must(t, "ikat booking 2", IkatRombonganBooking(ctx, tx, b2))
	if k := klaimOf(t, tx, b2); k != 0 {
		t.Fatalf("klaim booking tanpa pax tanpa agen = %d, want kosong", k)
	}

	// Pax tanpa agen yang batal tidak memicu klaim.
	lama2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Lama2"})
	testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: lama2})
	pic3 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PIC3", Direkrut: x})
	b3, _ := testdb.NewBookingPax(t, tx, sched, "baru", 2000, testdb.Pax{JamaahID: pic3}, testdb.Pax{JamaahID: lama2, Batal: true})
	must(t, "ikat booking 3", IkatRombonganBooking(ctx, tx, b3))
	if k := klaimOf(t, tx, b3); k != 0 {
		t.Fatalf("klaim dari pax batal = %d, want kosong", k)
	}

	// IkatJamaah langsung: bookingID 0 (pax belum disimpan) memperlakukan semua booking_pax sebagai booking lain.
	sudahPax := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "SudahPax"})
	bSendiri, _ := testdb.NewBookingPax(t, tx, sched, "baru", 1000, testdb.Pax{JamaahID: sudahPax})
	must(t, "ikat tanpa bookingID", IkatJamaah(ctx, tx, sudahPax, x, SumberReferral, 0))
	if st, _ := kaitanOf(t, tx, sudahPax); st != "tanpa_agen" {
		t.Fatalf("bookingID 0 dengan pax tersimpan = %s, want tanpa_agen", st)
	}
	must(t, "ikat dengan bookingID", IkatJamaah(ctx, tx, sudahPax, x, SumberReferral, bSendiri))
	if st, by := kaitanOf(t, tx, sudahPax); st != "terikat_agen" || by != x {
		t.Fatalf("bookingID sendiri = %s/%d, want terikat_agen/%d", st, by, x)
	}
}

// (b) Jamaah tanpa agen pilihan admin (kaitan_sumber jalur3) tidak terikat otomatis, walau belum pernah booking.
func TestTanpaAgenPilihanAdminTidakTerikatOtomatis(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})

	pilihan := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Pilihan", TanpaAgen: true})
	must(t, "ikat langsung", IkatJamaah(ctx, tx, pilihan, x, SumberReferral, 0))
	if st, by := kaitanOf(t, tx, pilihan); st != "tanpa_agen" || by != 0 {
		t.Fatalf("jalur3 tanpa_agen = %s/%d, want tetap tanpa_agen", st, by)
	}

	pic := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PIC", Direkrut: x})
	b, _ := testdb.NewBookingPax(t, tx, sched, "baru", 2000, testdb.Pax{JamaahID: pic}, testdb.Pax{JamaahID: pilihan})
	must(t, "ikat booking", IkatRombonganBooking(ctx, tx, b))
	if st, by := kaitanOf(t, tx, pilihan); st != "tanpa_agen" || by != 0 {
		t.Fatalf("jalur3 tanpa_agen lewat booking = %s/%d, want tetap tanpa_agen", st, by)
	}
	if k := klaimOf(t, tx, b); k != x {
		t.Fatalf("klaim = %d, want agen X %d", k, x)
	}
}

// (c) Pengaitan admin berlaku ke depan. Booking yang sudah lunas sebelum jamaah dikaitkan tidak mendapat komisi,
// juga saat turun dari lunas lalu lunas lagi. Booking yang masih dp saat dikaitkan mendapat komisi saat lunas.
func TestKomisiPengaitanAdminBerlakuKeDepan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	admin := adminUji(t, tx)
	adam := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Adam", StatusAgen: "aktif"})
	nuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nuh", StatusAgen: "aktif", Direkrut: adam, Upline: adam})
	kaitkan := func(jamaahID int64) {
		t.Helper()
		req := GantiKaitanRequest{Mode: modeKaitanAgen, AgenAkunID: &nuh, Alasan: "dibawa Nuh"}
		must(t, "kaitkan admin", gantiKaitanTx(ctx, tx, jamaahID, admin, req))
	}
	sinkron := func(bookingID int64, c context.Context) {
		t.Helper()
		must(t, "sinkron status", payment.SyncBookingStatusTx(c, tx, bookingID))
	}
	komisiAktif := func(bookingID int64) int {
		t.Helper()
		var n int
		must(t, "hitung komisi", tx.QueryRow(`SELECT COUNT(*) FROM transaksi_komisi WHERE booking_id=? AND dibatalkan_at IS NULL`, bookingID).Scan(&n))
		return n
	}

	// Booking lunas saat jamaah tanpa agen: tidak ada komisi.
	umar := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Umar", TanpaAgen: true})
	b, _ := testdb.NewBookingPax(t, tx, sched, "baru", 1000, testdb.Pax{JamaahID: umar})
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	bayar := testdb.AddPayment(t, tx, b, 1000, "confirmed")
	sinkron(b, ctx)
	if st := testdb.Status(t, tx, b); st != "lunas" {
		t.Fatalf("status = %s, want lunas", st)
	}
	if n := komisiAktif(b); n != 0 {
		t.Fatalf("komisi saat tanpa agen = %d, want 0", n)
	}
	// Lunas pertama terjadi lebih dulu daripada pengaitan (dalam satu transaksi uji keduanya bisa jatuh di detik
	// yang sama, jadi waktunya dimundurkan).
	testdb.Exec(t, tx, `UPDATE bookings SET pertama_lunas_at = NOW() - INTERVAL 1 HOUR WHERE id=?`, b)
	kaitkan(umar)

	// Turun dari lunas lalu lunas lagi: tetap tanpa komisi.
	testdb.Exec(t, tx, `UPDATE payments SET status='void' WHERE id=?`, bayar)
	sinkron(b, komisi.DenganSebabBatal(ctx, komisi.SebabVoid))
	if st := testdb.Status(t, tx, b); st == "lunas" {
		t.Fatalf("status masih lunas setelah void")
	}
	testdb.AddPayment(t, tx, b, 1000, "confirmed")
	sinkron(b, ctx)
	if st := testdb.Status(t, tx, b); st != "lunas" {
		t.Fatalf("status lunas lagi = %s", st)
	}
	if n := komisiAktif(b); n != 0 {
		t.Fatalf("komisi booking lunas sebelum dikaitkan = %d, want 0", n)
	}

	// Booking dp saat dikaitkan: lunas sesudahnya mendapat komisi, juga setelah turun lalu lunas lagi.
	ali := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ali", TanpaAgen: true})
	b2, _ := testdb.NewBookingPax(t, tx, sched, "baru", 1000, testdb.Pax{JamaahID: ali})
	must(t, "snapshot 2", komisi.SnapshotNominal(ctx, tx, b2))
	testdb.AddPayment(t, tx, b2, 400, "confirmed")
	sinkron(b2, ctx)
	if st := testdb.Status(t, tx, b2); st != "dp" {
		t.Fatalf("status b2 = %s, want dp", st)
	}
	kaitkan(ali)
	testdb.Exec(t, tx, `UPDATE jamaah_kaitan_log SET diganti_at = NOW() - INTERVAL 1 HOUR WHERE jamaah_id=?`, ali)
	pelunasan := testdb.AddPayment(t, tx, b2, 600, "confirmed")
	sinkron(b2, ctx)
	if st := testdb.Status(t, tx, b2); st != "lunas" {
		t.Fatalf("status b2 = %s, want lunas", st)
	}
	var penerima []int64
	rows, err := tx.Query(`SELECT akun_penerima_id FROM transaksi_komisi WHERE booking_id=? AND dibatalkan_at IS NULL ORDER BY jenis`, b2)
	must(t, "ledger b2", err)
	for rows.Next() {
		var id int64
		must(t, "scan", rows.Scan(&id))
		penerima = append(penerima, id)
	}
	rows.Close()
	// ORDER BY jenis: langsung, pembinaan.
	if len(penerima) != 2 || penerima[0] != nuh || penerima[1] != adam {
		t.Fatalf("komisi b2 = %v, want [Nuh %d, Adam %d]", penerima, nuh, adam)
	}
	testdb.Exec(t, tx, `UPDATE payments SET status='void' WHERE id=?`, pelunasan)
	sinkron(b2, komisi.DenganSebabBatal(ctx, komisi.SebabVoid))
	if n := komisiAktif(b2); n != 0 {
		t.Fatalf("komisi b2 setelah turun = %d, want 0", n)
	}
	testdb.AddPayment(t, tx, b2, 600, "confirmed")
	sinkron(b2, ctx)
	if n := komisiAktif(b2); n != 2 {
		t.Fatalf("komisi b2 lunas lagi = %d, want 2", n)
	}
}
