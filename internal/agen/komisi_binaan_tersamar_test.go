package agen

import (
	"context"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit keamanan 2026-11-26 (Q3): bonus pembinaan berasal dari jamaah milik downline. Di portal agen (dashboard dan
// mutasi; riwayat komisi memakai samarkanSumberBinaan yang sama) nama jamaah itu tersamar dan kode bookingnya kosong,
// sama seperti Detail Binaan. Komisi dari jamaah agen sendiri tetap utuh.
func TestKomisiPembinaanTersamarDiPortal(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineSamarQzp", StatusAgen: "aktif"})
	downline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "DownlineSamarQzp", StatusAgen: "aktif", Upline: upline, Direkrut: upline})

	jSendiri := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Jamaah Sendiri Qzp", Direkrut: upline})
	jDownline := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Jamaah Downline Qzp", Direkrut: downline})
	bSendiri, paxSendiri := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: jSendiri})
	bDownline, paxDownline := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: jDownline})
	testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal, created_at)
		VALUES (?,?,?,?,?,'langsung',500000,'2030-05-01 10:00:00')`, brand, upline, jSendiri, bSendiri, paxSendiri[0])
	testdb.Exec(t, tx, `INSERT INTO transaksi_komisi (brand_id, akun_penerima_id, jamaah_sumber_id, booking_id, booking_pax_id, jenis, nominal, created_at)
		VALUES (?,?,?,?,?,'pembinaan',100000,'2030-05-02 10:00:00')`, brand, upline, jDownline, bDownline, paxDownline[0])

	d, err := getDashboardAgen(ctx, tx, upline)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.KomisiTerbaru) != 2 {
		t.Fatalf("komisi terbaru: %+v", d.KomisiTerbaru)
	}
	for _, k := range d.KomisiTerbaru {
		switch k.Jenis {
		case "pembinaan":
			if strings.Contains(k.SumberNama, "Downline") || !strings.HasPrefix(k.SumberNama, "Ja**** Do****** Qz*") || k.BookingCode != "" {
				t.Errorf("pembinaan tidak tersamar: nama=%q kode=%q", k.SumberNama, k.BookingCode)
			}
		case "langsung":
			if !strings.HasPrefix(k.SumberNama, "Jamaah Sendiri Qzp") || k.BookingCode == "" {
				t.Errorf("komisi langsung ikut tersamar: nama=%q kode=%q", k.SumberNama, k.BookingCode)
			}
		}
	}

	mutasi, err := listMutasiAgen(ctx, tx, upline)
	if err != nil {
		t.Fatal(err)
	}
	judul := []string{}
	for _, m := range mutasi {
		judul = append(judul, m.Judul)
	}
	gabung := strings.Join(judul, " | ")
	if strings.Contains(gabung, "Downline") || !strings.Contains(gabung, "Komisi · Ja**** Do****** Qz*") ||
		!strings.Contains(gabung, "Komisi · Jamaah Sendiri Qzp") {
		t.Errorf("judul mutasi: %s", gabung)
	}
}
