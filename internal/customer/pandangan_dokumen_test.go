package customer

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestPandanganDokumenHanyaStatusTanpaIsiBerkas(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status, rejection_reason) VALUES (?, 'ktp', '/uploads/RAHASIA-ktp-ahmad.jpg', 'rejected', 'AlasanRahasia')`, s.ahmad)
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status) VALUES (?, 'paspor', '/uploads/RAHASIA-paspor-istri.jpg', 'approved')`, s.istri)

	jid, nama, tipe, ok, err := bookingPeran(ctx, tx, s.ahmadAkun, s.booking)
	if err != nil || !ok || jid != s.ahmad {
		t.Fatalf("bookingPeran: %d %v %v", jid, ok, err)
	}
	items, err := muatDokumenPeran(ctx, tx, jid, tipe)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(map[string]any{"jamaah": dokumenPeran{JamaahID: jid, Nama: nama, PaxType: tipe, Dokumen: items}})
	body := string(b)
	for _, terlarang := range []string{"RAHASIA", "file_url", "uploads", "AlasanRahasia", "IstriRahasia", s.kodeBooking, "54321", "7777"} {
		if strings.Contains(body, terlarang) {
			t.Errorf("body memuat %q: %s", terlarang, body)
		}
	}
	st := map[string]string{}
	for _, d := range items {
		st[d.Jenis] = d.Status
		if d.RejectionReason != nil {
			t.Errorf("alasan penolakan bocor: %s", body)
		}
	}
	if st["ktp"] != "rejected" || st["paspor"] != "belum_upload" {
		t.Errorf("status salah (paspor istri tidak boleh terbaca): %v", st)
	}
}

func TestPandanganDokumenSuratTanpaUnduhDanDataLain(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()
	testdb.Exec(t, tx, `INSERT INTO surat_jamaah (booking_id, jamaah_id, jenis, nomor_urut, nomor, terbit_at, unduh_terakhir_at) VALUES (?, ?, 'keterangan_umroh', 1, '001/TES/I/2099', NOW(), NOW())`, s.booking, s.ahmad)
	testdb.Exec(t, tx, `INSERT INTO surat_jamaah (booking_id, jamaah_id, jenis, nomor_urut, nomor, terbit_at, unduh_terakhir_at) VALUES (?, ?, 'keterangan_umroh', 2, '002/ISTRI/I/2099', NOW(), NOW())`, s.booking, s.istri)
	items, err := muatSuratPeran(ctx, tx, s.ahmad, s.booking)
	if err != nil || len(items) != 1 || items[0].Nomor != "001/TES/I/2099" {
		t.Fatalf("surat: %+v %v", items, err)
	}
	b, _ := json.Marshal(map[string]any{"surat": items})
	for _, terlarang := range []string{"ISTRI", s.kodeBooking, "tempat", "lahir", "unduh", "url"} {
		if strings.Contains(string(b), terlarang) {
			t.Errorf("body memuat %q: %s", terlarang, b)
		}
	}
}

