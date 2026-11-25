package booking

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// Tidak ada booking di luar sistem: setelah setiap jalur booking, seat_sisa harus tetap
// seat_total − seat yang dipegang booking (keputusan 2026-11-25).

var nikUrut atomic.Int64

func nikUji() string {
	return fmt.Sprintf("31%08d%06d", time.Now().UnixNano()%100_000_000, nikUrut.Add(1)%1_000_000)
}

type jadwalRumus struct {
	tx     *sql.Tx
	sched  int64
	brand  int64
	admin  int64
	ctx    context.Context
	quad   *string
	rumusT *testing.T
}

func siapkanRumus(t *testing.T) *jadwalRumus {
	t.Helper()
	_, tx := testdb.Tx(t)
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', harga_quad=30000000, harga_triple=32000000, harga_double=35000000, harga_infant=5000000 WHERE id=?`, sched)
	testdb.SamakanSisa(t, tx, sched)
	var admin int64
	if err := tx.QueryRow(`SELECT id FROM admin_users ORDER BY id LIMIT 1`).Scan(&admin); err != nil {
		t.Fatal(err)
	}
	quad := "Quad"
	return &jadwalRumus{tx: tx, sched: sched, brand: brand, admin: admin, ctx: context.Background(), quad: &quad, rumusT: t}
}

func (j *jadwalRumus) jamaah(nama string) int64 {
	id := testdb.NewJamaah(j.rumusT, j.tx, j.brand, testdb.JamaahOpts{Nama: nama, NIK: nikUji()})
	lahir := "1990-01-01"
	if nama == "Bayi" {
		lahir = "2098-06-01" // < 2 tahun saat berangkat 2099-01-01
	}
	testdb.Exec(j.rumusT, j.tx, `UPDATE jamaah SET tanggal_lahir=? WHERE id=?`, lahir, id)
	return id
}

// bookingAdmin membuat booking admin berisi n pax reguler dan satu infant.
func (j *jadwalRumus) bookingAdmin(n int) int64 {
	t := j.rumusT
	t.Helper()
	var pax []CreateBookingPaxItem
	for i := 0; i < n; i++ {
		pax = append(pax, CreateBookingPaxItem{JamaahID: j.jamaah(fmt.Sprintf("Reguler %d", i)), PaxType: "reguler", RoomType: j.quad})
	}
	pax = append(pax, CreateBookingPaxItem{JamaahID: j.jamaah("Bayi"), PaxType: "infant"})
	r := &Repository{}
	id, err := r.createBookingTx(j.ctx, j.tx, &CreateBookingRequest{ScheduleID: j.sched, PicJamaahID: pax[0].JamaahID, Pax: pax}, j.admin)
	if err != nil {
		t.Fatalf("booking admin: %v", err)
	}
	return id
}

func TestRumusSeatBookingAdminDanBatal(t *testing.T) {
	j := siapkanRumus(t)
	var sebelum, sesudah int
	if err := j.tx.QueryRow(`SELECT seat_sisa FROM schedules WHERE id=?`, j.sched).Scan(&sebelum); err != nil {
		t.Fatal(err)
	}
	id := j.bookingAdmin(2)
	testdb.CekRumusSeat(t, j.tx, j.sched, "booking admin 2 reguler + 1 infant")
	if err := j.tx.QueryRow(`SELECT seat_sisa FROM schedules WHERE id=?`, j.sched).Scan(&sesudah); err != nil || sesudah != sebelum-2 {
		t.Fatalf("seat_sisa %d → %d err=%v, want berkurang 2", sebelum, sesudah, err)
	}
	var seatCount int
	if err := j.tx.QueryRow(`SELECT seat_count FROM bookings WHERE id=?`, id).Scan(&seatCount); err != nil || seatCount != 2 {
		t.Fatalf("seat_count=%d err=%v, want 2 (infant tidak makan seat)", seatCount, err)
	}
	if err := updateBookingStatusTx(j.ctx, j.tx, id, "batal", nil); err != nil {
		t.Fatalf("batal: %v", err)
	}
	testdb.CekRumusSeat(t, j.tx, j.sched, "batal booking")
}

func TestRumusSeatBatalPax(t *testing.T) {
	j := siapkanRumus(t)
	id := j.bookingAdmin(2)
	var paxID int64
	if err := j.tx.QueryRow(`SELECT id FROM booking_pax WHERE booking_id=? AND pax_type='reguler' ORDER BY id DESC LIMIT 1`, id).Scan(&paxID); err != nil {
		t.Fatal(err)
	}
	r := &Repository{}
	if err := r.cancelPaxTx(j.ctx, j.tx, id, paxID, nil, true); err != nil {
		t.Fatalf("batal pax: %v", err)
	}
	testdb.CekRumusSeat(t, j.tx, j.sched, "batal satu pax")
}

func TestRumusSeatLepasDanHoldSeat(t *testing.T) {
	j := siapkanRumus(t)
	id := j.bookingAdmin(3)
	if err := cancelSeatBlockTx(j.ctx, j.tx, id); err != nil {
		t.Fatalf("lepas seat: %v", err)
	}
	testdb.CekRumusSeat(t, j.tx, j.sched, "lepas seat")
	exp := time.Now().Add(2 * time.Hour)
	if err := blockSeatTx(j.ctx, j.tx, id, &exp, "uji-rumus-seat"); err != nil {
		t.Fatalf("hold seat: %v", err)
	}
	testdb.CekRumusSeat(t, j.tx, j.sched, "hold seat")
	if err := blockSeatTx(j.ctx, j.tx, id, &exp, "uji-rumus-seat"); err != nil {
		t.Fatalf("hold seat ulang (idempoten): %v", err)
	}
	testdb.CekRumusSeat(t, j.tx, j.sched, "hold seat ulang")

	// Hold kedaluwarsa dilepas worker.
	testdb.Exec(t, j.tx, `UPDATE bookings SET seat_hold_expires_at = NOW() - INTERVAL 1 MINUTE WHERE id=?`, id)
	released, err := releaseSingleSeatHoldTx(j.ctx, j.tx, id)
	if err != nil || !released {
		t.Fatalf("lepas hold kedaluwarsa: released=%v err=%v", released, err)
	}
	testdb.CekRumusSeat(t, j.tx, j.sched, "hold kedaluwarsa")
}

func TestRumusSeatFinalisasiDraft(t *testing.T) {
	j := siapkanRumus(t)
	a, b := j.jamaah("Draft A"), j.jamaah("Draft B")
	id, _ := testdb.NewBookingPax(t, j.tx, j.sched, "draft", 60_000_000, testdb.Pax{JamaahID: a}, testdb.Pax{JamaahID: b})
	// Draft belum memegang seat.
	testdb.Exec(t, j.tx, `UPDATE bookings SET is_seat_blocked=FALSE, id_booking=NULL WHERE id=?`, id)
	testdb.CekRumusSeat(t, j.tx, j.sched, "draft")
	r := &Repository{}
	if err := r.finalizeBookingTx(j.ctx, j.tx, id, true); err != nil {
		t.Fatalf("finalisasi: %v", err)
	}
	testdb.CekRumusSeat(t, j.tx, j.sched, "finalisasi draft 2 pax")
	var blocked bool
	var seatCount int
	if err := j.tx.QueryRow(`SELECT is_seat_blocked, seat_count FROM bookings WHERE id=?`, id).Scan(&blocked, &seatCount); err != nil || !blocked || seatCount != 2 {
		t.Fatalf("finalisasi harus memegang 2 seat: blocked=%v seat_count=%d err=%v", blocked, seatCount, err)
	}
}
