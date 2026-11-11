package customer

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/testdb"
)

// fixtureAkses: jamaah korban dibuat akun pemilik (data lengkap terisi), lalu dipakai ulang di booking akun lain.
func fixtureAkses(t *testing.T, tx *sql.Tx) (pemilik, lain, korban, bookingPemilik, bookingLain int64) {
	t.Helper()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	pemilik = testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemilikAks"})
	lain = testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "LainAks"})
	korban = testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "KorbanAks", NIK: "3201234567890123"})
	testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id=?, no_paspor='X1234567', alamat='Jl. Rahasia 1', no_hp='081200001111', email='korban@uji.test', tempat_lahir='Garut' WHERE id=?`, pemilik, korban)
	bookingPemilik, _ = testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: korban})
	bookingLain, _ = testdb.NewBookingPax(t, tx, jadwal, "baru", 1000, testdb.Pax{JamaahID: korban})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, pemilik, bookingPemilik)
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=? WHERE id=?`, lain, bookingLain)
	return
}

func bacaDataJamaah(t *testing.T, tx *sql.Tx, akun, booking int64) *dataJamaah {
	t.Helper()
	d, err := scanDataJamaah(tx.QueryRow(dataJamaahSelect, akun, booking))
	if err != nil {
		t.Fatal(err)
	}
	return d
}

// C1: akun yang hanya memakai ulang jamaah lewat NIK + nama tidak membaca data lengkapnya.
func TestDataJamaahTerbatasUntukAkunBukanPembuat(t *testing.T) {
	_, tx := testdb.Tx(t)
	pemilik, lain, korban, bPemilik, bLain := fixtureAkses(t, tx)

	penuh := bacaDataJamaah(t, tx, pemilik, bPemilik)
	if penuh.AksesTerbatas || penuh.NoPaspor != "X1234567" || penuh.Alamat != "Jl. Rahasia 1" {
		t.Fatalf("pembuat harus melihat data lengkap: %+v", penuh)
	}

	terbatas := bacaDataJamaah(t, tx, lain, bLain)
	if !terbatas.AksesTerbatas || terbatas.JamaahID != korban || terbatas.NamaLengkap == "" || terbatas.NIKTersamar == "" {
		t.Fatalf("akses terbatas tetap memuat nama dan NIK tersamar: %+v", terbatas)
	}
	if terbatas.NoPaspor != "" || terbatas.Alamat != "" || terbatas.NoHP != "" || terbatas.TempatLahir != "" || len(terbatas.Kurang) != 0 {
		t.Fatalf("isian bocor ke akun bukan pembuat: %+v", terbatas)
	}
	if terbatas.Total == 0 || terbatas.Terisi != penuh.Terisi {
		t.Fatalf("angka kelengkapan harus tetap ada: terisi %d dari %d", terbatas.Terisi, terbatas.Total)
	}

	ctx := context.Background()
	for akun, mau := range map[int64]bool{pemilik: true, lain: false} {
		got, err := aksesPenuhJamaah(ctx, tx, akun, korban)
		if err != nil || got != mau {
			t.Errorf("aksesPenuhJamaah(akun %d) = %v, %v; mau %v", akun, got, err, mau)
		}
	}
}

// C1: akun bukan pembuat tidak bisa mengisi kolom kosong maupun mengajukan perubahan.
func TestSimpanDitolakUntukAksesTerbatas(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	pemilik, lain, korban, _, _ := fixtureAkses(t, tx)

	grup, err := kelompokDari(ctx, tx, lain, korban)
	if err != nil || len(grup) != 1 || grup[0].Penuh {
		t.Fatalf("grup akun lain = %+v, %v", grup, err)
	}
	if _, err := simpanPerubahanPelanggan(ctx, tx, lain, grup, map[string]string{"kota": "Depok", "alamat": "Jl. Baru"}); !errors.Is(err, errAksesTerbatas) {
		t.Fatalf("simpan akses terbatas harus ditolak: %v", err)
	}
	if got := nilaiKolom(t, tx, korban, "kota"); got != "" {
		t.Fatalf("kolom kosong terisi oleh akun bukan pembuat: %q", got)
	}

	grupPemilik, err := kelompokDari(ctx, tx, pemilik, korban)
	if err != nil || !grupPemilik[0].Penuh {
		t.Fatalf("grup pembuat = %+v, %v", grupPemilik, err)
	}
	if h, err := simpanPerubahanPelanggan(ctx, tx, pemilik, grupPemilik, map[string]string{"kota": "Depok"}); err != nil || h.Langsung != 1 {
		t.Fatalf("pembuat harus bisa mengisi kolom kosong: %+v, %v", h, err)
	}
}

