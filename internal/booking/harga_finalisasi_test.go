package booking

import (
	"context"
	"errors"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit menu Paket S-05: saat draft difinalisasi, harga tiap pax aktif mengikuti harga paket saat itu,
// bukan harga saat draft dibuat. Pax batal tidak disentuh; harga yang belum ada di paket menolak finalisasi.
func TestHargaPaxSaatFinalisasi(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET harga_quad=30000000, harga_triple=32000000, harga_double=35000000, harga_infant=5000000 WHERE id=?`, sched)

	a := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Quad"})
	b := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Double"})
	c := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Bayi"})
	d := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Batal"})
	// Harga lama 1.000 per pax: seolah draft dibuat sebelum harga paket berubah.
	booking, pax := testdb.NewBookingPax(t, tx, sched, "draft", 4000,
		testdb.Pax{JamaahID: a}, testdb.Pax{JamaahID: b}, testdb.Pax{JamaahID: c}, testdb.Pax{JamaahID: d, Batal: true})
	testdb.Exec(t, tx, `UPDATE booking_pax SET room_type='Double' WHERE id=?`, pax[1])
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type='infant', room_type=NULL, counts_for_seat=FALSE WHERE id=?`, pax[2])

	if err := hargaPaxSaatIniTx(ctx, tx, booking); err != nil {
		t.Fatalf("harga saat ini: %v", err)
	}
	harga := func(id int64) float64 {
		t.Helper()
		var h float64
		if err := tx.QueryRow(`SELECT harga_pax FROM booking_pax WHERE id=?`, id).Scan(&h); err != nil {
			t.Fatal(err)
		}
		return h
	}
	for i, mau := range []float64{30_000_000, 35_000_000, 5_000_000, 1000} {
		if got := harga(pax[i]); got != mau {
			t.Fatalf("pax %d: harga %.0f, mau %.0f", i, got, mau)
		}
	}

	// Harga bayi dikosongkan di paket: finalisasi ditolak dengan pesan bisnis, harga pax tidak berubah.
	testdb.Exec(t, tx, `UPDATE schedules SET harga_infant=NULL, harga_quad=31000000 WHERE id=?`, sched)
	var bisnis *BusinessError
	err := hargaPaxSaatIniTx(ctx, tx, booking)
	if !errors.As(err, &bisnis) || !strings.Contains(bisnis.Message, "Bayi") {
		t.Fatalf("harga bayi kosong harus ditolak dengan nama jamaah, dapat %v", err)
	}
	if got := harga(pax[0]); got != 30_000_000 {
		t.Fatalf("penolakan tidak boleh mengubah harga pax lain: %.0f", got)
	}
}
