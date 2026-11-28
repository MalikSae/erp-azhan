package media

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/gif"
	"image/png"
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func jpegBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	p := filepath.Join(t.TempDir(), "x.jpg")
	tulisJPEG(t, p, w, h, nil)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// pngBerisik: PNG buram dengan derau acak agar berukuran besar (meniru poster PNG dari desain).
func pngBerisik(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	r := rand.New(rand.NewSource(1))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3] = uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 255
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func ukuranBytes(t *testing.T, b []byte) (int, int) {
	t.Helper()
	cfg, _, err := image.DecodeConfig(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Width, cfg.Height
}

func TestOptimasiUnggahanLebarMaks(t *testing.T) {
	isi, ext, ok := optimasiUnggahan("schedule-brosur", ".jpg", jpegBytes(t, 3000, 1500))
	if !ok || ext != ".jpg" {
		t.Fatalf("ok=%v ext=%q", ok, ext)
	}
	if w, h := ukuranBytes(t, isi); w != 2000 || h != 1000 {
		t.Fatalf("ukuran %dx%d, mau 2000x1000", w, h)
	}
}

func TestOptimasiUnggahanSudahKecilTidakDiubah(t *testing.T) {
	if _, _, ok := optimasiUnggahan("haramain", ".jpg", jpegBytes(t, 1200, 800)); ok {
		t.Fatal("gambar 1200 px di bawah 1 MB tidak boleh disimpan ulang")
	}
}

func TestOptimasiUnggahanPNGBesarDisimpanUlang(t *testing.T) {
	asli := pngBerisik(t, 1080, 1350)
	if len(asli) <= ukuranUlangUnggah {
		t.Fatalf("fixture %d B harus di atas 1 MB", len(asli))
	}
	isi, ext, ok := optimasiUnggahan("schedule-brosur", ".png", asli)
	if !ok || ext != ".jpg" {
		t.Fatalf("ok=%v ext=%q", ok, ext)
	}
	if w, h := ukuranBytes(t, isi); w != 1080 || h != 1350 {
		t.Fatalf("ukuran berubah: %dx%d", w, h)
	}
	if len(isi) >= len(asli) {
		t.Fatalf("hasil %d B tidak lebih kecil dari %d B", len(isi), len(asli))
	}
}

func TestOptimasiUnggahanKategoriLainDanGIF(t *testing.T) {
	besar := jpegBytes(t, 3000, 1500)
	for _, k := range []string{"brand-logos", "airline-logos", "dokumen-jamaah", "payment-proofs"} {
		if _, _, ok := optimasiUnggahan(k, ".jpg", besar); ok {
			t.Fatalf("%s tidak boleh diubah", k)
		}
	}
	var buf bytes.Buffer
	pal := image.NewPaletted(image.Rect(0, 0, 3000, 10), []color.Color{color.Black, color.White})
	if err := gif.Encode(&buf, pal, nil); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := optimasiUnggahan("banner-promosi", ".gif", buf.Bytes()); ok {
		t.Fatal("GIF tidak boleh diubah")
	}
}

// Ujung ke ujung: poster 3000 px lewat processUpload tersimpan selebar 2000 px dan URL-nya menunjuk berkas itu.
func TestProcessUploadMengecilkanPoster(t *testing.T) {
	t.Chdir(t.TempDir())
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "poster.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write(jpegBytes(t, 3000, 3750)); err != nil {
		t.Fatal(err)
	}
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/media/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if err := req.ParseMultipartForm(MaxUploadBytes); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	(&Handler{}).processUpload(rec, req, "schedule-brosur", "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("kode %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(resp["url"], "/uploads/schedule-brosur/") || !strings.HasSuffix(resp["url"], ".jpg") {
		t.Fatalf("url %q", resp["url"])
	}
	disimpan, err := os.ReadFile(filepath.Join(".", strings.TrimPrefix(resp["url"], "/")))
	if err != nil {
		t.Fatal(err)
	}
	if w, h := ukuranBytes(t, disimpan); w != 2000 || h != 2500 {
		t.Fatalf("tersimpan %dx%d, mau 2000x2500", w, h)
	}
}
