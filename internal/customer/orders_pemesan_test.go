package customer

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Daftar booking membawa pemesan_ikut dan nama jamaah pertama: pembeda booking di app bila satu akun memesan untuk
// beberapa keluarga pada keberangkatan yang sama.
func TestScanOrderPemesanIkutDanJamaahPertama(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-10"})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemesanBanyak"})
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ringkas Satu"})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Ringkas Dua"})
	id, _ := testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: j1}, testdb.Pax{JamaahID: j2})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, akun, id)
	baca := func() *orderSummary {
		t.Helper()
		o, _, err := scanOrder(tx.QueryRowContext(ctx, orderSelect+` AND b.id = ? GROUP BY b.id`, akun, akun, id))
		if err != nil {
			t.Fatal(err)
		}
		return o
	}

	// Memesankan orang lain: PIC jamaah pertama, pemesan tidak ikut.
	testdb.Exec(t, tx, `UPDATE bookings SET pic_jamaah_id=?, pemesan_ikut=0 WHERE id=?`, j1, id)
	o := baca()
	if o.JamaahPertama != "Ringkas Satu (uji)" {
		t.Errorf("jamaah_pertama = %q", o.JamaahPertama)
	}
	if o.PemesanIkut == nil || *o.PemesanIkut {
		t.Errorf("pemesan_ikut = %v, mau false", o.PemesanIkut)
	}
	if o.JumlahJamaah != 2 {
		t.Errorf("jumlah_jamaah = %d, mau 2", o.JumlahJamaah)
	}

	// Pemesan ikut berangkat.
	testdb.Exec(t, tx, `UPDATE bookings SET pemesan_ikut=1 WHERE id=?`, id)
	if o := baca(); o.PemesanIkut == nil || !*o.PemesanIkut {
		t.Errorf("pemesan_ikut = %v, mau true", o.PemesanIkut)
	}

	// Booking lama: pemesan_ikut belum tercatat (NULL) dan PIC belum ada -> null dan nama kosong, bukan galat.
	testdb.Exec(t, tx, `UPDATE bookings SET pic_jamaah_id=NULL, pemesan_ikut=NULL WHERE id=?`, id)
	if o := baca(); o.PemesanIkut != nil || o.JamaahPertama != "" {
		t.Errorf("booking lama: pemesan_ikut %v, jamaah_pertama %q", o.PemesanIkut, o.JamaahPertama)
	}
}
