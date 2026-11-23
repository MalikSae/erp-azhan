package agen

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func TestDetailAgenBinaan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Date(2099, 5, 10, 10, 0, 0, 0, zonaWIB)
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})

	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineDetailQzp", StatusAgen: "aktif"})
	binaan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BinaanDetailQzp", StatusAgen: "aktif", Upline: upline, Direkrut: upline})
	cucu := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "CucuDetailQzp", StatusAgen: "aktif", Upline: binaan, Direkrut: binaan})
	uplineLain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineLainQzp", StatusAgen: "aktif"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET domisili = 'Kota Depok', disetujui_agen_at = '2099-01-02 09:00:00', no_hp = '081299990001' WHERE id = ?`, binaan)

	// Jamaah binaan: satu DP, satu lunas, satu sudah berangkat, satu booking baru (tidak dihitung).
	jDP := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahBinaanDPQzp", Direkrut: binaan})
	jLunas := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahBinaanLunasQzp", Direkrut: binaan})
	jLalu := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahBinaanLaluQzp", Direkrut: binaan})
	jBaru := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahBinaanBaruQzp", Direkrut: binaan})
	bDP, paxDP := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: jDP})
	testdb.NewBookingPax(t, tx, jadwal, "lunas", 1000, testdb.Pax{JamaahID: jLunas})
	testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: jBaru})
	jadwalLalu := jadwalLainBrand(t, tx, brand, jadwal, "2099-05-01")
	bLalu, paxLalu := testdb.NewBookingPax(t, tx, jadwalLalu, "lunas", 1000, testdb.Pax{JamaahID: jLalu})

	// Bonus pembinaan upline: tertahan (belum berangkat), tersedia (berangkat + hari tahan sudah lewat), batal (tidak dihitung).
	// Komisi langsung milik binaan tidak boleh ikut. CURRENT_DATE dipakai SQL, jadi jadwal lalu digeser ke masa lampau.
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal = '2020-01-01' WHERE id = ?`, jadwalLalu)
	tambah := func(penerima, jamaah, booking, pax int64, jenis string, nominal int, batal bool) {
		t.Helper()
		var dibatalkan any
		if batal {
			dibatalkan = "2030-05-05 10:00:00"
		}
		testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal, dibatalkan_at)
			VALUES (?,?,?,?,?,?,?,?)`, brand, penerima, jamaah, booking, pax, jenis, nominal, dibatalkan)
	}
	tambah(upline, jDP, bDP, paxDP[0], "pembinaan", 100000, false)
	tambah(upline, jLalu, bLalu, paxLalu[0], "pembinaan", 150000, false)
	tambah(upline, jDP, bDP, paxDP[0], "pembinaan", 70000, true)
	tambah(binaan, jDP, bDP, paxDP[0], "langsung", 500000, false)

	later := time.Date(2020, 6, 1, 10, 0, 0, 0, zonaWIB) // setelah jadwal lalu berangkat, sebelum jadwal utama
	d, err := detailAgenBinaan(ctx, tx, upline, binaan, later)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d.Nama, "BinaanDetailQzp") || d.Status != "aktif" || d.Kota == nil || *d.Kota != "Kota Depok" ||
		d.Bergabung == nil || *d.Bergabung != "2099-01-02" {
		t.Errorf("identitas: %+v", d)
	}
	if d.Jamaah != (JamaahBinaan{DP: 1, Lunas: 1, Berangkat: 1}) || d.PaketAktif != 1 {
		t.Errorf("jamaah (booking baru tidak dihitung): %+v paket %d", d.Jamaah, d.PaketAktif)
	}
	if d.Bonus != (BonusBinaan{Total: 250000, Tertahan: 100000, Tersedia: 150000}) {
		t.Errorf("bonus (batal dan komisi langsung binaan tidak ikut): %+v", d.Bonus)
	}
	if d.WhatsAppURL != "https://wa.me/6281299990001" {
		t.Errorf("whatsapp: %q", d.WhatsAppURL)
	}

	// Rincian: tiga baris terbaru dulu, baris batal tetap ada, nama jamaah tersamar.
	if len(d.Rincian) != 3 {
		t.Fatalf("rincian: %+v", d.Rincian)
	}
	wantRincian := []struct {
		nominal float64
		status  string
	}{{70000, "batal"}, {150000, "tersedia"}, {100000, "tertahan"}}
	for i, w := range wantRincian {
		r := d.Rincian[i]
		if r.Nominal != w.nominal || r.Status != w.status || (w.status == "batal") != (r.DibatalkanAt != nil) {
			t.Errorf("rincian[%d] = %+v, mau %.0f %s", i, r, w.nominal, w.status)
		}
		if !strings.HasPrefix(r.Jamaah, "Ja*") || strings.Contains(r.Jamaah, "JamaahBinaan") || r.Paket == "" || r.Tanggal == "" {
			t.Errorf("rincian[%d] identitas: %+v", i, r)
		}
	}

	// Bukan upline langsung: upline lain dan kakek (dua tingkat) tidak melihat apa pun.
	if _, err := detailAgenBinaan(ctx, tx, uplineLain, binaan, now); !errors.Is(err, ErrNotFound) {
		t.Errorf("upline lain: %v", err)
	}
	if _, err := detailAgenBinaan(ctx, tx, upline, cucu, now); !errors.Is(err, ErrNotFound) {
		t.Errorf("dua tingkat: %v", err)
	}

	// Belum punya jamaah minimal DP: rintisan. Keagenan nonaktif: nonaktif.
	if d, err := detailAgenBinaan(ctx, tx, binaan, cucu, now); err != nil || d.Status != "rintisan" || d.Bonus.Total != 0 {
		t.Errorf("rintisan: err=%v %+v", err, d)
	}
	testdb.Exec(t, tx, `UPDATE customer_accounts SET status_agen = 'nonaktif' WHERE id = ?`, binaan)
	if d, err := detailAgenBinaan(ctx, tx, upline, binaan, later); err != nil || d.Status != "nonaktif" {
		t.Errorf("nonaktif: err=%v %+v", err, d)
	}
}

func TestNamaTersamar(t *testing.T) {
	for nama, want := range map[string]string{
		"Siti Aminah": "Si** Am****",
		"Al":          "Al",
		"  Budi   S ": "Bu** S",
		"Émile Zola":  "Ém*** Zo**",
	} {
		if got := namaTersamar(nama); got != want {
			t.Errorf("namaTersamar(%q) = %q, mau %q", nama, got, want)
		}
	}
}
