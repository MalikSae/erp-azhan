package booking

import (
	"context"
	"database/sql"
	"strconv"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Visa, SISKOPATUH (per pax) dan tiket (per booking): tanggal selesai terisi saat dicentang, tetap saat dicentang ulang,
// kosong saat dicabut; notifikasi "tahapan" hanya sekali per tahap ke pemilik booking dan akun tertaut ke jamaah terkait.
func TestSimpanProgressTahapan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	pemesan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemesanTahapQzp"})
	tertaut := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "TertautTahapQzp"})
	tertautLain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "TertautLainTahapQzp"})
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahTahapSatuQzp"})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahTahapDuaQzp"})
	b, pax := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j1}, testdb.Pax{JamaahID: j2})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, pemesan, b)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j1, tertaut)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j2, tertautLain)

	tanggalVisa := func() sql.NullTime {
		t.Helper()
		var at sql.NullTime
		if err := tx.QueryRowContext(ctx, `SELECT progress_visa_at FROM booking_pax WHERE id=?`, pax[0]).Scan(&at); err != nil {
			t.Fatal(err)
		}
		return at
	}
	jumlahNotif := func(akun int64, kunci string) int {
		t.Helper()
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifikasi WHERE akun_id=? AND jenis='tahapan' AND kunci=?`, akun, kunci).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	kunciVisa := "tahap:visa:" + itoa(pax[0])

	// Centang visa jamaah 1.
	if err := simpanProgressPax(ctx, tx, b, pax[0], 0, map[string]bool{"visa": true}); err != nil {
		t.Fatal(err)
	}
	pertama := tanggalVisa()
	if !pertama.Valid {
		t.Fatal("progress_visa_at kosong setelah dicentang")
	}
	if jumlahNotif(pemesan, kunciVisa) != 1 || jumlahNotif(tertaut, kunciVisa) != 1 || jumlahNotif(tertautLain, kunciVisa) != 0 {
		t.Errorf("notifikasi visa: pemesan %d, tertaut %d, tertaut jamaah lain %d (mau 1, 1, 0)",
			jumlahNotif(pemesan, kunciVisa), jumlahNotif(tertaut, kunciVisa), jumlahNotif(tertautLain, kunciVisa))
	}
	var isi string
	if err := tx.QueryRowContext(ctx, `SELECT CONCAT(judul, ' | ', isi) FROM notifikasi WHERE akun_id=? AND kunci=?`, pemesan, kunciVisa).Scan(&isi); err != nil {
		t.Fatal(err)
	}
	if isi == "" || isi[:13] != "Visa selesai " {
		t.Errorf("judul dan isi notifikasi: %q", isi)
	}

	// Dicentang ulang: tanggal tetap, tanpa notifikasi kedua.
	if err := simpanProgressPax(ctx, tx, b, pax[0], 0, map[string]bool{"visa": true}); err != nil {
		t.Fatal(err)
	}
	if got := tanggalVisa(); !got.Valid || !got.Time.Equal(pertama.Time) {
		t.Errorf("tanggal berubah saat dicentang ulang: %v -> %v", pertama, got)
	}
	// Dicabut: tanggal kosong. Dicentang lagi: tanggal terisi, notifikasi tetap satu.
	if err := simpanProgressPax(ctx, tx, b, pax[0], 0, map[string]bool{"visa": false}); err != nil {
		t.Fatal(err)
	}
	if tanggalVisa().Valid {
		t.Error("progress_visa_at tidak dikosongkan saat dicabut")
	}
	if err := simpanProgressPax(ctx, tx, b, pax[0], 0, map[string]bool{"visa": true}); err != nil {
		t.Fatal(err)
	}
	if !tanggalVisa().Valid || jumlahNotif(pemesan, kunciVisa) != 1 {
		t.Errorf("centang ulang setelah dicabut: tanggal %v, notifikasi %d", tanggalVisa(), jumlahNotif(pemesan, kunciVisa))
	}

	// SISKOPATUH jamaah 2 bersama manasik: tanggal SISKOPATUH terisi, notifikasi ke pemilik dan akun tertaut jamaah 2.
	if err := simpanProgressPax(ctx, tx, b, pax[1], 0, map[string]bool{"siskopatuh": true, "manasik": true}); err != nil {
		t.Fatal(err)
	}
	var siskoAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT progress_siskopatuh_at FROM booking_pax WHERE id=?`, pax[1]).Scan(&siskoAt); err != nil {
		t.Fatal(err)
	}
	kunciSisko := "tahap:siskopatuh:" + itoa(pax[1])
	if !siskoAt.Valid || jumlahNotif(pemesan, kunciSisko) != 1 || jumlahNotif(tertautLain, kunciSisko) != 1 || jumlahNotif(tertaut, kunciSisko) != 0 {
		t.Errorf("siskopatuh: tanggal %v, notif pemesan %d, tertaut j2 %d, tertaut j1 %d", siskoAt,
			jumlahNotif(pemesan, kunciSisko), jumlahNotif(tertautLain, kunciSisko), jumlahNotif(tertaut, kunciSisko))
	}

	// Tiket per booking: satu notifikasi per akun (pemilik dan semua akun tertaut ke pax aktif).
	if err := simpanProgressBooking(ctx, tx, b, false, map[string]bool{"tiket": true}); err != nil {
		t.Fatal(err)
	}
	var tiketAt sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT progress_tiket_at FROM bookings WHERE id=?`, b).Scan(&tiketAt); err != nil {
		t.Fatal(err)
	}
	kunciTiket := "tahap:tiket:b" + itoa(b)
	if !tiketAt.Valid || jumlahNotif(pemesan, kunciTiket) != 1 || jumlahNotif(tertaut, kunciTiket) != 1 || jumlahNotif(tertautLain, kunciTiket) != 1 {
		t.Errorf("tiket: tanggal %v, notif pemesan %d, tertaut %d, tertaut lain %d", tiketAt,
			jumlahNotif(pemesan, kunciTiket), jumlahNotif(tertaut, kunciTiket), jumlahNotif(tertautLain, kunciTiket))
	}
	// Tiket yang sudah selesai dicentang lagi (tiketLama true): tanpa notifikasi baru, tanggal tetap.
	if err := simpanProgressBooking(ctx, tx, b, true, map[string]bool{"tiket": true}); err != nil {
		t.Fatal(err)
	}
	var tiketAt2 sql.NullTime
	_ = tx.QueryRowContext(ctx, `SELECT progress_tiket_at FROM bookings WHERE id=?`, b).Scan(&tiketAt2)
	if !tiketAt2.Time.Equal(tiketAt.Time) || jumlahNotif(pemesan, kunciTiket) != 1 {
		t.Errorf("tiket dicentang ulang: tanggal %v -> %v, notif %d", tiketAt, tiketAt2, jumlahNotif(pemesan, kunciTiket))
	}

	// Pax batal dan field tak dikenal ditolak.
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='batal' WHERE id=?`, pax[1])
	if err := simpanProgressPax(ctx, tx, b, pax[1], 0, map[string]bool{"visa": true}); err != ErrPaxBatal {
		t.Errorf("pax batal: %v", err)
	}
	if err := simpanProgressPax(ctx, tx, b, pax[0], 0, map[string]bool{"paspor": true}); err == nil {
		t.Error("field paspor harus ditolak")
	}
}


func itoa(n int64) string { return strconv.FormatInt(n, 10) }

// Booking draft dan batal tidak tampil di portal, jadi tidak diberi notifikasi tahapan (tanggal tetap tersimpan).
func TestNotifikasiTahapanTanpaDraftDanBatal(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	tertaut := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "TertautDraftQzp"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahDraftQzp"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, j, tertaut)

	for _, status := range []string{"draft", "batal"} {
		b, pax := testdb.NewBookingPax(t, tx, jadwal, status, 1000, testdb.Pax{JamaahID: j})
		testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='aktif' WHERE id=?`, pax[0])
		if err := simpanProgressPax(ctx, tx, b, pax[0], 0, map[string]bool{"visa": true}); err != nil {
			t.Fatalf("%s: %v", status, err)
		}
		var n int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifikasi WHERE akun_id=? AND kunci=?`, tertaut, "tahap:visa:"+itoa(pax[0])).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("booking %s: %d notifikasi, mau 0", status, n)
		}
	}
}
