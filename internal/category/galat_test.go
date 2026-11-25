package category

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Kategori yang masih dipakai paket ditolak dengan 409 lewat sentinel ErrDipakai, bukan pencocokan teks.
func TestErrDipakaiJadi409(t *testing.T) {
	w := httptest.NewRecorder()
	handleRepoError(w, ErrDipakai)
	var body map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &body)
	if w.Code != http.StatusConflict || body["error"] != "Tidak bisa dihapus, masih dipakai oleh paket umroh." {
		t.Fatalf("dapat %d %q", w.Code, w.Body.String())
	}
}
