// Package haramain: Jelajah Haramain (lihat migrasi 145). Tempat global di Makkah dan Madinah, ulasan jamaah tanpa
// bintang dari akun yang jamaahnya sudah berangkat, laporan ulasan, dan moderasi admin.
package haramain

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"

	"erp-azhan/api/internal/shared"
)

var (
	ErrTidakAda = errors.New("tempat tidak ditemukan")
	// ErrBelumBerangkat: hanya akun yang jamaahnya sudah berangkat yang boleh menulis ulasan.
	ErrBelumBerangkat = errors.New("ulasan hanya untuk jamaah yang sudah berangkat")
	ErrUlasanTidakAda = errors.New("ulasan tidak ditemukan")
	// ErrDipakaiPaket: a tempat picked as Lokasi Dikunjungi by any paket cannot be deleted.
	ErrDipakaiPaket = errors.New("Tempat dipakai paket.")
)

// ErrValidasi membawa pesan yang aman ditampilkan.
type ErrValidasi struct{ Pesan string }

func (e *ErrValidasi) Error() string { return e.Pesan }

func galat(format string, args ...any) error {
	return &ErrValidasi{Pesan: fmt.Sprintf(format, args...)}
}

// ZonaWIB: "sudah berangkat" dihitung dengan tanggal Indonesia Barat.
var ZonaWIB = time.FixedZone("WIB", 7*3600)

// Kategori tempat (daftar tetap).
var Kategori = map[string]string{
	"masjid": "Masjid", "bersejarah": "Tempat Bersejarah", "ziarah": "Ziarah",
	"belanja": "Pusat Belanja", "kuliner": "Kuliner", "lainnya": "Lainnya",
}

var Kota = map[string]string{"makkah": "Makkah", "madinah": "Madinah"}

const (
	maksGaleri     = 10
	maksFotoUlasan = 3
	maksTeksUlasan = 1000
	// KategoriFotoTempat dan KategoriFotoUlasan: kategori media publik.
	KategoriFotoTempat = "haramain"
	KategoriFotoUlasan = "ulasan-haramain"
	// maksUlasanTampil: ulasan terbaru yang dikirim di detail tempat.
	maksUlasanTampil = 50
)

var (
	polaFotoTempat = regexp.MustCompile(`^/uploads/haramain/[0-9a-f-]{36}\.(jpg|png|webp)$`)
	polaFotoUlasan = regexp.MustCompile(`^/uploads/ulasan-haramain/[0-9a-f-]{36}\.(jpg|png|webp)$`)
)

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func panjang(s string) int { return utf8.RuneCountInString(s) }

func opsional(s *string) *string {
	if s == nil {
		return nil
	}
	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}
	return &v
}

func adminAtauNull(id int64) any {
	if id > 0 {
		return id
	}
	return nil
}

func jsonDaftar(v []string) any {
	if len(v) == 0 {
		return nil
	}
	b, _ := json.Marshal(v)
	return string(b)
}

func bacaDaftar(raw sql.NullString) []string {
	out := []string{}
	if raw.Valid && raw.String != "" {
		_ = json.Unmarshal([]byte(raw.String), &out)
	}
	return out
}

// ---- Tempat ----

type Tempat struct {
	ID           int64    `json:"id"`
	Nama         string   `json:"nama"`
	NamaLengkap  *string  `json:"nama_lengkap"`
	Kategori     string   `json:"kategori"`
	Kota         string   `json:"kota"`
	FotoURL      string   `json:"foto_url"`
	Galeri       []string `json:"galeri"`
	Ringkas      string   `json:"ringkas"`
	Deskripsi    *string  `json:"deskripsi"`
	Alamat       *string  `json:"alamat"`
	GmapsURL     *string  `json:"gmaps_url"`
	Populer      bool     `json:"populer"`
	Aktif        bool     `json:"aktif"`
	Urutan       int      `json:"urutan"`
	JumlahUlasan int      `json:"jumlah_ulasan"`
}

