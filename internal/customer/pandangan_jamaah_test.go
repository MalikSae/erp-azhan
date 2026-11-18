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
	s.istri = testdb.NewJamaah(t, tx, s.brand, testdb.JamaahOpts{Nama: "IstriRahasia", Direkrut: s.ibrahim})
	testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id IN (?,?)`, s.ibrahim, s.ahmad, s.istri)
	s.ahmadAkun = testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "AhmadAkun", Direkrut: s.ibrahim, Upline: s.ibrahim})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, s.ahmad, s.ahmadAkun)
	s.booking, _ = testdb.NewBookingPax(t, tx, s.jadwal, "dp", 54321000, testdb.Pax{JamaahID: s.ahmad}, testdb.Pax{JamaahID: s.istri})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, s.ibrahim, s.booking)
	testdb.AddPayment(t, tx, s.booking, 7777000, "confirmed")
	testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'DiskonRahasia', 123000)`, s.booking)
	if err := tx.QueryRow(`SELECT id_booking FROM bookings WHERE id=?`, s.booking).Scan(&s.kodeBooking); err != nil {
		t.Fatal(err)
	}
	return s
}

func bodyPerjalanan(t *testing.T, tx *sql.Tx, akun int64) (string, []perjalananJamaah) {
	t.Helper()
	items, err := muatPerjalananJamaah(context.Background(), tx, akun)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"perjalanan": items})
	return string(b), items
}

func TestPandanganJamaahIsiBodyTerbatas(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	body, items := bodyPerjalanan(t, tx, s.ahmadAkun)
	if len(items) != 1 || items[0].BookingID != s.booking || items[0].Status != "dp" || items[0].Berangkat != "2099-03-01" || items[0].Paket == "" {
		t.Fatalf("kartu salah: %s", body)
	}
	if len(items[0].Pax) != 1 || items[0].Pax[0].JamaahID != s.ahmad {
		t.Fatalf("pax harus hanya miliknya: %s", body)
	}
	for _, terlarang := range []string{s.kodeBooking, "IstriRahasia", "54321", "7777", "123000", "DiskonRahasia", "harga", "total", "dp_", "sisa", "diskon", "payment", "bank", "rekening", "kantor", "booking_code"} {
		if strings.Contains(strings.ToLower(body), strings.ToLower(terlarang)) {
			t.Errorf("body memuat %q: %s", terlarang, body)
		}
	}
	var raw struct {
		P []map[string]json.RawMessage `json:"perjalanan"`
	}
	_ = json.Unmarshal([]byte(body), &raw)
	boleh := map[string]bool{"booking_id": true, "paket": true, "berangkat": true, "status": true, "pax": true, "can_pay": true, "can_cancel": true}
	for k := range raw.P[0] {
		if !boleh[k] {
			t.Errorf("field tak terduga %q", k)
		}
	}
	if items[0].CanPay || items[0].CanCancel {
		t.Error("CanPay/CanCancel harus false")
	}
}

func TestPandanganJamaahPembatalanDanDraftMenghilang(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	for _, st := range []string{"batal", "draft"} {
		testdb.Exec(t, tx, `UPDATE bookings SET status=? WHERE id=?`, st, s.booking)
		if _, items := bodyPerjalanan(t, tx, s.ahmadAkun); len(items) != 0 {
			t.Errorf("status %s harus hilang: %+v", st, items)
		}
	}
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, s.booking)
	if _, items := bodyPerjalanan(t, tx, s.ahmadAkun); len(items) != 1 || items[0].Status != "lunas" {
		t.Errorf("lunas: %+v", items)
	}
}

func TestPandanganJamaahAkunTakTertautTakBerubah(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	lain := testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "AkunBiasa"})
	body, items := bodyPerjalanan(t, tx, lain)
	if len(items) != 0 || body != `{"perjalanan":[]}` {
		t.Errorf("akun tak tertaut: %s", body)
	}
	if semua, err := muatAnggota(context.Background(), tx, lain); err != nil || len(semua) != 0 {
		t.Errorf("anggota akun tak tertaut: %+v %v", semua, err)
	}
	_, itemsI := bodyPerjalanan(t, tx, s.ibrahim)
	if len(itemsI) != 0 {
		t.Errorf("pemilik tidak punya kartu peran: %+v", itemsI)
	}
	semuaI, _ := muatAnggota(context.Background(), tx, s.ibrahim)
	if len(semuaI) != 2 {
		t.Fatalf("Ibrahim harus 2 jamaah: %+v", semuaI)
	}
	for _, a := range semuaI {
		if !a.Penuh || a.DariPeran {
			t.Errorf("Ibrahim tetap penuh: %+v", a)
		}
	}
}

