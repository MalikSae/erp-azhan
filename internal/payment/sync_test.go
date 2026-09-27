package payment

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// countLunasHook mengganti komisi.OnBookingLunas selama test dan menghitung
// berapa kali hook dipanggil per booking.
func countLunasHook(t *testing.T) map[int64]int {
	t.Helper()
	calls := map[int64]int{}
	orig := komisi.OnBookingLunas
	komisi.OnBookingLunas = func(ctx context.Context, tx *sql.Tx, bookingID int64) error {
		calls[bookingID]++
		return nil
	}
	t.Cleanup(func() { komisi.OnBookingLunas = orig })
	return calls
}

func TestSyncBookingStatusTransitions(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	calls := countLunasHook(t)

	sync := func(bookingID int64) {
		t.Helper()
		if err := repo.syncBookingStatusTx(ctx, tx, bookingID); err != nil {
			t.Fatalf("syncBookingStatusTx: %v", err)
		}
	}

	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, MinimalDP: 300})

	// baru, bayar di bawah minimal DP -> tetap baru.
	testdb.AddPayment(t, tx, b.ID, 100, "confirmed")
	sync(b.ID)
	if got := testdb.Status(t, tx, b.ID); got != "baru" {
		t.Fatalf("di bawah minimal DP: status=%s, want baru", got)
	}

	// Akumulasi mencapai minimal DP -> dp.
	testdb.AddPayment(t, tx, b.ID, 250, "confirmed")
	sync(b.ID)
	if got := testdb.Status(t, tx, b.ID); got != "dp" {
		t.Fatalf("mencapai minimal DP: status=%s, want dp", got)
	}
	if calls[b.ID] != 0 {
		t.Fatalf("hook lunas terpanggil %d kali sebelum lunas", calls[b.ID])
	}

	// Lunas -> hook terpanggil sekali.
	last := testdb.AddPayment(t, tx, b.ID, 650, "confirmed")
	sync(b.ID)
	if got := testdb.Status(t, tx, b.ID); got != "lunas" {
		t.Fatalf("pembayaran penuh: status=%s, want lunas", got)
	}
	if calls[b.ID] != 1 {
		t.Fatalf("hook lunas terpanggil %d kali, want 1", calls[b.ID])
	}

	// Sync ulang tanpa perubahan -> tidak memanggil hook lagi.
	sync(b.ID)
	if calls[b.ID] != 1 {
		t.Fatalf("sync ulang: hook terpanggil %d kali, want 1", calls[b.ID])
	}

	// Pembayaran terakhir dibatalkan -> mundur ke dp.
	testdb.Exec(t, tx, `UPDATE payments SET status='rejected' WHERE id=?`, last)
	sync(b.ID)
	if got := testdb.Status(t, tx, b.ID); got != "dp" {
		t.Fatalf("pembayaran ditolak: status=%s, want dp", got)
	}

	// Lunas lagi -> hook terpanggil kedua kali untuk booking yang sama.
	// Inilah alasan implementasi komisi wajib idempoten (agen-azhan.md 7.4).
	testdb.AddPayment(t, tx, b.ID, 650, "confirmed")
	sync(b.ID)
	if got := testdb.Status(t, tx, b.ID); got != "lunas" {
		t.Fatalf("lunas kedua: status=%s, want lunas", got)
	}
	if calls[b.ID] != 2 {
		t.Fatalf("lunas kedua: hook terpanggil %d kali, want 2", calls[b.ID])
	}
}

// TestSyncLunasMencatatKomisi memakai hook komisi asli: pembayaran lunas yang
// dikonfirmasi harus menghasilkan komisi langsung untuk agen pemilik pax.
func TestSyncLunasMencatatKomisi(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	ctx := context.Background()
	langsung := 750_000.0

	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: &langsung, BerangkatTanggal: "2099-01-01"})
	agen := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Agen", StatusAgen: "aktif"})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: agen})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	if err := komisi.SnapshotNominal(ctx, tx, b); err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	testdb.AddPayment(t, tx, b, 1000, "confirmed")
	if err := repo.syncBookingStatusTx(ctx, tx, b); err != nil {
		t.Fatalf("syncBookingStatusTx: %v", err)
	}
	var penerima int64
	var jenis string
	var nominal float64
	if err := tx.QueryRow(`SELECT jamaah_penerima_id, jenis, nominal FROM transaksi_komisi WHERE booking_id=?`, b).
		Scan(&penerima, &jenis, &nominal); err != nil {
		t.Fatalf("komisi tidak tercatat: %v", err)
	}
	if penerima != agen || jenis != "langsung" || nominal != langsung {
		t.Fatalf("komisi = (%d, %s, %.0f), want (%d, langsung, 750000)", penerima, jenis, nominal, agen)
	}
}

func TestSyncBookingStatusIgnoresBatal(t *testing.T) {
	db, tx := testdb.Tx(t)
	repo := NewRepository(db)
	calls := countLunasHook(t)

	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "batal", TotalHarga: 1000, MinimalDP: 300})
	testdb.AddPayment(t, tx, b.ID, 1000, "confirmed")
	if err := repo.syncBookingStatusTx(context.Background(), tx, b.ID); err != nil {
		t.Fatalf("syncBookingStatusTx: %v", err)
	}
	if got := testdb.Status(t, tx, b.ID); got != "batal" {
		t.Fatalf("status=%s, want batal", got)
	}
	if calls[b.ID] != 0 {
		t.Fatalf("hook lunas terpanggil %d kali untuk booking batal", calls[b.ID])
	}
}
