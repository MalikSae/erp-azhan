package perlengkapan

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Review Inventory 2026-11-25: gate validasi berhenti di isian pertama yang salah, dan koreksi stok
// lewat PUT /perlengkapan-stok wajib beralasan seperti POST /perlengkapan-mutasi/koreksi.
func TestGateValidasiStok(t *testing.T) {
	h := NewHandler(nil) // semua kasus berhenti sebelum repository dipanggil
	r := chi.NewRouter()
	r.Post("/pembelian", h.Pembelian)
	r.Post("/koreksi", h.Koreksi)
	r.Put("/stok/{item_id}", h.UpdateStok)

	cases := []struct {
		method, path, body, mau string
	}{
		{http.MethodPost, "/pembelian", `{"brand_id":1,"qty":1}`, "Item wajib dipilih."},
		{http.MethodPost, "/pembelian", `{"perlengkapan_item_id":1,"qty":1}`, "Brand wajib dipilih."},
		{http.MethodPost, "/pembelian", `{"perlengkapan_item_id":1,"brand_id":1,"qty":0}`, "Jumlah minimal 1."},
		{http.MethodPost, "/koreksi", `{"brand_id":1,"stok_baru":3,"catatan":"x"}`, "Item wajib dipilih."},
		{http.MethodPost, "/koreksi", `{"perlengkapan_item_id":1,"brand_id":1,"stok_baru":-1}`, "Stok wajib bernilai 0 atau lebih."},
		{http.MethodPost, "/koreksi", `{"perlengkapan_item_id":1,"brand_id":1,"stok_baru":3}`, "Alasan koreksi stok wajib diisi."},
		{http.MethodPut, "/stok/1?brand_id=1", `{"stok_tersedia":3}`, "Alasan koreksi stok wajib diisi."},
		{http.MethodPut, "/stok/1?brand_id=1", `{"stok_tersedia":3,"catatan":"  "}`, "Alasan koreksi stok wajib diisi."},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, strings.NewReader(c.body)))
		var res map[string]string
		_ = json.Unmarshal(rec.Body.Bytes(), &res)
		if rec.Code != http.StatusBadRequest || res["error"] != c.mau {
			t.Errorf("%s %s %s: %d %q, mau 400 %q", c.method, c.path, c.body, rec.Code, res["error"], c.mau)
		}
	}
}

func TestPesanInputTidakValid(t *testing.T) {
	err := fmt.Errorf("%w: Jumlah minimal 1.", ErrInvalidInput)
	if got := pesanInputTidakValid(err); got != "Jumlah minimal 1." {
		t.Errorf("pesan = %q, mau %q", got, "Jumlah minimal 1.")
	}
}
