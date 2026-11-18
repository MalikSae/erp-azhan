package banner

import (
	"context"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// tulisPNG menyimpan PNG polos berukuran w × h di dir/banner-promosi dan mengembalikan URL-nya.
func tulisPNG(t *testing.T, dir, nama string, w, h int) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, KategoriUnggah), 0o755); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, KategoriUnggah, nama))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return "/uploads/banner-promosi/" + nama
}

const uuidUji = "11111111-2222-4333-8444-"

func galatValidasi(err error) bool {
	var v *ErrValidasi
	return errors.As(err, &v)
}

func TestCekGambar(t *testing.T) {
	dir := t.TempDir()
	pas := tulisPNG(t, dir, uuidUji+"000000000001.png", 1200, 600)
	besar := tulisPNG(t, dir, uuidUji+"000000000002.png", 2400, 1200)
	persegi := tulisPNG(t, dir, uuidUji+"000000000003.png", 1200, 1200)
	kecil := tulisPNG(t, dir, uuidUji+"000000000004.png", 800, 400)
	if err := CekGambar(dir, pas); err != nil {
		t.Errorf("1200x600 ditolak: %v", err)
	}
	if err := CekGambar(dir, besar); err != nil {
		t.Errorf("2400x1200 ditolak: %v", err)
	}
	for nama, u := range map[string]string{"persegi": persegi, "kecil": kecil, "luar folder": "/uploads/brand-logos/x.png", "tidak ada": "/uploads/banner-promosi/" + uuidUji + "000000000009.png"} {
		if err := CekGambar(dir, u); !galatValidasi(err) {
			t.Errorf("%s: harus ditolak, err=%v", nama, err)
		}
	}
}

