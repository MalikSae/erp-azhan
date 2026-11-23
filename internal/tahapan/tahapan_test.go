package tahapan

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestMuatBooking(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	j1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "TahapSatuQzp"})
	j2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "TahapDuaQzp"})
	j3 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "TahapBatalQzp"})
	b, pax := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j1}, testdb.Pax{JamaahID: j2}, testdb.Pax{JamaahID: j3, Batal: true})

	// Jamaah 1: perlengkapan diserahkan, paspor disetujui, SISKOPATUH selesai bertanggal, visa belum.
	testdb.Exec(t, tx, `UPDATE booking_pax SET perlengkapan_status='sudah_diberikan', perlengkapan_diserahkan_at='2099-01-05 10:00:00',
		progress_siskopatuh=1, progress_siskopatuh_at='2099-02-01 09:00:00' WHERE id=?`, pax[0])
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, status, updated_at) VALUES (?, 'paspor', 'approved', '2030-01-10 08:00:00')`, j1)
	// Jamaah 2: paspor ditolak, visa dicentang sebelum ada kolom tanggal (tanggal kosong).
	testdb.Exec(t, tx, `UPDATE booking_pax SET progress_visa=1 WHERE id=?`, pax[1])
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, status) VALUES (?, 'paspor', 'rejected')`, j2)
	// Tiket per booking: berlaku untuk semua jamaah.
	testdb.Exec(t, tx, `UPDATE bookings SET progress_tiket=1, progress_tiket_at='2099-03-01 12:00:00' WHERE id=?`, b)

	semua, err := MuatBooking(ctx, tx, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(semua) != 2 || semua[0].PaxID != pax[0] || semua[1].PaxID != pax[1] {
		t.Fatalf("pax aktif saja, urut id: %+v", semua)
	}
	type mau struct{ kunci, status, tanggal string }
	cek := func(nama string, got []Tahap, want []mau) {
		t.Helper()
		if len(got) != len(want) {
			t.Fatalf("%s: %d tahap, mau %d", nama, len(got), len(want))
		}
		for i, w := range want {
			g := got[i]
			tgl := ""
			if g.SelesaiAt != nil {
				tgl = *g.SelesaiAt
			}
			if g.Kunci != w.kunci || g.Status != w.status || tgl != w.tanggal {
				t.Errorf("%s tahap %d = %s/%s/%q, mau %s/%s/%q", nama, i, g.Kunci, g.Status, tgl, w.kunci, w.status, w.tanggal)
			}
		}
	}
	cek("jamaah 1", semua[0].Tahapan, []mau{
		{Perlengkapan, Selesai, "2099-01-05"},
		{Paspor, Selesai, "2030-01-10"},
		{Siskopatuh, Selesai, "2099-02-01"},
		{Visa, Diproses, ""},
		{Tiket, Selesai, "2099-03-01"},
	})
	cek("jamaah 2", semua[1].Tahapan, []mau{
		{Perlengkapan, Diproses, ""},
		{Paspor, Perbaiki, ""},
		{Siskopatuh, Diproses, ""},
		{Visa, Selesai, ""},
		{Tiket, Selesai, "2099-03-01"},
	})

	if got, err := UntukJamaah(ctx, tx, b, j2); err != nil || len(got) != 5 || got[1].Status != Perbaiki {
		t.Errorf("untuk jamaah 2: %+v err=%v", got, err)
	}
	if got, err := UntukJamaah(ctx, tx, b, j3); err != nil || got != nil {
		t.Errorf("pax batal tidak punya tahapan: %+v err=%v", got, err)
	}
}

// Infant tidak mendapat jatah perlengkapan, jadi tahap Perlengkapan tidak ada untuknya (empat tahap, mulai Paspor).
func TestTahapanInfantTanpaPerlengkapan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	dewasa := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "TahapDewasaQzp"})
	bayi := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "TahapBayiQzp"})
	b, pax := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: dewasa}, testdb.Pax{JamaahID: bayi})
	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_type='infant', counts_for_seat=0 WHERE id=?`, pax[1])

	semua, err := MuatBooking(ctx, tx, b)
	if err != nil {
		t.Fatal(err)
	}
	if len(semua) != 2 || len(semua[0].Tahapan) != 5 || semua[0].Tahapan[0].Kunci != Perlengkapan {
		t.Fatalf("dewasa: %+v", semua)
	}
	got := semua[1].Tahapan
	if len(got) != 4 || got[0].Kunci != Paspor || got[3].Kunci != Tiket {
		t.Errorf("infant: %+v, mau 4 tahap tanpa perlengkapan", got)
	}
}
