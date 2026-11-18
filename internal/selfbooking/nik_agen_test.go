package selfbooking

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Agen aktif yang ikut berangkat: NIK jamaah pertama diambil dari profil agen; NIK request yang berbeda ditolak.
func TestIsiNIKPemesanAgenDariProfil(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBernik", StatusAgen: "aktif"})
	tanpaNIK := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenTanpaNIK", StatusAgen: "aktif"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen='9999999999990401' WHERE id=?`, agen)
	p, err := bacaPemesan(ctx, tx, brand, agen)
	if err != nil || !p.agen {
		t.Fatalf("bacaPemesan: %+v %v", p, err)
	}
	pTanpa, err := bacaPemesan(ctx, tx, brand, tanpaNIK)
	if err != nil {
		t.Fatal(err)
	}

	kosong := BookingRequest{}
	if err := isiNIKPemesanAgen(ctx, tx, p, &kosong); err != nil || kosong.PIC.NIK == nil || *kosong.PIC.NIK != "9999999999990401" {
		t.Fatalf("NIK kosong harus terisi dari profil: %v %v", err, kosong.PIC.NIK)
	}
	sama := "9999999999990401"
	ok := BookingRequest{PIC: PICInput{NIK: &sama}}
	if err := isiNIKPemesanAgen(ctx, tx, p, &ok); err != nil {
		t.Fatalf("NIK sama harus lolos: %v", err)
	}
	lain := "1111111111111111"
	beda := BookingRequest{PIC: PICInput{NIK: &lain}}
	if err := isiNIKPemesanAgen(ctx, tx, p, &beda); !errors.Is(err, ErrSayaSendiriBukanAnda) {
		t.Fatalf("NIK beda: err = %v, want ErrSayaSendiriBukanAnda", err)
	}
	// Agen tanpa NIK tercatat tidak bisa ikut berangkat.
	if err := isiNIKPemesanAgen(ctx, tx, pTanpa, &BookingRequest{}); !errors.Is(err, ErrNIKAgenBelumAda) {
		t.Fatalf("err = %v, want ErrNIKAgenBelumAda", err)
	}
	if ErrNIKAgenBelumAda.Error() != "Lengkapi NIK di profil agen." {
		t.Fatalf("pesan = %q", ErrNIKAgenBelumAda.Error())
	}
}
