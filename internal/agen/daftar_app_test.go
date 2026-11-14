package agen

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Pendaftaran agen dari app (keputusan 14 November 2026): akun jamaah (booking minimal DP) langsung aktif tanpa
// biaya; akun lain menunggu keputusan admin dengan tagihan biaya brand; MoU brand wajib disetujui dan disalin.
func TestDaftarAgenApp(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	if _, err := tx.ExecContext(ctx, `UPDATE brands SET biaya_pendaftaran_agen = 300000, mou_agen = '<p>MoU uji</p>' WHERE id = ?`, brand); err != nil {
		t.Fatal(err)
	}
	req := func(nik string) AjukanRequest {
		return AjukanRequest{NIK: nik, Domisili: "Kota Bandung", JenisKelamin: "L", TanggalLahir: "1990-01-01", SetujuSyaratKetentuan: true, SetujuMoU: true}
	}

	t.Run("MoU wajib disetujui", func(t *testing.T) {
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "TanpaMoU"})
		r := req("9999999999991401")
		r.SetujuMoU = false
		if err := ajukanTx(ctx, tx, akun, r); !errors.Is(err, ErrMoUWajib) {
			t.Fatalf("err = %v, want ErrMoUWajib", err)
		}
	})

	t.Run("akun bukan jamaah menunggu admin dengan tagihan", func(t *testing.T) {
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "BukanJamaah"})
		if err := ajukanTx(ctx, tx, akun, req("9999999999991402")); err != nil {
			t.Fatal(err)
		}
		status, bayar, nominal, otomatis, mou := bacaPengajuan(t, tx, akun)
		if status != StatusAgenPengajuan || bayar != "menunggu_verifikasi" || nominal != 300000 || otomatis {
			t.Fatalf("status=%s bayar=%s nominal=%v otomatis=%v", status, bayar, nominal, otomatis)
		}
		if mou != "<p>MoU uji</p>" {
			t.Fatalf("salinan MoU = %q", mou)
		}
	})

	t.Run("akun jamaah langsung aktif tanpa biaya", func(t *testing.T) {
		akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunJamaah"})
		pic := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Jamaah Pemilik Akun", TanpaAgen: true})
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, seat_count, status, customer_account_id)
			VALUES ('UJIAGN', ?, ?, 1, 'dp', ?)`, jadwal, pic, akun); err != nil {
			t.Fatal(err)
		}
		if err := ajukanTx(ctx, tx, akun, req("9999999999991403")); err != nil {
			t.Fatal(err)
		}
		status, bayar, _, otomatis, _ := bacaPengajuan(t, tx, akun)
		if status != "aktif" || bayar != "dibebaskan" || !otomatis {
			t.Fatalf("status=%s bayar=%s otomatis=%v", status, bayar, otomatis)
		}
		var kode sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT kode_referral FROM customer_accounts WHERE id = ?`, akun).Scan(&kode); err != nil {
			t.Fatal(err)
		}
		if !kode.Valid || kode.String == "" {
			t.Fatal("kode referral belum terbit")
		}
	})
}

func bacaPengajuan(t *testing.T, tx *sql.Tx, akun int64) (status, bayar string, nominal float64, otomatis bool, mou string) {
	t.Helper()
	var m sql.NullString
	err := tx.QueryRowContext(context.Background(), `
		SELECT a.status_agen, p.status, p.nominal_tagihan, p.disetujui_otomatis, p.mou_disetujui
		FROM customer_accounts a JOIN pembayaran_pendaftaran_agen p ON p.akun_id = a.id
		WHERE a.id = ? ORDER BY p.id DESC LIMIT 1`, akun).Scan(&status, &bayar, &nominal, &otomatis, &m)
	if err != nil {
		t.Fatal(err)
	}
	return status, bayar, nominal, otomatis, m.String
}