func TestBannerDanPromosi(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-03-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status = 'published' WHERE id = ?`, jadwal)
	// Banner lain milik brand ini di DB lokal tidak boleh memengaruhi uji (dibatalkan bersama transaksi).
	testdb.Exec(t, tx, `DELETE FROM banner_promosi WHERE brand_id = ?`, brand)
	dir := t.TempDir()
	now := time.Date(2099, 1, 10, 9, 0, 0, 0, ZonaWIB)
	str := func(s string) *string { return &s }
	paket := strconv.FormatInt(jadwal, 10)

	if _, err := Buat(ctx, tx, dir, 0, Input{BrandID: brand, Judul: "Tanpa gambar", Aktif: true}); !galatValidasi(err) {
		t.Errorf("tanpa gambar: %v", err)
	}
	g := tulisPNG(t, dir, uuidUji+"000000000010.png", 1200, 600)
	if _, err := Buat(ctx, tx, dir, 0, Input{BrandID: brand, Judul: "Paket asing", GambarURL: g, Aktif: true,
		Tujuan: Tujuan{Jenis: TujuanPaket, Nilai: str("999999999")}}); !galatValidasi(err) {
		t.Errorf("paket asing: %v", err)
	}
	if _, err := Buat(ctx, tx, dir, 0, Input{BrandID: brand, Judul: "Tautan http", GambarURL: g, Aktif: true,
		Tujuan: Tujuan{Jenis: TujuanTautan, Nilai: str("http://contoh.id")}}); !galatValidasi(err) {
		t.Errorf("tautan http: %v", err)
	}

	// Tujuh banner: satu nonaktif, satu terjadwal, satu berakhir, empat tayang + satu tayang ekstra (lewat batas 5).
	buat := func(judul string, aktif bool, mulai, selesai *string, tujuan Tujuan) int64 {
		t.Helper()
		id, err := Buat(ctx, tx, dir, 0, Input{BrandID: brand, Judul: judul, GambarURL: g, Aktif: aktif, Mulai: mulai, Selesai: selesai, Tujuan: tujuan})
		if err != nil {
			t.Fatalf("buat %s: %v", judul, err)
		}
		return id
	}
	b1 := buat("Paket", true, nil, nil, Tujuan{Jenis: TujuanPaket, Nilai: &paket})
	buat("Jenis", true, nil, str("2099-01-10"), Tujuan{Jenis: TujuanJenisPaket, Nilai: str("low-cost")})
	buat("WA", true, str("2099-01-01"), nil, Tujuan{Jenis: TujuanWhatsApp})
	buat("Tautan", true, nil, nil, Tujuan{Jenis: TujuanTautan, Nilai: str("https://contoh.id/promo")})
	nonaktif := buat("Nonaktif", false, nil, nil, Tujuan{})
	terjadwal := buat("Terjadwal", true, str("2099-02-01"), nil, Tujuan{})
	berakhir := buat("Berakhir", true, nil, str("2099-01-09"), Tujuan{})
	ekstra1 := buat("Ekstra1", true, nil, nil, Tujuan{})
	buat("Ekstra2", true, nil, nil, Tujuan{})

	daftar, err := Daftar(ctx, tx, brand, now)
	if err != nil {
		t.Fatal(err)
	}
	status := map[int64]string{}
	for _, b := range daftar {
		status[b.ID] = b.Status
	}
	if status[nonaktif] != "nonaktif" || status[terjadwal] != "terjadwal" || status[berakhir] != "berakhir" || status[b1] != "tayang" {
		t.Errorf("status: %+v", status)
	}

	p, err := AmbilPromosi(ctx, tx, brand, "", now)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Banner) != MaksTayang || p.Banner[0].ID != b1 {
		t.Fatalf("promosi: %+v", p.Banner)
	}
	for _, b := range p.Banner {
		if b.ID == nonaktif || b.ID == terjadwal || b.ID == berakhir {
			t.Errorf("banner tidak tayang ikut terkirim: %+v", b)
		}
	}

	// Tempat tampil: banner khusus web tidak ikut ke app, khusus app tidak ikut ke web.
	if _, err := Buat(ctx, tx, dir, 0, Input{BrandID: brand, Judul: "Salah", GambarURL: g, Aktif: true, TampilDi: "lainnya"}); !galatValidasi(err) {
		t.Errorf("tampil_di tidak sah: %v", err)
	}
	soloWeb, err := Buat(ctx, tx, dir, 0, Input{BrandID: brand, Judul: "Solo web", GambarURL: g, Aktif: true, TampilDi: TampilWeb})
	if err != nil {
		t.Fatal(err)
	}
	soloApp, err := Buat(ctx, tx, dir, 0, Input{BrandID: brand, Judul: "Solo app", GambarURL: g, Aktif: true, TampilDi: TampilApp})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE banner_promosi SET urutan = urutan + 100 WHERE brand_id = ? AND id NOT IN (?, ?)`, brand, soloWeb, soloApp); err != nil {
		t.Fatal(err)
	}
	pw, _ := AmbilPromosi(ctx, tx, brand, TampilWeb, now)
	pa, _ := AmbilPromosi(ctx, tx, brand, TampilApp, now)
	punya := func(p *Promosi, id int64) bool {
		for _, b := range p.Banner {
			if b.ID == id {
				return true
			}
		}
		return false
	}
	if !punya(pw, soloWeb) || punya(pw, soloApp) || !punya(pa, soloApp) || punya(pa, soloWeb) {
		t.Errorf("tampil_di: web=%+v app=%+v", pw.Banner, pa.Banner)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE banner_promosi SET urutan = urutan - 100 WHERE brand_id = ? AND id NOT IN (?, ?)`, brand, soloWeb, soloApp); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM banner_promosi WHERE id IN (?, ?)`, soloWeb, soloApp); err != nil {
		t.Fatal(err)
	}

	// Urutan: Ekstra1 dipindah ke paling atas.
	ids := []int64{ekstra1}
	for _, b := range daftar {
		if b.ID != ekstra1 {
			ids = append(ids, b.ID)
		}
	}
	if err := Urutkan(ctx, tx, brand, ids[:3]); !galatValidasi(err) {
		t.Errorf("urutan tidak lengkap: %v", err)
	}
	if err := Urutkan(ctx, tx, brand, ids); err != nil {
		t.Fatal(err)
	}
	if p, _ = AmbilPromosi(ctx, tx, brand, "", now); p.Banner[0].ID != ekstra1 {
		t.Errorf("urutan baru: %+v", p.Banner[0])
	}

	// Klik hanya untuk banner tayang.
	if err := CatatKlik(ctx, tx, b1, now); err != nil {
		t.Fatal(err)
	}
	if err := CatatKlik(ctx, tx, nonaktif, now); err != nil {
		t.Fatal(err)
	}
	var k1, k2 int
	_ = tx.QueryRowContext(ctx, `SELECT klik FROM banner_promosi WHERE id = ?`, b1).Scan(&k1)
	_ = tx.QueryRowContext(ctx, `SELECT klik FROM banner_promosi WHERE id = ?`, nonaktif).Scan(&k2)
	if k1 != 1 || k2 != 0 {
		t.Errorf("klik: tayang=%d nonaktif=%d", k1, k2)
	}

	// Ubah dan hapus lintas brand ditolak.
	lain := brand + 99999
	if err := Ubah(ctx, tx, dir, b1, &lain, 0, Input{Judul: "x", GambarURL: g}); !errors.Is(err, ErrTidakAda) {
		t.Errorf("ubah lintas brand: %v", err)
	}
	if err := Hapus(ctx, tx, b1, &lain); !errors.Is(err, ErrTidakAda) {
		t.Errorf("hapus lintas brand: %v", err)
	}
	if err := Ubah(ctx, tx, dir, b1, &brand, 0, Input{Judul: "Paket baru", GambarURL: g, Aktif: false}); err != nil {
		t.Fatal(err)
	}
	if err := Hapus(ctx, tx, b1, &brand); err != nil {
		t.Fatal(err)
	}
}
