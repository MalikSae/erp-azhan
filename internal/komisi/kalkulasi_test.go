package komisi_test

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

func rp(v float64) *float64 { return &v }

// ledger mengembalikan baris transaksi_komisi sebuah booking dalam format
// "penerima:jenis:nominal", terurut, supaya mudah dibandingkan. Penerima adalah akun.
func ledger(t *testing.T, tx *sql.Tx, bookingID int64, nama map[int64]string) []string {
	t.Helper()
	rows, err := tx.Query(`SELECT akun_penerima_id, jenis, nominal FROM transaksi_komisi WHERE booking_id=?`, bookingID)
	if err != nil {
		t.Fatalf("baca ledger: %v", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id int64
		var jenis string
		var nominal float64
		rows.Scan(&id, &jenis, &nominal)
		out = append(out, fmt.Sprintf("%s:%s:%.0f", nama[id], jenis, nominal))
	}
	sort.Strings(out)
	return out
}

func expect(t *testing.T, label string, got []string, want ...string) {
	t.Helper()
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s:\n got  %v\n want %v", label, got, want)
	}
}

// lunas men-set status booking dan menjalankan hook seperti jalur produksi.
func lunas(t *testing.T, tx *sql.Tx, bookingID int64) {
	t.Helper()
	ctx := context.Background()
	if err := komisi.SnapshotNominal(ctx, tx, bookingID); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, bookingID)
	if err := komisi.ProcessBookingLunas(ctx, tx, bookingID); err != nil {
		t.Fatalf("ProcessBookingLunas: %v", err)
	}
}

// nyalakanRepeatCashback menyalakan cabang repeat/cashback yang dimatikan di tahap awal, selama test berjalan.
func nyalakanRepeatCashback(t *testing.T) {
	t.Helper()
	komisi.KomisiRepeatCashbackAktif = true
	t.Cleanup(func() { komisi.KomisiRepeatCashbackAktif = false })
}

// TestSkenarioAcuan menjalankan tujuh skenario acuan spek keagenan bagian 5.2
// dengan komisi langsung 1.000.000 dan bonus pembinaan 300.000 per pax.
func TestSkenarioAcuan(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	nama := map[int64]string{}
	akun := func(o testdb.AkunOpts) int64 {
		id := testdb.NewAkun(t, tx, brand, o)
		nama[id] = o.Nama
		return id
	}

	// Skenario 1: agen A membawa jamaah X -> A komisi langsung.
	u := akun(testdb.AkunOpts{Nama: "U", StatusAgen: "aktif"}) // upline A
	a := akun(testdb.AkunOpts{Nama: "A", StatusAgen: "aktif", Direkrut: u, Upline: u})
	jamaahX := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: a})
	b1, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaahX})
	lunas(t, tx, b1)
	expect(t, "skenario 1", ledger(t, tx, b1, nama), "A:langsung:1000000", "U:pembinaan:300000")

	// Skenario 2: X mendaftar jadi agen, akunnya terikat ke A -> upline X adalah A.
	x := akun(testdb.AkunOpts{Nama: "AkunX", StatusAgen: "aktif", Direkrut: a, Upline: a})

	// Skenario 3: agen X membawa jamaah Y -> X komisi langsung, A bonus pembinaan.
	jamaahY := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Y", Direkrut: x})
	b3, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaahY})
	lunas(t, tx, b3)
	expect(t, "skenario 3", ledger(t, tx, b3, nama), "AkunX:langsung:1000000", "A:pembinaan:300000")

	// Skenario 4: agen X mendaftarkan dirinya lewat portal agen ("Saya sendiri"). Jamaah X terkunci ke A,
	// tetapi pax ini menghasilkan komisi langsung untuk X, bonus pembinaan untuk A (upline X).
	b4, pax4 := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaahX})
	testdb.Exec(t, tx, `UPDATE booking_pax SET saya_sendiri_akun_id=? WHERE id=?`, x, pax4[0])
	lunas(t, tx, b4)
	expect(t, "skenario 4", ledger(t, tx, b4, nama), "AkunX:langsung:1000000", "A:pembinaan:300000")

	// Skenario 5: jamaah Y booking ulang -> X komisi langsung, A bonus pembinaan, penuh.
	b5, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaahY})
	lunas(t, tx, b5)
	expect(t, "skenario 5", ledger(t, tx, b5, nama), "AkunX:langsung:1000000", "A:pembinaan:300000")

	// Skenario 6: agen X memesan untuk dirinya lewat web publik (tanpa "Saya sendiri"): dihitung booking ulang.
	// Komisi langsung ke agen yang mengunci jamaah X (A), bonus pembinaan ke upline A (U).
	b6, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaahX})
	lunas(t, tx, b6)
	expect(t, "skenario 6", ledger(t, tx, b6, nama), "A:langsung:1000000", "U:pembinaan:300000")

	// Skenario 7: agen Y2 mendaftarkan jamaah yang terkunci ke X: komisi ke X (pemilik kunci), bukan Y2.
	y2 := akun(testdb.AkunOpts{Nama: "AgenY2", StatusAgen: "aktif"})
	b7, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: jamaahY})
	testdb.Exec(t, tx, `UPDATE bookings SET dibuat_oleh_agen_akun_id=?, customer_account_id=? WHERE id=?`, y2, y2, b7)
	lunas(t, tx, b7)
	expect(t, "skenario 7", ledger(t, tx, b7, nama), "AkunX:langsung:1000000", "A:pembinaan:300000")
}

