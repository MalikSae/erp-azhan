package booking

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/testdb"
)

// SPEK-LAYANAN-TAMBAHAN-BOOKING-2026-11-27: total = harga pax + layanan - diskon.

func totalBooking(t *testing.T, tx *sql.Tx, id int64) float64 {
	t.Helper()
	var v float64
	if err := tx.QueryRow(`SELECT COALESCE(total_harga,0) FROM bookings WHERE id=?`, id).Scan(&v); err != nil {
		t.Fatal(err)
	}
	return v
}

func jumlahLayanan(t *testing.T, tx *sql.Tx, id int64) int {
	t.Helper()
	var n int
	if err := tx.QueryRow(`SELECT COUNT(*) FROM booking_layanan WHERE booking_id=?`, id).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func layananPertama(t *testing.T, tx *sql.Tx, id int64) int64 {
	t.Helper()
	var lid int64
	if err := tx.QueryRow(`SELECT id FROM booking_layanan WHERE booking_id=? ORDER BY id LIMIT 1`, id).Scan(&lid); err != nil {
		t.Fatal(err)
	}
	return lid
}

func TestLayananMenaikkanTotalTanpaMengubahDP(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 1000, MinimalDP: 300, Blocked: true})
	testdb.AddPayment(t, tx, b.ID, 400, "confirmed")

	dpSebelum, err := shared.RequiredBookingDP(ctx, tx, b.ID, 1000)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.addLayananTx(ctx, tx, b.ID, "Kereta cepat", 500, 0); err != nil {
		t.Fatalf("tambah layanan: %v", err)
	}
	if got := totalBooking(t, tx, b.ID); got != 1500 {
		t.Fatalf("total = %.0f, mau 1500", got)
	}
	dpSesudah, err := shared.RequiredBookingDP(ctx, tx, b.ID, 1500)
	if err != nil {
		t.Fatal(err)
	}
	if dpSesudah != dpSebelum {
		t.Fatalf("DP wajib berubah: %.0f -> %.0f", dpSebelum, dpSesudah)
	}
	if got := testdb.Status(t, tx, b.ID); got != "dp" {
		t.Fatalf("status = %s, mau dp", got)
	}
}

func TestLayananBookingLunasTurunKeDPLaluLunasLagi(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()

	turun, naik := 0, 0
	origTurun, origNaik := komisi.OnBookingTurunDariLunas, komisi.OnBookingLunas
	komisi.OnBookingTurunDariLunas = func(context.Context, *sql.Tx, int64) (int64, error) { turun++; return 0, nil }
	komisi.OnBookingLunas = func(context.Context, *sql.Tx, int64) error { naik++; return nil }
	t.Cleanup(func() { komisi.OnBookingTurunDariLunas, komisi.OnBookingLunas = origTurun, origNaik })

	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "lunas", TotalHarga: 1000, MinimalDP: 300, Blocked: true})
	testdb.AddPayment(t, tx, b.ID, 1000, "confirmed")

	if err := repo.addLayananTx(ctx, tx, b.ID, "Kereta cepat", 200, 0); err != nil {
		t.Fatalf("tambah layanan: %v", err)
	}
	if got := testdb.Status(t, tx, b.ID); got != "dp" || turun != 1 {
		t.Fatalf("status = %s, komisi dibatalkan %d kali; mau dp dan 1", got, turun)
	}
	var blocked bool
	var exp sql.NullTime
	if err := tx.QueryRow(`SELECT is_seat_blocked, seat_hold_expires_at FROM bookings WHERE id=?`, b.ID).Scan(&blocked, &exp); err != nil {
		t.Fatal(err)
	}
	if !blocked || exp.Valid {
		t.Fatalf("seat blocked=%v batas=%v, mau dipegang tanpa batas waktu", blocked, exp)
	}

	if err := repo.deleteLayananTx(ctx, tx, b.ID, layananPertama(t, tx, b.ID)); err != nil {
		t.Fatalf("hapus layanan: %v", err)
	}
	if got := testdb.Status(t, tx, b.ID); got != "lunas" || naik != 1 {
		t.Fatalf("status = %s, komisi dicatat %d kali; mau lunas dan 1", got, naik)
	}
}

