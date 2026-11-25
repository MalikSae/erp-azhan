package agen

import (
	"context"
	"errors"
	"testing"
	"time"

	"erp-azhan/api/internal/notifikasi"
	"erp-azhan/api/internal/testdb"
)

func TestProspekAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	now := time.Now()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, jadwal)
	penuh := testdb.CloneSchedule(t, tx, jadwal)
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', seat_sisa=0 WHERE id=?`, penuh)

	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenProspekQzp", StatusAgen: "aktif"})
	agenLain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLainProspekQzp", StatusAgen: "aktif"})

	// booking: booking baru milik akun, dengan hold tertentu dan waktu dibuat tertentu.
	booking := func(akun, sched int64, status string, hold *time.Time, dibuat time.Time, buatanAgen int64) int64 {
		j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahProspekQzp"})
		b, _ := testdb.NewBookingPax(t, tx, sched, status, 1000, testdb.Pax{JamaahID: j})
		var agenPembuat any
		if buatanAgen > 0 {
			agenPembuat = buatanAgen
		}
		testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=?, is_seat_blocked=?, seat_hold_expires_at=?, created_at=?, dibuat_oleh_agen_akun_id=? WHERE id=?`,
			akun, hold != nil, hold, dibuat, agenPembuat, b)
		return b
	}
	akun := func(nama string, perekrut int64) int64 {
		return testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: nama, Direkrut: perekrut})
	}

	holdCepat, holdLambat := now.Add(2*time.Hour), now.Add(20*time.Hour)
	aCepat := akun("ProspekHoldCepatQzp", agen)
	bCepat := booking(aCepat, jadwal, "baru", &holdCepat, now.Add(-22*time.Hour), 0)
	aLambat := akun("ProspekHoldLambatQzp", agen)
	booking(aLambat, jadwal, "baru", &holdLambat, now.Add(-4*time.Hour), 0)
	aHabis := akun("ProspekHoldHabisQzp", agen)
	booking(aHabis, jadwal, "baru", nil, now.Add(-48*time.Hour), 0)
	aPenuh := akun("ProspekJadwalPenuhQzp", agen)
	booking(aPenuh, penuh, "baru", nil, now.Add(-72*time.Hour), 0)
	aBaru := akun("ProspekBelumBookingQzp", agen)

	// Tidak ikut: sudah DP, buatan agen sendiri, akun agen lain, lebih dari 30 hari, akun yang sudah jadi agen.
	aDP := akun("ProspekSudahDPQzp", agen)
	booking(aDP, jadwal, "dp", nil, now.Add(-5*time.Hour), 0)
	aBuatan := akun("ProspekBuatanAgenQzp", agen)
	booking(aBuatan, jadwal, "baru", &holdLambat, now.Add(-1*time.Hour), agen)
	aLain := akun("ProspekAgenLainQzp", agenLain)
	booking(aLain, jadwal, "baru", &holdLambat, now.Add(-1*time.Hour), 0)
	aLama := akun("ProspekLamaQzp", agen)
	booking(aLama, jadwal, "baru", nil, now.AddDate(0, 0, -31), 0)
	binaan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "ProspekBinaanQzp", StatusAgen: "aktif", Direkrut: agen})

	var hpCepat string
	if err := tx.QueryRow(`SELECT no_hp FROM customer_accounts WHERE id=?`, aCepat).Scan(&hpCepat); err != nil {
		t.Fatal(err)
	}
	waCepat := "62" + hpCepat[1:]
	items, err := prospekAgen(ctx, tx, agen, now)
	if err != nil {
		t.Fatal(err)
	}
	var urut []int64
	tahap := map[int64]Prospek{}
	for _, p := range items {
		urut = append(urut, p.AkunID)
		tahap[p.AkunID] = p
	}
	ingin := []int64{aCepat, aLambat, aHabis, aPenuh, aBaru}
	if len(urut) != len(ingin) {
		t.Fatalf("jumlah prospek: ingin %v, dapat %v", ingin, urut)
	}
	for i := range ingin {
		if urut[i] != ingin[i] {
			t.Fatalf("urutan prospek: ingin %v, dapat %v", ingin, urut)
		}
	}
	if p := tahap[aCepat]; p.Tahap != TahapBelumDP || p.HoldSampai == nil || p.BookingID == nil || *p.BookingID != bCepat || p.WhatsApp != waCepat || p.JumlahJamaah != 1 {
		t.Errorf("belum DP: %+v", p)
	}
	if p := tahap[aHabis]; p.Tahap != TahapHoldHabis || p.Keterangan != nil || p.HoldSampai != nil {
		t.Errorf("hold habis: %+v", p)
	}
	if p := tahap[aPenuh]; p.Tahap != TahapHoldHabis || p.Keterangan == nil || *p.Keterangan != "jadwal_penuh" {
		t.Errorf("jadwal penuh: %+v", p)
	}
	if p := tahap[aBaru]; p.Tahap != TahapBelumBooking || p.BookingID != nil || p.Paket != nil {
		t.Errorf("belum booking: %+v", p)
	}
	for _, id := range []int64{aDP, aBuatan, aLain, aLama, binaan} {
		if _, ada := tahap[id]; ada {
			t.Errorf("akun %d tidak boleh jadi prospek", id)
		}
	}

	// Pendaftaran tutup: jadwal berangkat kurang dari 30 hari lagi.
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal=DATE_ADD(CURRENT_DATE, INTERVAL 10 DAY) WHERE id=?`, penuh)
	items, _ = prospekAgen(ctx, tx, agen, now)
	for _, p := range items {
		if p.AkunID == aPenuh && (p.Keterangan == nil || *p.Keterangan != "pendaftaran_tutup") {
			t.Errorf("pendaftaran tutup: %+v", p)
		}
	}

	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenNonaktifProspekQzp", StatusAgen: "nonaktif"})
	if _, err := prospekAgen(ctx, tx, nonaktif, now); !errors.Is(err, ErrBukanAgenAktif) {
		t.Errorf("agen nonaktif: %v", err)
	}

	// Notifikasi "Prospek baru": hanya ke agen perekrut, tidak untuk booking buatan agen.
	notifikasi.ProspekBaru(ctx, tx, bCepat)
	notifikasi.ProspekBaru(ctx, tx, bCepat) // tidak dobel
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM notifikasi WHERE akun_id=? AND jenis='prospek_baru'`, agen).Scan(&n); err != nil || n != 1 {
		t.Errorf("prospek baru: %d %v", n, err)
	}
}
