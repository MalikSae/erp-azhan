package customer

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestHapusAkun(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	akun := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunHapusQzp"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahHapusQzp"})
	b, _ := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j})
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id = ? WHERE id = ?`, akun, b)

	cek := func(want string) {
		t.Helper()
		got, err := alasanTolakHapus(ctx, tx, akun)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("alasan = %q, mau %q", got, want)
		}
	}
	cek("Masih ada booking aktif.")
	testdb.Exec(t, tx, `UPDATE bookings SET status = 'batal' WHERE id = ?`, b)
	testdb.Exec(t, tx, `UPDATE customer_accounts SET status_agen = 'aktif' WHERE id = ?`, akun)
	cek("Akun agen aktif.")
	testdb.Exec(t, tx, `UPDATE customer_accounts SET status_agen = 'nonaktif' WHERE id = ?`, akun)
	testdb.Exec(t, tx, `INSERT INTO pengajuan_pencairan (brand_id, akun_agen_id, nominal_diajukan, status, rekening_bank, rekening_nomor, rekening_atas_nama)
		VALUES (?,?,500000,'pending','BCA','1','A')`, brand, akun)
	cek("Masih ada pencairan diproses.")
	testdb.Exec(t, tx, `UPDATE pengajuan_pencairan SET status = 'ditolak' WHERE akun_agen_id = ?`, akun)
	cek("")

	var emailLama, hpLama string
	if err := tx.QueryRowContext(ctx, `SELECT email, no_hp FROM customer_accounts WHERE id = ?`, akun).Scan(&emailLama, &hpLama); err != nil {
		t.Fatal(err)
	}
	testdb.Exec(t, tx, `INSERT INTO customer_sessions (account_id, token_hash, expires_at) VALUES (?, ?, DATE_ADD(NOW(), INTERVAL 1 DAY))`, akun, "hash-uji-hapus-qzp")
	if err := anonimkanAkun(ctx, tx, akun); err != nil {
		t.Fatal(err)
	}
	var nama, email string
	var hpNormal sql.NullString
	var dihapus sql.NullTime
	if err := tx.QueryRowContext(ctx, `SELECT nama_lengkap, email, no_hp_normal, dihapus_at FROM customer_accounts WHERE id = ?`, akun).
		Scan(&nama, &email, &hpNormal, &dihapus); err != nil {
		t.Fatal(err)
	}
	if nama != "Akun dihapus" || email == emailLama || hpNormal.Valid || !dihapus.Valid {
		t.Errorf("setelah anonim: nama=%s email=%s hp=%v dihapus=%v", nama, email, hpNormal, dihapus)
	}
	var sesiAktif int
	_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM customer_sessions WHERE account_id = ? AND revoked_at IS NULL`, akun).Scan(&sesiAktif)
	if sesiAktif != 0 {
		t.Errorf("sesi masih aktif: %d", sesiAktif)
	}
	// Booking tetap merujuk akun.
	var pemilik int64
	_ = tx.QueryRowContext(ctx, `SELECT customer_account_id FROM bookings WHERE id = ?`, b).Scan(&pemilik)
	if pemilik != akun {
		t.Error("booking kehilangan rujukan akun")
	}
	// Email dan nomor lama bebas dipakai mendaftar ulang di brand yang sama.
	testdb.Exec(t, tx, `INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash)
		VALUES (?, 'Daftar Ulang', ?, ?, ?, 'x')`, brand, emailLama, NormalizeEmail(emailLama), hpLama)
}
