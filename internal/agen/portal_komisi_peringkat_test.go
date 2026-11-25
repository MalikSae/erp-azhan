package agen

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func TestKomisiPaketAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	override := 750000.0
	nol := 0.0
	a, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: &override, BerangkatTanggal: "2099-06-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, a)
	testdb.Exec(t, tx, `UPDATE brands SET ujroh_closing=500000 WHERE id=?`, brand)

	ikutBrand := testdb.CloneSchedule(t, tx, a)
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=NULL, status='published' WHERE id=?`, ikutBrand)
	tanpaKomisi := testdb.CloneSchedule(t, tx, a)
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=?, status='published' WHERE id=?`, nol, tanpaKomisi)
	draft := testdb.CloneSchedule(t, tx, a)
	testdb.Exec(t, tx, `UPDATE schedules SET status='draft' WHERE id=?`, draft)
	lampau := testdb.CloneSchedule(t, tx, a)
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', berangkat_tanggal='2020-01-01' WHERE id=?`, lampau)

	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenKomisiPaketQzp", StatusAgen: "aktif"})
	items, err := komisiPaketAgen(ctx, tx, agen)
	if err != nil {
		t.Fatal(err)
	}
	got := map[int64]float64{}
	for _, k := range items {
		got[k.ScheduleID] = k.KomisiLangsung
	}
	if got[a] != 750000 {
		t.Errorf("override jadwal: ingin 750000, dapat %v", got[a])
	}
	if got[ikutBrand] != 500000 {
		t.Errorf("bawaan brand: ingin 500000, dapat %v", got[ikutBrand])
	}
	for nama, id := range map[string]int64{"komisi 0": tanpaKomisi, "draft": draft, "lampau": lampau} {
		if _, ada := got[id]; ada {
			t.Errorf("%s tidak boleh tampil", nama)
		}
	}

	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenNonaktifQzp", StatusAgen: "nonaktif"})
	if _, err := komisiPaketAgen(ctx, tx, nonaktif); !errors.Is(err, ErrBukanAgenAktif) {
		t.Errorf("agen nonaktif: ingin ErrBukanAgenAktif, dapat %v", err)
	}
}

func TestPeringkatPortalAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Date(2037, 3, 15, 10, 0, 0, 0, zonaWIB)
	komisiNominal := 500000.0
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: &komisiNominal, BerangkatTanggal: "2099-06-01"})
	var brandLain int64
	if err := tx.QueryRow(`SELECT id FROM brands WHERE id <> ? ORDER BY id LIMIT 1`, brand).Scan(&brandLain); err != nil {
		t.Fatal(err)
	}

	seq := 0
	// tambah: n pax closing (komisi langsung) untuk agen, tercatat pada tanggal tgl, di brand kom.
	tambah := func(agen int64, n int, tgl string, jenis string, kom int64) {
		pax := make([]testdb.Pax, n)
		for i := range pax {
			seq++
			pax[i] = testdb.Pax{JamaahID: testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: fmt.Sprintf("JamaahPeringkat%dQzp", seq), Direkrut: agen})}
		}
		b, ids := testdb.NewBookingPax(t, tx, jadwal, "lunas", 1000, pax...)
		for i, id := range ids {
			testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal, created_at)
				VALUES (?,?,?,?,?,?,500000,?)`, kom, agen, pax[i].JamaahID, b, id, jenis, tgl+" 09:00:00")
		}
	}

	saya := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenSayaQzp", StatusAgen: "aktif"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET domisili='Depok' WHERE id=?`, saya)
	juara := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenJuaraQzp", StatusAgen: "aktif"})
	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenNonaktifPeringkatQzp", StatusAgen: "nonaktif"})
	pembina := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenPembinaQzp", StatusAgen: "aktif"})

	tambah(saya, 2, "2037-03-10", "langsung", brand)
	tambah(juara, 3, "2037-03-02", "langsung", brand)
	tambah(juara, 4, "2037-01-20", "langsung", brand) // hanya masuk periode tahun
	tambah(nonaktif, 5, "2037-03-05", "langsung", brand)
	tambah(pembina, 1, "2037-03-05", "pembinaan", brand) // 0 pax closing
	tambah(saya, 9, "2037-03-11", "langsung", brandLain) // brand lain tidak dihitung

	p, err := peringkatPortalAgen(ctx, tx, saya, "bulan", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Dari != "2037-03-01" || p.Sampai != "2037-03-15" || p.JumlahAgen != 2 {
		t.Fatalf("bulan: %+v", p)
	}
	if p.Saya.Peringkat == nil || *p.Saya.Peringkat != 2 || p.Saya.PaxClosing != 2 {
		t.Errorf("posisi saya bulan: %+v", p.Saya)
	}
	if p.Saya.Target == nil || p.Saya.Target.Peringkat != 1 || p.Saya.Target.KurangPax != 1 {
		t.Errorf("target saya bulan: %+v", p.Saya.Target)
	}
	if p.Teratas[0].Saya || p.Teratas[0].Peringkat != 1 {
		t.Errorf("#1 bukan juara: %+v", p.Teratas[0])
	}
	if len(p.Teratas) != 2 || p.Teratas[0].PaxClosing != 3 || !p.Teratas[1].Saya || p.Teratas[1].Kota == nil || *p.Teratas[1].Kota != "Depok" {
		t.Errorf("teratas bulan: %+v", p.Teratas)
	}

	p, err = peringkatPortalAgen(ctx, tx, saya, "tahun", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Dari != "2037-01-01" || len(p.Teratas) != 2 || p.Teratas[0].PaxClosing != 7 {
		t.Errorf("tahun: %+v", p)
	}

	if _, err := peringkatPortalAgen(ctx, tx, saya, "minggu", now); !errors.Is(err, ErrPeriodePeringkat) {
		t.Errorf("periode salah: %v", err)
	}

	// Agen tanpa pax closing: tanpa nomor.
	p, err = peringkatPortalAgen(ctx, tx, pembina, "bulan", now)
	if err != nil {
		t.Fatal(err)
	}
	if p.Saya.Peringkat != nil || p.Saya.PaxClosing != 0 || p.Saya.Target != nil {
		t.Errorf("tanpa pax: %+v", p.Saya)
	}

	// Di luar 10 besar: baris sendiri ditambahkan setelah 10 teratas.
	for i := 0; i < 10; i++ {
		lain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: fmt.Sprintf("AgenRamai%dQzp", i), StatusAgen: "aktif"})
		tambah(lain, 3, "2037-03-12", "langsung", brand)
	}
	p, err = peringkatPortalAgen(ctx, tx, saya, "bulan", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Teratas) != MaksPeringkatPortal+1 || !p.Teratas[MaksPeringkatPortal].Saya || *p.Saya.Peringkat != 12 || p.JumlahAgen != 12 {
		t.Errorf("di luar 10 besar: jumlah %d, saya %+v, akhir %+v", p.JumlahAgen, p.Saya, p.Teratas[len(p.Teratas)-1])
	}
	// Peringkat bersama: juara dan 10 agen ramai sama-sama 3 pax = #1; saya 2 pax = #12, target #1 kurang 1.
	for _, b := range p.Teratas[:MaksPeringkatPortal] {
		if b.Peringkat != 1 || b.PaxClosing != 3 {
			t.Errorf("seri 3 pax harus #1: %+v", b)
		}
	}
	if p.Saya.Target == nil || p.Saya.Target.Peringkat != 1 || p.Saya.Target.KurangPax != 1 {
		t.Errorf("target di luar 10 besar: %+v", p.Saya.Target)
	}
	// Juara sendiri: tanpa target.
	if pj, err := peringkatPortalAgen(ctx, tx, juara, "tahun", now); err != nil || pj.Saya.Peringkat == nil || *pj.Saya.Peringkat != 1 || pj.Saya.Target != nil {
		t.Errorf("juara tahun: %+v %v", pj, err)
	}
	for _, b := range p.Teratas[:MaksPeringkatPortal] {
		if b.Saya {
			t.Errorf("baris saya muncul dua kali")
		}
	}
}

func TestFotoAgenSebrand(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	saya := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenFotoSayaQzp", StatusAgen: "aktif"})
	lain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenFotoLainQzp", StatusAgen: "aktif"})
	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenFotoNonaktifQzp", StatusAgen: "nonaktif"})
	foto := "/api/admin/media/dokumen-jamaah/0f8fad5b-d9cb-469f-a165-70867728950e.jpg"
	testdb.Exec(t, tx, `UPDATE customer_accounts SET foto_url=? WHERE id IN (?, ?)`, foto, lain, nonaktif)

	berkas, err := fotoAgenSebrand(ctx, tx, saya, lain)
	if err != nil || berkas != "uploads/dokumen-jamaah/0f8fad5b-d9cb-469f-a165-70867728950e.jpg" {
		t.Errorf("agen sebrand: %q %v", berkas, err)
	}
	if berkas, _ := fotoAgenSebrand(ctx, tx, saya, nonaktif); berkas != "" {
		t.Errorf("agen nonaktif tidak boleh: %q", berkas)
	}
	if berkas, _ := fotoAgenSebrand(ctx, tx, saya, saya); berkas != "" {
		t.Errorf("tanpa foto: %q", berkas)
	}
	if _, err := fotoAgenSebrand(ctx, tx, nonaktif, lain); !errors.Is(err, ErrBukanAgenAktif) {
		t.Errorf("pemanggil nonaktif: %v", err)
	}
}
