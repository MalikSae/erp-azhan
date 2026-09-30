package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// JB-11: dokumen admin hanya boleh menunjuk berkas unggahan resmi.
func TestValidateAdminUpload(t *testing.T) {
	db := testdb.Portal(t)
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("uploads", "dokumen-jamaah"), 0700); err != nil {
		t.Fatal(err)
	}
	brandFile := "/api/admin/media/dokumen-jamaah/brand10.pdf"
	superFile := "/api/admin/media/dokumen-jamaah/super.pdf"
	for _, name := range []string{"brand10.pdf", "super.pdf"} {
		if err := os.WriteFile(filepath.Join("uploads", "dokumen-jamaah", name), []byte("%PDF-1.4"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO jamaah VALUES(1,10,'a'),(2,20,'b')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO media_uploads(path,brand_id,admin_user_id) VALUES(?,10,5),(?,NULL,1)`, brandFile, superFile); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	for _, ok := range []struct {
		path string
		id   int64
	}{{brandFile, 1}, {superFile, 2}} {
		if err := ValidateAdminUpload(ctx, db, ok.path, ok.id); err != nil {
			t.Fatalf("%s untuk jamaah %d ditolak: %v", ok.path, ok.id, err)
		}
	}
	for _, bad := range []struct {
		path string
		id   int64
	}{
		{brandFile, 2}, // unggahan brand lain
		{"https://contoh-luar.example/paspor.pdf", 1},
		{"/uploads/../../.env", 1},
		{"/api/admin/media/dokumen-jamaah/tidak-ada.pdf", 1},
	} {
		var uploadErr *UploadError
		if err := ValidateAdminUpload(ctx, db, bad.path, bad.id); !errors.As(err, &uploadErr) {
			t.Fatalf("%s untuk jamaah %d: err=%v, want UploadError", bad.path, bad.id, err)
		}
	}
}
