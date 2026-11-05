package selfbooking

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// "Saya sendiri": NIK jamaah pertama diambil dari NIK yang agen simpan saat mendaftar, apa pun yang dikirim browser.
func TestIsiNIKSayaSendiriDariNIKAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBernik", StatusAgen: "aktif"})
	tanpaNIK := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenTanpaNIK", StatusAgen: "aktif"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen='9999999999990401' WHERE id=?`, agen)

	lain := "1111111111111111"
	req := BookingRequest{PIC: PICInput{SayaSendiri: true, NIK: &lain}}
	if err := isiNIKSayaSendiri(ctx, tx, agen, &req); err != nil {
		t.Fatal(err)
	}
	if req.PIC.NIK == nil || *req.PIC.NIK != "9999999999990401" {
		t.Fatalf("NIK harus dari NIK agen, dapat %v", req.PIC.NIK)
	}

	kosong := BookingRequest{PIC: PICInput{SayaSendiri: true}}
	if err := isiNIKSayaSendiri(ctx, tx, agen, &kosong); err != nil || kosong.PIC.NIK == nil || *kosong.PIC.NIK != "9999999999990401" {
		t.Fatalf("NIK kosong harus terisi otomatis: %v %v", err, kosong.PIC.NIK)
	}

	// Bukan "Saya sendiri": data dari form tidak disentuh.
	biasa := BookingRequest{PIC: PICInput{NIK: &lain}}
	if err := isiNIKSayaSendiri(ctx, tx, agen, &biasa); err != nil || *biasa.PIC.NIK != lain {
		t.Fatalf("bukan saya sendiri harus tetap: %v %v", err, biasa.PIC.NIK)
	}

	// Agen tanpa NIK tercatat tidak bisa memakai pilihan ini.
	req2 := BookingRequest{PIC: PICInput{SayaSendiri: true, NIK: &lain}}
	if err := isiNIKSayaSendiri(ctx, tx, tanpaNIK, &req2); !errors.Is(err, ErrNIKAgenBelumAda) {
		t.Fatalf("err = %v, want ErrNIKAgenBelumAda", err)
	}
}
