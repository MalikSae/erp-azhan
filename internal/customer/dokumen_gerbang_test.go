package customer

import (
	"context"
	"net/http"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// D8: gerbang unggah dokumen (minimal DP, tidak batal, pax aktif) dibaca di dalam transaksi unggah.
func TestGerbangKirimDokumenDalamTransaksi(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemesanDok"})
	lain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BukanPemesanDok"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahDok"})
	b, pax := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, akun, b)

	cek := func(langkah string, akunID int64, want int) {
		t.Helper()
		kode, pesan, err := gerbangKirimDokumen(ctx, tx, akunID, b, j)
		if err != nil {
			t.Fatalf("%s: %v", langkah, err)
		}
		if kode != want {
			t.Fatalf("%s: kode=%d pesan=%q, want %d", langkah, kode, pesan, want)
		}
	}
	cek("dp", akun, 0)
	cek("akun lain", lain, http.StatusNotFound)
	testdb.Exec(t, tx, `UPDATE bookings SET status='baru' WHERE id=?`, b)
	cek("belum dp", akun, http.StatusConflict)
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, b)
	cek("batal", akun, http.StatusConflict)
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	cek("lunas", akun, 0)
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='batal' WHERE id=?`, pax[0])
	cek("pax batal", akun, http.StatusNotFound)
}
