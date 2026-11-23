package agen

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func TestDetailJamaahAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Date(2099, 5, 10, 10, 0, 0, 0, zonaWIB)
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})

	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenDetailQzp", StatusAgen: "aktif"})
	agenLain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLainQzp", StatusAgen: "aktif"})
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahDetailQzp", Direkrut: agen, NIK: "9900000000000001"})
	testdb.Exec(t, tx, `UPDATE jamaah SET tempat_lahir='Depok', tanggal_lahir='1990-01-01', no_hp='081200001111', dibuat_oleh_akun_id=? WHERE id=?`, agen, j1)
	b1, pax1 := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j1})
	testdb.Exec(t, tx, `UPDATE booking_pax SET progress_visa=1 WHERE id=?`, pax1[0])
	testdb.AddPayment(t, tx, b1, 400, "confirmed")
	testdb.AddPayment(t, tx, b1, 300, "pending")
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, status) VALUES (?, 'paspor', 'approved'), (?, 'kk', 'approved')`, j1, j1)
	testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal)
		VALUES (?,?,?,?,?,'langsung',500000)`, brand, agen, j1, b1, pax1[0])
	// Bonus pembinaan upline dari jamaah ini (penerima akun lain): tampil sebagai bonus_upline, bukan komisi agen.
	testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal)
		VALUES (?,?,?,?,?,'pembinaan',100000)`, brand, agenLain, j1, b1, pax1[0])

	d, err := detailJamaahAgen(ctx, tx, agen, b1, j1, now)
	if err != nil {
		t.Fatal(err)
	}
	if d.BonusUpline == nil || d.BonusUpline.Nominal != 100000 || d.BonusUpline.Status != "tertahan" {
		t.Errorf("bonus upline: %+v", d.BonusUpline)
	}
	if d.Status != "dp" || !strings.HasPrefix(d.Nama, "JamaahDetailQzp") || d.Jenis != "umroh" {
		t.Errorf("dasar: %+v", d)
	}
	if !strings.HasSuffix(d.NIKTersamar, "0001") || strings.Contains(d.NIKTersamar, "990000") {
		t.Errorf("NIK harus tersamar: %s", d.NIKTersamar)
	}
	if d.Pembayaran == nil || d.Pembayaran.Dibayar != 400 || d.Pembayaran.Sisa != 600 || d.Pembayaran.TenggatLunas == "" {
		t.Errorf("pembayaran (hanya confirmed): %+v", d.Pembayaran)
	}
	if d.Kelengkapan == nil || d.Kelengkapan.Data.Terisi != 3 || d.Kelengkapan.Data.Total != 10 ||
		d.Kelengkapan.Dokumen.Terisi != 1 || d.Kelengkapan.Dokumen.Total != 3 {
		t.Errorf("kelengkapan (kk bukan dokumen wajib): %+v", d.Kelengkapan)
	}
	if !d.Tahapan.Visa || d.Tahapan.Manasik || d.Tahapan.Perlengkapan {
		t.Errorf("tahapan: %+v", d.Tahapan)
	}
	if d.Komisi == nil || d.Komisi.Nominal != 500000 || d.Komisi.Status != "tertahan" {
		t.Errorf("komisi: %+v", d.Komisi)
	}
	if !d.BisaLengkapi || d.WhatsAppURL != "https://wa.me/6281200001111" {
		t.Errorf("aksi: bisa_lengkapi=%v wa=%q", d.BisaLengkapi, d.WhatsAppURL)
	}

	// Agen lain (bukan perekrut) tidak melihat apa pun.
	if _, err := detailJamaahAgen(ctx, tx, agenLain, b1, j1, now); !errors.Is(err, ErrNotFound) {
		t.Errorf("agen lain: %v", err)
	}
	// Jamaah dibuat akun lain (agen hanya perekrut): detail tetap ada, tetapi tanpa aksi Lengkapi data.
	testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id=?`, agenLain, j1)
	d, err = detailJamaahAgen(ctx, tx, agen, b1, j1, now)
	if err != nil || d.BisaLengkapi {
		t.Errorf("jamaah referral: err=%v bisa_lengkapi=%v", err, d != nil && d.BisaLengkapi)
	}
	// Komisi dibatalkan.
	testdb.Exec(t, tx, `UPDATE transaksi_komisi SET dibatalkan_at=NOW(), sebab_batal='batal' WHERE booking_id=?`, b1)
	d, _ = detailJamaahAgen(ctx, tx, agen, b1, j1, now)
	if d.Komisi == nil || d.Komisi.Status != "batal" {
		t.Errorf("komisi batal: %+v", d.Komisi)
	}

	// Booking baru (belum DP) dan booking batal tidak punya halaman.
	bBaru, _ := testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: j1})
	if _, err := detailJamaahAgen(ctx, tx, agen, bBaru, j1, now); !errors.Is(err, ErrNotFound) {
		t.Errorf("booking baru: %v", err)
	}
	bBatal, _ := testdb.NewBookingPax(t, tx, jadwal, "batal", 1000, testdb.Pax{JamaahID: j1})
	if _, err := detailJamaahAgen(ctx, tx, agen, bBatal, j1, now); !errors.Is(err, ErrNotFound) {
		t.Errorf("booking batal: %v", err)
	}

	// Sudah berangkat: pembayaran dan kelengkapan disembunyikan.
	berangkat := now.AddDate(0, 0, 30)
	d, err = detailJamaahAgen(ctx, tx, agen, b1, j1, berangkat)
	if err != nil || d.Status != "berangkat" || d.Pembayaran != nil || d.Kelengkapan != nil {
		t.Errorf("berangkat: err=%v %+v", err, d)
	}
}
