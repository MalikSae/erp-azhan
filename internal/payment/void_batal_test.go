package payment

import (
	"context"
	"database/sql"
	"testing"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// Code review area uang, 10 November 2026: booking lunas yang kemudian dibatalkan mempertahankan komisinya
// (K20). Bila pelunasannya lalu di-void, pembayaran itu dinyatakan tidak pernah ada, sehingga komisi ikut
// batal dengan sebab void. Tanpa void, komisi booking batal tetap aktif.
func TestVoidPadaBookingBatalMembatalkanKomisi(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	langsung := 750_000.0

	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: &langsung, BerangkatTanggal: "2099-01-01"})
	agen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Agen", StatusAgen: "aktif"})
	x := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "X", Direkrut: agen})
	b, _ := testdb.NewBookingPax(t, tx, sched, "dp", 1000, testdb.Pax{JamaahID: x})
	if err := komisi.SnapshotNominal(ctx, tx, b); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	pelunasan := testdb.AddPayment(t, tx, b, 1000, "confirmed")
	// Pembayaran kedua yang salah catat (ganda): setelah di-void, pelunasan masih menutup total harga.
	ganda := testdb.AddPayment(t, tx, b, 400, "confirmed")
	if err := SyncBookingStatusTx(ctx, tx, b); err != nil {
		t.Fatalf("sync: %v", err)
	}

	komisiAktif := func() (aktif int, sebab sql.NullString) {
		t.Helper()
		if err := tx.QueryRow(`SELECT COALESCE(SUM(dibatalkan_at IS NULL),0), MAX(sebab_batal) FROM transaksi_komisi WHERE booking_id=?`, b).
			Scan(&aktif, &sebab); err != nil {
			t.Fatal(err)
		}
		return aktif, sebab
	}
	if aktif, _ := komisiAktif(); aktif != 1 {
		t.Fatalf("komisi aktif setelah lunas = %d, want 1", aktif)
	}

	// Booking dibatalkan setelah lunas: komisi tetap (K20).
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, b)
	if aktif, _ := komisiAktif(); aktif != 1 {
		t.Fatalf("komisi aktif setelah booking batal = %d, want 1", aktif)
	}

	// Void pembayaran ganda: sisa terkonfirmasi (1000) masih menutup total harga, komisi tetap.
	if err := voidTx(ctx, tx, ganda, "tercatat dua kali", 0); err != nil {
		t.Fatalf("void ganda: %v", err)
	}
	if aktif, _ := komisiAktif(); aktif != 1 {
		t.Fatalf("komisi aktif setelah void pembayaran ganda = %d, want 1", aktif)
	}

	// Pelunasan di-void: komisi batal dengan sebab void, status booking tetap batal.
	if err := voidTx(ctx, tx, pelunasan, "transfer tidak masuk", 0); err != nil {
		t.Fatalf("void: %v", err)
	}
	aktif, sebab := komisiAktif()
	if aktif != 0 || sebab.String != komisi.SebabVoid {
		t.Fatalf("setelah void: komisi aktif = %d, sebab = %q; want 0, %q", aktif, sebab.String, komisi.SebabVoid)
	}
	if got := testdb.Status(t, tx, b); got != "batal" {
		t.Fatalf("status booking = %s, want batal", got)
	}
}
