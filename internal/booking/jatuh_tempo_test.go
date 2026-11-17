package booking

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// testTx membungkus transaksi uji beserta jadwal fixture-nya.
type testTx struct {
	*sql.Tx
	jadwal int64
	brand  int64
}

func (tx testTx) jamaah(t *testing.T, nama string) int64 {
	t.Helper()
	return testdb.NewJamaah(t, tx.Tx, tx.brand, testdb.JamaahOpts{Nama: nama})
}

// bookingDPUji: booking dp senilai total dengan tenggat lunas due dan pembayaran terverifikasi sebesar bayar.
func bookingDPUji(t *testing.T, due string, total, bayar float64) (testTx, int64) {
	t.Helper()
	_, sqlTx := testdb.Tx(t)
	jadwal, brand := testdb.Schedule(t, sqlTx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	tx := testTx{sqlTx, jadwal, brand}
	id, _ := testdb.NewBookingPax(t, sqlTx, jadwal, "dp", total, testdb.Pax{JamaahID: tx.jamaah(t, "Jatuh Tempo")})
	tambahCheckout(t, tx, id, due, "b")
	if bayar > 0 {
		testdb.AddPayment(t, sqlTx, id, bayar, "confirmed")
	}
	return tx, id
}

func tambahCheckout(t *testing.T, tx testTx, id int64, due, huruf string) {
	t.Helper()
	testdb.Exec(t, tx.Tx, `INSERT INTO booking_checkout (booking_id, dp_per_pax, due_at, invoice_token, terms_version)
		VALUES (?, 5000000, ?, REPEAT(?, 64), 'uji')`, id, due, huruf)
}

func TestBatalkanJatuhTempo(t *testing.T) {
	ctx := context.Background()
	hari := "2026-11-22"

	t.Run("lewat tenggat dan belum lunas dibatalkan, seat dilepas, pembayaran tetap", func(t *testing.T) {
		tx, id := bookingDPUji(t, "2026-11-21 00:00:00", 30000000, 5000000)
		// Fixture melebihkan kuota; pasang angka tetap supaya kenaikan seat terukur.
		testdb.Exec(t, tx.Tx, `UPDATE schedules SET seat_total = 50, seat_sisa = 10 WHERE id=?`, tx.jadwal)
		seatSebelum := 10
		ok, err := batalkanJatuhTempoTx(ctx, tx.Tx, id, hari)
		if err != nil || !ok {
			t.Fatalf("ok=%v err=%v", ok, err)
		}
		if s := testdb.Status(t, tx.Tx, id); s != "batal" {
			t.Errorf("status = %s, mau batal", s)
		}
		var blok bool
		var pax, payments int
		_ = tx.QueryRowContext(ctx, `SELECT is_seat_blocked FROM bookings WHERE id=?`, id).Scan(&blok)
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM booking_pax WHERE booking_id=? AND pax_status='aktif'`, id).Scan(&pax)
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments WHERE booking_id=? AND status='confirmed'`, id).Scan(&payments)
		if blok || pax != 0 {
			t.Errorf("seat masih tertahan (%v) atau pax aktif (%d)", blok, pax)
		}
		if payments != 1 {
			t.Errorf("pembayaran terverifikasi = %d, mau 1 (urusan keuangan manual, tidak disentuh)", payments)
		}
		var seatSesudah int
		_ = tx.QueryRowContext(ctx, `SELECT seat_sisa FROM schedules WHERE id=?`, tx.jadwal).Scan(&seatSesudah)
		if seatSesudah != seatSebelum+1 {
			t.Errorf("seat_sisa %d -> %d, mau naik 1", seatSebelum, seatSesudah)
		}
	})

	t.Run("hari tenggat masih termasuk", func(t *testing.T) {
		tx, id := bookingDPUji(t, "2026-11-22 00:00:00", 30000000, 5000000)
		if ok, err := batalkanJatuhTempoTx(ctx, tx.Tx, id, hari); err != nil || ok {
			t.Fatalf("ok=%v err=%v, mau dilewati", ok, err)
		}
		if s := testdb.Status(t, tx.Tx, id); s != "dp" {
			t.Errorf("status = %s, mau dp", s)
		}
	})

	t.Run("pembayaran menunggu verifikasi yang menutup tagihan: dilewati", func(t *testing.T) {
		tx, id := bookingDPUji(t, "2026-11-21 00:00:00", 30000000, 5000000)
		testdb.AddPayment(t, tx.Tx, id, 25000000, "pending")
		if ok, err := batalkanJatuhTempoTx(ctx, tx.Tx, id, hari); err != nil || ok {
			t.Fatalf("ok=%v err=%v, mau dilewati", ok, err)
		}
		if s := testdb.Status(t, tx.Tx, id); s != "dp" {
			t.Errorf("status = %s, mau dp", s)
		}
	})

	t.Run("pembayaran menunggu yang belum menutup tagihan: tetap dibatalkan", func(t *testing.T) {
		tx, id := bookingDPUji(t, "2026-11-21 00:00:00", 30000000, 5000000)
		testdb.AddPayment(t, tx.Tx, id, 2000000, "pending")
		if ok, err := batalkanJatuhTempoTx(ctx, tx.Tx, id, hari); err != nil || !ok {
			t.Fatalf("ok=%v err=%v, mau dibatalkan", ok, err)
		}
	})

	t.Run("booking yang pernah lunas tidak dibatalkan otomatis", func(t *testing.T) {
		tx, id := bookingDPUji(t, "2026-11-21 00:00:00", 30000000, 5000000)
		// Pernah lunas, lalu turun ke dp (penyesuaian harga setelah H-30): admin yang memutuskan.
		testdb.Exec(t, tx.Tx, `UPDATE bookings SET pertama_lunas_at = NOW() WHERE id=?`, id)
		ids, err := kandidatJatuhTempo(ctx, tx.Tx, hari)
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range ids {
			if v == id {
				t.Fatal("booking yang pernah lunas ikut jadi kandidat batal")
			}
		}
		if ok, err := batalkanJatuhTempoTx(ctx, tx.Tx, id, hari); err != nil || ok {
			t.Fatalf("ok=%v err=%v, mau dilewati", ok, err)
		}
		if s := testdb.Status(t, tx.Tx, id); s != "dp" {
			t.Errorf("status = %s, mau dp", s)
		}
	})

	t.Run("pencarian kandidat hanya booking dp lewat tenggat", func(t *testing.T) {
		tx, dp := bookingDPUji(t, "2026-11-21 00:00:00", 30000000, 5000000)
		lunas, _ := testdb.NewBookingPax(t, tx.Tx, tx.jadwal, "lunas", 10000000, testdb.Pax{JamaahID: tx.jamaah(t, "Lunas")})
		tambahCheckout(t, tx, lunas, "2026-11-01 00:00:00", "c")
		baru, _ := testdb.NewBookingPax(t, tx.Tx, tx.jadwal, "baru", 10000000, testdb.Pax{JamaahID: tx.jamaah(t, "Baru")})
		tambahCheckout(t, tx, baru, "2026-11-01 00:00:00", "d")
		ids, err := kandidatJatuhTempo(ctx, tx.Tx, hari)
		if err != nil {
			t.Fatal(err)
		}
		ada := map[int64]bool{}
		for _, v := range ids {
			ada[v] = true
		}
		if !ada[dp] || ada[lunas] || ada[baru] {
			t.Errorf("kandidat dp=%v lunas=%v baru=%v, mau true/false/false", ada[dp], ada[lunas], ada[baru])
		}
	})
}
