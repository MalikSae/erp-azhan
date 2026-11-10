package booking

import (
	"context"
	"fmt"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// T-06: perlengkapan hanya boleh diserahkan pada booking yang sudah resmi dan belum batal.
func TestTolakPerlengkapanUntukStatus(t *testing.T) {
	for _, status := range []string{"draft", "batal"} {
		if tolakPerlengkapanUntukStatus(status) == "" {
			t.Errorf("status %q harus ditolak", status)
		}
	}
	for _, status := range []string{"baru", "dp", "lunas"} {
		if msg := tolakPerlengkapanUntukStatus(status); msg != "" {
			t.Errorf("status %q harus diizinkan, dapat %q", status, msg)
		}
	}
}

// T-06: membatalkan booking yang kursinya sudah dilepas tetap membatalkan pax, tanpa mengembalikan kursi lagi.
func TestBatalBookingTanpaBlokirMembatalkanPax(t *testing.T) {
	for _, blocked := range []bool{false, true} {
		// Satu transaksi per kasus: fixture booking memakai kode berbasis waktu yang bisa bertabrakan
		// bila dibuat dua kali dalam transaksi yang sama.
		t.Run(fmt.Sprintf("blocked=%v", blocked), func(t *testing.T) {
			_, tx := testdb.Tx(t)
			ctx := context.Background()
			b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 30_000_000, MinimalDP: 5_000_000, Blocked: blocked})
			// Fixture menambah kuota melewati seat_total, sedangkan pengembalian kursi dibatasi seat_total.
			testdb.Exec(t, tx, `UPDATE schedules SET seat_sisa = seat_total - 1 WHERE id=?`, b.ScheduleID)
			var sebelum, sesudah int
			if err := tx.QueryRowContext(ctx, `SELECT seat_sisa FROM schedules WHERE id=?`, b.ScheduleID).Scan(&sebelum); err != nil {
				t.Fatal(err)
			}
			if err := updateBookingStatusTx(ctx, tx, b.ID, "batal", nil); err != nil {
				t.Fatalf("batal: %v", err)
			}
			var paxStatus string
			if err := tx.QueryRowContext(ctx, `SELECT pax_status FROM booking_pax WHERE id=?`, b.PaxID).Scan(&paxStatus); err != nil {
				t.Fatal(err)
			}
			if paxStatus != "batal" {
				t.Errorf("pax_status = %q, want batal", paxStatus)
			}
			if err := tx.QueryRowContext(ctx, `SELECT seat_sisa FROM schedules WHERE id=?`, b.ScheduleID).Scan(&sesudah); err != nil {
				t.Fatal(err)
			}
			want := sebelum
			if blocked {
				want = sebelum + 1
			}
			if sesudah != want {
				t.Errorf("seat_sisa %d -> %d, want %d", sebelum, sesudah, want)
			}
		})
	}
}
