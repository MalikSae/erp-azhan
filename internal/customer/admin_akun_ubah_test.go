package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func adminId(t *testing.T, tx *sql.Tx) int64 {
	t.Helper()
	var id int64
	must(t, "admin", tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&id))
	return id
}

func hpBaru() string { return fmt.Sprintf("0899%07d", time.Now().UnixNano()%10_000_000) }

func TestUbahProfilAkunMencatatLog(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	id := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Sebelum"})
	hp := hpBaru()

	must(t, "ubah", ubahProfilTx(ctx, tx, &brand, id, admin, UbahProfilRequest{NamaLengkap: "Sesudah Ubah", NoHP: hp, Domisili: "Kota Bandung", Alasan: "diminta pemilik akun"}))

	var nama, noHP string
	var dom sql.NullString
	must(t, "baca", tx.QueryRow(`SELECT nama_lengkap, no_hp, domisili FROM customer_accounts WHERE id=?`, id).Scan(&nama, &noHP, &dom))
	if nama != "Sesudah Ubah" || noHP != hp || dom.String != "Kota Bandung" {
		t.Fatalf("data = %q %q %v", nama, noHP, dom)
	}
	var n int
	must(t, "log", tx.QueryRow(`SELECT COUNT(*) FROM akun_data_log WHERE akun_id=? AND diubah_oleh=? AND alasan='diminta pemilik akun'`, id, admin).Scan(&n))
	if n != 3 {
		t.Fatalf("baris log = %d, want 3 (nama, no_hp, domisili)", n)
	}
	var lama, baru string
	must(t, "log nama", tx.QueryRow(`SELECT nilai_lama, nilai_baru FROM akun_data_log WHERE akun_id=? AND kolom='nama_lengkap'`, id).Scan(&lama, &baru))
	if !strings.HasPrefix(lama, "Sebelum") || baru != "Sesudah Ubah" {
		t.Fatalf("log nama = %q -> %q", lama, baru)
	}

	// Hanya kolom yang berubah yang dicatat.
	must(t, "ubah satu kolom", ubahProfilTx(ctx, tx, &brand, id, admin, UbahProfilRequest{NamaLengkap: "Sesudah Ubah", NoHP: hp, Domisili: "Kota Depok", Alasan: "pindah domisili"}))
	must(t, "log domisili", tx.QueryRow(`SELECT COUNT(*) FROM akun_data_log WHERE akun_id=? AND alasan='pindah domisili'`, id).Scan(&n))
	if n != 1 {
		t.Fatalf("baris log kedua = %d, want 1", n)
	}
	mustErrIs(t, "tanpa perubahan", ubahProfilTx(ctx, tx, &brand, id, admin, UbahProfilRequest{NamaLengkap: "Sesudah Ubah", NoHP: hp, Domisili: "Kota Depok", Alasan: "tidak ada yang beda"}), ErrTidakBerubah)
}

func TestUbahProfilAkunValidasiDanBatasBrand(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	id := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Akun"})
	ok := UbahProfilRequest{NamaLengkap: "Nama Baru", NoHP: hpBaru(), Domisili: "Depok", Alasan: "alasan cukup"}

	for nama, mut := range map[string]func(r *UbahProfilRequest){
		"alasan pendek":       func(r *UbahProfilRequest) { r.Alasan = "abc" },
		"nama berurl":         func(r *UbahProfilRequest) { r.NamaLengkap = "Budi http://x.id" },
		"nama terlalu pendek": func(r *UbahProfilRequest) { r.NamaLengkap = "B" },
		"nomor tidak valid":   func(r *UbahProfilRequest) { r.NoHP = "12ab" },
		"domisili panjang":    func(r *UbahProfilRequest) { r.Domisili = strings.Repeat("k", 101) },
	} {
		r := ok
		mut(&r)
		var ev *ErrValidasi
		if err := ubahProfilTx(ctx, tx, &brand, id, admin, r); !errors.As(err, &ev) {
			t.Errorf("%s: err = %v, want ErrValidasi", nama, err)
		}
	}

	brandLain := brand + 999_999
	mustErrIs(t, "brand lain", ubahProfilTx(ctx, tx, &brandLain, id, admin, ok), ErrNotFound)
	mustErrIs(t, "akun tidak ada", ubahProfilTx(ctx, tx, nil, id+999_999, admin, ok), ErrNotFound)
}

func TestUbahProfilAkunBentrokWhatsApp(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	id := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pengubah"})
	lain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemilikNomor"})
	var hpLain string
	must(t, "hp lain", tx.QueryRow(`SELECT no_hp FROM customer_accounts WHERE id=?`, lain).Scan(&hpLain))

	// Format berbeda (+62 menggantikan awalan 0) tetap dianggap nomor yang sama.
	var bentrok *ErrBentrokData
	err := ubahProfilTx(ctx, tx, &brand, id, admin, UbahProfilRequest{NamaLengkap: "Pengubah Uji", NoHP: "+62" + hpLain[1:], Domisili: "", Alasan: "uji bentrok nomor"})
	if !errors.As(err, &bentrok) || bentrok.Kolom != "no_hp" || bentrok.AkunID != lain || !strings.HasPrefix(bentrok.Nama, "PemilikNomor") {
		t.Fatalf("err = %v, bentrok = %+v", err, bentrok)
	}
	var nama string
	must(t, "nama tetap", tx.QueryRow(`SELECT nama_lengkap FROM customer_accounts WHERE id=?`, id).Scan(&nama))
	if !strings.HasPrefix(nama, "Pengubah") || strings.Contains(nama, "Uji") {
		t.Fatalf("perubahan harus dibatalkan seluruhnya saat bentrok, nama = %q", nama)
	}
}

