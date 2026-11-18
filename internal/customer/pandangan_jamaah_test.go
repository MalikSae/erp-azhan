package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/testdb"
)

// skenarioPeran: Ibrahim (agen) memesankan Ahmad dan istrinya; Ahmad tertaut lewat aktivasi.
type skenarioPeran struct {
	jadwal, brand      int64
	ibrahim, ahmadAkun int64
	ahmad, istri       int64
	booking            int64
	kodeBooking        string
}

func siapkanPeran(t *testing.T, tx *sql.Tx) skenarioPeran {
	t.Helper()
	var s skenarioPeran
	s.jadwal, s.brand = testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	s.ibrahim = testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "Ibrahim", StatusAgen: "aktif"})
	s.ahmad = testdb.NewJamaah(t, tx, s.brand, testdb.JamaahOpts{Nama: "Ahmad", Direkrut: s.ibrahim})
	s.istri = testdb.NewJamaah(t, tx, s.brand, testdb.JamaahOpts{Nama: "Istri", Direkrut: s.ibrahim})
	testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id IN (?,?)`, s.ibrahim, s.ahmad, s.istri)
	s.ahmadAkun = testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "AhmadAkun", Direkrut: s.ibrahim, Upline: s.ibrahim})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, s.ahmad, s.ahmadAkun)
	s.booking, _ = testdb.NewBookingPax(t, tx, s.jadwal, "dp", 54321000, testdb.Pax{JamaahID: s.ahmad}, testdb.Pax{JamaahID: s.istri})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, s.ibrahim, s.booking)
	testdb.AddPayment(t, tx, s.booking, 7777000, "confirmed")
	if err := tx.QueryRow(`SELECT id_booking FROM bookings WHERE id=?`, s.booking).Scan(&s.kodeBooking); err != nil {
		t.Fatal(err)
	}
	return s
}

func hpAkun(t *testing.T, tx *sql.Tx, akun int64) string {
	t.Helper()
	var hp string
	if err := tx.QueryRow(`SELECT no_hp FROM customer_accounts WHERE id=?`, akun).Scan(&hp); err != nil {
		t.Fatal(err)
	}
	return hp
}

func TestPeranJamaahMelihatBookingUtuh(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()

	items, err := daftarOrder(ctx, tx, s.ahmadAkun, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("Ahmad harus melihat 1 booking: %+v", items)
	}
	o := items[0]
	if o.ID != s.booking || o.Peran != peranJamaah || o.BookingCode != s.kodeBooking {
		t.Fatalf("booking peran salah: %+v", o)
	}
	if o.TotalHarga != 54321000 || o.Terbayar != 7777000 || o.Sisa != 54321000-7777000 {
		t.Errorf("nominal harus transparan: total %v terbayar %v sisa %v", o.TotalHarga, o.Terbayar, o.Sisa)
	}
	if o.CanCancel {
		t.Error("peran jamaah tidak boleh membatalkan")
	}
	want := pembuatBooking{Nama: "Ibrahim (uji)", Kontak: "agen", WhatsApp: hpAkun(t, tx, s.ibrahim)}
	if o.DibuatOleh == nil || *o.DibuatOleh != want {
		t.Errorf("dibuat_oleh = %+v, mau %+v", o.DibuatOleh, want)
	}
	b, _ := json.Marshal(o)
	if !strings.Contains(string(b), `"peran":"jamaah"`) || strings.Contains(string(b), "pemilikID") {
		t.Errorf("json: %s", b)
	}

	// Pemesan tetap melihat booking yang sama sebagai pemesan, tanpa dibuat_oleh.
	itemsI, err := daftarOrder(ctx, tx, s.ibrahim, "")
	if err != nil || len(itemsI) != 1 || itemsI[0].Peran != peranPemesan || itemsI[0].DibuatOleh != nil {
		t.Fatalf("pemesan: %+v %v", itemsI, err)
	}
	if bI, _ := json.Marshal(itemsI[0]); strings.Contains(string(bI), "dibuat_oleh") {
		t.Errorf("pemesan tidak menerima dibuat_oleh: %s", bI)
	}
}

func TestPeranJamaahHanyaBertindakAtasDirinya(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()

	o, _, err := ownedQ(ctx, tx, s.ahmadAkun, s.booking)
	if err != nil {
		t.Fatalf("Ahmad harus lolos owned: %v", err)
	}
	if !o.bolehJamaah(s.ahmad) || o.bolehJamaah(s.istri) {
		t.Errorf("Ahmad hanya boleh atas dirinya: ahmad=%v istri=%v", o.bolehJamaah(s.ahmad), o.bolehJamaah(s.istri))
	}
	oi, _, err := ownedQ(ctx, tx, s.ibrahim, s.booking)
	if err != nil || !oi.bolehJamaah(s.istri) || !oi.bolehJamaah(s.ahmad) {
		t.Fatalf("pemesan boleh atas semua jamaah: %v", err)
	}

	// Lepas tautan: Ahmad kehilangan akses.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=NULL WHERE id=?`, s.ahmadAkun)
	if _, _, err := ownedQ(ctx, tx, s.ahmadAkun, s.booking); !errors.Is(err, ErrNotFound) {
		t.Errorf("setelah lepas tautan harus 404: %v", err)
	}
}

