package payment

import (
	"context"
	"errors"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestValidasiAlasanVoid(t *testing.T) {
	cases := []struct{ nama, isi, hasil, pesan string }{
		{"valid", "  salah nominal  ", "salah nominal", ""},
		{"kosong", "   ", "", "wajib diisi"},
		{"pendek", "ya", "", "wajib diisi"},
		{"panjang", strings.Repeat("a", 501), "", "maksimal 500"},
	}
	for _, c := range cases {
		hasil, pesan := validasiAlasanVoid(c.isi)
		if (c.pesan == "" && (pesan != "" || hasil != c.hasil)) || (c.pesan != "" && !strings.Contains(pesan, c.pesan)) {
			t.Errorf("%s: got (%q, %q)", c.nama, hasil, pesan)
		}
	}
}

// T-03: void pembayaran terkonfirmasi menurunkan status booking, mempertahankan kursi, dan tidak bisa
// dipakai untuk pembayaran yang belum dikonfirmasi.
func TestVoidPembayaran(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 30_000_000, MinimalDP: 5_000_000, Blocked: true})
	dp := testdb.AddPayment(t, tx, b.ID, 5_000_000, "confirmed")
	pelunasan := testdb.AddPayment(t, tx, b.ID, 25_000_000, "confirmed")
	pending := testdb.AddPayment(t, tx, b.ID, 1_000_000, "pending")
	if err := SyncBookingStatusTx(ctx, tx, b.ID); err != nil {
		t.Fatal(err)
	}
	if got := testdb.Status(t, tx, b.ID); got != "lunas" {
		t.Fatalf("status awal = %s, want lunas", got)
	}

	if err := voidTx(ctx, tx, pending, "salah catat", 0); !errors.Is(err, ErrNotVoidable) {
		t.Fatalf("void pending: err = %v, want ErrNotVoidable", err)
	}
	if err := voidTx(ctx, tx, -1, "salah catat", 0); !errors.Is(err, ErrNotFound) {
		t.Fatalf("void id tidak ada: err = %v, want ErrNotFound", err)
	}

	if err := voidTx(ctx, tx, pelunasan, "salah nominal", 0); err != nil {
		t.Fatal(err)
	}
	var status, alasan string
	var voidedAt any
	if err := tx.QueryRowContext(ctx, `SELECT status, COALESCE(void_reason,''), voided_at FROM payments WHERE id=?`, pelunasan).Scan(&status, &alasan, &voidedAt); err != nil {
		t.Fatal(err)
	}
	if status != "void" || alasan != "salah nominal" || voidedAt == nil {
		t.Fatalf("pembayaran = %s %q %v", status, alasan, voidedAt)
	}
	if got := testdb.Status(t, tx, b.ID); got != "dp" {
		t.Fatalf("status setelah void pelunasan = %s, want dp", got)
	}
	// Permintaan ulang tidak mengubah apa pun.
	if err := voidTx(ctx, tx, pelunasan, "alasan lain", 0); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(void_reason,'') FROM payments WHERE id=?`, pelunasan).Scan(&alasan); err != nil {
		t.Fatal(err)
	}
	if alasan != "salah nominal" {
		t.Fatalf("alasan tertimpa oleh permintaan ulang: %q", alasan)
	}

	// Void pembayaran terakhir: booking kembali baru, kursi tetap terkunci tanpa batas waktu.
	if err := voidTx(ctx, tx, dp, "salah booking", 0); err != nil {
		t.Fatal(err)
	}
	var blocked bool
	var expiry any
	if err := tx.QueryRowContext(ctx, `SELECT status, is_seat_blocked, seat_hold_expires_at FROM bookings WHERE id=?`, b.ID).Scan(&status, &blocked, &expiry); err != nil {
		t.Fatal(err)
	}
	if status != "baru" || !blocked || expiry != nil {
		t.Fatalf("booking = %s blocked=%v expiry=%v, want baru/true/nil", status, blocked, expiry)
	}
}

// T-03: dana yang sudah dikembalikan tidak boleh melebihi pembayaran terkonfirmasi yang tersisa.
func TestVoidDitolakBilaMelebihiRefund(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "batal", TotalHarga: 30_000_000, MinimalDP: 5_000_000})
	bayar := testdb.AddPayment(t, tx, b.ID, 5_000_000, "confirmed")
	testdb.Exec(t, tx, `INSERT INTO booking_refunds (booking_id, jumlah, tanggal) VALUES (?, 3000000, CURDATE())`, b.ID)
	if err := voidTx(ctx, tx, bayar, "salah catat", 0); !errors.Is(err, ErrVoidBelowRefund) {
		t.Fatalf("err = %v, want ErrVoidBelowRefund", err)
	}
}