// Input: isi form tempat.
type Input struct {
	Nama        string   `json:"nama"`
	NamaLengkap *string  `json:"nama_lengkap"`
	Kategori    string   `json:"kategori"`
	Kota        string   `json:"kota"`
	FotoURL     string   `json:"foto_url"`
	Galeri      []string `json:"galeri"`
	Ringkas     string   `json:"ringkas"`
	Deskripsi   *string  `json:"deskripsi"`
	Alamat      *string  `json:"alamat"`
	GmapsURL    *string  `json:"gmaps_url"`
	Populer     bool     `json:"populer"`
	Aktif       bool     `json:"aktif"`
}

func (in *Input) validasi() error {
	in.Nama = strings.TrimSpace(in.Nama)
	in.Ringkas = strings.TrimSpace(in.Ringkas)
	in.FotoURL = strings.TrimSpace(in.FotoURL)
	in.NamaLengkap, in.Deskripsi, in.Alamat, in.GmapsURL = opsional(in.NamaLengkap), opsional(in.Deskripsi), opsional(in.Alamat), opsional(in.GmapsURL)
	// The description comes from the admin editor as limited HTML (judul, tebal, miring, daftar): keep only the safe
	// tags, and store an empty editor as NULL.
	if in.Deskripsi != nil {
		bersih := shared.SanitasiHTML(*in.Deskripsi)
		if shared.HTMLKosong(bersih) {
			in.Deskripsi = nil
		} else {
			in.Deskripsi = &bersih
		}
	}
	switch {
	case in.Nama == "":
		return galat("Nama wajib diisi")
	case panjang(in.Nama) > 100:
		return galat("Nama maksimal 100 karakter")
	case in.NamaLengkap != nil && panjang(*in.NamaLengkap) > 150:
		return galat("Nama lengkap maksimal 150 karakter")
	case Kategori[in.Kategori] == "":
		return galat("Pilih kategori")
	case Kota[in.Kota] == "":
		return galat("Pilih kota")
	case !polaFotoTempat.MatchString(in.FotoURL):
		return galat("Unggah foto utama")
	case in.Ringkas == "":
		return galat("Ringkasan wajib diisi")
	case panjang(in.Ringkas) > 100:
		return galat("Ringkasan maksimal 100 karakter")
	case in.Deskripsi != nil && panjang(*in.Deskripsi) > maksDeskripsi:
		return galat("Deskripsi terlalu panjang")
	case in.Alamat != nil && panjang(*in.Alamat) > 300:
		return galat("Alamat maksimal 300 karakter")
	case len(in.Galeri) > maksGaleri:
		return galat("Galeri maksimal %d foto", maksGaleri)
	}
	for _, g := range in.Galeri {
		if !polaFotoTempat.MatchString(g) {
			return galat("Foto galeri tidak valid. Unggah ulang")
		}
	}
	if in.GmapsURL != nil {
		u, err := url.Parse(*in.GmapsURL)
		if err != nil || u.Scheme != "https" || u.Host == "" || len(*in.GmapsURL) > 500 {
			return galat("Link Google Maps harus diawali https://")
		}
	}
	return nil
}

// maksDeskripsi counts the stored HTML, tags included.
const maksDeskripsi = 8000

const kolomTempat = `t.id, t.nama, t.nama_lengkap, t.kategori, t.kota, t.foto_url, t.galeri, t.ringkas, t.deskripsi, t.alamat,
	t.gmaps_url, t.populer, t.aktif, t.urutan,
	(SELECT COUNT(*) FROM haramain_ulasan u WHERE u.tempat_id = t.id AND NOT u.disembunyikan)`

func scanTempat(row interface{ Scan(...any) error }) (*Tempat, error) {
	var t Tempat
	var galeri sql.NullString
	var namaLengkap, deskripsi, alamat, gmaps sql.NullString
	if err := row.Scan(&t.ID, &t.Nama, &namaLengkap, &t.Kategori, &t.Kota, &t.FotoURL, &galeri, &t.Ringkas, &deskripsi, &alamat,
		&gmaps, &t.Populer, &t.Aktif, &t.Urutan, &t.JumlahUlasan); err != nil {
		return nil, err
	}
	t.Galeri = bacaDaftar(galeri)
	for p, n := range map[**string]sql.NullString{&t.NamaLengkap: namaLengkap, &t.Deskripsi: deskripsi, &t.Alamat: alamat, &t.GmapsURL: gmaps} {
		if n.Valid {
			v := n.String
			*p = &v
		}
	}
	return &t, nil
}

