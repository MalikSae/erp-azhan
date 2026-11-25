package agen

import (
	"context"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// Menu Jamaah agen (spek menu Jamaah 2026-11-25): booking buatan agen (termasuk belum DP) + jamaah terikat mulai DP,
// lencana "Anda" untuk agen sendiri, "Isi data" untuk jamaah yang datanya dibuat agen, dan "Perlu tindakan".
func TestMenuJamaahAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Now()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	lampau := testdb.CloneSchedule(t, tx, jadwal)
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2020-01-01' WHERE id=?`, lampau)

	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenMenuJamaahQzp", StatusAgen: "aktif"})
	pelanggan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PelangganMenuJamaahQzp", Direkrut: agen})

	booking := func(nama string, sched int64, status string, akun int64, buatanAgen bool, direkrut int64, pembuat int64) (int64, int64, int64) {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: nama, Direkrut: direkrut})
		var dibuatOleh any
		if pembuat > 0 {
			dibuatOleh = pembuat
		}
		testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id=?`, dibuatOleh, j)
		b, pax := testdb.NewBookingPax(t, tx, sched, status, 1000, testdb.Pax{JamaahID: j})
		var agenPembuat any
		if buatanAgen {
			agenPembuat = agen
		}
		testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=?, dibuat_oleh_agen_akun_id=? WHERE id=?`, akun, agenPembuat, b)
		return b, j, pax[0]
	}

	bBelumDP, jBelumDP, _ := booking("JamaahBelumDPQzp", jadwal, "baru", agen, true, agen, agen)
	_, jDiri, paxDiri := booking("JamaahDiriAgenQzp", jadwal, "dp", agen, true, 0, agen)
	testdb.Exec(t, tx, `UPDATE booking_pax SET saya_sendiri_akun_id=? WHERE id=?`, agen, paxDiri)
	_, jTautan, _ := booking("JamaahTautanQzp", jadwal, "lunas", pelanggan, false, agen, pelanggan)
	_, jProspek, _ := booking("JamaahProspekBaruQzp", jadwal, "baru", pelanggan, false, agen, pelanggan)
	_, jBerangkat, _ := booking("JamaahBerangkatQzp", lampau, "lunas", agen, true, agen, agen)

	items, err := listDataJamaah(ctx, tx, agen, now)
	if err != nil {
		t.Fatal(err)
	}
	per := map[int64]JamaahAgen{}
	for _, it := range items {
		per[it.JamaahID] = it
	}
	if it, ada := per[jBelumDP]; !ada || it.Status != "belum_dp" || !it.BisaIsiData || !it.PerluTindakan || it.Saya {
		t.Errorf("booking agen belum DP: %+v (ada=%v)", it, ada)
	}
	if it := per[jDiri]; !it.Saya || it.Status != "dp" {
		t.Errorf("agen sendiri: %+v", it)
	}
	if it, ada := per[jTautan]; !ada || it.BisaIsiData || it.Status != "lunas" || !it.PerluTindakan {
		t.Errorf("jamaah lewat tautan (data belum lengkap): %+v (ada=%v)", it, ada)
	}
	if _, ada := per[jProspek]; ada {
		t.Errorf("booking pelanggan belum DP adalah prospek, bukan jamaah")
	}
	if it := per[jBerangkat]; it.Status != "berangkat" || it.PerluTindakan {
		t.Errorf("sudah berangkat: %+v", it)
	}

	// Angka dashboard = jumlah baris daftar.
	d, err := getDashboardAgen(ctx, tx, agen)
	if err != nil {
		t.Fatal(err)
	}
	if d.JumlahJamaah != len(items) {
		t.Errorf("jumlah jamaah dashboard %d, daftar %d", d.JumlahJamaah, len(items))
	}

	// Detail terbuka untuk booking agen yang belum DP, berstatus belum_dp.
	det, err := detailJamaahAgen(ctx, tx, agen, bBelumDP, jBelumDP, now)
	if err != nil || det.Status != "belum_dp" || !det.BisaLengkapi {
		t.Errorf("detail belum DP: %+v %v", det, err)
	}
	// Detail booking pelanggan yang belum DP tetap tertutup.
	if _, err := detailJamaahAgen(ctx, tx, agen, 0, jProspek, now); err == nil {
		t.Errorf("detail prospek harus tertutup")
	}
}
