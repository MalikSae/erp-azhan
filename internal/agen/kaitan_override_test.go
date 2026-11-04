package agen

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// Pemindahan komisi: langsung ke agen baru, pembinaan ke upline agen baru; log mencatat jumlahnya.
func TestOverrideKaitanPindahkanKomisiLangsungDanPembinaan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01"})
	admin := adminUji(t, tx)
	uplineLama := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineLama", StatusAgen: "aktif"})
	lama := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLama", StatusAgen: "aktif", Upline: uplineLama})
	uplineBaru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineBaru", StatusAgen: "aktif"})
	baru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBaru", StatusAgen: "aktif", Upline: uplineBaru})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J", Direkrut: lama})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))

	req := GantiKaitanRequest{Mode: "agen", AgenAkunID: &baru, Alasan: "salah agen", PindahkanKomisi: true}
	must(t, "override", gantiKaitanTx(ctx, tx, j, admin, req))

	ledger := map[string]int64{}
	rows, err := tx.Query(`SELECT jenis, akun_penerima_id FROM transaksi_komisi WHERE booking_id=?`, b)
	must(t, "ledger", err)
	for rows.Next() {
		var jenis string
		var id int64
		rows.Scan(&jenis, &id)
		ledger[jenis] = id
	}
	rows.Close()
	if ledger["langsung"] != baru || ledger["pembinaan"] != uplineBaru || len(ledger) != 2 {
		t.Fatalf("ledger = %v, want langsung->%d pembinaan->%d", ledger, baru, uplineBaru)
	}
	logs, err := listKaitanLog(ctx, tx, j)
	must(t, "log", err)
	if len(logs) != 1 || logs[0].KomisiDipindah != 2 || logs[0].KomisiDihapus != 0 {
		t.Fatalf("log = %+v, want 2 dipindah", logs)
	}
	if st, by, _ := kaitanSumber(t, tx, j); st != "terikat_agen" || by != baru {
		t.Fatalf("kaitan = %s/%d", st, by)
	}
}

// Agen baru tanpa upline: bonus pembinaan lama tidak punya penerima dan dihapus (tercatat di log).
func TestOverrideKaitanPembinaanTanpaUplineBaruDihapus(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01"})
	admin := adminUji(t, tx)
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Upline", StatusAgen: "aktif"})
	lama := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLama", StatusAgen: "aktif", Upline: upline})
	baru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBaru", StatusAgen: "aktif"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J", Direkrut: lama})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))

	must(t, "override", gantiKaitanTx(ctx, tx, j, admin, GantiKaitanRequest{Mode: "agen", AgenAkunID: &baru, Alasan: "pindah agen", PindahkanKomisi: true}))
	var n int
	must(t, "hitung", tx.QueryRow(`SELECT COUNT(*) FROM transaksi_komisi WHERE booking_id=? AND jenis='pembinaan'`, b).Scan(&n))
	logs, _ := listKaitanLog(ctx, tx, j)
	if n != 0 || len(logs) != 1 || logs[0].KomisiDipindah != 1 || logs[0].KomisiDihapus != 1 {
		t.Fatalf("pembinaan=%d log=%+v, want 0 baris pembinaan, dipindah 1 dihapus 1", n, logs)
	}
}