// Daftar: semua tempat (hanyaAktif untuk publik), urut tampil.
func Daftar(ctx context.Context, q querier, hanyaAktif bool) ([]Tempat, error) {
	where := ""
	if hanyaAktif {
		where = "WHERE t.aktif"
	}
	rows, err := q.QueryContext(ctx, `SELECT `+kolomTempat+` FROM haramain_tempat t `+where+` ORDER BY t.urutan, t.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Tempat{}
	for rows.Next() {
		t, err := scanTempat(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

// Ambil memuat satu tempat; hanyaAktif menolak tempat nonaktif (publik).
func Ambil(ctx context.Context, q querier, id int64, hanyaAktif bool) (*Tempat, error) {
	query := `SELECT ` + kolomTempat + ` FROM haramain_tempat t WHERE t.id = ?`
	if hanyaAktif {
		query += ` AND t.aktif`
	}
	t, err := scanTempat(q.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTidakAda
	}
	return t, err
}

// Buat menambah tempat di urutan paling akhir.
func Buat(ctx context.Context, q querier, adminID int64, in Input) (int64, error) {
	if err := in.validasi(); err != nil {
		return 0, err
	}
	res, err := q.ExecContext(ctx, `
		INSERT INTO haramain_tempat (nama, nama_lengkap, kategori, kota, foto_url, galeri, ringkas, deskripsi, alamat,
			gmaps_url, populer, aktif, urutan, diubah_oleh)
		SELECT ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, COALESCE(MAX(urutan), 0) + 1, ? FROM haramain_tempat`,
		in.Nama, in.NamaLengkap, in.Kategori, in.Kota, in.FotoURL, jsonDaftar(in.Galeri), in.Ringkas, in.Deskripsi, in.Alamat,
		in.GmapsURL, in.Populer, in.Aktif, adminAtauNull(adminID))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Ubah memperbarui tempat.
func Ubah(ctx context.Context, q querier, id, adminID int64, in Input) error {
	lama, err := Ambil(ctx, q, id, false)
	if err != nil {
		return err
	}
	if err := in.validasi(); err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `
		UPDATE haramain_tempat SET nama = ?, nama_lengkap = ?, kategori = ?, kota = ?, foto_url = ?, galeri = ?, ringkas = ?,
			deskripsi = ?, alamat = ?, gmaps_url = ?, populer = ?, aktif = ?, diubah_oleh = ?
		WHERE id = ?`,
		in.Nama, in.NamaLengkap, in.Kategori, in.Kota, in.FotoURL, jsonDaftar(in.Galeri), in.Ringkas, in.Deskripsi, in.Alamat,
		in.GmapsURL, in.Populer, in.Aktif, adminAtauNull(adminID), id)
	if err != nil {
		return err
	}
	// Paket lokasi fields changed: bump linked pakets for the public updated_since filter.
	if lama.Nama != in.Nama || lama.Kota != in.Kota || lama.FotoURL != in.FotoURL || lama.Aktif != in.Aktif {
		if _, err := q.ExecContext(ctx, `
			UPDATE schedules s JOIN schedule_lokasi sl ON sl.schedule_id = s.id
			SET s.updated_at = NOW() WHERE sl.tempat_id = ?`, id); err != nil {
			return err
		}
	}
	return nil
}

// Hapus menghapus tempat beserta ulasannya. Tempat yang dipakai paket ditolak (ErrDipakaiPaket).
func Hapus(ctx context.Context, q querier, id int64) error {
	if _, err := Ambil(ctx, q, id, false); err != nil {
		return err
	}
	var dipakai bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM schedule_lokasi WHERE tempat_id = ?)`, id).Scan(&dipakai); err != nil {
		return err
	}
	if dipakai {
		return ErrDipakaiPaket
	}
	_, err := q.ExecContext(ctx, `DELETE FROM haramain_tempat WHERE id = ?`, id)
	// FK RESTRICT (1451): a paket linked the tempat after the check above.
	var me *mysql.MySQLError
	if errors.As(err, &me) && me.Number == 1451 {
		return ErrDipakaiPaket
	}
	return err
}

