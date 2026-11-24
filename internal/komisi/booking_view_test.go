package komisi_test

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// ringkasKomisi: "penerima:jenis:status:nominal" (nominal "-" bila nil), urut.
func ringkasKomisi(r *komisi.BookingAgenKomisi, nama map[int64]string) string {
	var out []string
	for _, k := range r.Komisi {
		n := "-"
		if k.Nominal != nil {
			n = fmt.Sprintf("%.0f", *k.Nominal)
		}
		out = append(out, fmt.Sprintf("%s:%s:%s:%s", nama[k.Penerima.ID], k.Jenis, k.Status, n))
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// (d) Ringkasan agen dan komisi Detail Booking: estimasi sebelum lunas, tercatat sesudahnya, nominal hanya untuk
// pemegang komisi.view, klaim hanya tampil bila ada pax aktif tanpa agen.
func TestRingkasanBookingAgenKomisi(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	adam := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Adam", StatusAgen: "aktif"})
	nuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nuh", StatusAgen: "aktif", Direkrut: adam, Upline: adam})
	hud := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Hud", StatusAgen: "aktif"})
	nama := map[int64]string{adam: "Adam", nuh: "Nuh", hud: "Hud"}

	umar := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Umar", Direkrut: nuh})
	salman := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Salman", TanpaAgen: true})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 2000, testdb.Pax{JamaahID: umar}, testdb.Pax{JamaahID: salman})
	testdb.Exec(t, tx, `UPDATE bookings SET dibuat_oleh_agen_akun_id=?, agen_klaim_akun_id=? WHERE id=?`, nuh, hud, b)
	if err := komisi.SnapshotNominal(ctx, tx, b); err != nil {
		t.Fatal(err)
	}

	// Sebelum lunas: estimasi dari snapshot.
	r, err := komisi.RingkasanBooking(ctx, tx, b, true)
	if err != nil {
		t.Fatalf("ringkasan dp: %v", err)
	}
	if got, want := ringkasKomisi(r, nama), "Adam:pembinaan:estimasi:300000,Nuh:langsung:estimasi:1000000"; got != want {
		t.Fatalf("estimasi:\n got  %s\n want %s", got, want)
	}
	if !r.LihatNominal || r.Total == nil || *r.Total != 1_300_000 {
		t.Fatalf("total estimasi = %v, lihat=%v", r.Total, r.LihatNominal)
	}
	if r.DibuatOleh == nil || r.DibuatOleh.ID != nuh {
		t.Fatalf("dibuat_oleh = %+v, want Nuh", r.DibuatOleh)
	}
	if r.Klaim == nil || r.Klaim.ID != hud {
		t.Fatalf("klaim = %+v, want Hud (ada pax tanpa agen)", r.Klaim)
	}
	if len(r.Pax) != 2 || r.Pax[0].Agen == nil || r.Pax[0].Agen.ID != nuh || r.Pax[0].KaitanStatus != "terikat_agen" ||
		r.Pax[1].Agen != nil || r.Pax[1].KaitanStatus != "tanpa_agen" || r.Pax[0].UplineBerubah {
		t.Fatalf("pax estimasi = %+v", r.Pax)
	}

	// Tanpa izin komisi.view: nominal dan total kosong.
	r, err = komisi.RingkasanBooking(ctx, tx, b, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ringkasKomisi(r, nama), "Adam:pembinaan:estimasi:-,Nuh:langsung:estimasi:-"; got != want || r.Total != nil || r.LihatNominal {
		t.Fatalf("tanpa nominal: %s total=%v lihat=%v", got, r.Total, r.LihatNominal)
	}

	// Sesudah lunas: komisi tercatat.
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	if err := komisi.ProcessBookingLunas(ctx, tx, b); err != nil {
		t.Fatal(err)
	}
	r, err = komisi.RingkasanBooking(ctx, tx, b, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ringkasKomisi(r, nama), "Adam:pembinaan:tercatat:300000,Nuh:langsung:tercatat:1000000"; got != want {
		t.Fatalf("tercatat:\n got  %s\n want %s", got, want)
	}
	if r.Total == nil || *r.Total != 1_300_000 {
		t.Fatalf("total tercatat = %v", r.Total)
	}
	r, err = komisi.RingkasanBooking(ctx, tx, b, false)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := ringkasKomisi(r, nama), "Adam:pembinaan:tercatat:-,Nuh:langsung:tercatat:-"; got != want || r.Total != nil {
		t.Fatalf("tercatat tanpa nominal: %s total=%v", got, r.Total)
	}

	// Kaitan jamaah berubah setelah komisi tercatat: agen pax tetap penerima tercatat, ditandai upline_berubah.
	testdb.Exec(t, tx, `UPDATE jamaah SET direkrut_oleh_akun_id=? WHERE id=?`, hud, umar)
	r, err = komisi.RingkasanBooking(ctx, tx, b, true)
	if err != nil {
		t.Fatal(err)
	}
	if p := r.Pax[0]; p.Agen == nil || p.Agen.ID != nuh || !p.UplineBerubah {
		t.Fatalf("pax setelah kaitan berubah = %+v, want agen Nuh dengan upline_berubah", p)
	}

	// Pax tanpa agen sudah dikaitkan: klaim tidak ditampilkan lagi.
	testdb.Exec(t, tx, `UPDATE jamaah SET direkrut_oleh_akun_id=?, kaitan_status='terikat_agen' WHERE id=?`, hud, salman)
	r, err = komisi.RingkasanBooking(ctx, tx, b, true)
	if err != nil {
		t.Fatal(err)
	}
	if r.Klaim != nil {
		t.Fatalf("klaim = %+v, want nil tanpa pax tanpa agen", r.Klaim)
	}

	// Booking tidak ada: sql.ErrNoRows.
	if _, err := komisi.RingkasanBooking(ctx, tx, b+999_999, true); err == nil {
		t.Fatal("booking tidak ada tanpa error")
	}
}
