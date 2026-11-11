package booking

import (
	"context"
	"errors"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
	"github.com/go-chi/chi/v5"
)

// Code review Booking, 11 November 2026 (A1): nominal diskon pecahan dan nama yang melebihi lebar kolom
// ditolak di pintu masuk, sebelum menyentuh data.
func TestAddBookingDiscountMenolakIsianTidakSah(t *testing.T) {
	// Handler tanpa repository: bila pemeriksaan isian terlewati, pemanggilan ini panik.
	h := &Handler{}
	r := chi.NewRouter()
	r.Post("/bookings/{id}/discounts", h.AddBookingDiscount)

	cases := map[string]string{
		"nominal pecahan":      `{"nama":"Promo","nominal":1000000.5}`,
		"nominal nol":          `{"nama":"Promo","nominal":0}`,
		"nama kosong":          `{"nama":"   ","nominal":1000}`,
		"nama terlalu panjang": `{"nama":"` + strings.Repeat("a", maksNamaDiskon+1) + `","nominal":1000}`,
	}
	for nama, body := range cases {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/bookings/1/discounts", strings.NewReader(body)))
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status %d, mau 400 (%s)", nama, rec.Code, rec.Body.String())
		}
	}
}

// Code review Booking, 11 November 2026 (R-04): pembatalan pax tersisa memakai helper yang melaporkan
// setiap kegagalan. Log perlengkapan pax terhapus; pax tanpa perlengkapan tidak ikut terdaftar.
func TestReturPerlengkapanPaxTx(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "dp", TotalHarga: 1000, Blocked: true})

	var itemID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM perlengkapan_items ORDER BY id LIMIT 1`).Scan(&itemID); err != nil {
		t.Skipf("tidak ada perlengkapan_items di database uji: %v", err)
	}

	if ids, err := paxAktifBerperlengkapanTx(ctx, tx, b.ID); err != nil || len(ids) != 0 {
		t.Fatalf("pax belum menerima perlengkapan: ids=%v err=%v", ids, err)
	}

	testdb.Exec(t, tx, `UPDATE booking_pax SET perlengkapan_status='sudah_diberikan' WHERE id=?`, b.PaxID)
	testdb.Exec(t, tx, `INSERT INTO booking_pax_perlengkapan_logs (booking_pax_id, perlengkapan_item_id, qty) VALUES (?,?,1)`, b.PaxID, itemID)

	ids, err := paxAktifBerperlengkapanTx(ctx, tx, b.ID)
	if err != nil || len(ids) != 1 || ids[0] != b.PaxID {
		t.Fatalf("ids=%v err=%v, mau [%d]", ids, err, b.PaxID)
	}
	// Booking sudah dibayar: log dihapus tanpa mengembalikan stok.
	if err := returPerlengkapanPaxTx(ctx, tx, 0, b.PaxID, false); err != nil {
		t.Fatal(err)
	}
	var sisa int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM booking_pax_perlengkapan_logs WHERE booking_pax_id=?`, b.PaxID).Scan(&sisa); err != nil {
		t.Fatal(err)
	}
	if sisa != 0 {
		t.Fatalf("log perlengkapan tersisa %d baris", sisa)
	}
	// Pax tanpa log: tetap berhasil, tidak ada yang dihapus.
	if err := returPerlengkapanPaxTx(ctx, tx, 0, b.PaxID, true); err != nil {
		t.Fatalf("pax tanpa log: %v", err)
	}

	testdb.Exec(t, tx, `UPDATE booking_pax SET pax_status='batal' WHERE id=?`, b.PaxID)
	if ids, err := paxAktifBerperlengkapanTx(ctx, tx, b.ID); err != nil || len(ids) != 0 {
		t.Fatalf("pax batal tidak boleh terdaftar: ids=%v err=%v", ids, err)
	}
}

// Keputusan 11 November 2026 (C3): seat terkonfirmasi hanya untuk booking yang sudah dibayar.
func TestSeatTerkonfirmasi(t *testing.T) {
	for status, mau := range map[string]bool{"draft": false, "baru": false, "dp": true, "lunas": true, "batal": false} {
		if got := seatTerkonfirmasi(status); got != mau {
			t.Errorf("seatTerkonfirmasi(%q) = %v, mau %v", status, got, mau)
		}
	}
}

// Keputusan 11 November 2026 (C2): perubahan yang menurunkan harga pax tidak boleh membuat diskon melewati
// batas 10%, kecuali pelakunya pemegang izin diskon besar.
func TestCekBatasDiskonSetelahHargaTurunTx(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 30_000_000, MinimalDP: 5_000_000, Blocked: true})
	testdb.Exec(t, tx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, 'Uji', 3000000)`, b.ID)

	awal, err := jumlahHargaPaxTx(ctx, tx, b.ID)
	if err != nil || awal != 30_000_000 {
		t.Fatalf("harga awal = %v, %v", awal, err)
	}
	// Diskon tepat 10% dan harga tidak berubah: lolos.
	if err := cekBatasDiskonSetelahHargaTurunTx(ctx, tx, b.ID, awal, false); err != nil {
		t.Fatalf("harga tetap: %v", err)
	}

	// Harga turun ke 20 juta (pindah kamar lebih murah): diskon 3 juta menjadi 15%.
	testdb.Exec(t, tx, `UPDATE booking_pax SET harga_pax = 20000000 WHERE id = ?`, b.PaxID)
	var biz *BusinessError
	if err := cekBatasDiskonSetelahHargaTurunTx(ctx, tx, b.ID, awal, false); !errors.As(err, &biz) {
		t.Fatalf("harga turun melewati batas: err = %v, mau BusinessError", err)
	}
	if err := cekBatasDiskonSetelahHargaTurunTx(ctx, tx, b.ID, awal, true); err != nil {
		t.Fatalf("pemegang izin diskon besar harus lolos: %v", err)
	}
	// Booking yang diskonnya sudah di atas batas tetap boleh diubah selama harga tidak turun.
	if err := cekBatasDiskonSetelahHargaTurunTx(ctx, tx, b.ID, 20_000_000, false); err != nil {
		t.Fatalf("harga tidak turun pada booking berdiskon besar: %v", err)
	}
	if err := cekBatasDiskonSetelahHargaTurunTx(ctx, tx, b.ID, 15_000_000, false); err != nil {
		t.Fatalf("harga naik: %v", err)
	}
	// Finalisasi memakai harga sebelum tak hingga: selalu diperiksa.
	if err := cekBatasDiskonSetelahHargaTurunTx(ctx, tx, b.ID, math.Inf(1), false); !errors.As(err, &biz) {
		t.Fatalf("finalisasi dengan diskon di atas batas: err = %v, mau BusinessError", err)
	}
	// Harga turun tetapi diskon masih dalam batas: lolos.
	testdb.Exec(t, tx, `UPDATE booking_discounts SET nominal = 2000000 WHERE booking_id = ?`, b.ID)
	if err := cekBatasDiskonSetelahHargaTurunTx(ctx, tx, b.ID, awal, false); err != nil {
		t.Fatalf("diskon 10%% dari harga baru: %v", err)
	}
}
