package media

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Bukti pembayaran admin hanya boleh menunjuk berkas unggahan admin resmi di brand yang sama.
func TestValidateAdminPaymentProof(t *testing.T) {
	db := testdb.Portal(t)
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(filepath.Join("uploads", "payment-proofs"), 0700); err != nil {
		t.Fatal(err)
	}
	const brand10 = "/api/admin/media/payment-proofs/11111111-1111-1111-1111-111111111111.pdf"
	const brand20 = "/api/admin/media/payment-proofs/22222222-2222-2222-2222-222222222222.pdf"
	const customer = "/api/admin/media/payment-proofs/33333333-3333-3333-3333-333333333333.pdf"
	const hilang = "/api/admin/media/payment-proofs/44444444-4444-4444-4444-444444444444.pdf"
	for _, n := range []string{"11111111-1111-1111-1111-111111111111.pdf", "22222222-2222-2222-2222-222222222222.pdf", "33333333-3333-3333-3333-333333333333.pdf"} {
		if err := os.WriteFile(filepath.Join("uploads", "payment-proofs", n), []byte("%PDF-1.4"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	// brand10 dan brand20 diunggah admin; customer diunggah akun pelanggan (admin_user_id NULL); hilang tercatat tetapi berkasnya tidak ada.
	if _, err := db.Exec(`INSERT INTO media_uploads(path,brand_id,admin_user_id) VALUES(?,10,5),(?,20,6),(?,10,NULL),(?,10,5)`, brand10, brand20, customer, hilang); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	b10, b20 := int64(10), int64(20)

	for _, ok := range []struct {
		nama  string
		path  string
		brand *int64
	}{
		{"admin brand 10, berkas brand 10", brand10, &b10},
		{"admin brand 20, berkas brand 20", brand20, &b20},
		{"super admin, berkas brand 10", brand10, nil},
		{"super admin, berkas brand 20", brand20, nil},
		{"bentuk lama /uploads/", "/uploads/payment-proofs/11111111-1111-1111-1111-111111111111.pdf", &b10},
	} {
		if err := ValidateAdminPaymentProof(ctx, db, ok.path, ok.brand); err != nil {
			t.Fatalf("%s ditolak: %v", ok.nama, err)
		}
	}

	for _, bad := range []struct {
		nama  string
		path  string
		brand *int64
	}{
		{"URL eksternal", "https://evil.example/x.pdf", &b10},
		{"URL eksternal super admin", "https://evil.example/x.pdf", nil},
		{"skema lain", "javascript:alert(1)", nil},
		{"kategori dokumen-jamaah", "/api/admin/media/dokumen-jamaah/11111111-1111-1111-1111-111111111111.pdf", nil},
		{"path traversal", "/api/admin/media/payment-proofs/../../etc/passwd", nil},
		{"berkas brand lain", brand20, &b10},
		{"unggahan pelanggan", customer, &b10},
		{"tercatat tetapi berkas tidak ada", hilang, &b10},
		{"tidak tercatat", "/api/admin/media/payment-proofs/55555555-5555-5555-5555-555555555555.pdf", nil},
		{"kosong", "", nil},
	} {
		err := ValidateAdminPaymentProof(ctx, db, bad.path, bad.brand)
		var uploadErr *UploadError
		if !errors.As(err, &uploadErr) {
			t.Fatalf("%s: harus ditolak dengan UploadError, dapat %v", bad.nama, err)
		}
	}
}