func TestPaxBayiDanBatalTidakMenghasilkanKomisi(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Upline", StatusAgen: "aktif"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "aktif", Upline: upline})
	nama := map[int64]string{upline: "Upline", agen: "Agen"}
	p1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "P1", Direkrut: agen})
	bayi := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Bayi", Direkrut: agen})
	p3 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "P3", Direkrut: agen})
	b, pax := testdb.NewBookingPax(t, tx, sched, "dp", 3000,
		testdb.Pax{JamaahID: p1}, testdb.Pax{JamaahID: bayi}, testdb.Pax{JamaahID: p3, Batal: true})
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type='infant' WHERE id=?`, pax[1])
	lunas(t, tx, b)
	// Hanya pax reguler aktif (P1): bayi tidak menghasilkan komisi langsung maupun pembinaan; pax batal tidak dihitung.
	expect(t, "bayi dan pax batal", ledger(t, tx, b, nama), "Agen:langsung:1000000", "Upline:pembinaan:300000")
}

func TestBookingUlangDihitungSepertiBiasa(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Upline", StatusAgen: "aktif"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "aktif", Upline: upline})
	pelanggan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemesan"})
	nama := map[int64]string{upline: "Upline", agen: "Agen", pelanggan: "Pemesan"}
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J", Direkrut: agen})

	b1, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	lunas(t, tx, b1)
	b2, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, pelanggan, b2)
	lunas(t, tx, b2)
	// Cabang repeat dan cashback mati: booking kedua sama dengan booking pertama, tanpa cashback.
	expect(t, "booking pertama", ledger(t, tx, b1, nama), "Agen:langsung:1000000", "Upline:pembinaan:300000")
	expect(t, "booking ulang", ledger(t, tx, b2, nama), "Agen:langsung:1000000", "Upline:pembinaan:300000")
}

// Logika repeat order 50/50 dan cashback tetap ada di bawah flag (spek 5.1): bila dinyalakan, perilaku lama
// berlaku, dengan cashback ke akun pemesan booking.
func TestCabangRepeatCashbackTetapAdaDiBawahFlag(t *testing.T) {
	_, tx := testdb.Tx(t)
	nyalakanRepeatCashback(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Upline", StatusAgen: "aktif"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "aktif", Upline: upline})
	pemesan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemesan"})
	nama := map[int64]string{upline: "Upline", agen: "Agen", pemesan: "Pemesan"}
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J", Direkrut: agen})

	b1, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	lunas(t, tx, b1)
	expect(t, "pax baru", ledger(t, tx, b1, nama), "Agen:langsung:1000000", "Upline:pembinaan:300000")
	b2, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, pemesan, b2)
	lunas(t, tx, b2)
	expect(t, "repeat order", ledger(t, tx, b2, nama), "Agen:repeat_order:500000", "Pemesan:cashback:500000")
}

func TestIdempotenSaatLunasBerulang(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	adam := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Adam", StatusAgen: "aktif"})
	nuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nuh", StatusAgen: "aktif", Upline: adam})
	hud := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Hud", Direkrut: nuh})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: hud})

	// lunas -> dp -> lunas -> hook dipanggil lagi: ledger tetap 2 baris.
	lunas(t, tx, b)
	testdb.Exec(t, tx, `UPDATE bookings SET status='dp' WHERE id=?`, b)
	lunas(t, tx, b)
	if err := komisi.ProcessBookingLunas(context.Background(), tx, b); err != nil {
		t.Fatalf("panggilan ketiga: %v", err)
	}
	nama := map[int64]string{adam: "Adam", nuh: "Nuh"}
	expect(t, "idempoten", ledger(t, tx, b, nama), "Nuh:langsung:1000000", "Adam:pembinaan:300000")
}

func TestAgenNonaktifDanJadwalTanpaProgram(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Upline", StatusAgen: "aktif"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "nonaktif", Upline: upline})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: agen})
	nama := map[int64]string{upline: "Upline", agen: "Agen"}

	// Agen nonaktif tidak dapat langsung; upline aktif tetap dapat pembinaan.
	b1, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	lunas(t, tx, b1)
	expect(t, "agen nonaktif", ledger(t, tx, b1, nama), "Upline:pembinaan:300000")

	// Jadwal tanpa nominal (NULL) -> tidak ikut program, ledger kosong.
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=NULL, nominal_bonus_pembinaan=NULL WHERE id=?`, sched)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET status_agen='aktif' WHERE id=?`, agen)
	y := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Y", Direkrut: agen})
	b3, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: y})
	lunas(t, tx, b3)
	expect(t, "jadwal tanpa program", ledger(t, tx, b3, nama))
}

func TestJamaahTanpaAgenTidakMenghasilkanKomisi(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	yusuf := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Yusuf", TanpaAgen: true})
	belum := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Belum"})
	for _, j := range []int64{yusuf, belum, yusuf} {
		b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
		lunas(t, tx, b)
		expect(t, "tanpa agen", ledger(t, tx, b, nil))
	}
}

func TestSnapshotTidakBerubahSaatJadwalDiedit(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "aktif"})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: agen})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})

	// Snapshot saat booking dibuat, lalu jadwal diedit sebelum lunas (D9).
	if err := komisi.SnapshotNominal(context.Background(), tx, b); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=5000000 WHERE id=?`, sched)
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	if err := komisi.ProcessBookingLunas(context.Background(), tx, b); err != nil {
		t.Fatalf("ProcessBookingLunas: %v", err)
	}
	expect(t, "snapshot", ledger(t, tx, b, map[int64]string{agen: "Agen"}), "Agen:langsung:1000000")
}