func TestPandanganDokumenGerbangPeran(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	ctx := context.Background()
	lain := testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "AkunBiasa"})
	// akun tak tertaut dan pemilik (bukan peran): bukan kartu peran.
	for _, a := range []int64{lain, s.ibrahim} {
		if _, _, _, ok, err := bookingPeran(ctx, tx, a, s.booking); err != nil || ok {
			t.Errorf("akun %d tak boleh lolos: %v %v", a, ok, err)
		}
	}
	// booking lain yang tidak memuat jamaah tertaut.
	b2, _ := testdb.NewBookingPax(t, tx, s.jadwal, "dp", 1000000, testdb.Pax{JamaahID: s.istri})
	if _, _, _, ok, _ := bookingPeran(ctx, tx, s.ahmadAkun, b2); ok {
		t.Error("booking tanpa jamaah tertaut harus ditolak")
	}
	// batal/draft menghilang.
	for _, st := range []string{"batal", "draft"} {
		testdb.Exec(t, tx, `UPDATE bookings SET status=? WHERE id=?`, st, s.booking)
		if _, _, _, ok, _ := bookingPeran(ctx, tx, s.ahmadAkun, s.booking); ok {
			t.Errorf("status %s harus ditolak", st)
		}
	}
	testdb.Exec(t, tx, `UPDATE bookings SET status='dp' WHERE id=?`, s.booking)
	if _, _, _, ok, _ := bookingPeran(ctx, tx, s.ahmadAkun, s.booking); !ok {
		t.Fatal("dp harus lolos")
	}
	// Lepas Tautan langsung menutup akses.
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=NULL WHERE id=?`, s.ahmadAkun)
	if _, _, _, ok, _ := bookingPeran(ctx, tx, s.ahmadAkun, s.booking); ok {
		t.Error("setelah lepas tautan harus ditolak")
	}
}

// Dokumen/KirimDokumen/SuratDaftar/SuratUnduh memakai owned(): peran jamaah tetap ditolak.
func TestPandanganDokumenHandlerLamaTetapPemilikSaja(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	if _, _, err := ownedQ(context.Background(), tx, s.ahmadAkun, s.booking); err == nil {
		t.Fatal("Ahmad tidak boleh lolos owned (Dokumen, KirimDokumen, Surat*)")
	}
}

func panggilPeran(t *testing.T, h func(dbq, http.ResponseWriter, *http.Request), q dbq, akun, booking int64) (int, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rc := chi.NewRouteContext()
	rc.URLParams.Add("id", fmt.Sprint(booking))
	ctx := context.WithValue(req.Context(), chi.RouteCtxKey, rc)
	ctx = context.WithValue(ctx, accountKey, &Account{ID: akun})
	rec := httptest.NewRecorder()
	h(q, rec, req.WithContext(ctx))
	return rec.Code, rec.Body.String()
}

func TestPandanganDokumenHandlerHTTP(t *testing.T) {
	_, tx := testdb.Tx(t)
	s := siapkanPeran(t, tx)
	testdb.Exec(t, tx, `INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status) VALUES (?, 'ktp', '/uploads/RAHASIA-ktp.jpg', 'submitted')`, s.ahmad)
	testdb.Exec(t, tx, `INSERT INTO surat_jamaah (booking_id, jamaah_id, jenis, nomor_urut, nomor, terbit_at, unduh_terakhir_at) VALUES (?, ?, 'keterangan_umroh', 1, '001/TES/I/2099', NOW(), NOW())`, s.booking, s.ahmad)

	code, body := panggilPeran(t, dokumenPeranHandler, tx, s.ahmadAkun, s.booking)
	if code != 200 || !strings.Contains(body, `"jamaah_id":`) || !strings.Contains(body, `"status":"submitted"`) || !strings.Contains(body, `"rejection_reason":null`) {
		t.Fatalf("dokumen: %d %s", code, body)
	}
	for _, terlarang := range []string{"RAHASIA", "file_url", "IstriRahasia", s.kodeBooking, "54321", "7777"} {
		if strings.Contains(body, terlarang) {
			t.Errorf("body dokumen memuat %q: %s", terlarang, body)
		}
	}
	code, body = panggilPeran(t, suratPeranHandler, tx, s.ahmadAkun, s.booking)
	if code != 200 || !strings.Contains(body, `"nomor":"001/TES/I/2099"`) || !strings.Contains(body, `"slug":"keterangan-umroh"`) {
		t.Fatalf("surat: %d %s", code, body)
	}
	for _, terlarang := range []string{"IstriRahasia", s.kodeBooking, "lahir", "url"} {
		if strings.Contains(body, terlarang) {
			t.Errorf("body surat memuat %q: %s", terlarang, body)
		}
	}

	lain := testdb.NewAkun(t, tx, s.brand, testdb.AkunOpts{Nama: "AkunBiasa"})
	for _, h := range []func(dbq, http.ResponseWriter, *http.Request){dokumenPeranHandler, suratPeranHandler} {
		for _, a := range []int64{lain, s.ibrahim} {
			code, body := panggilPeran(t, h, tx, a, s.booking)
			if code != 404 || !strings.Contains(body, "Perjalanan tidak ditemukan.") {
				t.Errorf("akun %d harus 404: %d %s", a, code, body)
			}
		}
	}
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=NULL WHERE id=?`, s.ahmadAkun)
	if code, _ := panggilPeran(t, dokumenPeranHandler, tx, s.ahmadAkun, s.booking); code != 404 {
		t.Errorf("setelah Lepas Tautan harus 404: %d", code)
	}
}
