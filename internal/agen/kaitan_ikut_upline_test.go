package agen

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// Keputusan 12 November 2026: saat admin mengganti upline akun, jamaah yang booking pertamanya dibuat
// akun itu dan masih terikat ke upline lama ikut pindah. Komisi yang sudah tercatat tetap di agen lama.
//
// Simulasi: akun Rudi (upline Hannah) membooking Budi, Siti, Ahmad, Rina dalam satu booking.
//   Budi  : orang baru, booking pertamanya lewat Rudi, terikat Hannah        → ikut pindah
//   Siti  : booking pertamanya lewat akun lain, terikat Gibran (link Gibran) → tetap
//   Ahmad : dipasang admin ke Malik (jalur3)                                 → tetap
//   Rina  : terikat Hannah lewat akunnya sendiri (booking pertama di akun Rina) → tetap
//   Dewi  : booking pertamanya lewat Rudi, terikat Hannah, tapi dipilih admin (jalur3) → tetap

type simUpline struct {
	brand, sched                  int64
	admin                         int64
	hannah, gibran, malik, yusuf  int64
	rudi                          int64
	budi, siti, ahmad, rina, dewi int64
	bookingRudi                   int64
}

func siapkanSimUpline(t *testing.T, tx *sql.Tx) simUpline {
	t.Helper()
	ctx := context.Background()
	var s simUpline
	s.sched, s.brand = testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1_000_000), BerangkatTanggal: "2099-01-01"})
	s.admin = adminUji(t, tx)
	s.hannah = testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "Hannah", StatusAgen: "aktif"})
	s.gibran = testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "Gibran", StatusAgen: "aktif"})
	s.malik = testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "Malik", StatusAgen: "aktif"})
	s.yusuf = testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "Yusuf", StatusAgen: "aktif"})
	s.rudi = testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "Rudi", Direkrut: s.hannah})
	akunRina := testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "AkunRina", Direkrut: s.hannah})
	akunSiti := testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "AkunSiti", Direkrut: s.gibran})

	jamaah := func(nama string, agen int64, sumber string) int64 {
		id := testdb.NewJamaah(t, tx, s.brand, testdb.JamaahOpts{Nama: nama, Direkrut: agen})
		testdb.Exec(t, tx, `UPDATE jamaah SET kaitan_sumber = ? WHERE id = ?`, sumber, id)
		return id
	}
	s.budi = jamaah("Budi", s.hannah, "jalur2")
	s.siti = jamaah("Siti", s.gibran, "jalur2")
	s.ahmad = jamaah("Ahmad", s.malik, "jalur3")
	s.rina = jamaah("Rina", s.hannah, "jalur2")
	s.dewi = jamaah("Dewi", s.hannah, "jalur3")

	booking := func(akun int64, status string, paxes ...testdb.Pax) int64 {
		b, _ := testdb.NewBookingPax(t, tx, s.sched, status, 1000, paxes...)
		testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, akun, b)
		return b
	}
	// Booking pertama Rina dan Siti dibuat di akun mereka sendiri, sebelum booking Rudi.
	booking(akunRina, "batal", testdb.Pax{JamaahID: s.rina})
	booking(akunSiti, "batal", testdb.Pax{JamaahID: s.siti})
	s.bookingRudi = booking(s.rudi, "dp",
		testdb.Pax{JamaahID: s.budi}, testdb.Pax{JamaahID: s.siti}, testdb.Pax{JamaahID: s.ahmad},
		testdb.Pax{JamaahID: s.rina}, testdb.Pax{JamaahID: s.dewi})

	// Booking Rudi lunas: komisi tercatat untuk agen tempat masing-masing jamaah terikat.
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, s.bookingRudi))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, s.bookingRudi)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, s.bookingRudi))
	return s
}

func penerimaKomisi(t *testing.T, tx *sql.Tx, bookingID, jamaahID int64) int64 {
	t.Helper()
	var id int64
	must(t, "penerima komisi", tx.QueryRow(`SELECT akun_penerima_id FROM transaksi_komisi
		WHERE booking_id = ? AND jamaah_sumber_id = ? AND jenis = 'langsung' AND dibatalkan_at IS NULL`, bookingID, jamaahID).Scan(&id))
	return id
}

