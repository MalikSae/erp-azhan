package selfbooking

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit keamanan 2026-11-26 (Q4): Jamaah Saya (pemilih jamaah di form booking agen) mengikuti aturan akses data jamaah
// minimal DP. Anggota rombongan prospek yang belum DP tidak muncul; data buatan agen sendiri dan jamaah dari booking
// yang dibuat agen tetap muncul.
func TestJamaahSayaMinimalDP(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenJamaahSayaQzp", StatusAgen: "aktif"})

	prospek := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "ProspekBelumDpQzp", Direkrut: agen})
	testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: prospek})

	sudahDP := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "SudahDpQzp", Direkrut: agen})
	testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: sudahDP})

	buatanAgen := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "BuatanAgenQzp", Direkrut: agen})
	testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id = ? WHERE id = ?`, agen, buatanAgen)

	dariBookingAgen := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "DariBookingAgenQzp", Direkrut: agen})
	b, _ := testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: dariBookingAgen})
	testdb.Exec(t, tx, `UPDATE bookings SET dibuat_oleh_agen_akun_id = ? WHERE id = ?`, agen, b)

	items, err := listJamaahSaya(ctx, tx, agen, "")
	if err != nil {
		t.Fatal(err)
	}
	ada := map[int64]bool{}
	for _, it := range items {
		ada[it.ID] = true
	}
	if ada[prospek] {
		t.Errorf("jamaah prospek yang belum DP ikut tampil di Jamaah Saya")
	}
	for nama, id := range map[string]int64{"sudah DP": sudahDP, "buatan agen": buatanAgen, "dari booking agen": dariBookingAgen} {
		if !ada[id] {
			t.Errorf("jamaah %s tidak tampil", nama)
		}
	}
}
