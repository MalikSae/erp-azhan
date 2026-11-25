package dokumen

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

// Gate validasi status berjalan sebelum menyentuh database, jadi cukup Handler tanpa repo.
func TestUpdateDokumenStatusValidasi(t *testing.T) {
	cases := []struct {
		id, body, want string
	}{
		{"abc", `{}`, `{"error":"ID tidak valid."}`},
		{"5", `{"status":"lunas","version":1}`, `{"error":"Status tidak valid."}`},
		{"5", `{"status":"submitted"}`, `{"error":"Versi dokumen wajib diisi."}`},
		{"5", `{"status":"rejected","version":1}`, `{"error":"Alasan penolakan wajib diisi, maksimal 500 karakter."}`},
	}
	h := NewHandler(nil)
	for _, c := range cases {
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", c.id)
		req := httptest.NewRequest(http.MethodPut, "/api/admin/dokumen/"+c.id+"/status", strings.NewReader(c.body))
		req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
		rec := httptest.NewRecorder()
		h.UpdateDokumenStatus(rec, req)
		if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusBadRequest || got != c.want {
			t.Errorf("id=%s body=%s: %d %s, ingin 400 %s", c.id, c.body, rec.Code, got, c.want)
		}
		t.Logf("id=%s body=%s -> %d %s", c.id, c.body, rec.Code, strings.TrimSpace(rec.Body.String()))
	}
}
