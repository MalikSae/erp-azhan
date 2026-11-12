package selfbooking

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/testdb"
)

// Batas hold seat agen (keputusan 12 November 2026): booking portal agen dibatasi 20 seat per jadwal dan 45
// total, hanya menghitung seat dewasa dari booking 'baru' buatan agen yang hold seat-nya belum lewat.
func TestCekBatasSeatAgen(t *testing.T) {
	tx, scheduleID, brandID, agenID := setupAgenBooking(t)
	ctx := context.Background()
	var jadwalLain int64
	if err := tx.QueryRow(`SELECT id FROM schedules WHERE id<>? ORDER BY id LIMIT 1`, scheduleID).Scan(&jadwalLain); err != nil {
		t.Skip("butuh minimal dua jadwal")
	}
	pic := testdb.NewJamaah(t, tx, brandID, testdb.JamaahOpts{Nama: "PIC batas agen"})
	tahan := time.Now().Add(12 * time.Hour)
	tambah := func(sched int64, seat int, status string, hold time.Time, agenPembuat any) {
		kode, err := shared.UniqueCode(ctx, tx, "bookings", "id_booking", "Z", 5)
		if err != nil {
			t.Fatal(err)
		}
		testdb.Exec(t, tx, `INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, seat_count, status, is_seat_blocked, seat_hold_expires_at, dibuat_oleh_agen_akun_id)
			VALUES (?,?,?,?,?,TRUE,?,?)`, kode, sched, pic, seat, status, hold, agenPembuat)
	}

	tambah(scheduleID, 15, "baru", tahan, agenID)
	tambah(scheduleID, 9, "baru", time.Now().Add(-time.Hour), agenID) // hold lewat: tidak dihitung
	tambah(scheduleID, 9, "dp", tahan, agenID)                        // sudah DP: tidak dihitung
	tambah(scheduleID, 9, "baru", tahan, nil)                         // bukan buatan agen: tidak dihitung
	if err := cekBatasSeatAgen(ctx, tx, agenID, scheduleID, 5); err != nil {
		t.Fatalf("15 + 5 = 20 seat per jadwal masih dalam batas: %v", err)
	}
	err := cekBatasSeatAgen(ctx, tx, agenID, scheduleID, 6)
	var be *BatasSeatAgenError
	if !errors.Is(err, ErrBatasSeatAgen) || !errors.As(err, &be) || be.SisaPerJadwal != 5 {
		t.Fatalf("15 + 6 > 20 harus ditolak dengan sisa 5: %v", err)
	}

	// Batas total lintas jadwal: 15 + 20 = 35 ditahan; 10 lagi pas 45, 11 lewat.
	tambah(jadwalLain, 20, "baru", tahan, agenID)
	var jadwalKetiga int64
	if err := tx.QueryRow(`SELECT id FROM schedules WHERE id NOT IN (?,?) ORDER BY id LIMIT 1`, scheduleID, jadwalLain).Scan(&jadwalKetiga); err == nil {
		if err := cekBatasSeatAgen(ctx, tx, agenID, jadwalKetiga, 10); err != nil {
			t.Fatalf("35 + 10 = 45 total masih dalam batas: %v", err)
		}
		if err := cekBatasSeatAgen(ctx, tx, agenID, jadwalKetiga, 11); !errors.Is(err, ErrBatasSeatAgen) {
			t.Fatalf("35 + 11 > 45 total harus ditolak: %v", err)
		}
	}

	// Batas khusus dari admin menggantikan bawaan.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET batas_seat_agen_per_jadwal=60, batas_seat_agen_total=100 WHERE id=?`, agenID)
	if err := cekBatasSeatAgen(ctx, tx, agenID, scheduleID, 40); err != nil {
		t.Fatalf("batas khusus 60 per jadwal: 15 + 40 harus lolos: %v", err)
	}
	if err := cekBatasSeatAgen(ctx, tx, agenID, scheduleID, 46); !errors.Is(err, ErrBatasSeatAgen) {
		t.Fatalf("batas khusus 60 per jadwal: 15 + 46 harus ditolak: %v", err)
	}
}

// Booking portal agen lewat ProcessBookingAgen ikut ditolak saat batas tercapai.
func TestProcessBookingAgenMenolakLewatBatasSeat(t *testing.T) {
	tx, scheduleID, brandID, agenID := setupAgenBooking(t)
	ctx := context.Background()
	testdb.Exec(t, tx, `UPDATE customer_accounts SET batas_seat_agen_per_jadwal=1, batas_seat_agen_total=1 WHERE id=?`, agenID)
	nik := "3174010101013333"
	req := agenReq(scheduleID, PICInput{NamaLengkap: "Jamaah Batas", JenisKelamin: "L", RoomType: "Quad", NIK: &nik},
		AnggotaInput{PaxType: "reguler", NamaLengkap: "Anggota Batas", JenisKelamin: "P", RoomType: stringPtr("Quad"), NIK: stringPtr("3174010101013334")})
	_, err := processBookingTx(ctx, tx, brandID, req, inisiatorAgen(agenID))
	if !errors.Is(err, ErrBatasSeatAgen) {
		t.Fatalf("2 seat dengan batas 1: err=%v, want ErrBatasSeatAgen", err)
	}
}

// Jawaban HTTP saat batas tercapai: 429 dengan kode stabil dan pesan berisi sisa seat.
func TestWriteAgenBookingErrorBatasSeat(t *testing.T) {
	h := &Handler{}
	rec := httptest.NewRecorder()
	h.writeAgenBookingError(rec, &BatasSeatAgenError{BatasPerJadwal: 20, BatasTotal: 45, SisaPerJadwal: 3, SisaTotal: 10})
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), KodeBatasSeatAgen) || !strings.Contains(rec.Body.String(), "menahan 3 seat") {
		t.Fatalf("got %d %s", rec.Code, rec.Body.String())
	}
	var body struct {
		BatasPerJadwal int `json:"batas_per_jadwal"`
		BatasTotal     int `json:"batas_total"`
		SisaPerJadwal  int `json:"sisa_per_jadwal"`
		SisaTotal      int `json:"sisa_total"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body.BatasPerJadwal != 20 || body.BatasTotal != 45 || body.SisaPerJadwal != 3 || body.SisaTotal != 10 {
		t.Fatalf("sisa seat tidak dikirim sebagai data: %+v", body)
	}
}
