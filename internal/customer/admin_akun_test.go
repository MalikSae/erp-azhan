package customer

import (
	"context"
	"errors"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestListAkunAdminFilterDanBatasBrand(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "VxqUpline", StatusAgen: "aktif"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "VxqAgen", StatusAgen: "aktif", Direkrut: upline, Upline: upline})
	biasa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "VxqBiasa", Direkrut: upline})
	testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "VxqMandiri"})

	hasil, err := listAkun(ctx, tx, &brand, FilterAkun{Cari: "Vxq", PerPage: 2})
	must(t, "list", err)
	if hasil.Total != 4 || len(hasil.Items) != 2 || hasil.Page != 1 {
		t.Fatalf("total=%d items=%d page=%d, want 4/2/1", hasil.Total, len(hasil.Items), hasil.Page)
	}
	hal2, err := listAkun(ctx, tx, &brand, FilterAkun{Cari: "Vxq", PerPage: 2, Page: 2})
	must(t, "halaman 2", err)
	if len(hal2.Items) != 2 || hal2.Items[0].AkunID == hasil.Items[0].AkunID {
		t.Fatalf("halaman 2 harus berisi baris lain: %+v", hal2.Items)
	}

	aktif, err := listAkun(ctx, tx, &brand, FilterAkun{Cari: "Vxq", StatusAgen: "aktif"})
	must(t, "filter aktif", err)
	if aktif.Total != 2 {
		t.Fatalf("agen aktif = %d, want 2", aktif.Total)
	}
	ada, err := listAkun(ctx, tx, &brand, FilterAkun{Cari: "Vxq", Upline: "ada"})
	must(t, "upline ada", err)
	tanpa, err := listAkun(ctx, tx, &brand, FilterAkun{Cari: "Vxq", Upline: "tanpa"})
	must(t, "upline tanpa", err)
	if ada.Total != 2 || tanpa.Total != 2 {
		t.Fatalf("upline ada=%d tanpa=%d, want 2 dan 2", ada.Total, tanpa.Total)
	}
	var baris *AkunBaris
	for i := range ada.Items {
		if ada.Items[i].AkunID == agen {
			baris = &ada.Items[i]
		}
	}
	if baris == nil || baris.UplineID == nil || *baris.UplineID != upline || baris.UplineNama == nil || !strings.HasPrefix(*baris.UplineNama, "VxqUpline") {
		t.Fatalf("baris agen = %+v", baris)
	}
	_ = biasa

	brandLain := brand + 999_999
	kosong, err := listAkun(ctx, tx, &brandLain, FilterAkun{Cari: "Vxq"})
	must(t, "brand lain", err)
	if kosong.Total != 0 || len(kosong.Items) != 0 {
		t.Fatalf("brand lain harus kosong: %+v", kosong)
	}
	// Karakter LIKE di pencarian dianggap huruf biasa.
	wildcard, err := listAkun(ctx, tx, &brand, FilterAkun{Cari: "%"})
	must(t, "wildcard", err)
	if wildcard.Total != 0 {
		t.Fatalf("pencarian %% tidak boleh cocok dengan semua akun: %d", wildcard.Total)
	}
	if _, err := listAkun(ctx, tx, &brand, FilterAkun{StatusAgen: "bukan-status"}); !errors.Is(err, ErrFilterAkun) {
		t.Fatalf("status tidak dikenal harus ditolak: %v", err)
	}
}

func TestDetailAkunAdmin(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	upline := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineDetail", StatusAgen: "aktif"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenDetail", StatusAgen: "aktif", Direkrut: upline, Upline: upline})
	bawahan := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Bawahan", StatusAgen: "aktif", Direkrut: agen, Upline: agen})
	_ = bawahan
	milik := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "MilikAgen", Direkrut: agen})
	testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "MilikAgen2", Direkrut: agen})
	diri := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "DiriSendiri", TanpaAgen: true})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=?, nik_agen='9999999999990077', domisili='Kota Bandung' WHERE id=?`, diri, agen)
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: milik})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, agen, b)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET foto_url=? WHERE id=?`, fotoA, agen)

	d, err := getAkunDetail(ctx, tx, &brand, agen)
	must(t, "detail", err)
	if d.UplineID == nil || *d.UplineID != upline || d.JumlahJamaahAgen != 2 || d.JumlahDiajak != 1 || d.JumlahBooking != 1 {
		t.Fatalf("detail = %+v", d)
	}
	if d.Jamaah == nil || d.Jamaah.ID != diri {
		t.Fatalf("jamaah akun = %+v, want %d", d.Jamaah, diri)
	}
	if d.FotoURL == nil || *d.FotoURL != fotoA {
		t.Fatalf("foto_url = %v, want %s", d.FotoURL, fotoA)
	}
	if d.NIKAgen == nil || *d.NIKAgen != "9999**********77" {
		t.Fatalf("NIK agen harus disamarkan: %v", d.NIKAgen)
	}
	if d.Domisili == nil || *d.Domisili != "Kota Bandung" {
		t.Fatalf("domisili = %v", d.Domisili)
	}

	brandLain := brand + 999_999
	if _, err := getAkunDetail(ctx, tx, &brandLain, agen); !errors.Is(err, ErrNotFound) {
		t.Fatalf("admin brand lain: %v, want ErrNotFound", err)
	}
	if _, err := getAkunDetail(ctx, tx, nil, agen+999_999); !errors.Is(err, ErrNotFound) {
		t.Fatalf("akun tidak ada: %v, want ErrNotFound", err)
	}
}

func TestSamarkanNIK(t *testing.T) {
	for in, want := range map[string]string{"9999999999990077": "9999**********77", "123456": "******", "": ""} {
		if got := samarkanNIK(in); got != want {
			t.Errorf("samarkanNIK(%q) = %q, want %q", in, got, want)
		}
	}
}

func must(t *testing.T, what string, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
}