func TestPeranJamaahPaxBatalDanDraft(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()

	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='batal' WHERE booking_id=? AND jamaah_id=?`, s.booking, s.ahmad)
	if _, _, err := ownedQ(ctx, tx, s.ahmadAkun, s.booking); !errors.Is(err, ErrNotFound) {
		t.Errorf("pax Ahmad batal: booking tidak terlihat lagi: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='aktif' WHERE booking_id=? AND jamaah_id=?`, s.booking, s.ahmad)
	testdb.Exec(t, tx, `UPDATE bookings SET status='draft' WHERE id=?`, s.booking)
	if items, _ := daftarOrder(ctx, tx, s.ahmadAkun, ""); len(items) != 0 {
		t.Errorf("draft tidak terlihat: %+v", items)
	}
}

func TestSusunPembuatKontak(t *testing.T) {
	cases := []struct {
		nama, hp, status string
		want             pembuatBooking
	}{
		{"Ibrahim", "0812", "aktif", pembuatBooking{"Ibrahim", "agen", "0812"}},
		{"Ibrahim", "0812", "nonaktif", pembuatBooking{"Ibrahim", "admin", "0899"}},
		{"Budi", "0813", "tidak_aktif", pembuatBooking{"Budi", "admin", "0899"}},
		{"", "", "", pembuatBooking{"Hana", "admin", "0899"}},
	}
	for _, c := range cases {
		if got := susunPembuat(c.nama, c.hp, c.status, "Hana", "0899"); *got != c.want {
			t.Errorf("%+v: dapat %+v", c, *got)
		}
	}
}

func TestPeranJamaahAkunTakTertautTakBerubah(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()
	lain := testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "AkunBiasa"})
	if items, err := daftarOrder(ctx, tx, lain, ""); err != nil || len(items) != 0 {
		t.Errorf("akun tak tertaut: %+v %v", items, err)
	}
	if _, _, err := ownedQ(ctx, tx, lain, s.booking); !errors.Is(err, ErrNotFound) {
		t.Errorf("akun lain harus 404: %v", err)
	}
	if semua, err := muatAnggota(ctx, tx, lain); err != nil || len(semua) != 0 {
		t.Errorf("anggota akun tak tertaut: %+v %v", semua, err)
	}
	semuaI, _ := muatAnggota(ctx, tx, s.ibrahim)
	if len(semuaI) != 2 {
		t.Fatalf("Ibrahim harus 2 jamaah: %+v", semuaI)
	}
	for _, a := range semuaI {
		if !a.Penuh || a.DariPeran {
			t.Errorf("Ibrahim tetap penuh: %+v", a)
		}
	}
}

func TestPeranJamaahDataDiriPenuh(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()

	semua, err := muatAnggota(ctx, tx, s.ahmadAkun)
	if err != nil || len(semua) != 1 {
		t.Fatalf("hanya baris Ahmad di Data Jamaah: %+v %v", semua, err)
	}
	a := semua[0]
	if a.ID != s.ahmad || !a.Penuh || !a.DariPeran {
		t.Fatalf("baris Ahmad harus penuh: %+v", a)
	}
	if r := ringkasBooking(a); r.Kode != s.kodeBooking {
		t.Errorf("kode booking ikut tampil: %+v", r)
	}
	if penuh, err := aksesPenuhJamaah(ctx, tx, s.ahmadAkun, s.ahmad); err != nil || !penuh {
		t.Errorf("akses penuh atas diri: %v %v", penuh, err)
	}
	if penuh, err := aksesPenuhJamaah(ctx, tx, s.ahmadAkun, s.istri); err != nil || penuh {
		t.Errorf("istri tetap terbatas: %v %v", penuh, err)
	}
	if _, err := kelompokDari(ctx, tx, s.ahmadAkun, s.istri); !errors.Is(err, errJamaahTidakAda) {
		t.Errorf("istri tidak terjangkau di Data Jamaah: %v", err)
	}
	grup, err := kelompokDari(ctx, tx, s.ahmadAkun, s.ahmad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := simpanPerubahanPelanggan(ctx, tx, s.ahmadAkun, grup, map[string]string{"kota": "Depok"}); err != nil && !errors.Is(err, ErrTidakBerubah) {
		t.Fatalf("Ahmad boleh mengisi data dirinya: %v", err)
	}

	// Booking sendiri Ahmad dengan jamaah tertaut yang sama: satu baris, tanpa dobel.
	sendiri, _ := testdb.NewBookingPax(t, tx, s.jadwal, "baru", 1000, testdb.Pax{JamaahID: s.ahmad})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, s.ahmadAkun, sendiri)
	semua, err = muatAnggota(ctx, tx, s.ahmadAkun)
	if err != nil || len(semua) != 1 || semua[0].BookingID != sendiri || semua[0].DariPeran {
		t.Fatalf("satu baris dari booking sendiri: %+v %v", semua, err)
	}
}

func TestPandanganJamaahKlaimPembuatTidakGugur(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()
	sendiri, _ := testdb.NewBookingPax(t, tx, s.jadwal, "baru", 1000, testdb.Pax{JamaahID: s.ahmad}, testdb.Pax{JamaahID: s.istri})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, s.ahmadAkun, sendiri)
	if err := shared.GugurkanKlaimPembuatBooking(ctx, tx, sendiri); err != nil {
		t.Fatal(err)
	}
	var v sql.NullInt64
	_ = tx.QueryRow(`SELECT dibuat_oleh_akun_id FROM jamaah WHERE id=?`, s.ahmad).Scan(&v)
	if !v.Valid || v.Int64 != s.ibrahim {
		t.Errorf("klaim Ibrahim atas Ahmad harus tetap: %+v", v)
	}
}
