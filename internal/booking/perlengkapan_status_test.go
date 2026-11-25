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

// T-06: membatalkan seluruh booking memperlakukan perlengkapan pax aktif sama dengan batal per pax:
// log dihapus dan kolom perlengkapan direset; stok brand kembali hanya bila booking belum dibayar.
func TestBatalBookingMengembalikanPerlengkapan(t *testing.T) {
	for _, tc := range []struct {
		status      string
		stokKembali bool
	}{{"baru", true}, {"dp", false}} {
		t.Run(tc.status, func(t *testing.T) {
			_, tx := testdb.Tx(t)
			ctx := context.Background()
			b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: tc.status, TotalHarga: 1000, Blocked: true})

			var itemID, brandID int64
			if err := tx.QueryRowContext(ctx, `SELECT id FROM perlengkapan_items ORDER BY id LIMIT 1`).Scan(&itemID); err != nil {
				t.Skipf("tidak ada perlengkapan_items di database uji: %v", err)
			}
			if err := tx.QueryRowContext(ctx, `SELECT brand_id FROM schedules WHERE id=?`, b.ScheduleID).Scan(&brandID); err != nil {
				t.Fatal(err)
			}
			testdb.Exec(t, tx, `INSERT INTO perlengkapan_stok (brand_id, perlengkapan_item_id, stok_tersedia) VALUES (?,?,5)
				ON DUPLICATE KEY UPDATE stok_tersedia = 5`, brandID, itemID)
			testdb.Exec(t, tx, `UPDATE booking_pax SET perlengkapan_status='sudah_diberikan', perlengkapan_diserahkan_nama='Petugas Uji' WHERE id=?`, b.PaxID)
			testdb.Exec(t, tx, `INSERT INTO booking_pax_perlengkapan_logs (booking_pax_id, perlengkapan_item_id, qty) VALUES (?,?,2)`, b.PaxID, itemID)

			if err := updateBookingStatusTx(ctx, tx, b.ID, "batal", nil); err != nil {
				t.Fatalf("batal: %v", err)
			}

			var stok, logs, retur int
			var paxStatus, perlStatus string
			var diserahkanNama *string
			if err := tx.QueryRowContext(ctx, `SELECT stok_tersedia FROM perlengkapan_stok WHERE brand_id=? AND perlengkapan_item_id=?`, brandID, itemID).Scan(&stok); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM booking_pax_perlengkapan_logs WHERE booking_pax_id=?`, b.PaxID).Scan(&logs); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM perlengkapan_mutasi WHERE jenis='retur' AND referensi_tipe='booking_pax' AND referensi_id=?`, b.PaxID).Scan(&retur); err != nil {
				t.Fatal(err)
			}
			if err := tx.QueryRowContext(ctx, `SELECT pax_status, perlengkapan_status, perlengkapan_diserahkan_nama FROM booking_pax WHERE id=?`, b.PaxID).Scan(&paxStatus, &perlStatus, &diserahkanNama); err != nil {
				t.Fatal(err)
			}

			wantStok, wantRetur := 5, 0
			if tc.stokKembali {
				wantStok, wantRetur = 7, 1
			}
			if stok != wantStok || retur != wantRetur {
				t.Errorf("stok=%d retur=%d, mau stok=%d retur=%d", stok, retur, wantStok, wantRetur)
			}
			if logs != 0 {
				t.Errorf("log perlengkapan tersisa %d baris", logs)
			}
			if paxStatus != "batal" || perlStatus != "belum_diberikan" || diserahkanNama != nil {
				t.Errorf("pax_status=%q perlengkapan_status=%q diserahkan_nama=%v", paxStatus, perlStatus, diserahkanNama)
			}
		})
	}
}
