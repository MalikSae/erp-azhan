package customer

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/perlengkapan"
	"erp-azhan/api/internal/testdb"
)

func TestStatusPerlengkapan(t *testing.T) {
	cases := []struct {
		diberikan, diajukan, diterima bool
		want                          string
	}{
		{false, false, false, "belum"},
		{false, true, false, "diajukan"},
		{true, true, false, "diserahkan"},
		{true, false, false, "diserahkan"}, // diserahkan tanpa pengajuan (jamaah datang langsung)
		{true, true, true, "diterima"},
	}
	for _, c := range cases {
		if got := statusPerlengkapan(c.diberikan, c.diajukan, c.diterima); got != c.want {
			t.Errorf("statusPerlengkapan(%v,%v,%v) = %q, mau %q", c.diberikan, c.diajukan, c.diterima, got, c.want)
		}
	}
}

func TestIsiUntukJenisKelamin(t *testing.T) {
	set := []perlengkapan.SetItem{
		{Nama: "Koper", Qty: 1, Untuk: "semua"},
		{Nama: "Kain ihram", Qty: 2, Untuk: "pria"},
		{Nama: "Mukena", Qty: 1, Untuk: "wanita"},
	}
	pria := isiUntuk(set, "L")
	wanita := isiUntuk(set, "P")
	if len(pria) != 2 || pria[1].Nama != "Kain ihram" {
		t.Errorf("isi pria: %+v", pria)
	}
	if len(wanita) != 2 || wanita[1].Nama != "Mukena" {
		t.Errorf("isi wanita: %+v", wanita)
	}
}

func TestAlurPerlengkapanPelanggan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunPerlQzp"})
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PerlSatuQzp"})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PerlDuaQzp"})
	bayi := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PerlBayiQzp"})
	asing := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PerlAsingQzp"})
	b, pax := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j1}, testdb.Pax{JamaahID: j2}, testdb.Pax{JamaahID: bayi})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, akun, b)
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type='infant' WHERE id=?`, pax[2])

	daftar, err := daftarPerlengkapan(ctx, tx, b, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(daftar) != 2 {
		t.Fatalf("bayi tidak boleh tercantum: %+v", daftar)
	}
	for _, d := range daftar {
		if d.Status != "belum" {
			t.Errorf("status awal %s: %s", d.Nama, d.Status)
		}
	}

	// Jamaah di luar booking atau bayi ditolak, dan tidak ada yang tersimpan (semua atau tidak sama sekali).
	if err := ajukanPerlengkapanTx(ctx, tx, akun, b, []int64{j1, asing}); !errors.Is(err, errJamaahBukanBooking) {
		t.Errorf("jamaah asing: %v", err)
	}
	if err := ajukanPerlengkapanTx(ctx, tx, akun, b, []int64{bayi}); !errors.Is(err, errJamaahBukanBooking) {
		t.Errorf("bayi: %v", err)
	}

	if err := ajukanPerlengkapanTx(ctx, tx, akun, b, []int64{j1}); err != nil {
		t.Fatal(err)
	}
	if err := ajukanPerlengkapanTx(ctx, tx, akun, b, []int64{j1}); !errors.Is(err, errSudahDiajukan) {
		t.Errorf("ajukan ganda: %v", err)
	}
	// Konfirmasi sebelum diserahkan ditolak.
	if err := terimaPerlengkapanTx(ctx, tx, akun, b, j1); !errors.Is(err, errBelumDiserahkan) {
		t.Errorf("terima sebelum diserahkan: %v", err)
	}

	// Petugas menyerahkan (setara MarkPerlengkapanDiberikan; stok diuji di paket booking).
	testdb.Exec(t, tx, `UPDATE booking_pax SET perlengkapan_status='sudah_diberikan', perlengkapan_tanggal=CURDATE(),
		perlengkapan_diserahkan_at=NOW(), perlengkapan_diserahkan_nama='Petugas Uji' WHERE id=?`, pax[0])
	daftar, _ = daftarPerlengkapan(ctx, tx, b, nil)
	if daftar[0].Status != "diserahkan" || daftar[0].DiserahkanNama == nil || *daftar[0].DiserahkanNama != "Petugas Uji" || daftar[0].DiajukanAt == nil {
		t.Errorf("setelah diserahkan: %+v", daftar[0])
	}

	if err := terimaPerlengkapanTx(ctx, tx, akun, b, j1); err != nil {
		t.Fatal(err)
	}
	if err := terimaPerlengkapanTx(ctx, tx, akun, b, j1); !errors.Is(err, errSudahDiterima) {
		t.Errorf("terima ganda: %v", err)
	}
	daftar, _ = daftarPerlengkapan(ctx, tx, b, nil)
	if daftar[0].Status != "diterima" || daftar[0].DiterimaAt == nil {
		t.Errorf("setelah diterima: %+v", daftar[0])
	}
	if daftar[1].Status != "belum" {
		t.Errorf("jamaah kedua ikut berubah: %+v", daftar[1])
	}
}

func TestBisaAjukanPerlengkapan(t *testing.T) {
	if bisaAjukanPerlengkapan(&orderSummary{Status: "baru", Tab: "menunggu"}) {
		t.Error("booking belum DP tidak boleh mengajukan")
	}
	if bisaAjukanPerlengkapan(&orderSummary{Status: "batal", Tab: "batal"}) {
		t.Error("booking batal tidak boleh mengajukan")
	}
	if !bisaAjukanPerlengkapan(&orderSummary{Status: "dp", Tab: "sebagian"}) {
		t.Error("booking DP boleh mengajukan")
	}
}
