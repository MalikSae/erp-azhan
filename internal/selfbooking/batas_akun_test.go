package selfbooking

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/testdb"
)

// K2: seat yang ditahan booking akun yang belum dibayar dibatasi per jadwal dan total, dihitung
// dari database di dalam transaksi booking (bukan limiter memori yang dicatat di akhir).
func TestBatasSeatAktifPerAkunPerJadwal(t *testing.T) {
	tx, scheduleID, brandID, _ := setupAgenBooking(t)
	ctx := context.Background()
	akun := akunUji(t, tx, brandID, "Pemesan Borong", nomorUji(906))

	rombongan := func(n, mulai int) BookingRequest {
		req := BookingRequest{ScheduleID: scheduleID, PemesanTidakIkut: true,
			PIC: PICInput{NamaLengkap: fmt.Sprintf("Jamaah Borong %d", mulai), JenisKelamin: "L", RoomType: "Quad"}}
		for i := 1; i < n; i++ {
			req.Anggota = append(req.Anggota, AnggotaInput{PaxType: "reguler", NamaLengkap: fmt.Sprintf("Jamaah Borong %d", mulai+i), JenisKelamin: "L", RoomType: quad()})
		}
		return req
	}
	// Dua panggilan beruntun: 5 seat lolos, 5 seat berikutnya (total 10 > 9 per jadwal) ditolak.
	if _, err := processBookingTx(ctx, tx, brandID, rombongan(5, 0), inisiator{accountID: akun}); err != nil {
		t.Fatalf("booking pertama: %v", err)
	}
	if _, err := processBookingTx(ctx, tx, brandID, rombongan(5, 10), inisiator{accountID: akun}); !errors.Is(err, ErrBatasSeatAkun) {
		t.Fatalf("booking kedua melewati batas seat per jadwal: err=%v, want ErrBatasSeatAkun", err)
	}
	// Sisa jatah 4 seat di jadwal ini masih bisa dipakai.
	if _, err := processBookingTx(ctx, tx, brandID, rombongan(4, 20), inisiator{accountID: akun}); err != nil {
		t.Fatalf("booking dalam sisa batas: %v", err)
	}
}

func TestCekBatasAkunTotalDanHarian(t *testing.T) {
	tx, scheduleID, brandID, _ := setupAgenBooking(t)
	ctx := context.Background()
	akun := akunUji(t, tx, brandID, "Pemesan Batas", nomorUji(907))
	var jadwalLain, pic int64
	if err := tx.QueryRow(`SELECT id FROM schedules WHERE id<>? ORDER BY id LIMIT 1`, scheduleID).Scan(&jadwalLain); err != nil {
		t.Skip("butuh minimal dua jadwal")
	}
	pic = testdb.NewJamaah(t, tx, brandID, testdb.JamaahOpts{Nama: "PIC batas"})
	tahan := time.Now().Add(12 * time.Hour)
	tambah := func(sched int64, seat int, status string, hold time.Time, dibuat time.Time) {
		kode, err := shared.UniqueCode(ctx, tx, "bookings", "id_booking", "Z", 5)
		if err != nil {
			t.Fatal(err)
		}
		testdb.Exec(t, tx, `INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, seat_count, status, is_seat_blocked, seat_hold_expires_at, customer_account_id, created_at)
			VALUES (?,?,?,?,?,TRUE,?,?,?)`, kode, sched, pic, seat, status, hold, akun, dibuat)
	}
	batas := batasAkun{seatPerJadwal: 9, seatTotal: 12, bookingHarian: 3}
	lama := time.Now().Add(-48 * time.Hour)

	tambah(jadwalLain, 8, "baru", tahan, lama)
	tambah(jadwalLain, 9, "baru", time.Now().Add(-time.Hour), lama) // tahan seat sudah lewat: tidak dihitung
	tambah(jadwalLain, 9, "dp", tahan, lama)                        // sudah dibayar: tidak dihitung
	if err := cekBatasAkun(ctx, tx, akun, scheduleID, 4, batas); err != nil {
		t.Fatalf("8 + 4 = 12 seat masih dalam batas total: %v", err)
	}
	if err := cekBatasAkun(ctx, tx, akun, scheduleID, 5, batas); !errors.Is(err, ErrBatasSeatAkun) {
		t.Fatalf("8 + 5 > 12 harus ditolak: %v", err)
	}

	// Batas booking harian dihitung dari database (tidak bisa diakali permintaan paralel).
	for i := 0; i < 3; i++ {
		tambah(jadwalLain, 0, "batal", tahan, time.Now().Add(-time.Hour))
	}
	if err := cekBatasAkun(ctx, tx, akun, scheduleID, 1, batas); !errors.Is(err, ErrBatasBookingHarian) {
		t.Fatalf("booking ke-4 dalam 24 jam: %v, want ErrBatasBookingHarian", err)
	}
}