func TestPandanganJamaahOwnedTetapPemilikSaja(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()
	// owned() adalah gerbang Detail, Cancel, CreatePayment, dan Dokumen: Ahmad ditolak, Ibrahim lolos.
	if _, _, err := ownedQ(ctx, tx, s.ahmadAkun, s.booking); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Ahmad tidak boleh lolos owned: %v", err)
	}
	if o, _, err := ownedQ(ctx, tx, s.ibrahim, s.booking); err != nil || o == nil {
		t.Fatalf("Ibrahim harus lolos: %v", err)
	}
}

func TestPandanganJamaahDataJamaahTanpaDobel(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()

	semua, err := muatAnggota(ctx, tx, s.ahmadAkun)
	if err != nil || len(semua) != 1 {
		t.Fatalf("hanya baris Ahmad, istri tidak terlihat: %+v %v", semua, err)
	}
	a := semua[0]
	if a.ID != s.ahmad || a.Penuh || !a.DariPeran {
		t.Fatalf("baris terbatas peran: %+v", a)
	}
	if r := ringkasBooking(a); r.Kode != "" {
		t.Errorf("kode booking bocor: %+v", r)
	}
	if _, err := kelompokDari(ctx, tx, s.ahmadAkun, s.istri); !errors.Is(err, errJamaahTidakAda) {
		t.Errorf("istri tidak boleh terjangkau: %v", err)
	}
	grup, err := kelompokDari(ctx, tx, s.ahmadAkun, s.ahmad)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := simpanPerubahanPelanggan(ctx, tx, s.ahmadAkun, grup, map[string]string{"kota": "Depok"}); !errors.Is(err, errAksesTerbatas) {
		t.Fatalf("ubah data harus ditolak: %v", err)
	}
	d, err := (&JamaahAkun{}).bangunDetail(ctx, tx, s.ahmadAkun, grup)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(d)
	if !d.AksesTerbatas || strings.Contains(string(b), s.kodeBooking) || strings.Contains(string(b), "IstriRahasia") {
		t.Errorf("detail bocor: %s", b)
	}

	// Booking sendiri Ahmad dengan jamaah tertaut yang sama: satu baris, tanpa dobel.
	sendiri, _ := testdb.NewBookingPax(t, tx, s.jadwal, "baru", 1000, testdb.Pax{JamaahID: s.ahmad})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, s.ahmadAkun, sendiri)
	semua, err = muatAnggota(ctx, tx, s.ahmadAkun)
	if err != nil || len(semua) != 1 || semua[0].BookingID != sendiri || semua[0].DariPeran {
		t.Fatalf("satu baris dari booking sendiri: %+v %v", semua, err)
	}
	// Baris jamaah lain senama dari booking sendiri bergabung menjadi satu orang (sama-sama terbatas).
	dup := testdb.NewJamaah(t, tx, s.brand, testdb.JamaahOpts{Nama: "Ahmad"})
	sendiri2, _ := testdb.NewBookingPax(t, tx, s.jadwal, "baru", 1000, testdb.Pax{JamaahID: dup})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, s.ahmadAkun, sendiri2)
	semua, _ = muatAnggota(ctx, tx, s.ahmadAkun)
	if n := len(kelompokOrang(semua)); n != 1 {
		t.Errorf("harus tergabung satu orang, dapat %d: %+v", n, semua)
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
	klaim := func(id int64) sql.NullInt64 {
		var v sql.NullInt64
		_ = tx.QueryRow(`SELECT dibuat_oleh_akun_id FROM jamaah WHERE id=?`, id).Scan(&v)
		return v
	}
	if v := klaim(s.ahmad); !v.Valid || v.Int64 != s.ibrahim {
		t.Errorf("klaim Ibrahim atas Ahmad harus tetap: %+v", v)
	}
}

func TestStatusPeranPemetaan(t *testing.T) {
	for in, want := range map[string]string{"baru": "baru", "dp": "dp", "lunas": "lunas", "dokumen_lengkap": "lunas", "siap_berangkat": "lunas"} {
		if got := statusPeran(in); got != want {
			t.Errorf("statusPeran(%s) = %s, mau %s", in, got, want)
		}
	}
}