// Baris akses penuh dan akses terbatas yang senama tidak digabung menjadi satu orang.
func TestKelompokOrangMemisahkanAkses(t *testing.T) {
	semua := []anggotaJamaah{
		{ID: 3, Nama: "Siti Aminah", Lahir: "1990-01-01", Penuh: true},
		{ID: 2, Nama: "siti  aminah", Lahir: "1990-01-01", Penuh: false},
		{ID: 1, Nama: "Siti Aminah", Lahir: "", Penuh: true},
	}
	grup := kelompokOrang(semua)
	if len(grup) != 2 || len(grup[0]) != 2 || grup[0][1].ID != 1 || grup[1][0].ID != 2 {
		t.Fatalf("kelompok = %+v", grup)
	}
}

// Detail jamaah di /api/account/jamaah/{id} tidak memuat isian, email, atau pengajuan untuk akses terbatas.
func TestBangunDetailAksesTerbatas(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, lain, korban, _, _ := fixtureAkses(t, tx)
	grup, err := kelompokDari(ctx, tx, lain, korban)
	if err != nil {
		t.Fatal(err)
	}
	d, err := (&JamaahAkun{}).bangunDetail(ctx, tx, lain, grup)
	if err != nil {
		t.Fatal(err)
	}
	if !d.AksesTerbatas || d.Email != "" || len(d.Data) != 0 || len(d.Kurang) != 0 || d.Pengajuan != nil || d.NamaLengkap == "" {
		t.Fatalf("detail akses terbatas = %+v", d)
	}
}

// Surat memuat tempat dan tanggal lahir: tidak tersedia untuk akses terbatas, dan rekomendasi paspor tidak ditawarkan.
func TestEvaluasiSuratAksesTerbatas(t *testing.T) {
	o := &orderSummary{Status: "lunas", Tab: "lunas", TotalHarga: 1000, Terbayar: 1000}
	items := evaluasiSurat(o, &dataJamaah{JamaahID: 1, PaxType: "reguler", AksesTerbatas: true}, true, nil)
	if len(items) != 1 || items[0].Bisa || items[0].Alasan == "" || len(items[0].Kurang) != 0 {
		t.Fatalf("surat akses terbatas = %+v", items)
	}
}

func jumlahPembuat(t *testing.T, tx *sql.Tx, jamaah int64) sql.NullInt64 {
	t.Helper()
	var v sql.NullInt64
	if err := tx.QueryRow(`SELECT dibuat_oleh_akun_id FROM jamaah WHERE id=?`, jamaah).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

// Klaim pembuat gugur bila jamaah dipakai booking pihak lain sebelum pembuat punya pembayaran terkonfirmasi.
func TestGugurkanKlaimPembuat(t *testing.T) {
	ctx := context.Background()

	t.Run("booking sendiri tidak menggugurkan", func(t *testing.T) {
		_, tx := testdb.Tx(t)
		pemilik, _, korban, bPemilik, _ := fixtureAkses(t, tx)
		if err := shared.GugurkanKlaimPembuatBooking(ctx, tx, bPemilik); err != nil {
			t.Fatal(err)
		}
		if got := jumlahPembuat(t, tx, korban); got.Int64 != pemilik {
			t.Fatalf("pembuat = %v, mau %d", got, pemilik)
		}
	})

	t.Run("booking akun lain menggugurkan klaim tanpa pembayaran", func(t *testing.T) {
		_, tx := testdb.Tx(t)
		pemilik, _, korban, bPemilik, bLain := fixtureAkses(t, tx)
		testdb.AddPayment(t, tx, bPemilik, 500, "pending") // menunggu verifikasi belum dihitung
		if err := shared.GugurkanKlaimPembuatBooking(ctx, tx, bLain); err != nil {
			t.Fatal(err)
		}
		if got := jumlahPembuat(t, tx, korban); got.Valid {
			t.Fatalf("klaim harus gugur, pembuat masih %d", got.Int64)
		}
		if penuh, _ := aksesPenuhJamaah(ctx, tx, pemilik, korban); penuh {
			t.Fatal("setelah gugur, bekas pembuat harus akses terbatas")
		}
	})

	t.Run("pembayaran terkonfirmasi mempertahankan klaim", func(t *testing.T) {
		_, tx := testdb.Tx(t)
		pemilik, _, korban, bPemilik, bLain := fixtureAkses(t, tx)
		testdb.AddPayment(t, tx, bPemilik, 500, "confirmed")
		if err := shared.GugurkanKlaimPembuatBooking(ctx, tx, bLain); err != nil {
			t.Fatal(err)
		}
		if got := jumlahPembuat(t, tx, korban); got.Int64 != pemilik {
			t.Fatalf("pembuat = %v, mau %d", got, pemilik)
		}
	})

	t.Run("booking admin menggugurkan klaim tanpa pembayaran", func(t *testing.T) {
		_, tx := testdb.Tx(t)
		_, _, korban, _, bLain := fixtureAkses(t, tx)
		testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=NULL WHERE id=?`, bLain)
		if err := shared.GugurkanKlaimPembuatBooking(ctx, tx, bLain); err != nil {
			t.Fatal(err)
		}
		if got := jumlahPembuat(t, tx, korban); got.Valid {
			t.Fatalf("klaim harus gugur, pembuat masih %d", got.Int64)
		}
	})
}
