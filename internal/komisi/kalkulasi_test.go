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
// "penerima:jenis:nominal", terurut, supaya mudah dibandingkan.
func ledger(t *testing.T, tx *sql.Tx, bookingID int64, nama map[int64]string) []string {
	t.Helper()
	rows, err := tx.Query(`SELECT jamaah_penerima_id, jenis, nominal FROM transaksi_komisi WHERE booking_id=?`, bookingID)
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

// TestSimulasiBagian6 menjalankan contoh kasus agen-azhan.md §6 dengan
// komisi langsung 1.000.000 dan bonus pembinaan 300.000 per pax.
func TestSimulasiBagian6(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	nama := map[int64]string{}
	j := func(o testdb.JamaahOpts) int64 {
		id := testdb.NewJamaah(t, tx, brand, o)
		nama[id] = o.Nama
		return id
	}

	// Langkah 1–2: Adam agen; Idris + istri dimiliki Adam, 2 pax lunas.
	adam := j(testdb.JamaahOpts{Nama: "Adam", StatusAgen: "aktif"})
	idris := j(testdb.JamaahOpts{Nama: "Idris", Direkrut: adam})
	istri := j(testdb.JamaahOpts{Nama: "IstriIdris", Direkrut: adam})
	b2, _ := testdb.NewBookingPax(t, tx, sched, "dp", 2000, testdb.Pax{JamaahID: idris}, testdb.Pax{JamaahID: istri})
	lunas(t, tx, b2)
	expect(t, "langkah 2", ledger(t, tx, b2, nama), "Adam:langsung:1000000", "Adam:langsung:1000000")

	// Langkah 3–4: Nuh agen (upline Adam); Hud milik Nuh, 1 pax lunas.
	nuh := j(testdb.JamaahOpts{Nama: "Nuh", StatusAgen: "aktif", Direkrut: adam, Upline: adam})
	hud := j(testdb.JamaahOpts{Nama: "Hud", Direkrut: nuh})
	b4, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: hud})
	lunas(t, tx, b4)
	expect(t, "langkah 4", ledger(t, tx, b4, nama), "Nuh:langsung:1000000", "Adam:pembinaan:300000")

	// Langkah 5: Hud repeat order -> Nuh 50%, cashback Hud 50%, tanpa pembinaan.
	b5, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: hud})
	lunas(t, tx, b5)
	expect(t, "langkah 5", ledger(t, tx, b5, nama), "Nuh:repeat_order:500000", "Hud:cashback:500000")

	// Langkah 6–7: Hud jadi agen (upline Nuh); Sholeh milik Hud.
	// Nuh dapat pembinaan, Adam tidak (cap 1 tier).
	testdb.Exec(t, tx, `UPDATE jamaah SET status_agen='aktif', upline_jamaah_id=? WHERE id=?`, nuh, hud)
	sholeh := j(testdb.JamaahOpts{Nama: "Sholeh", Direkrut: hud})
	b7, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: sholeh})
	lunas(t, tx, b7)
	expect(t, "langkah 7", ledger(t, tx, b7, nama), "Hud:langsung:1000000", "Nuh:pembinaan:300000")

	// Langkah 10: 3 pax, 1 batal -> komisi hanya 2 pax aktif.
	p1 := j(testdb.JamaahOpts{Nama: "P1", Direkrut: adam})
	p2 := j(testdb.JamaahOpts{Nama: "P2", Direkrut: adam})
	p3 := j(testdb.JamaahOpts{Nama: "P3", Direkrut: adam})
	b10, _ := testdb.NewBookingPax(t, tx, sched, "dp", 3000,
		testdb.Pax{JamaahID: p1}, testdb.Pax{JamaahID: p2}, testdb.Pax{JamaahID: p3, Batal: true})
	lunas(t, tx, b10)
	expect(t, "langkah 10", ledger(t, tx, b10, nama), "Adam:langsung:1000000", "Adam:langsung:1000000")

	// Langkah 11: Yusuf tanpa agen, booking dua kali -> tidak ada komisi apa pun.
	yusuf := j(testdb.JamaahOpts{Nama: "Yusuf", TanpaAgen: true})
	b11a, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: yusuf})
	lunas(t, tx, b11a)
	b11b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: yusuf})
	lunas(t, tx, b11b)
	expect(t, "langkah 11a", ledger(t, tx, b11a, nama))
	expect(t, "langkah 11b", ledger(t, tx, b11b, nama))

	// Langkah 12–13: Zaid milik Hud (lunas pertama), lalu Idris & Zaid booking
	// bareng -> dua agen berbeda dalam satu booking, masing-masing repeat order.
	zaid := j(testdb.JamaahOpts{Nama: "Zaid", Direkrut: hud})
	b12, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: zaid})
	lunas(t, tx, b12)
	expect(t, "langkah 12", ledger(t, tx, b12, nama), "Hud:langsung:1000000", "Nuh:pembinaan:300000")
	b13, _ := testdb.NewBookingPax(t, tx, sched, "dp", 2000, testdb.Pax{JamaahID: idris}, testdb.Pax{JamaahID: zaid})
	lunas(t, tx, b13)
	expect(t, "langkah 13", ledger(t, tx, b13, nama),
		"Adam:repeat_order:500000", "Hud:repeat_order:500000", "Idris:cashback:500000", "Zaid:cashback:500000")
}