func TestHapusLayananDiBawahPembayaranDitolak(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 1000, MinimalDP: 300, Blocked: true})
	if err := repo.addLayananTx(ctx, tx, b.ID, "Kereta cepat", 500, 0); err != nil {
		t.Fatal(err)
	}
	testdb.AddPayment(t, tx, b.ID, 1200, "confirmed")
	if err := repo.recalculateTotalTx(ctx, tx, b.ID); err != nil {
		t.Fatal(err)
	}

	testdb.Exec(t, tx, `SAVEPOINT sebelum_hapus`)
	var bisnis *BusinessError
	err := repo.deleteLayananTx(ctx, tx, b.ID, layananPertama(t, tx, b.ID))
	if !errors.As(err, &bisnis) || bisnis.Message != "Total tidak boleh di bawah pembayaran Rp 1.200." {
		t.Fatalf("err = %v, mau penolakan total di bawah pembayaran", err)
	}
	// Repository membatalkan transaksinya saat galat; di sini dikembalikan lewat savepoint.
	testdb.Exec(t, tx, `ROLLBACK TO SAVEPOINT sebelum_hapus`)
	if n, total := jumlahLayanan(t, tx, b.ID), totalBooking(t, tx, b.ID); n != 1 || total != 1500 {
		t.Fatalf("sesudah penolakan: layanan %d, total %.0f; mau 1 dan 1500", n, total)
	}
}

func TestHapusLayananSampaiTotalNolDitolak(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 300, Blocked: true})
	testdb.Exec(t, tx, `INSERT INTO booking_layanan (booking_id, nama, nominal) VALUES (?, 'Kereta cepat', 500)`, b.ID)
	testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'Uji', 1000)`, b.ID)
	if err := repo.recalculateTotalTx(ctx, tx, b.ID); err != nil {
		t.Fatal(err)
	}
	if got := totalBooking(t, tx, b.ID); got != 500 {
		t.Fatalf("total = %.0f, mau 500", got)
	}
	var bisnis *BusinessError
	err := repo.deleteLayananTx(ctx, tx, b.ID, layananPertama(t, tx, b.ID))
	if !errors.As(err, &bisnis) || bisnis.Message != "Total booking tidak boleh Rp 0." {
		t.Fatalf("err = %v, mau penolakan total Rp 0", err)
	}
}

func TestLayananBookingBatalTerkunci(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "batal", TotalHarga: 1500})
	testdb.Exec(t, tx, `INSERT INTO booking_layanan (booking_id, nama, nominal) VALUES (?, 'Kereta cepat', 500)`, b.ID)

	if err := repo.addLayananTx(ctx, tx, b.ID, "Telefric", 100, 0); !errors.Is(err, ErrBookingBatal) {
		t.Fatalf("tambah di booking batal: err = %v, mau ErrBookingBatal", err)
	}
	if err := repo.deleteLayananTx(ctx, tx, b.ID, layananPertama(t, tx, b.ID)); !errors.Is(err, ErrBookingBatal) {
		t.Fatalf("hapus di booking batal: err = %v, mau ErrBookingBatal", err)
	}
	if n, total := jumlahLayanan(t, tx, b.ID), totalBooking(t, tx, b.ID); n != 1 || total != 1500 {
		t.Fatalf("booking batal berubah: layanan %d, total %.0f", n, total)
	}
}

func TestBatasDiskonTidakIkutLayanan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 300, Blocked: true})
	testdb.Exec(t, tx, `INSERT INTO booking_layanan (booking_id, nama, nominal) VALUES (?, 'Kereta cepat', 10000)`, b.ID)
	testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'Uji', 150)`, b.ID)
	var bisnis *BusinessError
	if err := cekBatasDiskonTx(ctx, tx, b.ID); !errors.As(err, &bisnis) {
		t.Fatalf("diskon 15%% dari harga pax harus ditolak walau ada layanan besar, dapat %v", err)
	}
}

// Alur finalisasi yang sebenarnya: harga pax ikut harga paket saat ini, layanan draft tetap dan ikut total.
func TestLayananDraftTerbawaSaatFinalisasi(t *testing.T) {
	j := siapkanRumus(t)
	repo := &Repository{}
	a := j.jamaah("Draft")
	booking, _ := testdb.NewBookingPax(t, j.tx, j.sched, "draft", 1000, testdb.Pax{JamaahID: a})
	testdb.Exec(t, j.tx, `UPDATE bookings SET is_seat_blocked=FALSE, id_booking=NULL WHERE id=?`, booking)

	if err := repo.addLayananTx(j.ctx, j.tx, booking, "Kereta cepat", 750000, 0); err != nil {
		t.Fatalf("tambah layanan di draft: %v", err)
	}
	if got := testdb.Status(t, j.tx, booking); got != "draft" {
		t.Fatalf("status = %s, mau tetap draft", got)
	}
	if err := repo.finalizeBookingTx(j.ctx, j.tx, booking, true); err != nil {
		t.Fatalf("finalisasi: %v", err)
	}
	if n, total := jumlahLayanan(t, j.tx, booking), totalBooking(t, j.tx, booking); n != 1 || total != 30_750_000 {
		t.Fatalf("layanan %d, total %.0f; mau 1 dan 30750000", n, total)
	}
}
