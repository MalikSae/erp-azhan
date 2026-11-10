package schedule

import (
	"context"
	"errors"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit menu Paket S-04: tanggal berangkat paket tanpa booking aktif bebas diubah. Dengan booking aktif:
// alasan wajib, tanggal baru tidak boleh lampau, paket yang sudah berangkat terkunci, dan perubahan tercatat.
func TestUbahTanggalBerangkat(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE schedule_id=?`, sched)

	jumlahLog := func() int {
		t.Helper()
		var n int
		if err := tx.QueryRow(`SELECT COUNT(*) FROM schedule_tanggal_log WHERE schedule_id=?`, sched).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	ubah := func(lama, baru, alasan string) error {
		return ubahTanggalBerangkatTx(ctx, tx, sched, lama, &ScheduleInput{BerangkatTanggal: baru, AlasanUbahTanggal: alasan, ActorID: 0})
	}

	// Tanpa booking aktif: bebas, tanpa alasan, tanpa catatan, termasuk ke tanggal lampau.
	if err := ubah("2099-01-01", "2000-01-01", ""); err != nil {
		t.Fatalf("tanpa booking aktif harus lolos: %v", err)
	}
	if jumlahLog() != 0 {
		t.Fatal("tanpa booking aktif tidak boleh mencatat log")
	}

	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X"})
	testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})

	// Tanggal tidak berubah: tidak diperiksa.
	if err := ubah("2099-01-01", "2099-01-01", ""); err != nil {
		t.Fatalf("tanggal sama harus lolos: %v", err)
	}

	tolak := func(label string, err error, potongan string) {
		t.Helper()
		if !errors.Is(err, ErrConflict) || !strings.Contains(err.Error(), potongan) {
			t.Fatalf("%s: harus ditolak dengan %q, dapat %v", label, potongan, err)
		}
	}
	tolak("tanpa alasan", ubah("2099-01-01", "2099-02-01", ""), "alasan")
	tolak("tanggal baru lampau", ubah("2099-01-01", "2000-01-01", "salah ketik"), "tidak boleh sudah lewat")
	tolak("sudah berangkat", ubah("2000-01-01", "2099-02-01", "reschedule"), "sudah berangkat")
	if jumlahLog() != 0 {
		t.Fatal("perubahan yang ditolak tidak boleh mencatat log")
	}

	// Beralasan dan ke tanggal depan: lolos dan tercatat.
	if err := ubah("2099-01-01", "2099-02-01", "Maskapai memundurkan jadwal"); err != nil {
		t.Fatalf("beralasan harus lolos: %v", err)
	}
	var lama, baru, alasan string
	var aktif int
	if err := tx.QueryRow(`SELECT DATE_FORMAT(tanggal_lama,'%Y-%m-%d'), DATE_FORMAT(tanggal_baru,'%Y-%m-%d'), booking_aktif, alasan
		FROM schedule_tanggal_log WHERE schedule_id=?`, sched).Scan(&lama, &baru, &aktif, &alasan); err != nil {
		t.Fatal(err)
	}
	if lama != "2099-01-01" || baru != "2099-02-01" || aktif != 1 || alasan != "Maskapai memundurkan jadwal" {
		t.Fatalf("log salah: %s %s %d %q", lama, baru, aktif, alasan)
	}
}
