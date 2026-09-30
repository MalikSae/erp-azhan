package media

import (
	"context"
	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/testdb"
	"github.com/go-chi/chi/v5"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestPrivateDocumentsAndUploadOwnership(t *testing.T) {
	db := testdb.Portal(t)
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("uploads", "dokumen-jamaah"), 0700); err != nil {
		t.Fatal(err)
	}
	name := "synthetic-document.pdf"
	raw := "/api/admin/media/dokumen-jamaah/" + name
	if err := os.WriteFile(filepath.Join("uploads", "dokumen-jamaah", name), []byte("%PDF-1.4 synthetic"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO jamaah VALUES(1,10,'a'),(2,20,'b')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO media_uploads(path,brand_id,jamaah_id) VALUES(?,10,1)`, raw); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dokumen_jamaah(id,jamaah_id,jenis,file_url,status) VALUES(1,1,'paspor',?,'submitted')`, raw); err != nil {
		t.Fatal(err)
	}
	if err := ValidatePortalUpload(context.Background(), db, raw, 1); err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct {
		path string
		id   int64
	}{{raw, 2}, {"https://external.invalid/file.pdf", 1}, {"/api/admin/media/dokumen-jamaah/../secret", 1}, {"/api/admin/media/dokumen-jamaah/missing.pdf", 1}} {
		if ValidatePortalUpload(context.Background(), db, item.path, item.id) == nil {
			t.Fatal("invalid ownership accepted")
		}
	}
	h := NewHandler(db)
	router := chi.NewRouter()
	router.Get("/dokumen/{id}/file", h.ServeDocument)
	for _, item := range []struct {
		id   int64
		want int
	}{{1, 200}, {2, 404}} {
		req := httptest.NewRequest("GET", "/dokumen/1/file", nil)
		req = req.WithContext(context.WithValue(req.Context(), identity.PortalJamaahIDKey, item.id))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		t.Logf("GET own-document account=%d: %d %s", item.id, w.Code, w.Body.String())
		if w.Code != item.want {
			t.Fatalf("got %d want %d", w.Code, item.want)
		}
		if w.Code == 200 && w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("private cache policy missing")
		}
	}
}
