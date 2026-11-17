// Package banner: Banner Promosi di Beranda app Hana (lihat migrasi 144). Banner adalah gambar jadi
// buatan desainer dengan satu tujuan saat ditekan, jadwal tayang, urutan, dan hitungan klik.
package banner

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // dekoder header gambar banner
	_ "image/png"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	_ "golang.org/x/image/webp"
)

var (
	ErrTidakAda = errors.New("banner tidak ditemukan")
)

// ErrValidasi membawa pesan yang aman ditampilkan ke admin.
type ErrValidasi struct{ Pesan string }

func (e *ErrValidasi) Error() string { return e.Pesan }

func galat(format string, args ...any) error {
	return &ErrValidasi{Pesan: fmt.Sprintf(format, args...)}
}

// ZonaWIB: jadwal tayang dibaca sebagai tanggal Indonesia Barat.
var ZonaWIB = time.FixedZone("WIB", 7*3600)

func hariIni(now time.Time) string { return now.In(ZonaWIB).Format("2006-01-02") }

const (
	// MaksTayang: banner aktif yang dikirim ke app.
	MaksTayang = 5
	// MaksUkuranGambar: 1 MB.
	MaksUkuranGambar = 1 << 20
	// LebarMinimal: gambar 2:1 minimal 1200 × 600 px.
	LebarMinimal = 1200
	// KategoriUnggah: kategori media untuk gambar banner (publik, /uploads/banner-promosi/...).
	KategoriUnggah = "banner-promosi"
)

// Tujuan saat banner ditekan.
const (
	TujuanTidakAda   = "tidak_ada"
	TujuanPaket      = "paket"
	TujuanJenisPaket = "jenis_paket"
	TujuanWhatsApp   = "whatsapp"
	TujuanTautan     = "tautan"
)

// JenisPaket: filter tab Paket di app (data/paket.ts, JenisPaket).
var JenisPaket = map[string]bool{"umroh": true, "low-cost": true, "private": true, "wisata": true, "haji": true}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Tujuan: aksi saat ditekan. Nilai: id jadwal, jenis paket, pesan WhatsApp (opsional), atau URL https.
type Tujuan struct {
	Jenis string  `json:"tujuan"`
	Nilai *string `json:"tujuan_nilai"`
}

func (t *Tujuan) rapikan() {
	t.Jenis = strings.TrimSpace(t.Jenis)
	if t.Jenis == "" {
		t.Jenis = TujuanTidakAda
	}
	if t.Nilai != nil {
		v := strings.TrimSpace(*t.Nilai)
		t.Nilai = &v
		if v == "" {
			t.Nilai = nil
		}
	}
}