// Urutkan menyimpan urutan semua tempat.
func Urutkan(ctx context.Context, q querier, ids []int64) error {
	var jumlah int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM haramain_tempat`).Scan(&jumlah); err != nil {
		return err
	}
	seen := map[int64]bool{}
	for _, id := range ids {
		seen[id] = true
	}
	if len(ids) != jumlah || len(seen) != jumlah {
		return galat("Urutan tidak lengkap. Muat ulang halaman")
	}
	for i, id := range ids {
		if _, err := q.ExecContext(ctx, `UPDATE haramain_tempat SET urutan = ? WHERE id = ?`, i+1, id); err != nil {
			return err
		}
	}
	return nil
}

// ---- Ulasan ----

type Ulasan struct {
	ID       int64    `json:"id"`
	TempatID int64    `json:"tempat_id"`
	Nama     string   `json:"nama"`
	Teks     string   `json:"teks"`
	Foto     []string `json:"foto"`
	DibuatAt string   `json:"dibuat_at"` // YYYY-MM-DD HH:MM
	DiubahAt string   `json:"diubah_at"`
	// Kolom admin.
	TempatNama    string `json:"tempat_nama,omitempty"`
	Disembunyikan bool   `json:"disembunyikan,omitempty"`
	JumlahLaporan int    `json:"jumlah_laporan,omitempty"`
}

const kolomUlasan = `u.id, u.tempat_id, a.nama_lengkap, u.teks, u.foto, DATE_FORMAT(u.dibuat_at, '%Y-%m-%d %H:%i'),
	DATE_FORMAT(u.diubah_at, '%Y-%m-%d %H:%i'), t.nama, u.disembunyikan, u.jumlah_laporan`

func scanUlasan(row interface{ Scan(...any) error }) (*Ulasan, error) {
	var u Ulasan
	var foto sql.NullString
	if err := row.Scan(&u.ID, &u.TempatID, &u.Nama, &u.Teks, &foto, &u.DibuatAt, &u.DiubahAt, &u.TempatNama, &u.Disembunyikan, &u.JumlahLaporan); err != nil {
		return nil, err
	}
	u.Foto = bacaDaftar(foto)
	return &u, nil
}

func daftarUlasan(ctx context.Context, q querier, where string, args ...any) ([]Ulasan, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+kolomUlasan+` FROM haramain_ulasan u
		JOIN customer_accounts a ON a.id = u.akun_id JOIN haramain_tempat t ON t.id = u.tempat_id
		WHERE `+where, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Ulasan{}
	for rows.Next() {
		u, err := scanUlasan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *u)
	}
	return out, rows.Err()
}

// UlasanTempat: ulasan yang tampil di detail tempat (tidak disembunyikan), terbaru dulu.
func UlasanTempat(ctx context.Context, q querier, tempatID int64) ([]Ulasan, error) {
	items, err := daftarUlasan(ctx, q, `u.tempat_id = ? AND NOT u.disembunyikan ORDER BY u.dibuat_at DESC, u.id DESC LIMIT ?`, tempatID, maksUlasanTampil)
	for i := range items {
		items[i].TempatNama = ""
	}
	return items, err
}

// SudahBerangkat: akun punya booking lunas yang tanggal berangkatnya sudah lewat (WIB).
func SudahBerangkat(ctx context.Context, q querier, akunID int64, now time.Time) (bool, error) {
	var ada bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM bookings b JOIN schedules s ON s.id = b.schedule_id
		WHERE b.customer_account_id = ? AND b.status = 'lunas' AND s.berangkat_tanggal < ?)`,
		akunID, now.In(ZonaWIB).Format("2006-01-02")).Scan(&ada)
	return ada, err
}

// UlasanSaya: ulasan akun di tempat itu, atau nil.
func UlasanSaya(ctx context.Context, q querier, tempatID, akunID int64) (*Ulasan, error) {
	items, err := daftarUlasan(ctx, q, `u.tempat_id = ? AND u.akun_id = ?`, tempatID, akunID)
	if err != nil || len(items) == 0 {
		return nil, err
	}
	items[0].TempatNama = ""
	return &items[0], nil
}

// InputUlasan: isi ulasan dari app.
type InputUlasan struct {
	Teks string   `json:"teks"`
	Foto []string `json:"foto"`
}

