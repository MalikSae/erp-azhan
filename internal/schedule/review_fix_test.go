package schedule

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Simpan form tanpa ubah kuota tidak menyamakan sisa seat diam-diam: sisa tersimpan dipertahankan.
// Kuota diturunkan: sisa dihitung dari booking, seat tampil ikut turun ke kuota baru, keduanya tercatat.
func TestPrepareScheduleUpdateSeat(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	var total, sisa int
	if err := tx.QueryRow(`SELECT seat_total, seat_sisa FROM schedules WHERE id=?`, sched).Scan(&total, &sisa); err != nil {
		t.Fatal(err)
	}
	var dipegang int
	if err := tx.QueryRow(`SELECT COALESCE(SUM(seat_count),0) FROM bookings WHERE schedule_id=? AND is_seat_blocked=1 AND status<>'batal'`, sched).Scan(&dipegang); err != nil {
		t.Fatal(err)
	}
	if sisa == total-dipegang {
		t.Fatalf("fixture harus punya sisa yang tidak cocok dengan booking (sisa=%d)", sisa)
	}

	inp := ScheduleInput{SeatTotal: total, BerangkatTanggal: "2099-01-01"}
	if err := prepareScheduleUpdate(ctx, tx, sched, &inp, nil, brand); err != nil {
		t.Fatalf("kuota tetap: %v", err)
	}
	if inp.SeatSisa != sisa {
		t.Fatalf("kuota tetap: seat_sisa=%d, mau %d (tersimpan)", inp.SeatSisa, sisa)
	}

	testdb.Exec(t, tx, `UPDATE schedules SET seat_total=?, seat_tampil=? WHERE id=?`, dipegang+10, dipegang+10, sched)
	inp = ScheduleInput{SeatTotal: dipegang + 2, BerangkatTanggal: "2099-01-01"}
	if err := prepareScheduleUpdate(ctx, tx, sched, &inp, nil, brand); err != nil {
		t.Fatalf("kuota turun: %v", err)
	}
	if inp.SeatSisa != 2 {
		t.Fatalf("kuota turun: seat_sisa=%d, mau 2", inp.SeatSisa)
	}
	var tampil sql.NullInt64
	if err := tx.QueryRow(`SELECT seat_tampil FROM schedules WHERE id=?`, sched).Scan(&tampil); err != nil {
		t.Fatal(err)
	}
	if !tampil.Valid || int(tampil.Int64) != dipegang+2 {
		t.Fatalf("seat_tampil=%v, mau %d", tampil, dipegang+2)
	}
	var nTampil, nKuota int
	if err := tx.QueryRow(`SELECT COALESCE(SUM(jenis='tampil'),0), COALESCE(SUM(jenis='kuota'),0) FROM schedule_seat_adjustments WHERE schedule_id=?`, sched).Scan(&nTampil, &nKuota); err != nil {
		t.Fatal(err)
	}
	if nTampil != 1 || nKuota != 1 {
		t.Fatalf("riwayat tampil=%d kuota=%d, mau 1 dan 1", nTampil, nKuota)
	}

	// seat_tampil NULL tetap NULL saat kuota turun.
	testdb.Exec(t, tx, `UPDATE schedules SET seat_tampil=NULL WHERE id=?`, sched)
	inp = ScheduleInput{SeatTotal: dipegang + 1, BerangkatTanggal: "2099-01-01"}
	if err := prepareScheduleUpdate(ctx, tx, sched, &inp, nil, brand); err != nil {
		t.Fatalf("kuota turun tanpa seat tampil: %v", err)
	}
	if err := tx.QueryRow(`SELECT seat_tampil FROM schedules WHERE id=?`, sched).Scan(&tampil); err != nil || tampil.Valid {
		t.Fatalf("seat_tampil=%v err=%v, mau NULL", tampil, err)
	}
}

// Arsip lewat PUT /status: paket belum berangkat dengan booking aktif ditolak di dalam transaksi status.
func TestUbahStatusTxArsip(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, sched)
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 20_000_000})
	if b.ScheduleID != sched {
		t.Fatalf("fixture booking di paket lain: %d vs %d", b.ScheduleID, sched)
	}
	status := func() string {
		var s string
		if err := tx.QueryRow(`SELECT status FROM schedules WHERE id=?`, sched).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	err := ubahStatusTx(ctx, tx, sched, StatusArchived, nil, "2026-11-25")
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("booking aktif: err=%v, mau ErrConflict", err)
	}
	if status() != StatusPublished {
		t.Fatal("status tidak boleh berubah saat ditolak")
	}
	other := brand + 1000
	if err := ubahStatusTx(ctx, tx, sched, StatusDraft, &other, "2026-11-25"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("brand lain: err=%v, mau ErrNotFound", err)
	}
	if err := ubahStatusTx(ctx, tx, sched, StatusDraft, &brand, "2026-11-25"); err != nil || status() != StatusDraft {
		t.Fatalf("draft dengan booking aktif harus boleh: err=%v status=%s", err, status())
	}

	// Sudah berangkat: boleh diarsipkan walau booking masih aktif.
	testdb.Exec(t, tx, `UPDATE schedules SET berangkat_tanggal='2000-01-01' WHERE id=?`, sched)
	if err := ubahStatusTx(ctx, tx, sched, StatusArchived, nil, "2026-11-25"); err != nil || status() != StatusArchived {
		t.Fatalf("arsip setelah berangkat: err=%v status=%s", err, status())
	}
	if err := ubahStatusTx(ctx, tx, -1, StatusDraft, nil, "2026-11-25"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("id tak ada: err=%v, mau ErrNotFound", err)
	}
}
