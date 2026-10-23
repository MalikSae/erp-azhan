package media

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/testdb"
	"github.com/go-chi/chi/v5"
)

// PNG 1x1 minimal (magic bytes cukup untuk DetectContentType).
var pngUji = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89")

func multipartUji(t *testing.T, category string, content []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if category != "" {
		_ = mw.WriteField("category", category)
	}
	fw, err := mw.CreateFormFile("file", "a.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = fw.Write(content)
	_ = mw.Close()
	return &buf, mw.FormDataContentType()
}

func adminCtx(ctx context.Context, perms ...string) context.Context {
	ctx = context.WithValue(ctx, identity.AdminUserIDKey, int64(9))
	ctx = context.WithValue(ctx, identity.BrandIDKey, (*int64)(nil))
	return context.WithValue(ctx, identity.PermsKey, perms)
}

func uploadAdmin(h *Handler, category string, content []byte, t *testing.T, perms ...string) *httptest.ResponseRecorder {
	body, ct := multipartUji(t, category, content)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/upload", body)
	req.Header.Set("Content-Type", ct)
	req = req.WithContext(adminCtx(req.Context(), perms...))
	rec := httptest.NewRecorder()
	h.UploadMedia(rec, req)
	return rec
}

// Butir 5: admin hanya boleh mengunggah ke kategori yang dikenal, dengan izin sesuai kategori.
func TestAdminUploadIzinPerKategori(t *testing.T) {
	t.Chdir(t.TempDir())
	h := NewHandler(nil) // kategori publik tidak menyentuh database
	if rec := uploadAdmin(h, "kategori-baru", pngUji, t, "brand.edit", "masterdata.edit"); rec.Code != http.StatusBadRequest {
		t.Fatalf("kategori tak dikenal: %d %s", rec.Code, rec.Body.String())
	}
	if rec := uploadAdmin(h, "brand-logos", pngUji, t, "masterdata.edit"); rec.Code != http.StatusForbidden {
		t.Fatalf("brand-logos tanpa brand.edit: %d", rec.Code)
	}
	if rec := uploadAdmin(h, "dokumen-jamaah", pngUji, t, "brand.edit"); rec.Code != http.StatusForbidden {
		t.Fatalf("dokumen-jamaah tanpa dokumen.edit: %d", rec.Code)
	}
	if rec := uploadAdmin(h, "brand-logos", pngUji, t, "brand.edit"); rec.Code != http.StatusCreated {
		t.Fatalf("brand-logos dengan brand.edit: %d %s", rec.Code, rec.Body.String())
	}
	if rec := uploadAdmin(h, "hotel-photos", pngUji, t, "masterdata.edit"); rec.Code != http.StatusCreated {
		t.Fatalf("hotel-photos dengan masterdata.edit: %d %s", rec.Code, rec.Body.String())
	}
	// Ukuran maksimal 10 MB.
	besar := append(append([]byte{}, pngUji...), make([]byte, MaxUploadBytes+1)...)
	if rec := uploadAdmin(h, "brand-logos", besar, t, "brand.edit"); rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("berkas > 10 MB: %d", rec.Code)
	}
}

// Butir 5: unggahan pelanggan dibatasi kuota harian per akun dan per IP.
func TestCustomerUploadKuota(t *testing.T) {
	t.Chdir(t.TempDir())
	h := NewHandler(nil)
	post := func(ip string) *httptest.ResponseRecorder {
		body, ct := multipartUji(t, "", pngUji)
		req := httptest.NewRequest(http.MethodPost, "/api/account/media/upload", body)
		req.Header.Set("Content-Type", ct)
		req.RemoteAddr = ip + ":1234"
		req = req.WithContext(identity.WithCustomer(req.Context(), 55, 1))
		rec := httptest.NewRecorder()
		h.UploadPortalMedia(rec, req)
		return rec
	}
	for i := 0; i < customerUploadFilesPerDay; i++ {
		if !h.customerQuota.reserve("akun:55", 1) {
			t.Fatalf("reserve %d ditolak", i)
		}
	}
	// Kuota berkas habis: ditolak sebelum menulis berkas/menyentuh database (db nil).
	if rec := post("198.51.100.20"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("kuota berkas habis: %d %s", rec.Code, rec.Body.String())
	}
	q := newUploadQuota(20, 100<<20, 24*60*60*1e9)
	if !q.reserve("k", 60<<20) || q.reserve("k", 50<<20) || !q.reserve("k", 40<<20) {
		t.Fatal("kuota byte harian tidak ditegakkan")
	}
	// Limiter per IP.
	for i := 0; i < customerUploadIPPerHour; i++ {
		h.customerIP.Record("198.51.100.21")
	}
	if rec := post("198.51.100.21"); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("batas per IP: %d", rec.Code)
	}
}

// Butir 7b: GET /api/admin/media wajib izin sesuai kategori, juga untuk user holding.
func TestServeProtectedWajibIzinKategori(t *testing.T) {
	db := testdb.Portal(t)
	t.Chdir(t.TempDir())
	for _, dir := range []string{"dokumen-jamaah", "payment-proofs"} {
		if err := os.MkdirAll(filepath.Join("uploads", dir), 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range []string{"dokumen-jamaah/ktp.pdf", "dokumen-jamaah/bukti.png", "payment-proofs/bukti.png"} {
		if err := os.WriteFile(filepath.Join("uploads", filepath.FromSlash(f)), []byte("%PDF-1.4"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO jamaah VALUES(1,10,'a')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO dokumen_jamaah(jamaah_id,jenis,file_url,status) VALUES(1,'ktp','/api/admin/media/dokumen-jamaah/ktp.pdf','submitted')`); err != nil {
		t.Fatal(err)
	}
	h := NewHandler(db)
	router := chi.NewRouter()
	router.Get("/api/admin/media/{category}/{filename}", h.ServeProtected)
	get := func(path string, perms ...string) int {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req = req.WithContext(adminCtx(req.Context(), perms...))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}
	cases := []struct {
		path  string
		perms []string
		want  int
	}{
		{"/api/admin/media/dokumen-jamaah/ktp.pdf", nil, 404},
		{"/api/admin/media/dokumen-jamaah/ktp.pdf", []string{"payment.view"}, 404}, // dokumen identitas butuh dokumen.view
		{"/api/admin/media/dokumen-jamaah/ktp.pdf", []string{"dokumen.view"}, 200},
		{"/api/admin/media/dokumen-jamaah/bukti.png", []string{"payment.view"}, 200}, // bukti transfer pelanggan
		{"/api/admin/media/payment-proofs/bukti.png", []string{"dokumen.view"}, 404},
		{"/api/admin/media/payment-proofs/bukti.png", []string{"payment.view"}, 200},
	}
	for _, c := range cases {
		if got := get(c.path, c.perms...); got != c.want {
			t.Errorf("%s perms=%v: %d, want %d", c.path, c.perms, got, c.want)
		}
	}
}