func TestGantiUplineMembawaJamaahBawaanAkun(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	s := siapkanSimUpline(t, tx)

	n, err := jamaahIkutUpline(ctx, tx, s.rudi, s.hannah, false)
	must(t, "pratinjau", err)
	if len(n) != 1 || n[0] != s.budi {
		t.Fatalf("pratinjau = %v, want hanya Budi (%d)", n, s.budi)
	}

	var dipindah int
	must(t, "ganti upline Rudi ke Yusuf", gantiAgenAkunDanJamaahTx(ctx, tx, nil, s.rudi, s.admin,
		GantiAgenAkunRequest{AgenAkunID: &s.yusuf, Alasan: "salah pasang upline"}, &dipindah))
	if dipindah != 1 {
		t.Fatalf("dipindah = %d, want 1", dipindah)
	}

	cek := func(nama string, jamaah, wantAgen int64) {
		t.Helper()
		if st, by, _ := kaitanSumber(t, tx, jamaah); st != "terikat_agen" || by != wantAgen {
			t.Fatalf("%s: kaitan = %s/%d, want terikat_agen/%d", nama, st, by, wantAgen)
		}
	}
	cek("Budi", s.budi, s.yusuf)
	cek("Siti", s.siti, s.gibran)
	cek("Ahmad", s.ahmad, s.malik)
	cek("Rina", s.rina, s.hannah)
	cek("Dewi", s.dewi, s.hannah)

	// Komisi yang sudah tercatat tetap milik agen lama.
	if got := penerimaKomisi(t, tx, s.bookingRudi, s.budi); got != s.hannah {
		t.Fatalf("komisi Budi pindah ke %d, want tetap Hannah (%d)", got, s.hannah)
	}

	// Riwayat Budi mencatat perpindahan beserta sebabnya; asal kaitan tidak berubah.
	logs, err := listKaitanLog(ctx, tx, s.budi)
	must(t, "log Budi", err)
	if len(logs) != 1 || !strings.Contains(logs[0].Alasan, "Ikut ganti upline akun Rudi") || logs[0].KomisiDipindah != 0 {
		t.Fatalf("log Budi = %+v", logs)
	}
	if _, _, sumber := kaitanSumber(t, tx, s.budi); sumber != "jalur2" {
		t.Fatalf("sumber Budi = %q, want tetap jalur2", sumber)
	}

	// Ganti lagi: Budi ikut lagi karena ikatannya masih berasal dari akun Rudi.
	must(t, "ganti upline Rudi ke Gibran", gantiAgenAkunDanJamaahTx(ctx, tx, nil, s.rudi, s.admin,
		GantiAgenAkunRequest{AgenAkunID: &s.gibran, Alasan: "pindah lagi"}, &dipindah))
	if dipindah != 1 {
		t.Fatalf("dipindah kedua = %d, want 1", dipindah)
	}
	cek("Budi (kedua)", s.budi, s.gibran)
	cek("Siti (kedua)", s.siti, s.gibran)
}

func TestLepasUplineMembuatJamaahBawaanTanpaAgen(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	s := siapkanSimUpline(t, tx)

	var dipindah int
	must(t, "lepas upline Rudi", gantiAgenAkunDanJamaahTx(ctx, tx, nil, s.rudi, s.admin,
		GantiAgenAkunRequest{AgenAkunID: nil, Alasan: "akun tanpa upline"}, &dipindah))
	if dipindah != 1 {
		t.Fatalf("dipindah = %d, want 1", dipindah)
	}
	// Budi sudah punya komisi tercatat, tetap boleh jadi tanpa agen di kasus ini.
	if st, by, _ := kaitanSumber(t, tx, s.budi); st != "tanpa_agen" || by != 0 {
		t.Fatalf("Budi: kaitan = %s/%d, want tanpa_agen", st, by)
	}
	if got := penerimaKomisi(t, tx, s.bookingRudi, s.budi); got != s.hannah {
		t.Fatalf("komisi Budi pindah ke %d, want tetap Hannah", got)
	}
	if st, by, _ := kaitanSumber(t, tx, s.rina); st != "terikat_agen" || by != s.hannah {
		t.Fatalf("Rina ikut berubah: %s/%d", st, by)
	}
}

// Booking draft (belum mengikat siapa pun) yang lebih dulu dibuat tidak boleh menentukan asal ikatan.
// Eko punya draft admin lebih dulu, lalu booking pertamanya yang sungguhan dibuat akun Rudi: Eko ikut pindah.
func TestGantiUplineMengabaikanBookingDraft(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	s := siapkanSimUpline(t, tx)

	eko := testdb.NewJamaah(t, tx, s.brand, testdb.JamaahOpts{Nama: "Eko", Direkrut: s.hannah})
	testdb.Exec(t, tx, `UPDATE jamaah SET kaitan_sumber = 'jalur2' WHERE id = ?`, eko)
	draft, _ := testdb.NewBookingPax(t, tx, s.sched, "draft", 1000, testdb.Pax{JamaahID: eko})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = NULL WHERE id = ?`, draft)
	b, _ := testdb.NewBookingPax(t, tx, s.sched, "baru", 1000, testdb.Pax{JamaahID: eko})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, s.rudi, b)

	ids, err := jamaahIkutUpline(ctx, tx, s.rudi, s.hannah, false)
	must(t, "pratinjau", err)
	if len(ids) != 2 || ids[0] != s.budi || ids[1] != eko {
		t.Fatalf("pratinjau = %v, want Budi (%d) dan Eko (%d)", ids, s.budi, eko)
	}
}
