package booking

import (
	"context"
	"errors"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit menu Agen S-10: finalisasi booking dashboard menolak pax reguler yang jamaahnya belum punya NIK.
// Pax infant dan pax yang sudah batal dikecualikan.
func TestWajibNIKPaxReguler(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	berNIK := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Punya NIK", NIK: "3201010101010001"})
	tanpaNIK := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Tanpa NIK"})
	bayi := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Bayi"})

	b, pax := testdb.NewBookingPax(t, tx, sched, "draft", 3000,
		testdb.Pax{JamaahID: berNIK}, testdb.Pax{JamaahID: tanpaNIK}, testdb.Pax{JamaahID: bayi})
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type='infant', room_type=NULL, counts_for_seat=FALSE WHERE id=?`, pax[2])

	var bisnis *BusinessError
	err := wajibNIKPaxRegulerTx(ctx, tx, b)
	if !errors.As(err, &bisnis) {
		t.Fatalf("pax reguler tanpa NIK harus ditolak dengan pesan bisnis, dapat %v", err)
	}
	if !strings.Contains(bisnis.Message, "Tanpa NIK") || strings.Contains(bisnis.Message, "Bayi") || strings.Contains(bisnis.Message, "Punya NIK") {
		t.Fatalf("pesan hanya boleh menyebut pax reguler tanpa NIK: %q", bisnis.Message)
	}

	// NIK berisi spasi saja tetap dianggap kosong.
	testdb.Exec(t, tx, `UPDATE jamaah SET nik='   ' WHERE id=?`, tanpaNIK)
	if err := wajibNIKPaxRegulerTx(ctx, tx, b); !errors.As(err, &bisnis) {
		t.Fatalf("NIK spasi harus ditolak, dapat %v", err)
	}

	// Pax yang sudah batal tidak diperiksa.
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='batal' WHERE id=?`, pax[1])
	if err := wajibNIKPaxRegulerTx(ctx, tx, b); err != nil {
		t.Fatalf("pax batal tidak boleh menghalangi: %v", err)
	}

	// NIK dilengkapi: lolos.
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='aktif' WHERE id=?`, pax[1])
	testdb.Exec(t, tx, `UPDATE jamaah SET nik='3201010101010002' WHERE id=?`, tanpaNIK)
	if err := wajibNIKPaxRegulerTx(ctx, tx, b); err != nil {
		t.Fatalf("semua pax reguler ber-NIK harus lolos: %v", err)
	}
}
