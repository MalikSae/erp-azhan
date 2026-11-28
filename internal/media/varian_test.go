package media

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
)

func tulisJPEG(t *testing.T, path string, w, h int, exif []byte) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.RGBA{200, 30, 30, 255} // kiri merah
			if x >= w/2 {
				c = color.RGBA{30, 30, 200, 255} // kanan biru
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	if exif != nil {
		data = append(append(append([]byte{}, data[:2]...), exif...), data[2:]...)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// segmenOrientasi membangun segmen APP1 EXIF (big endian) dengan satu tag Orientation.
func segmenOrientasi(o uint16) []byte {
	tiff := []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1}
	e := make([]byte, 12)
	binary.BigEndian.PutUint16(e[0:], 0x0112)
	binary.BigEndian.PutUint16(e[2:], 3)
	binary.BigEndian.PutUint32(e[4:], 1)
	binary.BigEndian.PutUint16(e[8:], o)
	tiff = append(append(tiff, e...), 0, 0, 0, 0)
	isi := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(isi)+2))
	return append(seg, isi...)
}

func ukuran(t *testing.T, path string) (int, int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	return cfg.Width, cfg.Height
}

func TestVarianMengecilkanJPEG(t *testing.T) {
	root := t.TempDir()
	tulisJPEG(t, filepath.Join(root, "schedule-brosur", "a.jpg"), 1080, 1350, nil)

	p, err := pathVarian(root, "schedule-brosur", "a.jpg", 400)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(p) != ".jpg" {
		t.Fatalf("ekstensi %s, mau .jpg", p)
	}
	if w, h := ukuran(t, p); w != 400 || h != 500 {
		t.Fatalf("ukuran %dx%d, mau 400x500", w, h)
	}
	// Panggilan kedua memakai berkas yang sama.
	p2, err := pathVarian(root, "schedule-brosur", "a.jpg", 400)
	if err != nil || p2 != p {
		t.Fatalf("panggilan kedua %q %v", p2, err)
	}
}

func TestVarianTidakMembesarkan(t *testing.T) {
	root := t.TempDir()
	tulisJPEG(t, filepath.Join(root, "haramain", "kecil.jpg"), 300, 200, nil)
	if _, err := pathVarian(root, "haramain", "kecil.jpg", 400); err != errTanpaVarian {
		t.Fatalf("err %v, mau errTanpaVarian", err)
	}
	// Penanda membuat panggilan berikut tidak mendekode ulang.
	if _, err := os.Stat(filepath.Join(root, folderVarian, "haramain", "kecil-w400.asli")); err != nil {
		t.Fatalf("penanda tidak dibuat: %v", err)
	}
}

func TestVarianKategoriDanFormat(t *testing.T) {
	root := t.TempDir()
	tulisJPEG(t, filepath.Join(root, "brand-logos", "a.jpg"), 1000, 1000, nil)
	if _, err := pathVarian(root, "brand-logos", "a.jpg", 400); err != errTanpaVarian {
		t.Fatalf("logo: err %v", err)
	}
	if _, err := pathVarian(root, "haramain", "a.gif", 400); err != errTanpaVarian {
		t.Fatalf("gif: err %v", err)
	}
}

func TestVarianTransparanJadiPNG(t *testing.T) {
	root := t.TempDir()
	img := image.NewNRGBA(image.Rect(0, 0, 1000, 500)) // transparan penuh
	var buf bytes.Buffer
	if err := (&png.Encoder{CompressionLevel: png.NoCompression}).Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	_ = os.MkdirAll(filepath.Join(root, "banner-promosi"), 0o755)
	_ = os.WriteFile(filepath.Join(root, "banner-promosi", "t.png"), buf.Bytes(), 0o644)

	p, err := pathVarian(root, "banner-promosi", "t.png", 800)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Ext(p) != ".png" {
		t.Fatalf("ekstensi %s, mau .png", p)
	}
	if w, h := ukuran(t, p); w != 800 || h != 400 {
		t.Fatalf("ukuran %dx%d", w, h)
	}
}

func TestVarianMengikutiOrientasiEXIF(t *testing.T) {
	root := t.TempDir()
	// Berkas 1600x800 dengan Orientation 6 (putar 90° searah jarum jam): tampil tegak 800x1600.
	tulisJPEG(t, filepath.Join(root, "hotel-photos", "hp.jpg"), 1600, 800, segmenOrientasi(6))
	data, _ := os.ReadFile(filepath.Join(root, "hotel-photos", "hp.jpg"))
	if o := orientasiEXIF(data); o != 6 {
		t.Fatalf("orientasi %d, mau 6", o)
	}
	p, err := pathVarian(root, "hotel-photos", "hp.jpg", 400)
	if err != nil {
		t.Fatal(err)
	}
	if w, h := ukuran(t, p); w != 400 || h != 800 {
		t.Fatalf("ukuran %dx%d, mau 400x800", w, h)
	}
	// Kiri berkas (merah) menjadi atas gambar tegak.
	f, _ := os.Open(p)
	defer f.Close()
	img, err := jpeg.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	r, _, b, _ := img.At(200, 50).RGBA()
	if r <= b {
		t.Fatalf("atas gambar bukan merah: r=%d b=%d", r>>8, b>>8)
	}
}

func TestLebarDiminta(t *testing.T) {
	for raw, want := range map[string]int{"400": 400, "600": 600, "800": 800, "1200": 1200, "500": 0, "": 0, "x": 0, "-400": 0} {
		if got := lebarDiminta(raw); got != want {
			t.Errorf("%q: %d, mau %d", raw, got, want)
		}
	}
}

func TestServePublicVarian(t *testing.T) {
	dir := t.TempDir()
	wd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	name := "6b27a769-1a2b-4c3d-8e9f-0123456789ab.jpg"
	tulisJPEG(t, filepath.Join("uploads", "schedule-brosur", name), 1080, 1350, nil)

	h := &Handler{}
	r := chi.NewRouter()
	r.Get("/uploads/{category}/{filename}", h.ServePublic)

	minta := func(q string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/schedule-brosur/"+name+q, nil))
		return rec
	}
	asli := minta("")
	kecil := minta("?w=400")
	salah := minta("?w=401")
	if asli.Code != 200 || kecil.Code != 200 || salah.Code != 200 {
		t.Fatalf("kode %d %d %d", asli.Code, kecil.Code, salah.Code)
	}
	if kecil.Body.Len() >= asli.Body.Len() {
		t.Fatalf("varian %d B tidak lebih kecil dari asli %d B", kecil.Body.Len(), asli.Body.Len())
	}
	if salah.Body.Len() != asli.Body.Len() {
		t.Fatalf("lebar tidak dikenal harus menyajikan asli")
	}
	if cc := kecil.Header().Get("Cache-Control"); cc != "public, max-age=31536000, immutable" {
		t.Fatalf("Cache-Control varian %q", cc)
	}
	// Folder varian tidak bisa diakses langsung.
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/uploads/_varian/x.jpg", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("_varian terbuka: %d", rec.Code)
	}
}