func TestIdempotenSaatLunasBerulang(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	adam := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Adam", StatusAgen: "aktif"})
	nuh := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Nuh", StatusAgen: "aktif", Upline: adam})
	hud := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Hud", Direkrut: nuh})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: hud})

	// lunas -> dp -> lunas -> hook dipanggil lagi: ledger tetap 2 baris,
	// dan pax tetap dianggap "baru" (bukan repeat terhadap dirinya sendiri).
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
	upline := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Upline", StatusAgen: "aktif"})
	agen := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Agen", StatusAgen: "nonaktif", Upline: upline})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: agen})
	nama := map[int64]string{upline: "Upline", agen: "Agen", x: "X"}

	// Pax baru: agen nonaktif tidak dapat langsung; upline aktif tetap dapat pembinaan.
	b1, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	lunas(t, tx, b1)
	expect(t, "agen nonaktif, pax baru", ledger(t, tx, b1, nama), "Upline:pembinaan:300000")

	// Repeat: agen nonaktif tidak dapat repeat order, cashback tetap (L1).
	b2, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	lunas(t, tx, b2)
	expect(t, "agen nonaktif, repeat", ledger(t, tx, b2, nama), "X:cashback:500000")

	// Jadwal tanpa nominal (NULL) -> tidak ikut program, ledger kosong.
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=NULL, nominal_bonus_pembinaan=NULL WHERE id=?`, sched)
	testdb.Exec(t, tx, `UPDATE jamaah SET status_agen='aktif' WHERE id=?`, agen)
	y := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Y", Direkrut: agen})
	b3, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: y})
	lunas(t, tx, b3)
	expect(t, "jadwal tanpa program", ledger(t, tx, b3, nama))
}

func TestSnapshotTidakBerubahSaatJadwalDiedit(t *testing.T) {
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	agen := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Agen", StatusAgen: "aktif"})
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
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2000-01-01",
	})
	agen := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Agen", StatusAgen: "aktif"})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: agen})

	// Booking 1 berangkat di masa lalu -> komisi tersedia.
	b1, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	lunas(t, tx, b1)
	// Booking 2 (repeat) di jadwal yang sama tapi tanggal diubah ke masa depan
	// setelah booking 1 dihitung -> keduanya ikut tanggal jadwal saat saldo dihitung.
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2099-01-01' WHERE id=?`, sched)
	b2, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
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
		testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, jamaah_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama)
			VALUES (?,?,500000,?,'BSI','123','Agen')`, brand, agen, st)
	}
	s, _ = komisi.HitungSaldo(ctx, tx, agen)
	if s.Tersedia != 500_000 || s.Tertahan != 0 || s.KreditCashback != 0 {
		t.Fatalf("setelah berangkat: %+v, want tersedia 500000 (1.5jt - 500rb disetujui - 500rb pending)", s)
	}

	// Cashback milik X: tidak masuk saldo pencairan, berkurang saat dipakai.
	s, _ = komisi.HitungSaldo(ctx, tx, x)
	if s.Tersedia != 0 || s.KreditCashback != 500_000 {
		t.Fatalf("cashback X: %+v, want kredit 500000 dan tersedia 0", s)
	}
	testdb.Exec(t, tx, `INSERT INTO pemakaian_cashback (brand_id, jamaah_id, booking_id, nominal, dipakai_oleh)
		VALUES (?,?,?,200000,(SELECT id FROM admin_users ORDER BY id LIMIT 1))`, brand, x, b2)
	s, _ = komisi.HitungSaldo(ctx, tx, x)
	if s.KreditCashback != 300_000 {
		t.Fatalf("cashback setelah dipakai: %+v, want 300000", s)
	}
}
