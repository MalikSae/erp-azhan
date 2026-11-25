package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// JB-11: dokumen admin hanya boleh menunjuk berkas unggahan admin itu sendiri
// atau unggahan admin brand jamaah.
func TestValidateAdminUpload(t *testing.T) {
	db := testdb.Portal(t)
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("uploads", "dokumen-jamaah"), 0700); err != nil {
		t.Fatal(err)
	}
	brandFile := "/api/admin/media/dokumen-jamaah/brand10.pdf"
	superFile := "/api/admin/media/dokumen-jamaah/super.pdf"
	jamaahFile := "/api/admin/media/dokumen-jamaah/portal.pdf"
	for _, name := range []string{"brand10.pdf", "super.pdf", "portal.pdf"} {
		if err := os.WriteFile(filepath.Join("uploads", "dokumen-jamaah", name), []byte("%PDF-1.4"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO jamaah VALUES(1,10,'a'),(2,20,'b')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO media_uploads(path,brand_id,admin_user_id,jamaah_id) VALUES(?,10,5,NULL),(?,NULL,1,NULL),(?,10,NULL,1)`,
		brandFile, superFile, jamaahFile); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	type kasus struct {
		nama        string
		path        string
		jamaah      int64
		adminUserID int64
	}
	for _, ok := range []kasus{
		{"unggahan sendiri, brand jamaah", brandFile, 1, 5},
		{"unggahan admin lain, brand jamaah", brandFile, 1, 6},
		{"unggahan sendiri tanpa brand", superFile, 2, 1},
	} {
		if err := ValidateAdminUpload(ctx, db, ok.path, ok.jamaah, ok.adminUserID); err != nil {
			t.Fatalf("%s ditolak: %v", ok.nama, err)
		}
	}
	for _, bad := range []kasus{
		{"unggahan brand lain", brandFile, 2, 5},
		{"unggahan tanpa brand milik admin lain", superFile, 2, 7},
		{"unggahan bukan lewat jalur admin", jamaahFile, 1, 5},
		{"URL luar", "https://contoh-luar.example/paspor.pdf", 1, 5},
		{"path buatan", "/uploads/../../.env", 1, 5},
		{"berkas tidak ada", "/api/admin/media/dokumen-jamaah/tidak-ada.pdf", 1, 5},
	} {
		var uploadErr *UploadError
		if err := ValidateAdminUpload(ctx, db, bad.path, bad.jamaah, bad.adminUserID); !errors.As(err, &uploadErr) {
			t.Fatalf("%s: err=%v, want UploadError", bad.nama, err)
		}
	}
}
