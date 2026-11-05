package agen

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Pengajuan menyimpan jenis kelamin dan tanggal lahir di akun, dan antrean admin membacanya.
func TestPengajuanMemuatBiodata(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "CalonAgenBio"})
	req := AjukanRequest{NIK: "9999999999990701", Domisili: "Kota Bandung", JenisKelamin: "P", TanggalLahir: "1992-03-15", SetujuSyaratKetentuan: true}
	if err := ajukanTx(ctx, tx, akun, req); err != nil {
		t.Fatal(err)
	}
	items, err := listPengajuan(ctx, tx, &brand)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range items {
		if p.AkunID != akun {
			continue
		}
		if p.JenisKelamin == nil || *p.JenisKelamin != "P" || p.TanggalLahir == nil || *p.TanggalLahir != "1992-03-15" {
			t.Fatalf("biodata = %v %v", p.JenisKelamin, p.TanggalLahir)
		}
		return
	}
	t.Fatal("pengajuan tidak ada di antrean")
}