// Komisi lama yang sudah dicairkan tidak boleh membuat saldo agen lama negatif: seluruh override dibatalkan.
func TestOverrideKaitanDitolakBilaKomisiSudahDicairkan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1_000_000), BerangkatTanggal: "2000-01-01"})
	admin := adminUji(t, tx)
	lama := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLama", StatusAgen: "aktif"})
	baru := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenBaru", StatusAgen: "aktif"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "J", Direkrut: lama})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))
	testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama)
		VALUES (?,?,1000000,'disetujui','BSI','123','Agen')`, brand, lama)

	sp := gantiKaitanTx(ctx, tx, j, admin, GantiKaitanRequest{Mode: "agen", AgenAkunID: &baru, Alasan: "pindah agen", PindahkanKomisi: true})
	mustErr(t, "sudah dicairkan", sp, ErrKomisiSudahDicairkan)
}

func TestOverrideAgenAkun(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminUji(t, tx)
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenA", StatusAgen: "aktif"})
	b := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenB", StatusAgen: "aktif"})
	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Nonaktif", StatusAgen: "nonaktif"})
	z := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Z"}) // akun tanpa agen
	direkrutZ := func() sql.NullInt64 {
		var d sql.NullInt64
		must(t, "baca", tx.QueryRow(`SELECT direkrut_oleh_akun_id FROM customer_accounts WHERE id=?`, z).Scan(&d))
		return d
	}
	uplineZ := func() sql.NullInt64 {
		var u sql.NullInt64
		must(t, "baca upline", tx.QueryRow(`SELECT upline_akun_id FROM customer_accounts WHERE id=?`, z).Scan(&u))
		return u
	}
	ganti := func(id int64, agen *int64) error {
		return gantiAgenAkunTx(ctx, tx, nil, id, admin, GantiAgenAkunRequest{AgenAkunID: agen, Alasan: "dipasangkan admin"})
	}

	// Memasangkan akun tanpa agen (K8: hanya admin). Akun yang belum menjadi agen belum punya upline.
	must(t, "pasang ke A", ganti(z, &a))
	if d := direkrutZ(); d.Int64 != a {
		t.Fatalf("direkrut = %v, want A", d)
	}
	if u := uplineZ(); u.Valid {
		t.Fatalf("upline akun non-agen = %v, want NULL", u)
	}
	mustErr(t, "tanpa perubahan", ganti(z, &a), ErrKaitanTidakBerubah)
	must(t, "ganti ke B", ganti(z, &b))
	mustErr(t, "agen nonaktif", ganti(z, &nonaktif), ErrAgenTidakValid)
	mustErr(t, "diri sendiri", ganti(z, &z), ErrAgenTidakValid)
	mustErr(t, "akun tidak ada", ganti(z+999_999, &a), ErrNotFound)

	// Rantai berputar ditolak: A diikat Z, lalu Z diikat A.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET direkrut_oleh_akun_id=? WHERE id=?`, z, a)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET status_agen='aktif' WHERE id=?`, z)
	mustErr(t, "siklus", ganti(z, &a), ErrKaitanAkunSiklus)

	// Akun yang sudah menjadi agen: upline selalu ikut pindah, juga saat dilepas.
	must(t, "lepas", ganti(z, nil))
	if d := direkrutZ(); d.Valid {
		t.Fatalf("direkrut = %v, want NULL", d)
	}
	if u := uplineZ(); u.Valid {
		t.Fatalf("upline setelah dilepas = %v, want NULL", u)
	}
	must(t, "pasang ke B", ganti(z, &b))
	if u := uplineZ(); u.Int64 != b {
		t.Fatalf("upline = %v, want B", u)
	}
	logs, err := listAkunKaitanLog(ctx, tx, nil, z)
	must(t, "log", err)
	if len(logs) != 4 || !logs[0].UplineDiubah || logs[0].Alasan == "" {
		t.Fatalf("log akun = %+v", logs)
	}
	if _, err := listAkunKaitanLog(ctx, tx, nil, z+999_999); err != ErrNotFound {
		t.Fatalf("log akun tidak ada: %v", err)
	}
	hasil, err := cariAkunKaitan(ctx, tx, &brand, "Z (uji)")
	must(t, "cari", err)
	if len(hasil) != 1 || hasil[0].AgenID == nil || *hasil[0].AgenID != b {
		t.Fatalf("cari akun = %+v", hasil)
	}
}

// X mengajak Y, Y agen dengan upline X. Setelah admin memindahkan Y ke Q, upline Y menjadi Q, dan bonus pembinaan
// yang sudah tercatat atas X tidak ditarik kembali.
func TestOverrideAgenAkunMemindahkanUplineTanpaMenarikBonusLama(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(1_000_000), BonusPembinaan: rp(300_000), BerangkatTanggal: "2099-01-01"})
	admin := adminUji(t, tx)
	x := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "X", StatusAgen: "aktif"})
	y := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Y", StatusAgen: "aktif", Direkrut: x, Upline: x})
	q := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Q", StatusAgen: "aktif"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "W", Direkrut: y})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: j})
	must(t, "snapshot", komisi.SnapshotNominal(ctx, tx, b))
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, b)
	must(t, "komisi", komisi.ProcessBookingLunas(ctx, tx, b))

	must(t, "pindah Y ke Q", gantiAgenAkunTx(ctx, tx, nil, y, admin, GantiAgenAkunRequest{AgenAkunID: &q, Alasan: "sengketa diputus admin"}))

	var direkrut, upline sql.NullInt64
	must(t, "baca Y", tx.QueryRow(`SELECT direkrut_oleh_akun_id, upline_akun_id FROM customer_accounts WHERE id=?`, y).Scan(&direkrut, &upline))
	if direkrut.Int64 != q || upline.Int64 != q {
		t.Fatalf("Y: direkrut=%v upline=%v, want keduanya Q", direkrut, upline)
	}
	var penerima int64
	must(t, "bonus lama", tx.QueryRow(`SELECT akun_penerima_id FROM transaksi_komisi WHERE booking_id=? AND jenis='pembinaan'`, b).Scan(&penerima))
	if penerima != x {
		t.Fatalf("bonus pembinaan tercatat = %d, want tetap X (%d)", penerima, x)
	}
}

// Admin yang dibatasi satu brand tidak boleh mengubah atau membaca log akun brand lain.
func TestOverrideAgenAkunDibatasiBrandAdmin(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminUji(t, tx)
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenA", StatusAgen: "aktif"})
	z := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Z"})
	brandLain := brand + 999_999
	req := GantiAgenAkunRequest{AgenAkunID: &a, Alasan: "dipasangkan admin"}

	mustErr(t, "ganti oleh admin brand lain", gantiAgenAkunTx(ctx, tx, &brandLain, z, admin, req), ErrNotFound)
	_, err := listAkunKaitanLog(ctx, tx, &brandLain, z)
	mustErr(t, "log oleh admin brand lain", err, ErrNotFound)
	must(t, "ganti oleh admin brand sendiri", gantiAgenAkunTx(ctx, tx, &brand, z, admin, req))
	if _, err := listAkunKaitanLog(ctx, tx, &brand, z); err != nil {
		t.Fatalf("log oleh admin brand sendiri: %v", err)
	}
}
