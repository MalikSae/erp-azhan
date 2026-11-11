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
	must(t, "baca kaitan", tx.QueryRow(`SELECT kaitan_status, direkrut_oleh_akun_id FROM jamaah WHERE id=?`, jamaahID).Scan(&status, &direkrut))
	return status, direkrut.Int64
}

func setKode(t *testing.T, tx *sql.Tx, agenID int64, kode string) {
	t.Helper()
	testdb.Exec(t, tx, `UPDATE customer_accounts SET kode_referral=? WHERE id=?`, kode, agenID)
}

func TestResolveKodeReferral(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	aktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Aktif", StatusAgen: "aktif"})
	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nonaktif", StatusAgen: "nonaktif"})
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

// Jamaah diikat ke akun agen (first-set wins); akun yang diikat agen lalu menjadi
// agen punya upline agen yang mengikat akunnya.
func TestKunciJamaahDanUplineAkun(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE brands SET biaya_pendaftaran_agen=0 WHERE id=?`, brand)
	var adminID int64
	must(t, "admin", tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&adminID))

	adam := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Adam", StatusAgen: "aktif"})
	lain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLain", StatusAgen: "aktif"})
	nuh := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Nuh"})

	must(t, "ikat ke Adam", IkatJamaah(ctx, tx, nuh, adam, SumberReferral))
	must(t, "coba rebut", IkatJamaah(ctx, tx, nuh, lain, SumberReferral)) // tidak boleh menimpa
	if st, by := kaitanOf(t, tx, nuh); st != "terikat_agen" || by != adam {
		t.Fatalf("Nuh = %s/%d, want terikat_agen/%d (first-set wins)", st, by, adam)
	}

	// Akun yang diikat Adam saat dibuat lalu mengajukan diri menjadi agen: upline-nya Adam.
	akunNuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunNuh", Direkrut: adam})
	must(t, "ajukan", ajukanTx(ctx, tx, akunNuh, ajuan))
	must(t, "setujui", setujuiTx(ctx, tx, akunNuh, &brand, adminID))
	var upline sql.NullInt64
	must(t, "upline", tx.QueryRow(`SELECT upline_akun_id FROM customer_accounts WHERE id=?`, akunNuh).Scan(&upline))
	if upline.Int64 != adam {
		t.Fatalf("upline AkunNuh = %d, want Adam %d", upline.Int64, adam)
	}

	// 'tanpa_agen' tidak pernah diikat otomatis.
	yusuf := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Yusuf", TanpaAgen: true})
	must(t, "ikat tanpa_agen", IkatJamaah(ctx, tx, yusuf, adam, SumberReferral))
	if st, _ := kaitanOf(t, tx, yusuf); st != "tanpa_agen" {
		t.Fatalf("Yusuf tanpa_agen berubah menjadi %s", st)
	}
}

// IkatRombongan mengikat semua pax yang belum terikat; yang sudah terikat tidak berubah.
func TestIkatRombongan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})
	y := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenY", StatusAgen: "aktif"})

	baru1 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Baru1"})
	baru2 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Baru2"})
	milikY := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "MilikY", Direkrut: y})
	must(t, "rombongan", IkatRombongan(ctx, tx, []int64{baru1, baru2, milikY}, x, SumberJalur1))
	for _, j := range []int64{baru1, baru2} {
		if _, by := kaitanOf(t, tx, j); by != x {
			t.Fatalf("pax baru terikat ke %d, want %d", by, x)
		}
	}
	if _, by := kaitanOf(t, tx, milikY); by != y {
		t.Fatalf("pax milik Y berubah ke %d", by)
	}

	// Tanpa agen: tidak ada yang diikat.
	baru3 := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Baru3"})
	must(t, "rombongan kosong", IkatRombongan(ctx, tx, []int64{baru3}, 0, SumberReferral))
	if st, _ := kaitanOf(t, tx, baru3); st != "belum_ditentukan" {
		t.Fatalf("Baru3 = %s, want belum_ditentukan", st)
	}
}

// Booking admin (tanpa akun pemesan): pax yang belum terikat ikut agen pemilik kunci PIC.
func TestIkatRombonganBookingIkutKunciPIC(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenX", StatusAgen: "aktif"})
	pic := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "PIC", Direkrut: x})
	anggota := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Anggota"})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 2000, testdb.Pax{JamaahID: pic}, testdb.Pax{JamaahID: anggota})
	must(t, "ikat booking", IkatRombonganBooking(ctx, tx, b))
	if st, by := kaitanOf(t, tx, anggota); st != "terikat_agen" || by != x {
		t.Fatalf("anggota = %s/%d, want terikat_agen/%d", st, by, x)
	}
}

// Jalur 3 per-pax: kaitkan ke agen berbeda / tanpa agen dalam satu booking,
// lalu komisi saat lunas mengikuti kaitan masing-masing.
func TestJalur3PerPax(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{
		KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01",
	})
	adam := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Adam", StatusAgen: "aktif"})
	nuh := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nuh", StatusAgen: "aktif", Direkrut: adam, Upline: adam})
	hud := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Hud", StatusAgen: "aktif", Direkrut: nuh, Upline: nuh})
	biasa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Biasa"})

	umar := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Umar"})
	salman := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Salman"})
	bilal := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Bilal"})
	must(t, "Umar", TetapkanKaitanJalur3(ctx, tx, umar, "agen", nuh))
	must(t, "Salman", TetapkanKaitanJalur3(ctx, tx, salman, "tanpa_agen", 0))
	must(t, "Bilal", TetapkanKaitanJalur3(ctx, tx, bilal, "agen", hud))

	// Validasi picker: akun yang bukan agen aktif / agen brand lain ditolak.
	if err := ValidasiAgenAktif(ctx, tx, brand, biasa); err != ErrAgenTidakValid {
		t.Fatalf("akun biasa lolos sebagai agen: %v", err)
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
	rows, err := tx.Query(`SELECT akun_penerima_id, jenis FROM transaksi_komisi WHERE booking_id=?`, b)
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
	// Umar -> Nuh (langsung; upline Adam), Bilal -> Hud (langsung; upline Nuh), Salman tanpa agen.
	want := []string{"Adam:pembinaan", "Hud:langsung", "Nuh:langsung", "Nuh:pembinaan"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("ledger jalur 3:\n got  %v\n want %v (Salman tanpa komisi)", got, want)
	}
}
