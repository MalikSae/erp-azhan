package komisi_test

import (
	"database/sql"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Penentu "diri agen" (E3, ADR 0004): saya_sendiri_akun_id atau jamaah tertaut ke akun. NIK-sama bukan penentu.
type fxDiri struct {
	tx    *sql.Tx
	sched int64
	brand int64
	nama  map[int64]string
}

func newFxDiri(t *testing.T) *fxDiri {
	t.Helper()
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	return &fxDiri{tx: tx, sched: sched, brand: brand, nama: map[int64]string{}}
}

func (f *fxDiri) akun(t *testing.T, nama, status string, direkrut, upline int64) int64 {
	t.Helper()
	id := testdb.NewAkun(t, f.tx, f.brand, testdb.AkunOpts{Nama: nama, StatusAgen: status, Direkrut: direkrut, Upline: upline})
	f.nama[id] = nama
	return id
}

func (f *fxDiri) lunas(t *testing.T, o testdb.JamaahOpts, tautKe int64, sayaSendiri int64) []string {
	t.Helper()
	j := testdb.NewJamaah(t, f.tx, f.brand, o)
	if tautKe > 0 {
		testdb.Exec(t, f.tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j, tautKe)
	}
	b, pax := testdb.NewBookingPax(t, f.tx, f.sched, "dp", 1000, testdb.Pax{JamaahID: j})
	if sayaSendiri > 0 {
		testdb.Exec(t, f.tx, `UPDATE booking_pax SET saya_sendiri_akun_id=? WHERE id=?`, sayaSendiri, pax[0])
	}
	lunas(t, f.tx, b)
	return ledger(t, f.tx, b, f.nama)
}

// Jamaah tertaut terikat ke pemilik kunci: komisi langsung ke pemilik kunci (bukan upline_akun_id akun),
// bonus pembinaan ke upline pemilik kunci. Berlaku untuk akun yang belum atau sudah agen.
func TestDiriTertautTerikatMemakaiPemilikKunci(t *testing.T) {
	f := newFxDiri(t)
	atas := f.akun(t, "Atas", "aktif", 0, 0)
	ibrahim := f.akun(t, "Ibrahim", "aktif", atas, atas)
	z := f.akun(t, "Z", "aktif", 0, 0)

	// Akun biasa (belum agen) tertaut ke jamaahnya.
	ahmad := f.akun(t, "Ahmad", "tidak_aktif", ibrahim, 0)
	expect(t, "akun biasa tertaut", f.lunas(t, testdb.JamaahOpts{Nama: "JA", Direkrut: ibrahim}, ahmad, 0),
		"Ibrahim:langsung:1000000", "Atas:pembinaan:300000")

	// Ahmad agen aktif, upline akun diganti ke Z: jamaahnya tetap terikat ke Ibrahim, komisi tetap ke Ibrahim.
	agen := f.akun(t, "AhmadAgen", "aktif", z, z)
	expect(t, "agen tertaut, upline akun Z", f.lunas(t, testdb.JamaahOpts{Nama: "JA2", Direkrut: ibrahim}, agen, 0),
		"Ibrahim:langsung:1000000", "Atas:pembinaan:300000")

	// Saya sendiri atas jamaah yang terikat ke agen lain (W): pemilik kunci W, bukan upline agen (Z).
	w := f.akun(t, "W", "aktif", 0, 0)
	expect(t, "saya sendiri, jamaah terikat W", f.lunas(t, testdb.JamaahOpts{Nama: "JW", Direkrut: w}, 0, agen),
		"W:langsung:1000000")
}

// Pemilik kunci nonaktif: komisi langsung hangus, bonus pembinaan tetap ke upline aktif di atasnya (tanpa backfill).
func TestDiriPemilikKunciNonaktifHangus(t *testing.T) {
	f := newFxDiri(t)
	atas := f.akun(t, "Atas", "aktif", 0, 0)
	ibrahim := f.akun(t, "Ibrahim", "nonaktif", atas, atas)
	ahmad := f.akun(t, "Ahmad", "aktif", ibrahim, ibrahim)
	j := testdb.NewJamaah(t, f.tx, f.brand, testdb.JamaahOpts{Nama: "JA", Direkrut: ibrahim})
	testdb.Exec(t, f.tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j, ahmad)
	b, _ := testdb.NewBookingPax(t, f.tx, f.sched, "dp", 1000, testdb.Pax{JamaahID: j})
	lunas(t, f.tx, b)
	expect(t, "pemilik kunci nonaktif", ledger(t, f.tx, b, f.nama), "Atas:pembinaan:300000")
}

// Diri agen dengan jamaah tanpa_agen tidak berkomisi, lewat tautan maupun saya_sendiri.
func TestDiriTanpaAgenTidakBerkomisi(t *testing.T) {
	f := newFxDiri(t)
	u := f.akun(t, "U", "aktif", 0, 0)
	y := f.akun(t, "Y", "aktif", u, u)
	expect(t, "tertaut tanpa_agen", f.lunas(t, testdb.JamaahOpts{Nama: "JT", TanpaAgen: true}, y, 0))
	expect(t, "saya sendiri tanpa_agen", f.lunas(t, testdb.JamaahOpts{Nama: "JS", TanpaAgen: true}, 0, y))
}

// belum_ditentukan: diri agen mengikuti upline_akun_id agen (perilaku saat ini, ditandai di tes).
func TestDiriBelumDitentukanMengikutiUplineAkun(t *testing.T) {
	f := newFxDiri(t)
	atas := f.akun(t, "Atas", "aktif", 0, 0)
	u := f.akun(t, "U", "aktif", atas, atas)
	y := f.akun(t, "Y", "aktif", u, u)
	expect(t, "saya sendiri belum_ditentukan", f.lunas(t, testdb.JamaahOpts{Nama: "JB"}, 0, y),
		"U:langsung:1000000", "Atas:pembinaan:300000")
	// Tanpa tautan dan tanpa saya_sendiri, belum_ditentukan tidak berkomisi.
	expect(t, "bukan diri, belum_ditentukan", f.lunas(t, testdb.JamaahOpts{Nama: "JC"}, 0, 0))
}

// NIK sama dengan nik_agen tanpa tautan atau saya_sendiri bukan diri agen (booking admin/CRM): komisi ke pemilik kunci jamaah.
// Admin menyelesaikan lewat tautan atau Ganti Upline.
func TestNIKSamaBukanDiriAgen(t *testing.T) {
	f := newFxDiri(t)
	const nik = "9999999999990901"
	atas := f.akun(t, "Atas", "aktif", 0, 0)
	u := f.akun(t, "U", "aktif", atas, atas)
	y := f.akun(t, "Y", "aktif", u, u)
	testdb.Exec(t, f.tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik, y)
	w := f.akun(t, "W", "aktif", 0, 0)

	// Jamaah ber-NIK agen, terikat ke W, dipesan admin: komisi ke W (bukan ke upline Y).
	expect(t, "NIK sama, terikat W", f.lunas(t, testdb.JamaahOpts{Nama: "JNIK", Direkrut: w, NIK: nik}, 0, 0), "W:langsung:1000000")
	// Terikat ke Y sendiri: komisi langsung ke Y (bukan booking pribadi), pembinaan ke U.
	expect(t, "NIK sama, terikat Y", f.lunas(t, testdb.JamaahOpts{Nama: "JNIK2", Direkrut: y, NIK: "9999999999990902"}, 0, 0),
		"Y:langsung:1000000", "U:pembinaan:300000")
}

// Komisi untuk keberangkatan berbeda tetap dibayar; repeat dan cashback tetap mati.
func TestKomisiKeberangkatanBerbedaDibayarLagi(t *testing.T) {
	f := newFxDiri(t)
	ibrahim := f.akun(t, "Ibrahim", "aktif", 0, 0)
	ahmad := f.akun(t, "Ahmad", "aktif", ibrahim, ibrahim)
	j := testdb.NewJamaah(t, f.tx, f.brand, testdb.JamaahOpts{Nama: "JA", Direkrut: ibrahim})
	testdb.Exec(t, f.tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j, ahmad)

	var sched2 int64
	if err := f.tx.QueryRow(`SELECT id FROM schedules WHERE brand_id=? AND id<>? ORDER BY id LIMIT 1`, f.brand, f.sched).Scan(&sched2); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, f.tx, `UPDATE schedules SET nominal_komisi_langsung=1000000, nominal_bonus_pembinaan=300000, berangkat_tanggal='2099-02-01' WHERE id=?`, sched2)

	b1, _ := testdb.NewBookingPax(t, f.tx, f.sched, "dp", 1000, testdb.Pax{JamaahID: j})
	b2, _ := testdb.NewBookingPax(t, f.tx, sched2, "dp", 1000, testdb.Pax{JamaahID: j})
	lunas(t, f.tx, b1)
	lunas(t, f.tx, b2)
	expect(t, "B1", ledger(t, f.tx, b1, f.nama), "Ibrahim:langsung:1000000")
	expect(t, "B2 keberangkatan berbeda", ledger(t, f.tx, b2, f.nama), "Ibrahim:langsung:1000000")
}

