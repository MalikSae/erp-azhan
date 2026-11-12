package agen

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func ip(n int) *int { return &n }

func TestValidasiBatasSeat(t *testing.T) {
	alasan := "Rombongan masjid besar"
	ok := []UbahBatasSeatRequest{
		{PerJadwal: ip(60), Total: ip(100), Alasan: alasan},
		{PerJadwal: ip(MaksBatasSeatAgenPerJadwal), Total: ip(MaksBatasSeatAgenTotal), Alasan: alasan},
		{Alasan: alasan}, // keduanya bawaan
		{Total: ip(80), Alasan: alasan},
	}
	for _, r := range ok {
		if err := validasiBatasSeat(r); err != nil {
			t.Errorf("%+v harus valid: %v", r, err)
		}
	}
	salah := map[string]UbahBatasSeatRequest{
		"per jadwal 0":              {PerJadwal: ip(0), Alasan: alasan},
		"per jadwal > 200":          {PerJadwal: ip(201), Total: ip(500), Alasan: alasan},
		"total > 500":               {Total: ip(501), Alasan: alasan},
		"per jadwal > total":        {PerJadwal: ip(50), Total: ip(40), Alasan: alasan},
		"per jadwal > total bawaan": {PerJadwal: ip(46), Alasan: alasan},
	}
	for nama, r := range salah {
		if err := validasiBatasSeat(r); !errors.Is(err, ErrBatasSeatTidakValid) {
			t.Errorf("%s: err=%v, want ErrBatasSeatTidakValid", nama, err)
		}
	}
	if err := validasiBatasSeat(UbahBatasSeatRequest{PerJadwal: ip(30), Alasan: "ok"}); !errors.Is(err, ErrAlasanBatasWajib) {
		t.Errorf("alasan pendek: err=%v, want ErrAlasanBatasWajib", err)
	}
}

func TestUbahBatasSeatAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	agenID := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen Batas", StatusAgen: "aktif"})
	bukanAgen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Bukan Agen"})
	alasan := "Rombongan pengajian 60 orang"

	b, err := getBatasSeatAgen(ctx, tx, agenID, &brand, true)
	if err != nil || b.PerJadwal != BatasSeatAgenPerJadwalBawaan || b.Total != BatasSeatAgenTotalBawaan || b.KhususPerJadwal != nil {
		t.Fatalf("awal harus batas bawaan: %+v err=%v", b, err)
	}

	if err := ubahBatasSeatTx(ctx, tx, agenID, &brand, 0, UbahBatasSeatRequest{PerJadwal: ip(60), Total: ip(100), Alasan: alasan}); err != nil {
		t.Fatal(err)
	}
	b, err = getBatasSeatAgen(ctx, tx, agenID, &brand, true)
	if err != nil || b.PerJadwal != 60 || b.Total != 100 || len(b.Riwayat) != 1 || b.Riwayat[0].PerJadwalBaru == nil || *b.Riwayat[0].PerJadwalBaru != 60 || b.Riwayat[0].PerJadwalLama != nil {
		t.Fatalf("setelah ubah: %+v err=%v", b, err)
	}

	if err := ubahBatasSeatTx(ctx, tx, agenID, &brand, 0, UbahBatasSeatRequest{PerJadwal: ip(60), Total: ip(100), Alasan: alasan}); !errors.Is(err, ErrBatasSeatSama) {
		t.Fatalf("nilai sama: err=%v, want ErrBatasSeatSama", err)
	}
	lain := brand + 1_000_000
	if err := ubahBatasSeatTx(ctx, tx, agenID, &lain, 0, UbahBatasSeatRequest{PerJadwal: ip(30), Alasan: alasan}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("admin brand lain: err=%v, want ErrNotFound", err)
	}
	if _, err := getBatasSeatAgen(ctx, tx, agenID, &lain, true); !errors.Is(err, ErrNotFound) {
		t.Fatalf("admin brand lain membaca: err=%v, want ErrNotFound", err)
	}
	if err := ubahBatasSeatTx(ctx, tx, bukanAgen, &brand, 0, UbahBatasSeatRequest{PerJadwal: ip(30), Alasan: alasan}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("akun bukan agen: err=%v, want ErrNotFound", err)
	}

	// Kembali ke bawaan (null) juga tercatat.
	if err := ubahBatasSeatTx(ctx, tx, agenID, nil, 0, UbahBatasSeatRequest{Alasan: "Rombongan sudah berangkat"}); err != nil {
		t.Fatal(err)
	}
	b, _ = getBatasSeatAgen(ctx, tx, agenID, nil, true)
	if b.PerJadwal != BatasSeatAgenPerJadwalBawaan || b.KhususTotal != nil || len(b.Riwayat) != 2 {
		t.Fatalf("kembali ke bawaan: %+v", b)
	}
}
