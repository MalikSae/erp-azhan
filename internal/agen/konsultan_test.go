package agen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestCariKonsultan(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	aktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Konsultan Aktif", StatusAgen: "aktif"})
	nonaktif := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Konsultan Nonaktif", StatusAgen: "nonaktif"})
	bukanAgen := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Bukan Agen"})
	setKode(t, tx, aktif, "KSLAKT")
	setKode(t, tx, nonaktif, "KSLNON")
	setKode(t, tx, bukanAgen, "KSLBKN")
	testdb.Exec(t, tx, `UPDATE customer_accounts SET no_hp='0812-3456-7890' WHERE id=?`, aktif)

	v, err := CariKonsultan(ctx, tx, brand, " kslakt ")
	if err != nil {
		t.Fatal(err)
	}
	if v == nil || !strings.HasPrefix(v.NamaLengkap, "Konsultan Aktif") || v.WhatsApp != "6281234567890" {
		t.Fatalf("agen aktif = %+v, mau Konsultan Aktif / 6281234567890", v)
	}

	for _, c := range []struct {
		brand int64
		kode  string
	}{
		{brand, "KSLNON"},           // agen nonaktif
		{brand, "KSLBKN"},           // bukan agen
		{brand + 999_999, "KSLAKT"}, // brand lain
		{brand, "TDKADA"},
		{brand, ""},
		{brand, "KSLAK"},
		{0, "KSLAKT"},
	} {
		v, err := CariKonsultan(ctx, tx, c.brand, c.kode)
		if err != nil {
			t.Fatal(err)
		}
		if v != nil {
			t.Errorf("CariKonsultan(%d, %q) = %+v, mau nil", c.brand, c.kode, v)
		}
	}
}

func TestJatahKonsultanPerIP(t *testing.T) {
	h := NewHandler(nil)
	// Kode yang sama berulang dari satu IP tidak menghabiskan jatah.
	for i := 0; i < konsultanKodePerIPPerJam*3; i++ {
		if !h.ambilJatahKonsultan("10.0.0.1", 3, "ABC123") {
			t.Fatalf("permintaan ke-%d untuk kode yang sama ditolak", i+1)
		}
	}
	// Kode berbeda: berhenti di batas per jam (kode pertama sudah terpakai satu).
	diizinkan := 1
	for i := 0; i < konsultanKodePerIPPerJam+10; i++ {
		kode := "K" + string(rune('A'+i%26)) + string(rune('A'+i/26)) + "XYZ"
		if h.ambilJatahKonsultan("10.0.0.1", 3, kode) {
			diizinkan++
		}
	}
	if diizinkan != konsultanKodePerIPPerJam {
		t.Errorf("kode berbeda diizinkan = %d, mau %d", diizinkan, konsultanKodePerIPPerJam)
	}
	// Kode yang ditolak tidak ditandai: tetap ditolak, tidak lolos sebagai "sudah ditanyakan".
	if h.ambilJatahKonsultan("10.0.0.1", 3, "KZZXYZ") {
		t.Error("kode di atas batas lolos pada permintaan kedua")
	}
	if !h.ambilJatahKonsultan("10.0.0.2", 3, "KZZXYZ") {
		t.Error("IP lain ikut terbatas")
	}
}

// Kode tidak valid dijawab tanpa menyentuh database: 200 {"konsultan":null}, tanpa cache.
func TestKonsultanPublikKodeTidakValid(t *testing.T) {
	h := NewHandler(nil)
	for _, q := range []string{"", "?brand_id=1", "?brand_id=1&kode=AB-12C", "?brand_id=x&kode=AB12CD", "?kode=AB12CD"} {
		rec := httptest.NewRecorder()
		h.KonsultanPublik(rec, httptest.NewRequest(http.MethodGet, "/api/public/v1/agen/konsultan"+q, nil))
		if rec.Code != http.StatusOK || strings.TrimSpace(rec.Body.String()) != `{"konsultan":null}` {
			t.Errorf("%q -> %d %s", q, rec.Code, rec.Body.String())
		}
		if rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%q Cache-Control = %q", q, rec.Header().Get("Cache-Control"))
		}
	}
}