// Satu komisi langsung dan satu bonus pembinaan per jamaah per keberangkatan, juga bila booking pertama
// batal setelah lunas (komisi lunas tidak ditarik saat batal) lalu jamaah memesan lagi di jadwal yang sama.
func TestKomisiSatuPerKeberangkatanSetelahBookingBatal(t *testing.T) {
	f := newFxDiri(t)
	atas := f.akun(t, "Atas", "aktif", 0, 0)
	ibrahim := f.akun(t, "Ibrahim", "aktif", atas, atas)
	j := testdb.NewJamaah(t, f.tx, f.brand, testdb.JamaahOpts{Nama: "JA", Direkrut: ibrahim})

	b1, _ := testdb.NewBookingPax(t, f.tx, f.sched, "dp", 1000, testdb.Pax{JamaahID: j})
	lunas(t, f.tx, b1)
	expect(t, "B1", ledger(t, f.tx, b1, f.nama), "Ibrahim:langsung:1000000", "Atas:pembinaan:300000")
	testdb.Exec(t, f.tx, `UPDATE bookings SET status='batal' WHERE id=?`, b1)

	b2, _ := testdb.NewBookingPax(t, f.tx, f.sched, "dp", 1000, testdb.Pax{JamaahID: j})
	lunas(t, f.tx, b2)
	expect(t, "B2 jadwal sama setelah B1 batal", ledger(t, f.tx, b2, f.nama))
	expect(t, "B1 tetap", ledger(t, f.tx, b1, f.nama), "Ibrahim:langsung:1000000", "Atas:pembinaan:300000")

	// Komisi B1 yang sudah dibatalkan (turun dari lunas) tidak menghalangi B2.
	testdb.Exec(t, f.tx, `UPDATE transaksi_komisi SET dibatalkan_at=NOW(), sebab_batal='status_turun' WHERE booking_id=?`, b1)
	lunas(t, f.tx, b2)
	expect(t, "B2 setelah komisi B1 dibatalkan", ledger(t, f.tx, b2, f.nama), "Ibrahim:langsung:1000000", "Atas:pembinaan:300000")
}
