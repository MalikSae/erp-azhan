package agen

import (
	"context"
	"database/sql"
	"sort"
	"strings"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

func rp(v float64) *float64 { return &v }

func kaitanOf(t *testing.T, tx *sql.Tx, jamaahID int64) (string, int64) {
	t.Helper()
	var status string
	var direkrut sql.NullInt64
	must(t, "baca kaitan", tx.QueryRow(`SELECT kaitan_status, direkrut_oleh_jamaah_id FROM jamaah WHERE id=?`, jamaahID).Scan(&status, &direkrut))
	return status, direkrut.Int64
}

func setKode(t *testing.T, tx *sql.Tx, agenID int64, kode string) {
	t.Helper()
	testdb.Exec(t, tx, `UPDATE jamaah SET kode_referral=? WHERE id=?`, kode, agenID)
}

func TestResolveKodeReferral(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	aktif := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Aktif", StatusAgen: "aktif"})
	nonaktif := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Nonaktif", StatusAgen: "nonaktif"})
	setKode(t, tx, aktif, "UJIAKT")
	setKode(t, tx, nonaktif, "UJINON")

	cases := []struct {
		brand int64
		kode  string
		want  int64
	}{
		{brand, "UJIAKT", aktif},
		{brand, " ujiakt ", aktif},     // dinormalisasi
		{brand, "UJINON", 0},           // agen nonaktif diabaikan (L8)
		{brand + 999_999, "UJIAKT", 0}, // brand lain diabaikan (L8)
		{brand, "TIDAKADA", 0},
		{brand, "", 0},
	}
	for _, c := range cases {
		got, err := ResolveKodeReferral(ctx, tx, c.brand, c.kode)
		must(t, "resolve", err)
		if got != c.want {
			t.Fatalf("ResolveKodeReferral(%q) = %d, want %d", c.kode, got, c.want)
		}
	}
}

// Langkah 3/3b: jamaah mendaftar lewat link agen (first-set wins), lalu jadi
// agen -> upline otomatis agen perekrut.
func TestLangkah3JalurReferralDanUpline(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE brands SET biaya_pendaftaran_agen=0 WHERE id=?`, brand)
	var adminID int64
	must(t, "admin", tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&adminID))

	adam := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Adam", StatusAgen: "aktif"})
	lain := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AgenLain", StatusAgen: "aktif"})
	nuh := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Nuh"})

	must(t, "ikat ke Adam", IkatJamaah(ctx, tx, nuh, adam))
	must(t, "coba rebut", IkatJamaah(ctx, tx, nuh, lain)) // tidak boleh menimpa
	if st, by := kaitanOf(t, tx, nuh); st != "terikat_agen" || by != adam {
		t.Fatalf("Nuh = %s/%d, want terikat_agen/%d (first-set wins)", st, by, adam)
	}

	must(t, "ajukan", ajukanTx(ctx, tx, nuh, ajuan))
	must(t, "setujui", setujuiTx(ctx, tx, nuh, &brand, adminID))
	var upline sql.NullInt64
	must(t, "upline", tx.QueryRow(`SELECT upline_jamaah_id FROM jamaah WHERE id=?`, nuh).Scan(&upline))
	if upline.Int64 != adam {
		t.Fatalf("upline Nuh = %d, want Adam %d", upline.Int64, adam)
	}

	// Tidak pernah mengikat diri sendiri; 'tanpa_agen' tidak pernah diikat otomatis.
	must(t, "ikat diri sendiri", IkatJamaah(ctx, tx, adam, adam))
	if st, _ := kaitanOf(t, tx, adam); st != "belum_ditentukan" {
		t.Fatalf("Adam terikat ke dirinya sendiri: %s", st)
	}
	yusuf := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Yusuf", TanpaAgen: true})
	must(t, "ikat tanpa_agen", IkatJamaah(ctx, tx, yusuf, adam))
	if st, _ := kaitanOf(t, tx, yusuf); st != "tanpa_agen" {
		t.Fatalf("Yusuf tanpa_agen berubah menjadi %s", st)
	}
}

func TestIkatRombongan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AgenX", StatusAgen: "aktif"})
	y := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AgenY", StatusAgen: "aktif"})

	// Tanpa kode referral: pax baru ikut agen PIC; pax milik agen lain tidak berubah.
	pic := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PIC", Direkrut: x})
	baru := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Baru"})
	milikY := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "MilikY", Direkrut: y})
	must(t, "rombongan", IkatRombongan(ctx, tx, pic, []int64{baru, milikY}, 0))
	if _, by := kaitanOf(t, tx, baru); by != x {
		t.Fatalf("pax baru ikut agen %d, want PIC agen %d", by, x)
	}
	if _, by := kaitanOf(t, tx, milikY); by != y {
		t.Fatalf("pax milik Y berubah ke %d", by)
	}

	// Dengan kode referral: agen booking = agen referral.
	pic2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PIC2"})
	baru2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Baru2"})
	must(t, "rombongan referral", IkatRombongan(ctx, tx, pic2, []int64{baru2}, y))
	if _, by := kaitanOf(t, tx, pic2); by != y {
		t.Fatalf("PIC2 = %d, want referral %d", by, y)
	}
	if _, by := kaitanOf(t, tx, baru2); by != y {
		t.Fatalf("Baru2 = %d, want referral %d", by, y)
	}

	// PIC tanpa agen & tanpa referral: tidak ada yang diikat.
	pic3 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PIC3"})
	baru3 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Baru3"})
	must(t, "rombongan kosong", IkatRombongan(ctx, tx, pic3, []int64{baru3}, 0))
	if st, _ := kaitanOf(t, tx, baru3); st != "belum_ditentukan" {
		t.Fatalf("Baru3 = %s, want belum_ditentukan", st)
	}
}

// Langkah 11, 12, 14: Jalur 3 per-pax — kaitkan ke agen berbeda / tanpa agen
// dalam satu booking, lalu komisi saat lunas mengikuti kaitan masing-masing.
func TestLangkah14Jalur3PerPax(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	adam := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Adam", StatusAgen: "aktif"})
	nuh := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Nuh", StatusAgen: "aktif", Direkrut: adam, Upline: adam})
	hud := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Hud", StatusAgen: "aktif", Direkrut: nuh, Upline: nuh})

	umar := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Umar"})
	salman := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Salman"})
	bilal := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Bilal"})
	must(t, "Umar -> Nuh", ValidasiAgenAktif(ctx, tx, brand, nuh))
	must(t, "Umar", TetapkanKaitanJalur3(ctx, tx, umar, "agen", nuh))
	must(t, "Salman", TetapkanKaitanJalur3(ctx, tx, salman, "tanpa_agen", 0))
	must(t, "Bilal", TetapkanKaitanJalur3(ctx, tx, bilal, "agen", hud))

	// Validasi picker: bukan agen aktif / brand lain ditolak.
	if err := ValidasiAgenAktif(ctx, tx, brand, umar); err != ErrAgenTidakValid {
		t.Fatalf("jamaah biasa lolos sebagai agen: %v", err)
	}
	if err := ValidasiAgenAktif(ctx, tx, brand+999_999, nuh); err != ErrAgenTidakValid {
		t.Fatalf("agen brand lain lolos: %v", err)
	}

	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 3000,
		testdb.Pax{JamaahID: umar}, testdb.Pax{JamaahID: salman}, testdb.Pax{JamaahID: bilal})
	// Auto-bind booking admin tidak mengubah pilihan eksplisit Jalur 3.
	must(t, "auto-bind", IkatRombonganBooking(ctx, tx, b))
	if st, _ := kaitanOf(t, tx, salman); st != "tanpa_agen" {
		t.Fatalf("Salman berubah menjadi %s", st)
	}

	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))

	nama := map[int64]string{adam: "Adam", nuh: "Nuh", hud: "Hud"}
	rows, err := tx.Query(`SELECT jamaah_penerima_id, jenis FROM transaksi_komisi WHERE booking_id=?`, b)
	must(t, "ledger", err)
	var got []string
	for rows.Next() {
		var id int64
		var jenis string
		rows.Scan(&id, &jenis)
		got = append(got, nama[id]+":"+jenis)
	}
	rows.Close()
	sort.Strings(got)
	want := []string{"Adam:pembinaan", "Hud:langsung", "Nuh:langsung", "Nuh:pembinaan"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ledger langkah 14:\n got  %v\n want %v (Salman tanpa komisi)", got, want)
	}
}