func TestUbahNIKAgenMengikutiStatusDanIzin(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	pengajuan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pengaju", StatusAgen: "pengajuan"})
	aktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenAktif", StatusAgen: "aktif"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen='9999999999990101' WHERE id=?`, pengajuan)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen='9999999999990102' WHERE id=?`, aktif)
	req := UbahNIKRequest{NIK: "9999999999990103", Alasan: "salah ketik saat daftar"}
	hanyaPengajuan := IzinNIK{Pengajuan: true}
	hanyaAktif := IzinNIK{Aktif: true}

	must(t, "pengajuan oleh penangan pengajuan", ubahNIKAgenTx(ctx, tx, &brand, pengajuan, admin, hanyaPengajuan, req))
	var nik string
	must(t, "baca", tx.QueryRow(`SELECT nik_agen FROM customer_accounts WHERE id=?`, pengajuan).Scan(&nik))
	if nik != "9999999999990103" {
		t.Fatalf("nik = %s", nik)
	}
	var lama, baru string
	must(t, "log", tx.QueryRow(`SELECT nilai_lama, nilai_baru FROM akun_data_log WHERE akun_id=? AND kolom='nik_agen'`, pengajuan).Scan(&lama, &baru))
	if lama != "9999**********01" || baru != "9999**********03" {
		t.Fatalf("log NIK harus disamarkan: %q -> %q", lama, baru)
	}

	// Agen aktif: penangan pengajuan saja tidak cukup, harus Admin Master.
	mustErrIs(t, "aktif tanpa izin", ubahNIKAgenTx(ctx, tx, &brand, aktif, admin, hanyaPengajuan, req), ErrDilarang)
	must(t, "aktif oleh Admin Master", ubahNIKAgenTx(ctx, tx, &brand, aktif, admin, hanyaAktif, req.dengan("9999999999990104")))
	// Sebaliknya, Admin Master saja tidak boleh mengubah NIK pada tahap pengajuan lewat jalur pengajuan.
	mustErrIs(t, "pengajuan tanpa izin", ubahNIKAgenTx(ctx, tx, &brand, pengajuan, admin, hanyaAktif, req.dengan("9999999999990105")), ErrDilarang)
}

func TestUbahNIKAgenValidasiBentrokDanBatasBrand(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	izin := IzinNIK{Pengajuan: true, Aktif: true}
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunA", StatusAgen: "pengajuan"})
	b := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunB", StatusAgen: "pengajuan"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen='9999999999990111' WHERE id=?`, a)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen='9999999999990112' WHERE id=?`, b)

	var ev *ErrValidasi
	if err := ubahNIKAgenTx(ctx, tx, &brand, a, admin, izin, UbahNIKRequest{NIK: "12345", Alasan: "alasan cukup"}); !errors.As(err, &ev) {
		t.Fatalf("NIK pendek: %v", err)
	}
	if err := ubahNIKAgenTx(ctx, tx, &brand, a, admin, izin, UbahNIKRequest{NIK: "9999999999990113", Alasan: "abc"}); !errors.As(err, &ev) {
		t.Fatalf("alasan pendek: %v", err)
	}
	mustErrIs(t, "sama", ubahNIKAgenTx(ctx, tx, &brand, a, admin, izin, UbahNIKRequest{NIK: "9999999999990111", Alasan: "alasan cukup"}), ErrTidakBerubah)

	var bentrok *ErrBentrokData
	err := ubahNIKAgenTx(ctx, tx, &brand, a, admin, izin, UbahNIKRequest{NIK: "9999999999990112", Alasan: "alasan cukup"})
	if !errors.As(err, &bentrok) || bentrok.Kolom != "nik_agen" || bentrok.AkunID != b || !strings.HasPrefix(bentrok.Nama, "AkunB") {
		t.Fatalf("bentrok = %v / %+v", err, bentrok)
	}

	brandLain := brand + 999_999
	mustErrIs(t, "brand lain", ubahNIKAgenTx(ctx, tx, &brandLain, a, admin, izin, UbahNIKRequest{NIK: "9999999999990113", Alasan: "alasan cukup"}), ErrNotFound)
}

func TestDetailAkunMemuatRiwayatData(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	admin := adminId(t, tx)
	id := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Berubah"})
	must(t, "ubah", ubahProfilTx(ctx, tx, &brand, id, admin, UbahProfilRequest{NamaLengkap: "Berubah Lagi", NoHP: hpBaru(), Domisili: "Depok", Alasan: "diminta pemilik"}))

	d, err := getAkunDetail(ctx, tx, &brand, id)
	must(t, "detail", err)
	if len(d.RiwayatData) != 3 || d.RiwayatData[0].Alasan != "diminta pemilik" || d.RiwayatData[0].DiubahOleh == "" {
		t.Fatalf("riwayat = %+v", d.RiwayatData)
	}
}

func TestNormalisasiHPSamaDenganKolomDatabase(t *testing.T) {
	for in, want := range map[string]string{"08123456789": "628123456789", "+62 812-3456-789": "628123456789", "628123456789": "628123456789", "": ""} {
		if got := normalisasiHP(in); got != want {
			t.Errorf("normalisasiHP(%q) = %q, want %q", in, got, want)
		}
	}
}

func mustErrIs(t *testing.T, what string, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("%s: err = %v, want %v", what, err, want)
	}
}

func (r UbahNIKRequest) dengan(nik string) UbahNIKRequest {
	r.NIK = nik
	return r
}