func TestSaldoTersediaTertahanDanCashback(t *testing.T) {
	_, tx := testdb.Tx(t)
	nyalakanRepeatCashback(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2000-01-01",
	})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "aktif"})
	pemesan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pemesan"})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: agen})

	// Booking 1 berangkat di masa lalu -> komisi tersedia.
	b1, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	lunas(t, tx, b1)
	// Booking 2 (repeat, flag dinyalakan) di jadwal yang sama tapi tanggal diubah ke masa depan.
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2099-01-01' WHERE id=?`, sched)
	b2, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, pemesan, b2)
	lunas(t, tx, b2)

	s, err := komisi.HitungSaldo(ctx, tx, agen)
	if err != nil {
		t.Fatalf("HitungSaldo: %v", err)
	}
	// Jadwal kini di masa depan -> semua komisi agen (1.000.000 + 500.000) tertahan.
	if s.Tersedia != 0 || s.Tertahan != 1_500_000 {
		t.Fatalf("berangkat masa depan: %+v, want tersedia 0 tertahan 1500000", s)
	}

	// Setelah berangkat: tersedia, dikurangi pencairan pending & disetujui.
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2000-01-01' WHERE id=?`, sched)
	for _, st := range []string{"disetujui", "pending", "ditolak"} {
		testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama)
			VALUES (?,?,500000,?,'BSI','123','Agen')`, brand, agen, st)
	}
	s, _ = komisi.HitungSaldo(ctx, tx, agen)
	if s.Tersedia != 500_000 || s.Tertahan != 0 || s.KreditCashback != 0 {
		t.Fatalf("setelah berangkat: %+v, want tersedia 500000 (1.5jt - 500rb disetujui - 500rb pending)", s)
	}

	// Cashback milik akun pemesan: tidak masuk saldo pencairan, berkurang saat dipakai.
	s, _ = komisi.HitungSaldo(ctx, tx, pemesan)
	if s.Tersedia != 0 || s.KreditCashback != 500_000 {
		t.Fatalf("cashback pemesan: %+v, want kredit 500000 dan tersedia 0", s)
	}
	testdb.Exec(t, tx, `INSERT INTO pemakaian_cashback (brand_id, akun_id, booking_id, nominal, dipakai_oleh)
		VALUES (?,?,?,200000,(SELECT id FROM admin_users ORDER BY id LIMIT 1))`, brand, pemesan, b2)
	s, _ = komisi.HitungSaldo(ctx, tx, pemesan)
	if s.KreditCashback != 300_000 {
		t.Fatalf("cashback setelah dipakai: %+v, want 300000", s)
	}
}
