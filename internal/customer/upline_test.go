package customer

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestCariUplineHanyaAgenAktifYangMengajak(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	aktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineAktif", StatusAgen: "aktif"})
	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineNonaktif", StatusAgen: "nonaktif"})
	tanpa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "TanpaUpline"})
	diajak := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Diajak", Direkrut: aktif})
	diajakNonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "DiajakNonaktif", Direkrut: nonaktif})
	leader := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenDiajak", StatusAgen: "aktif", Direkrut: aktif, Upline: aktif})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET no_hp='0812-3456-7890', foto_url=? WHERE id=?`, fotoA, aktif)

	u, err := cariUpline(ctx, tx, diajak)
	must(t, "cari upline", err)
	if u == nil || u.NamaLengkap == "" || u.WhatsApp != "6281234567890" || !u.AdaFoto || u.fotoURL != fotoA {
		t.Fatalf("upline = %+v", u)
	}
	// Agen yang diajak agen lain juga punya upline (Leader Anda).
	if u, err = cariUpline(ctx, tx, leader); err != nil || u == nil || u.WhatsApp != "6281234567890" {
		t.Fatalf("upline agen: %v %+v", err, u)
	}

	// Tanpa upline, atau upline sudah nonaktif: tidak ada data.
	for nama, id := range map[string]int64{"tanpa upline": tanpa, "upline nonaktif": diajakNonaktif, "akun tidak ada": diajak + 999_999} {
		if u, err := cariUpline(ctx, tx, id); err != nil || u != nil {
			t.Errorf("%s: u=%+v err=%v, want nil", nama, u, err)
		}
	}

	// Tanpa foto: AdaFoto false.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET foto_url=NULL WHERE id=?`, aktif)
	if u, err = cariUpline(ctx, tx, diajak); err != nil || u == nil || u.AdaFoto {
		t.Fatalf("tanpa foto: %v %+v", err, u)
	}
}
