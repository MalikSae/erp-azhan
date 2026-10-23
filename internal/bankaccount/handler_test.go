package bankaccount

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Butir 7a: endpoint publik wajib brand, hanya rekening aktif, dan hanya field untuk pelanggan.
func TestListPublicWajibBrandDanFieldTerbatas(t *testing.T) {
	h := NewHandler(nil)
	for _, q := range []string{"", "?brand_id=", "?brand_id=abc", "?brand_id=0", "?active=false"} {
		rec := httptest.NewRecorder()
		h.ListPublic(rec, httptest.NewRequest(http.MethodGet, "/api/public/bank-accounts"+q, nil))
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%q: got %d, want 400", q, rec.Code)
		}
	}

	db := testdb.DB(t) // hanya membaca
	h = NewHandler(NewRepository(db))
	var brandID int64
	var aktif int
	if err := db.QueryRow(`SELECT brand_id, SUM(is_active) FROM bank_accounts GROUP BY brand_id ORDER BY COUNT(*) DESC LIMIT 1`).Scan(&brandID, &aktif); err != nil {
		t.Skipf("tidak ada data rekening: %v", err)
	}
	for _, param := range []string{"brand_id", "brand"} {
		rec := httptest.NewRecorder()
		h.ListPublic(rec, httptest.NewRequest(http.MethodGet, "/api/public/bank-accounts?"+param+"="+strconv.FormatInt(brandID, 10)+"&active=false", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", param, rec.Code, rec.Body.String())
		}
		var items []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != aktif {
			t.Fatalf("%s: %d rekening, want %d rekening aktif saja", param, len(items), aktif)
		}
		for _, it := range items {
			for k := range it {
				if !strings.Contains(" id bank_name logo_url account_number account_holder ", " "+k+" ") {
					t.Fatalf("field internal bocor: %q", k)
				}
			}
		}
	}
}