// SimpanUlasan menulis atau memperbarui ulasan akun di tempat itu.
func SimpanUlasan(ctx context.Context, q querier, tempatID, akunID int64, in InputUlasan, now time.Time) error {
	in.Teks = strings.TrimSpace(in.Teks)
	switch {
	case in.Teks == "":
		return galat("Tulis ulasan")
	case panjang(in.Teks) > maksTeksUlasan:
		return galat("Ulasan maksimal %d karakter", maksTeksUlasan)
	case len(in.Foto) > maksFotoUlasan:
		return galat("Foto maksimal %d", maksFotoUlasan)
	}
	for _, f := range in.Foto {
		if !polaFotoUlasan.MatchString(f) {
			return galat("Foto tidak valid. Unggah ulang")
		}
	}
	if _, err := Ambil(ctx, q, tempatID, true); err != nil {
		return err
	}
	ok, err := SudahBerangkat(ctx, q, akunID, now)
	if err != nil {
		return err
	}
	if !ok {
		return ErrBelumBerangkat
	}
	_, err = q.ExecContext(ctx, `
		INSERT INTO haramain_ulasan (tempat_id, akun_id, teks, foto) VALUES (?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE teks = VALUES(teks), foto = VALUES(foto)`,
		tempatID, akunID, in.Teks, jsonDaftar(in.Foto))
	return err
}

// HapusUlasan menghapus ulasan milik akun.
func HapusUlasan(ctx context.Context, q querier, tempatID, akunID int64) error {
	res, err := q.ExecContext(ctx, `DELETE FROM haramain_ulasan WHERE tempat_id = ? AND akun_id = ?`, tempatID, akunID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrUlasanTidakAda
	}
	return nil
}

// Laporkan mencatat laporan akun atas ulasan orang lain (satu kali per akun).
func Laporkan(ctx context.Context, q querier, ulasanID, akunID int64) error {
	var pemilik int64
	err := q.QueryRowContext(ctx, `SELECT akun_id FROM haramain_ulasan WHERE id = ? AND NOT disembunyikan`, ulasanID).Scan(&pemilik)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrUlasanTidakAda
	}
	if err != nil {
		return err
	}
	if pemilik == akunID {
		return galat("Tidak bisa melaporkan ulasan sendiri")
	}
	res, err := q.ExecContext(ctx, `INSERT IGNORE INTO haramain_laporan (ulasan_id, akun_id) VALUES (?, ?)`, ulasanID, akunID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		_, err = q.ExecContext(ctx, `UPDATE haramain_ulasan SET jumlah_laporan = jumlah_laporan + 1, laporan_terakhir_at = NOW(),
			diubah_at = diubah_at WHERE id = ?`, ulasanID)
	}
	return err
}

// ---- Admin ulasan ----

// UlasanAdmin: tab "semua" (terbaru) atau "dilaporkan" (laporan belum ditinjau, belum disembunyikan).
func UlasanAdmin(ctx context.Context, q querier, dilaporkan bool) ([]Ulasan, error) {
	if dilaporkan {
		return daftarUlasan(ctx, q, `u.laporan_terakhir_at IS NOT NULL AND (u.ditinjau_at IS NULL OR u.laporan_terakhir_at > u.ditinjau_at)
			AND NOT u.disembunyikan ORDER BY u.laporan_terakhir_at DESC LIMIT 500`)
	}
	return daftarUlasan(ctx, q, `TRUE ORDER BY u.dibuat_at DESC, u.id DESC LIMIT 500`)
}

// AturTampil menyembunyikan atau menampilkan ulasan.
func AturTampil(ctx context.Context, q querier, ulasanID, adminID int64, sembunyikan bool) error {
	var oleh any
	if sembunyikan {
		oleh = adminAtauNull(adminID)
	}
	res, err := q.ExecContext(ctx, `UPDATE haramain_ulasan SET disembunyikan = ?, disembunyikan_oleh = ?, ditinjau_at = NOW(),
		diubah_at = diubah_at WHERE id = ?`, sembunyikan, oleh, ulasanID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrUlasanTidakAda
	}
	return nil
}

// Abaikan menandai laporan ulasan sudah ditinjau tanpa menyembunyikannya.
func Abaikan(ctx context.Context, q querier, ulasanID int64) error {
	res, err := q.ExecContext(ctx, `UPDATE haramain_ulasan SET ditinjau_at = NOW(), diubah_at = diubah_at WHERE id = ?`, ulasanID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrUlasanTidakAda
	}
	return nil
}