// validasiTujuan memastikan tujuan sah untuk brand itu. Paket harus milik brand dan tidak diarsipkan.
func validasiTujuan(ctx context.Context, q querier, brandID int64, t *Tujuan) error {
	t.rapikan()
	nilai := ""
	if t.Nilai != nil {
		nilai = *t.Nilai
	}
	switch t.Jenis {
	case TujuanTidakAda:
		t.Nilai = nil
	case TujuanPaket:
		id, err := strconv.ParseInt(nilai, 10, 64)
		if err != nil || id <= 0 {
			return galat("Pilih paket tujuan")
		}
		var ada bool
		if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schedules WHERE id = ? AND brand_id = ? AND status <> 'archived')`,
			id, brandID).Scan(&ada); err != nil {
			return err
		}
		if !ada {
			return galat("Paket tujuan tidak ditemukan")
		}
	case TujuanJenisPaket:
		if !JenisPaket[nilai] {
			return galat("Pilih jenis paket tujuan")
		}
	case TujuanWhatsApp:
		if utf8.RuneCountInString(nilai) > 200 {
			return galat("Pesan WhatsApp maksimal 200 karakter")
		}
	case TujuanTautan:
		u, err := url.Parse(nilai)
		if err != nil || u.Scheme != "https" || u.Host == "" || len(nilai) > 500 {
			return galat("Tautan harus diawali https://")
		}
	default:
		return galat("Tujuan tidak valid")
	}
	return nil
}

// ---- Banner ----

type Banner struct {
	ID        int64   `json:"id"`
	BrandID   int64   `json:"brand_id"`
	Judul     string  `json:"judul"`
	GambarURL string  `json:"gambar_url"`
	Tujuan            // tujuan, tujuan_nilai
	Mulai     *string `json:"mulai"`   // YYYY-MM-DD
	Selesai   *string `json:"selesai"` // YYYY-MM-DD
	Aktif     bool    `json:"aktif"`
	Urutan    int     `json:"urutan"`
	Klik      int     `json:"klik"`
	// Status turunan untuk daftar admin: tayang | terjadwal | berakhir | nonaktif.
	Status string `json:"status"`
}

func statusBanner(b *Banner, today string) string {
	switch {
	case !b.Aktif:
		return "nonaktif"
	case b.Selesai != nil && *b.Selesai < today:
		return "berakhir"
	case b.Mulai != nil && *b.Mulai > today:
		return "terjadwal"
	default:
		return "tayang"
	}
}

// Input: isi form banner.
type Input struct {
	BrandID   int64   `json:"brand_id"`
	Judul     string  `json:"judul"`
	GambarURL string  `json:"gambar_url"`
	Tujuan            // tujuan, tujuan_nilai
	Mulai     *string `json:"mulai"`
	Selesai   *string `json:"selesai"`
	Aktif     bool    `json:"aktif"`
}

var polaGambar = regexp.MustCompile(`^/uploads/banner-promosi/[0-9a-f-]{36}\.(jpg|png|webp)$`)

// CekGambar memeriksa berkas banner di folder unggahan: maksimal 1 MB, rasio tepat 2:1, lebar minimal 1200 px.
// dirUnggah biasanya "uploads".
func CekGambar(dirUnggah, gambarURL string) error {
	if !polaGambar.MatchString(gambarURL) {
		return galat("Unggah gambar banner")
	}
	path := filepath.Join(dirUnggah, KategoriUnggah, filepath.Base(gambarURL))
	f, err := os.Open(path)
	if err != nil {
		return galat("Gambar banner tidak ditemukan. Unggah ulang")
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if info.Size() > MaksUkuranGambar {
		return galat("Ukuran gambar maksimal 1 MB")
	}
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return galat("Gambar harus JPG, PNG, atau WebP")
	}
	if cfg.Width != 2*cfg.Height || cfg.Width < LebarMinimal {
		return galat("Ukuran gambar harus 1200 × 600 px (rasio 2:1)")
	}
	return nil
}

func tanggalOpsional(p *string) (*string, error) {
	if p == nil || strings.TrimSpace(*p) == "" {
		return nil, nil
	}
	v := strings.TrimSpace(*p)
	if _, err := time.Parse("2006-01-02", v); err != nil {
		return nil, galat("Tanggal tidak valid")
	}
	return &v, nil
}

// validasi merapikan dan memeriksa isian banner (tanpa memeriksa berkas gambar).
func (in *Input) validasi(ctx context.Context, q querier) error {
	in.Judul = strings.TrimSpace(in.Judul)
	in.GambarURL = strings.TrimSpace(in.GambarURL)
	switch {
	case in.Judul == "":
		return galat("Judul wajib diisi")
	case utf8.RuneCountInString(in.Judul) > 100:
		return galat("Judul maksimal 100 karakter")
	}
	var err error
	if in.Mulai, err = tanggalOpsional(in.Mulai); err != nil {
		return err
	}
	if in.Selesai, err = tanggalOpsional(in.Selesai); err != nil {
		return err
	}
	if in.Mulai != nil && in.Selesai != nil && *in.Selesai < *in.Mulai {
		return galat("Tanggal selesai harus setelah tanggal mulai")
	}
	return validasiTujuan(ctx, q, in.BrandID, &in.Tujuan)
}

const kolomBanner = `b.id, b.brand_id, b.judul, b.gambar_url, b.tujuan, b.tujuan_nilai,
	DATE_FORMAT(b.mulai, '%Y-%m-%d'), DATE_FORMAT(b.selesai, '%Y-%m-%d'), b.aktif, b.urutan, b.klik`

func scanBanner(row interface{ Scan(...any) error }) (*Banner, error) {
	var b Banner
	var nilai, mulai, selesai sql.NullString
	if err := row.Scan(&b.ID, &b.BrandID, &b.Judul, &b.GambarURL, &b.Tujuan.Jenis, &nilai, &mulai, &selesai, &b.Aktif, &b.Urutan, &b.Klik); err != nil {
		return nil, err
	}
	if nilai.Valid {
		b.Tujuan.Nilai = &nilai.String
	}
	if mulai.Valid {
		b.Mulai = &mulai.String
	}
	if selesai.Valid {
		b.Selesai = &selesai.String
	}
	return &b, nil
}

// Daftar: semua banner brand, urut tampil.
func Daftar(ctx context.Context, q querier, brandID int64, now time.Time) ([]Banner, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+kolomBanner+` FROM banner_promosi b WHERE b.brand_id = ? ORDER BY b.urutan, b.id`, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	today := hariIni(now)
	out := []Banner{}
	for rows.Next() {
		b, err := scanBanner(rows)
		if err != nil {
			return nil, err
		}
		b.Status = statusBanner(b, today)
		out = append(out, *b)
	}
	return out, rows.Err()
}

// ambil memuat banner dalam lingkup brand (scope nil = semua brand).
func ambil(ctx context.Context, q querier, id int64, scope *int64) (*Banner, error) {
	query, args := `SELECT `+kolomBanner+` FROM banner_promosi b WHERE b.id = ?`, []any{id}
	if scope != nil {
		query += ` AND b.brand_id = ?`
		args = append(args, *scope)
	}
	b, err := scanBanner(q.QueryRowContext(ctx, query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTidakAda
	}
	return b, err
}

func adminAtauNull(id int64) any {
	if id > 0 {
		return id
	}
	return nil
}

// Buat menambah banner di urutan paling akhir brand.
func Buat(ctx context.Context, q querier, dirUnggah string, adminID int64, in Input) (int64, error) {
	if err := in.validasi(ctx, q); err != nil {
		return 0, err
	}
	if err := CekGambar(dirUnggah, in.GambarURL); err != nil {
		return 0, err
	}
	res, err := q.ExecContext(ctx, `
		INSERT INTO banner_promosi (brand_id, judul, gambar_url, tujuan, tujuan_nilai, mulai, selesai, aktif, urutan, dibuat_oleh, diubah_oleh)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(MAX(urutan), 0) + 1, ?, ? FROM banner_promosi WHERE brand_id = ?`,
		in.BrandID, in.Judul, in.GambarURL, in.Tujuan.Jenis, in.Tujuan.Nilai, in.Mulai, in.Selesai, in.Aktif,
		adminAtauNull(adminID), adminAtauNull(adminID), in.BrandID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Ubah memperbarui banner. Brand banner tidak bisa dipindah.
func Ubah(ctx context.Context, q querier, dirUnggah string, id int64, scope *int64, adminID int64, in Input) error {
	lama, err := ambil(ctx, q, id, scope)
	if err != nil {
		return err
	}
	in.BrandID = lama.BrandID
	if err := in.validasi(ctx, q); err != nil {
		return err
	}
	if in.GambarURL != lama.GambarURL {
		if err := CekGambar(dirUnggah, in.GambarURL); err != nil {
			return err
		}
	}
	_, err = q.ExecContext(ctx, `
		UPDATE banner_promosi SET judul = ?, gambar_url = ?, tujuan = ?, tujuan_nilai = ?, mulai = ?, selesai = ?, aktif = ?, diubah_oleh = ?
		WHERE id = ?`,
		in.Judul, in.GambarURL, in.Tujuan.Jenis, in.Tujuan.Nilai, in.Mulai, in.Selesai, in.Aktif, adminAtauNull(adminID), id)
	return err
}

// Hapus menghapus banner.
func Hapus(ctx context.Context, q querier, id int64, scope *int64) error {
	if _, err := ambil(ctx, q, id, scope); err != nil {
		return err
	}
	_, err := q.ExecContext(ctx, `DELETE FROM banner_promosi WHERE id = ?`, id)
	return err
}

// Urutkan menyimpan urutan banner sebuah brand. ids harus berisi tepat semua banner brand itu.
func Urutkan(ctx context.Context, q querier, brandID int64, ids []int64) error {
	var jumlah int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM banner_promosi WHERE brand_id = ?`, brandID).Scan(&jumlah); err != nil {
		return err
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	if len(seen) != jumlah || len(ids) != jumlah {
		return galat("Urutan tidak lengkap. Muat ulang halaman")
	}
	for i, id := range ids {
		res, err := q.ExecContext(ctx, `UPDATE banner_promosi SET urutan = ? WHERE id = ? AND brand_id = ?`, i+1, id, brandID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			var ada bool
			if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM banner_promosi WHERE id = ? AND brand_id = ?)`, id, brandID).Scan(&ada); err != nil {
				return err
			}
			if !ada {
				return galat("Urutan tidak lengkap. Muat ulang halaman")
			}
		}
	}
	return nil
}

// ---- Publik (app) ----

// BannerTayang: banner untuk app.
type BannerTayang struct {
	ID        int64  `json:"id"`
	Judul     string `json:"judul"`
	GambarURL string `json:"gambar_url"`
	Tujuan           // tujuan, tujuan_nilai
}

// Promosi: banner Beranda app.
type Promosi struct {
	Banner []BannerTayang `json:"banner"`
}

// syaratTayang: aktif dan dalam jadwal (tanggal WIB).
const syaratTayang = `b.aktif AND (b.mulai IS NULL OR b.mulai <= ?) AND (b.selesai IS NULL OR b.selesai >= ?)`

// AmbilPromosi memuat maksimal 5 banner yang sedang tayang.
func AmbilPromosi(ctx context.Context, q querier, brandID int64, now time.Time) (*Promosi, error) {
	today := hariIni(now)
	rows, err := q.QueryContext(ctx, `SELECT b.id, b.judul, b.gambar_url, b.tujuan, b.tujuan_nilai FROM banner_promosi b
		WHERE b.brand_id = ? AND `+syaratTayang+` ORDER BY b.urutan, b.id LIMIT ?`, brandID, today, today, MaksTayang)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	p := &Promosi{Banner: []BannerTayang{}}
	for rows.Next() {
		var b BannerTayang
		var nilai sql.NullString
		if err := rows.Scan(&b.ID, &b.Judul, &b.GambarURL, &b.Tujuan.Jenis, &nilai); err != nil {
			return nil, err
		}
		if nilai.Valid {
			b.Tujuan.Nilai = &nilai.String
		}
		p.Banner = append(p.Banner, b)
	}
	return p, rows.Err()
}

// CatatKlik menambah hitungan klik banner yang sedang tayang. Banner lain diabaikan tanpa galat.
func CatatKlik(ctx context.Context, q querier, id int64, now time.Time) error {
	today := hariIni(now)
	_, err := q.ExecContext(ctx, `UPDATE banner_promosi b SET b.klik = b.klik + 1, b.diubah_at = b.diubah_at WHERE b.id = ? AND `+syaratTayang, id, today, today)
	return err
}
