// Package manasik: manasik milik satu brand yang dikaitkan ke satu atau lebih paket (jadwal keberangkatan), konfirmasi
// hadir (RSVP) per jamaah dari portal pelanggan, dan absensi hari-H yang mengisi penanda progres "sudah manasik".
// Satu paket paling banyak satu manasik. Bayi tidak ikut manasik. Lihat migrasi 140.
package manasik

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-sql-driver/mysql"

	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/notifikasi"
)

// Pesan galat tampil apa adanya di dashboard, web, dan app: huruf kapital, titik, tanpa alasan (docs/ux-copy.md).
var (
	ErrTidakAda = errors.New("Manasik tidak ditemukan.")
	// ErrSudahDimulai: paket tidak bisa ditambah atau dilepas mulai 00.00 WIB hari manasik, supaya data absensi tetap utuh.
	ErrSudahDimulai = errors.New("Paket sudah terkunci.")
	// ErrJadwalTerkunci: mulai 00.00 WIB hari manasik, tanggal dan jam tidak bisa diubah.
	ErrJadwalTerkunci = errors.New("Tanggal dan jam sudah terkunci.")
	// ErrAdaAbsensi: manasik yang sudah punya jamaah tercatat hadir tidak bisa dihapus.
	ErrAdaAbsensi = errors.New("Manasik sudah punya absensi.")
	// ErrAbsensiBelumDibuka: absensi baru bisa diisi mulai hari manasik.
	ErrAbsensiBelumDibuka = errors.New("Absensi belum dibuka.")
	// ErrPaketSudahBermanasik: paket sudah terkait ke manasik lain (PK manasik_jadwal = schedule_id).
	ErrPaketSudahBermanasik = errors.New("Paket sudah punya manasik.")
)

// ErrValidasi membawa pesan yang aman ditampilkan ke admin.
type ErrValidasi struct{ Pesan string }

func (e *ErrValidasi) Error() string { return e.Pesan }

func validasiGalat(format string, args ...any) error {
	return &ErrValidasi{Pesan: fmt.Sprintf(format, args...)}
}

// ZonaWIB: tanggal dan jam manasik dibaca sebagai waktu Indonesia Barat.
var ZonaWIB = time.FixedZone("WIB", 7*3600)

// hariIni: tanggal hari ini (WIB) dalam format YYYY-MM-DD.
func hariIni(now time.Time) string { return now.In(ZonaWIB).Format("2006-01-02") }

const maksAgenda = 30

var polaJam = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type Manasik struct {
	ID         int64    `json:"id"`
	BrandID    int64    `json:"brand_id"`
	Judul      string   `json:"judul"`
	Tanggal    string   `json:"tanggal"`     // YYYY-MM-DD
	JamMulai   string   `json:"jam_mulai"`   // HH:MM
	JamSelesai string   `json:"jam_selesai"` // HH:MM
	Agenda     []string `json:"agenda"`
	Ustadz     *string  `json:"ustadz"`
	Tempat     string   `json:"tempat"`
	Alamat     string   `json:"alamat"`
	GmapsURL   *string  `json:"gmaps_url"`
}

// Input: isi formulir manasik dari dashboard.
type Input struct {
	Judul      string   `json:"judul"`
	Tanggal    string   `json:"tanggal"`
	JamMulai   string   `json:"jam_mulai"`
	JamSelesai string   `json:"jam_selesai"`
	Agenda     []string `json:"agenda"`
	Ustadz     string   `json:"ustadz"`
	Tempat     string   `json:"tempat"`
	Alamat     string   `json:"alamat"`
	GmapsURL   string   `json:"gmaps_url"`
}

// rapikan memangkas spasi dan membuang butir agenda kosong.
func (in *Input) rapikan() {
	in.Judul = strings.TrimSpace(in.Judul)
	in.Tanggal = strings.TrimSpace(in.Tanggal)
	in.JamMulai = strings.TrimSpace(in.JamMulai)
	in.JamSelesai = strings.TrimSpace(in.JamSelesai)
	in.Ustadz = strings.TrimSpace(in.Ustadz)
	in.Tempat = strings.TrimSpace(in.Tempat)
	in.Alamat = strings.TrimSpace(in.Alamat)
	in.GmapsURL = strings.TrimSpace(in.GmapsURL)
	agenda := make([]string, 0, len(in.Agenda))
	for _, a := range in.Agenda {
		if a = strings.TrimSpace(a); a != "" {
			agenda = append(agenda, a)
		}
	}
	in.Agenda = agenda
}

func panjang(s string) int { return utf8.RuneCountInString(s) }

// validasi mengembalikan galat validasi pertama, atau nil bila isian sah. Panggil setelah rapikan.
func (in *Input) validasi() error {
	switch {
	case in.Judul == "":
		return validasiGalat("Judul manasik wajib diisi.")
	case panjang(in.Judul) > 150:
		return validasiGalat("Judul manasik maksimal 150 karakter.")
	}
	if _, err := time.ParseInLocation("2006-01-02", in.Tanggal, ZonaWIB); err != nil {
		return validasiGalat("Tanggal manasik tidak valid.")
	}
	if !polaJam.MatchString(in.JamMulai) || !polaJam.MatchString(in.JamSelesai) {
		return validasiGalat("Jam manasik tidak valid.")
	}
	if in.JamSelesai <= in.JamMulai {
		return validasiGalat("Jam selesai harus setelah jam mulai.")
	}
	if len(in.Agenda) > maksAgenda {
		return validasiGalat("Agenda maksimal %d butir.", maksAgenda)
	}
	for _, a := range in.Agenda {
		if panjang(a) > 200 || strings.ContainsAny(a, "\r\n") {
			return validasiGalat("Butir agenda maksimal 200 karakter.")
		}
	}
	switch {
	case panjang(in.Ustadz) > 150:
		return validasiGalat("Nama ustadz maksimal 150 karakter.")
	case in.Tempat == "":
		return validasiGalat("Nama tempat wajib diisi.")
	case panjang(in.Tempat) > 150:
		return validasiGalat("Nama tempat maksimal 150 karakter.")
	case in.Alamat == "":
		return validasiGalat("Alamat wajib diisi.")
	case panjang(in.Alamat) > 500:
		return validasiGalat("Alamat maksimal 500 karakter.")
	case panjang(in.GmapsURL) > 500:
		return validasiGalat("Link Google Maps maksimal 500 karakter.")
	}
	if in.GmapsURL != "" {
		u, err := url.Parse(in.GmapsURL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return validasiGalat("Link Google Maps tidak valid.")
		}
	}
	return nil
}

// Terkunci: mulai 00.00 WIB hari manasik, paket tidak bisa ditambah atau dilepas dan tanggal/jam tidak bisa diubah
// (keputusan Q4 audit 2026-11-26). Judul, ustadz, tempat, alamat, link peta, dan agenda tetap bisa diubah.
func (m *Manasik) Terkunci(now time.Time) bool { return hariIni(now) >= m.Tanggal }

// Mulai: waktu manasik dimulai (WIB). Konfirmasi hadir ditutup pada waktu ini.
func (m *Manasik) Mulai() time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", m.Tanggal+" "+m.JamMulai, ZonaWIB)
	if err != nil {
		return time.Time{}
	}
	return t
}

// AbsensiTerbuka: absensi bisa diisi mulai hari manasik (WIB) dan tetap terbuka sesudahnya untuk koreksi.
func (m *Manasik) AbsensiTerbuka(now time.Time) bool { return hariIni(now) >= m.Tanggal }

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

const kolomManasik = `m.id, m.brand_id, m.judul, DATE_FORMAT(m.tanggal,'%Y-%m-%d'), TIME_FORMAT(m.jam_mulai,'%H:%i'),
	TIME_FORMAT(m.jam_selesai,'%H:%i'), COALESCE(m.agenda,''), m.ustadz, m.tempat, m.alamat, m.gmaps_url`

func scanManasik(row interface{ Scan(...any) error }) (*Manasik, error) {
	var m Manasik
	var agenda string
	var ustadz, gmaps sql.NullString
	if err := row.Scan(&m.ID, &m.BrandID, &m.Judul, &m.Tanggal, &m.JamMulai, &m.JamSelesai, &agenda, &ustadz, &m.Tempat, &m.Alamat, &gmaps); err != nil {
		return nil, err
	}
	m.Agenda = []string{}
	for _, a := range strings.Split(agenda, "\n") {
		if a = strings.TrimSpace(a); a != "" {
			m.Agenda = append(m.Agenda, a)
		}
	}
	if ustadz.Valid && ustadz.String != "" {
		m.Ustadz = &ustadz.String
	}
	if gmaps.Valid && gmaps.String != "" {
		m.GmapsURL = &gmaps.String
	}
	return &m, nil
}

func kosongJadiNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func adminAtauNull(adminID int64) any {
	if adminID > 0 {
		return adminID
	}
	return nil
}

// ambilAdmin memuat manasik dalam lingkup brand admin (nil = semua brand).
func ambilAdmin(ctx context.Context, q querier, id int64, brandID *int64) (*Manasik, error) {
	return muatAdmin(ctx, q, id, brandID, "")
}

// ambilAdminKunci seperti ambilAdmin, tetapi mengunci baris manasik (FOR UPDATE) supaya ubah, hapus, tambah, dan lepas
// paket yang bersamaan berjalan bergantian. Jalankan dalam transaksi.
func ambilAdminKunci(ctx context.Context, q querier, id int64, brandID *int64) (*Manasik, error) {
	return muatAdmin(ctx, q, id, brandID, " FOR UPDATE")
}

func muatAdmin(ctx context.Context, q querier, id int64, brandID *int64, kunci string) (*Manasik, error) {
	query, args := `SELECT `+kolomManasik+` FROM manasik m WHERE m.id = ?`, []any{id}
	if brandID != nil {
		query += ` AND m.brand_id = ?`
		args = append(args, *brandID)
	}
	m, err := scanManasik(q.QueryRowContext(ctx, query+kunci, args...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTidakAda
	}
	return m, err
}

// bookingBerlaku: booking tidak draft, tidak batal, dan bukan booking "baru" yang hold seat-nya sudah lewat (kedaluwarsa
// tanpa DP tetap berstatus baru di database). Sama dengan aturan tab "batal" di portal pelanggan (customer/orders.go).
const bookingBerlaku = `b.status NOT IN ('draft','batal')
	AND (b.status <> 'baru' OR (b.is_seat_blocked AND (b.seat_hold_expires_at IS NULL OR b.seat_hold_expires_at > NOW())))`

// bookingPeserta: peserta manasik (rekap, RSVP, absensi, daftar jamaah) mulai booking DP (keputusan Q5 audit 2026-11-26).
const bookingPeserta = `b.status IN ('dp','lunas')`

// paxManasik: jamaah dewasa aktif dari booking dp/lunas. Bayi tidak ikut manasik.
const paxManasik = `bp.pax_status = 'aktif' AND bp.pax_type <> 'infant' AND ` + bookingPeserta

// ---- Paket ----

// Paket: paket (jadwal keberangkatan) yang ikut atau bisa ikut sebuah manasik.
type Paket struct {
	ScheduleID       int64  `json:"schedule_id"`
	Nama             string `json:"nama"`
	BrandID          int64  `json:"brand_id"`
	BerangkatTanggal string `json:"berangkat_tanggal"`
	JumlahJamaah     int    `json:"jumlah_jamaah"`
}

// Jamaah dihitung sekali walau punya lebih dari satu booking di paket yang sama.
const kolomPaket = `s.id, s.jadwal_nama, s.brand_id, DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d'),
	(SELECT COUNT(DISTINCT bp.jamaah_id) FROM booking_pax bp JOIN bookings b ON b.id = bp.booking_id WHERE b.schedule_id = s.id AND ` + paxManasik + `)`

func daftarPaket(ctx context.Context, q querier, query string, args ...any) ([]Paket, error) {
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Paket{}
	for rows.Next() {
		var p Paket
		if err := rows.Scan(&p.ScheduleID, &p.Nama, &p.BrandID, &p.BerangkatTanggal, &p.JumlahJamaah); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// PaketManasik: paket yang ikut manasik, urut tanggal berangkat.
func PaketManasik(ctx context.Context, q querier, manasikID int64) ([]Paket, error) {
	return daftarPaket(ctx, q, `SELECT `+kolomPaket+` FROM manasik_jadwal mj JOIN schedules s ON s.id = mj.schedule_id
		WHERE mj.manasik_id = ? ORDER BY s.berangkat_tanggal, s.id`, manasikID)
}

// PaketTersedia: paket yang bisa dikaitkan ke manasik baru: belum berangkat, tidak diarsipkan, dan belum punya manasik.
// brandID membatasi ke satu brand (lingkup admin brand atau filter holding); nil = semua brand.
func PaketTersedia(ctx context.Context, q querier, brandID *int64, now time.Time) ([]Paket, error) {
	query, args := `SELECT `+kolomPaket+` FROM schedules s
		WHERE s.berangkat_tanggal > ? AND s.status <> 'archived'
			AND NOT EXISTS (SELECT 1 FROM manasik_jadwal mj WHERE mj.schedule_id = s.id)`, []any{hariIni(now)}
	if brandID != nil {
		query += ` AND s.brand_id = ?`
		args = append(args, *brandID)
	}
	return daftarPaket(ctx, q, query+` ORDER BY s.berangkat_tanggal, s.id`, args...)
}

// cekPaketBaru memastikan setiap paket ada, dalam lingkup brand, satu brand dengan brandManasik (0 = ambil dari paket
// pertama), belum berangkat, belum punya manasik, dan berangkat setelah tanggal manasik. Mengembalikan brand paket.
func cekPaketBaru(ctx context.Context, q querier, scheduleIDs []int64, scope *int64, brandManasik int64, tanggal string, now time.Time) (int64, error) {
	if len(scheduleIDs) == 0 {
		return 0, validasiGalat("Pilih minimal satu paket.")
	}
	seen := map[int64]bool{}
	for _, sid := range scheduleIDs {
		if seen[sid] {
			continue
		}
		seen[sid] = true
		var nama, berangkat, status string
		var brand int64
		var sudahAda bool
		err := q.QueryRowContext(ctx, `SELECT s.jadwal_nama, s.brand_id, DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d'), s.status,
				EXISTS (SELECT 1 FROM manasik_jadwal mj WHERE mj.schedule_id = s.id)
			FROM schedules s WHERE s.id = ? FOR UPDATE`, sid).Scan(&nama, &brand, &berangkat, &status, &sudahAda)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && scope != nil && brand != *scope) {
			return 0, validasiGalat("Paket tidak ditemukan.")
		}
		if err != nil {
			return 0, err
		}
		switch {
		case brandManasik == 0:
			brandManasik = brand
		case brand != brandManasik:
			return 0, validasiGalat("Paket %s milik brand lain.", nama)
		}
		switch {
		case status == "archived":
			return 0, validasiGalat("Paket %s sudah diarsipkan.", nama)
		case sudahAda:
			return 0, validasiGalat("Paket %s sudah punya manasik.", nama)
		case berangkat <= hariIni(now):
			return 0, validasiGalat("Paket %s sudah berangkat.", nama)
		case berangkat <= tanggal:
			return 0, validasiGalat("Tanggal manasik harus sebelum keberangkatan paket %s.", nama)
		}
	}
	return brandManasik, nil
}

// cekTanggalTerhadapPaket: tanggal manasik harus sebelum keberangkatan paling awal paket yang sudah terkait. Baris paket
// dikunci bersama (FOR SHARE) supaya tanggal berangkat tidak dimajukan bersamaan (lihat BentrokBerangkat).
func cekTanggalTerhadapPaket(ctx context.Context, q querier, manasikID int64, tanggal string) error {
	rows, err := q.QueryContext(ctx, `SELECT s.jadwal_nama, DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d')
		FROM manasik_jadwal mj JOIN schedules s ON s.id = mj.schedule_id
		WHERE mj.manasik_id = ? ORDER BY s.berangkat_tanggal, s.id FOR SHARE`, manasikID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var bentrok string
	for rows.Next() {
		var nama, berangkat string
		if err := rows.Scan(&nama, &berangkat); err != nil {
			return err
		}
		if bentrok == "" && berangkat <= tanggal {
			bentrok = nama
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if bentrok != "" {
		return validasiGalat("Tanggal manasik harus sebelum keberangkatan paket %s.", bentrok)
	}
	return nil
}

// ---- Buat, ubah, hapus ----

// Buat membuat manasik dan mengaitkan paketnya. Jalankan dalam transaksi supaya kaitan paket semua atau tidak sama sekali.
func Buat(ctx context.Context, q querier, scope *int64, adminID int64, in Input, scheduleIDs []int64, now time.Time) (int64, error) {
	in.rapikan()
	if err := in.validasi(); err != nil {
		return 0, err
	}
	if in.Tanggal < hariIni(now) {
		return 0, validasiGalat("Tanggal manasik minimal hari ini.")
	}
	brand, err := cekPaketBaru(ctx, q, scheduleIDs, scope, 0, in.Tanggal, now)
	if err != nil {
		return 0, err
	}
	res, err := q.ExecContext(ctx, `
		INSERT INTO manasik (brand_id, judul, tanggal, jam_mulai, jam_selesai, agenda, ustadz, tempat, alamat, gmaps_url, dibuat_oleh, diubah_oleh)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		brand, in.Judul, in.Tanggal, in.JamMulai, in.JamSelesai, kosongJadiNull(strings.Join(in.Agenda, "\n")),
		kosongJadiNull(in.Ustadz), in.Tempat, in.Alamat, kosongJadiNull(in.GmapsURL), adminAtauNull(adminID), adminAtauNull(adminID))
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := kaitkan(ctx, q, id, scheduleIDs); err != nil {
		return 0, err
	}
	notifikasi.Manasik(ctx, q, id, nil)
	return id, nil
}

// kaitkan mengaitkan paket ke manasik. Paket yang sudah punya manasik ditolak (ErrPaketSudahBermanasik), bukan dilewati.
func kaitkan(ctx context.Context, q querier, manasikID int64, scheduleIDs []int64) error {
	seen := map[int64]bool{}
	for _, sid := range scheduleIDs {
		if seen[sid] {
			continue
		}
		seen[sid] = true
		if _, err := q.ExecContext(ctx, `INSERT INTO manasik_jadwal (schedule_id, manasik_id) VALUES (?, ?)`, sid, manasikID); err != nil {
			var me *mysql.MySQLError
			if errors.As(err, &me) && me.Number == 1062 {
				return ErrPaketSudahBermanasik
			}
			return err
		}
	}
	return nil
}

// Ubah memperbarui isi manasik. Jawaban kehadiran jamaah tetap disimpan. Mulai hari manasik tanggal dan jam terkunci;
// tanggal baru minimal hari ini. Tidak mengirim notifikasi. Jalankan dalam transaksi.
func Ubah(ctx context.Context, q querier, id int64, scope *int64, adminID int64, in Input, now time.Time) error {
	in.rapikan()
	if err := in.validasi(); err != nil {
		return err
	}
	m, err := ambilAdminKunci(ctx, q, id, scope)
	if err != nil {
		return err
	}
	jadwalBerubah := in.Tanggal != m.Tanggal || in.JamMulai != m.JamMulai || in.JamSelesai != m.JamSelesai
	if jadwalBerubah && m.Terkunci(now) {
		return ErrJadwalTerkunci
	}
	if in.Tanggal != m.Tanggal && in.Tanggal < hariIni(now) {
		return validasiGalat("Tanggal manasik minimal hari ini.")
	}
	if err := cekTanggalTerhadapPaket(ctx, q, id, in.Tanggal); err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `
		UPDATE manasik SET judul = ?, tanggal = ?, jam_mulai = ?, jam_selesai = ?, agenda = ?, ustadz = ?, tempat = ?,
			alamat = ?, gmaps_url = ?, diubah_oleh = ?
		WHERE id = ?`,
		in.Judul, in.Tanggal, in.JamMulai, in.JamSelesai, kosongJadiNull(strings.Join(in.Agenda, "\n")),
		kosongJadiNull(in.Ustadz), in.Tempat, in.Alamat, kosongJadiNull(in.GmapsURL), adminAtauNull(adminID), id)
	return err
}

// TambahPaket mengaitkan paket lain (brand yang sama) ke manasik sebelum hari manasik; jamaah paket baru diberi
// notifikasi. Jalankan dalam transaksi.
func TambahPaket(ctx context.Context, q querier, id int64, scope *int64, scheduleIDs []int64, now time.Time) error {
	m, err := ambilAdminKunci(ctx, q, id, scope)
	if err != nil {
		return err
	}
	if m.Terkunci(now) {
		return ErrSudahDimulai
	}
	if _, err := cekPaketBaru(ctx, q, scheduleIDs, scope, m.BrandID, m.Tanggal, now); err != nil {
		return err
	}
	if err := kaitkan(ctx, q, id, scheduleIDs); err != nil {
		return err
	}
	notifikasi.Manasik(ctx, q, id, scheduleIDs)
	return nil
}

// LepasPaket melepas paket dari manasik sebelum hari manasik; jawaban kehadiran jamaah paket itu ikut dihapus.
// Manasik minimal punya satu paket. Jalankan dalam transaksi.
func LepasPaket(ctx context.Context, q querier, id int64, scope *int64, scheduleID int64, now time.Time) error {
	m, err := ambilAdminKunci(ctx, q, id, scope)
	if err != nil {
		return err
	}
	if m.Terkunci(now) {
		return ErrSudahDimulai
	}
	var jumlah int
	var terkait bool
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(SUM(schedule_id = ?), 0) > 0 FROM manasik_jadwal WHERE manasik_id = ?`,
		scheduleID, id).Scan(&jumlah, &terkait); err != nil {
		return err
	}
	if !terkait {
		return validasiGalat("Paket tidak ada di manasik ini.")
	}
	if jumlah <= 1 {
		return validasiGalat("Manasik minimal punya satu paket.")
	}
	if _, err := q.ExecContext(ctx, `DELETE r FROM manasik_rsvp r JOIN bookings b ON b.id = r.booking_id
		WHERE r.manasik_id = ? AND b.schedule_id = ?`, id, scheduleID); err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `DELETE FROM manasik_jadwal WHERE manasik_id = ? AND schedule_id = ?`, id, scheduleID)
	return err
}

// adaAbsensi: ada jamaah paket manasik yang sudah ditandai "sudah manasik". Penanda hanya diisi absensi manasik
// (keputusan Q6), tetapi data lama bisa saja terisi sebelum hari manasik, jadi tidak bergantung pada jendela absensi.
func adaAbsensi(ctx context.Context, q querier, id int64) (bool, error) {
	var ada bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM manasik_jadwal mj
			JOIN bookings b ON b.schedule_id = mj.schedule_id
			JOIN booking_pax bp ON bp.booking_id = b.id
		WHERE mj.manasik_id = ? AND `+paxManasik+` AND bp.progress_manasik)`, id).Scan(&ada)
	return ada, err
}

// Hapus menghapus manasik beserta kaitan paket dan jawaban kehadirannya. Ditolak bila sudah ada absensi. Jalankan dalam
// transaksi.
func Hapus(ctx context.Context, q querier, id int64, scope *int64, now time.Time) error {
	if _, err := ambilAdminKunci(ctx, q, id, scope); err != nil {
		return err
	}
	ada, err := adaAbsensi(ctx, q, id)
	if err != nil {
		return err
	}
	if ada {
		return ErrAdaAbsensi
	}
	_, err = q.ExecContext(ctx, `DELETE FROM manasik WHERE id = ?`, id)
	return err
}

// ---- Daftar dan detail ----

// Ringkas: satu baris di daftar manasik.
type Ringkas struct {
	ID           int64  `json:"id"`
	BrandID      int64  `json:"brand_id"`
	BrandNama    string `json:"brand_nama"`
	Judul        string `json:"judul"`
	Tanggal      string `json:"tanggal"`
	JamMulai     string `json:"jam_mulai"`
	JamSelesai   string `json:"jam_selesai"`
	Tempat       string `json:"tempat"`
	JumlahPaket  int    `json:"jumlah_paket"`
	JumlahJamaah int    `json:"jumlah_jamaah"`
	BerniatHadir int    `json:"berniat_hadir"`
	Hadir        int    `json:"hadir"`
}

// Daftar memuat manasik dalam lingkup brand. selesai=false: hari ini dan sesudahnya (urut terdekat);
// selesai=true: sebelum hari ini (urut terbaru).
func Daftar(ctx context.Context, q querier, brandID *int64, selesai bool, now time.Time) ([]Ringkas, error) {
	where, urut := `m.tanggal >= ?`, `m.tanggal, m.jam_mulai, m.id`
	if selesai {
		where, urut = `m.tanggal < ?`, `m.tanggal DESC, m.jam_mulai DESC, m.id DESC`
	}
	args := []any{hariIni(now)}
	if brandID != nil {
		where += ` AND m.brand_id = ?`
		args = append(args, *brandID)
	}
	rows, err := q.QueryContext(ctx, `
		SELECT m.id, m.brand_id, br.name, m.judul, DATE_FORMAT(m.tanggal,'%Y-%m-%d'), TIME_FORMAT(m.jam_mulai,'%H:%i'),
			TIME_FORMAT(m.jam_selesai,'%H:%i'), m.tempat,
			(SELECT COUNT(*) FROM manasik_jadwal mj WHERE mj.manasik_id = m.id),
			COALESCE(px.jamaah, 0), COALESCE(px.berniat, 0), COALESCE(px.hadir, 0)
		FROM manasik m
		JOIN brands br ON br.id = m.brand_id
		LEFT JOIN (
			SELECT mj.manasik_id, COUNT(DISTINCT bp.jamaah_id) AS jamaah,
				COUNT(DISTINCT IF(r.hadir, bp.jamaah_id, NULL)) AS berniat,
				COUNT(DISTINCT IF(bp.progress_manasik, bp.jamaah_id, NULL)) AS hadir
			FROM manasik_jadwal mj
			JOIN bookings b ON b.schedule_id = mj.schedule_id
			JOIN booking_pax bp ON bp.booking_id = b.id
			LEFT JOIN manasik_rsvp r ON r.manasik_id = mj.manasik_id AND r.jamaah_id = bp.jamaah_id
			WHERE `+paxManasik+`
			GROUP BY mj.manasik_id
		) px ON px.manasik_id = m.id
		WHERE `+where+` ORDER BY `+urut, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Ringkas{}
	for rows.Next() {
		var r Ringkas
		if err := rows.Scan(&r.ID, &r.BrandID, &r.BrandNama, &r.Judul, &r.Tanggal, &r.JamMulai, &r.JamSelesai, &r.Tempat,
			&r.JumlahPaket, &r.JumlahJamaah, &r.BerniatHadir, &r.Hadir); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// JamaahManasik: satu jamaah dewasa paket manasik beserta jawaban di app dan catatan kehadirannya.
type JamaahManasik struct {
	PaxID       int64   `json:"pax_id"`
	JamaahID    int64   `json:"jamaah_id"`
	Nama        string  `json:"nama"`
	ScheduleID  int64   `json:"schedule_id"`
	BookingID   int64   `json:"booking_id"`
	BookingCode string  `json:"booking_code"`
	Jawaban     *bool   `json:"jawaban"` // nil = belum menjawab
	Hadir       bool    `json:"hadir"`   // penanda progres "sudah manasik"
	DicatatOleh *string `json:"dicatat_oleh"`
	DicatatAt   *string `json:"dicatat_at"` // YYYY-MM-DD HH:MM
}

// Rekap: jumlah jawaban di app dan kehadiran nyata.
type Rekap struct {
	Jamaah       int `json:"jamaah"`
	BerniatHadir int `json:"berniat_hadir"`
	TidakHadir   int `json:"tidak_hadir"`
	Belum        int `json:"belum"`
	Hadir        int `json:"hadir"`
}

// Detail: semua data halaman detail manasik.
type Detail struct {
	Manasik        *Manasik        `json:"manasik"`
	BrandNama      string          `json:"brand_nama"`
	Paket          []Paket         `json:"paket"`
	Rekap          Rekap           `json:"rekap"`
	Jamaah         []JamaahManasik `json:"jamaah"`
	SudahDimulai   bool            `json:"sudah_dimulai"`
	AbsensiTerbuka bool            `json:"absensi_terbuka"`
	AdaAbsensi     bool            `json:"ada_absensi"` // true: manasik tidak bisa dihapus
}

// AmbilDetail memuat manasik, paket, rekap, dan daftar jamaahnya.
func AmbilDetail(ctx context.Context, q querier, id int64, scope *int64, now time.Time) (*Detail, error) {
	m, err := ambilAdmin(ctx, q, id, scope)
	if err != nil {
		return nil, err
	}
	d := &Detail{Manasik: m, SudahDimulai: m.Terkunci(now), AbsensiTerbuka: m.AbsensiTerbuka(now)}
	if err := q.QueryRowContext(ctx, `SELECT name FROM brands WHERE id = ?`, m.BrandID).Scan(&d.BrandNama); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	if d.Paket, err = PaketManasik(ctx, q, id); err != nil {
		return nil, err
	}
	if d.AdaAbsensi, err = adaAbsensi(ctx, q, id); err != nil {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `
		SELECT bp.id, j.id, j.nama_lengkap, b.schedule_id, b.id, b.id_booking, r.hadir, COALESCE(bp.progress_manasik, FALSE),
			COALESCE(NULLIF(TRIM(au.display_name), ''), au.email), DATE_FORMAT(bp.progress_manasik_at, '%Y-%m-%d %H:%i')
		FROM manasik_jadwal mj
		JOIN schedules s ON s.id = mj.schedule_id
		JOIN bookings b ON b.schedule_id = mj.schedule_id
		JOIN booking_pax bp ON bp.booking_id = b.id
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN manasik_rsvp r ON r.manasik_id = mj.manasik_id AND r.jamaah_id = bp.jamaah_id
		LEFT JOIN admin_users au ON au.id = bp.progress_manasik_oleh
		WHERE mj.manasik_id = ? AND `+paxManasik+`
		ORDER BY s.berangkat_tanggal, s.id, j.nama_lengkap, bp.id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d.Jamaah = []JamaahManasik{}
	sudah := map[int64]bool{}
	for rows.Next() {
		var it JamaahManasik
		var jawaban sql.NullBool
		var oleh, at sql.NullString
		if err := rows.Scan(&it.PaxID, &it.JamaahID, &it.Nama, &it.ScheduleID, &it.BookingID, &it.BookingCode, &jawaban,
			&it.Hadir, &oleh, &at); err != nil {
			return nil, err
		}
		// Jamaah dengan lebih dari satu booking di paket manasik tampil dan dihitung sekali (baris pertama).
		if sudah[it.JamaahID] {
			continue
		}
		sudah[it.JamaahID] = true
		d.Rekap.Jamaah++
		switch {
		case !jawaban.Valid:
			d.Rekap.Belum++
		case jawaban.Bool:
			d.Rekap.BerniatHadir++
		default:
			d.Rekap.TidakHadir++
		}
		if jawaban.Valid {
			v := jawaban.Bool
			it.Jawaban = &v
		}
		if it.Hadir {
			d.Rekap.Hadir++
		}
		if oleh.Valid {
			it.DicatatOleh = &oleh.String
		}
		if at.Valid {
			it.DicatatAt = &at.String
		}
		d.Jamaah = append(d.Jamaah, it)
	}
	return d, rows.Err()
}

// CatatHadir mengisi atau mengosongkan penanda "sudah manasik" seorang jamaah paket manasik, mulai hari manasik.
func CatatHadir(ctx context.Context, q querier, id int64, scope *int64, adminID, paxID int64, hadir bool, now time.Time) error {
	m, err := ambilAdmin(ctx, q, id, scope)
	if err != nil {
		return err
	}
	if !m.AbsensiTerbuka(now) {
		return ErrAbsensiBelumDibuka
	}
	var jamaahID int64
	err = q.QueryRowContext(ctx, `SELECT bp.jamaah_id FROM booking_pax bp JOIN bookings b ON b.id = bp.booking_id
		JOIN manasik_jadwal mj ON mj.schedule_id = b.schedule_id AND mj.manasik_id = ?
		WHERE bp.id = ? AND `+paxManasik+` FOR UPDATE`, id, paxID).Scan(&jamaahID)
	if errors.Is(err, sql.ErrNoRows) {
		return validasiGalat("Jamaah tidak terdaftar di manasik ini.")
	}
	if err != nil {
		return err
	}
	// Kehadiran per orang: semua pax jamaah itu di paket manasik ikut dicatat, supaya jamaah dengan dua booking
	// tidak tampil hadir di satu booking dan belum di booking lain.
	rows, err := q.QueryContext(ctx, `SELECT bp.id FROM booking_pax bp JOIN bookings b ON b.id = bp.booking_id
		JOIN manasik_jadwal mj ON mj.schedule_id = b.schedule_id AND mj.manasik_id = ?
		WHERE bp.jamaah_id = ? AND `+paxManasik+` FOR UPDATE`, id, jamaahID)
	if err != nil {
		return err
	}
	var paxIDs []int64
	for rows.Next() {
		var pid int64
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		paxIDs = append(paxIDs, pid)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, pid := range paxIDs {
		// Penanda manasik milik modul booking; nilai sama menghasilkan false dan tetap dianggap berhasil.
		if _, err := booking.CatatProgressManasik(ctx, q, pid, hadir, adminID); err != nil {
			return err
		}
	}
	return nil
}

// ManasikPaket: ringkasan manasik sebuah paket untuk halaman detail paket (nil = belum punya manasik).
type ManasikPaket struct {
	ID         int64  `json:"id"`
	Judul      string `json:"judul"`
	Tanggal    string `json:"tanggal"`
	JamMulai   string `json:"jam_mulai"`
	JamSelesai string `json:"jam_selesai"`
}

// UntukPaket memuat ringkasan manasik paket dalam lingkup brand admin.
func UntukPaket(ctx context.Context, q querier, scheduleID int64, scope *int64) (*ManasikPaket, error) {
	query, args := `SELECT m.id, m.judul, DATE_FORMAT(m.tanggal,'%Y-%m-%d'), TIME_FORMAT(m.jam_mulai,'%H:%i'),
			TIME_FORMAT(m.jam_selesai,'%H:%i')
		FROM manasik_jadwal mj JOIN manasik m ON m.id = mj.manasik_id WHERE mj.schedule_id = ?`, []any{scheduleID}
	if scope != nil {
		query += ` AND m.brand_id = ?`
		args = append(args, *scope)
	}
	var p ManasikPaket
	err := q.QueryRowContext(ctx, query, args...).Scan(&p.ID, &p.Judul, &p.Tanggal, &p.JamMulai, &p.JamSelesai)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

// ---- Portal pelanggan ----

// ambilUntukJadwal memuat manasik yang dikaitkan ke sebuah paket; ErrTidakAda bila belum ada.
func ambilUntukJadwal(ctx context.Context, q querier, scheduleID int64) (*Manasik, error) {
	m, err := scanManasik(q.QueryRowContext(ctx, `SELECT `+kolomManasik+` FROM manasik_jadwal mj JOIN manasik m ON m.id = mj.manasik_id
		WHERE mj.schedule_id = ?`, scheduleID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTidakAda
	}
	return m, err
}

// JamaahRSVP: satu jamaah dewasa booking beserta jawabannya (nil = belum menjawab).
type JamaahRSVP struct {
	JamaahID int64  `json:"jamaah_id"`
	Nama     string `json:"nama"`
	Hadir    *bool  `json:"hadir"`
}

// bookingAkun: paket sebuah booking milik akun pelanggan, atau booking orang lain yang memuat jamaah tertaut
// ke akun itu (peran jamaah). ErrTidakAda bila bukan keduanya atau tidak berlaku lagi. peserta: booking sudah DP
// (dp/lunas), syarat ikut manasik.
func bookingAkun(ctx context.Context, q querier, accountID, bookingID int64) (scheduleID int64, peserta bool, err error) {
	err = q.QueryRowContext(ctx, `SELECT b.schedule_id, `+bookingPeserta+` FROM bookings b WHERE b.id = ? AND (b.customer_account_id = ? OR EXISTS (
			SELECT 1 FROM booking_pax bpj JOIN customer_accounts caj ON caj.jamaah_id = bpj.jamaah_id
			WHERE bpj.booking_id = b.id AND bpj.pax_status = 'aktif' AND caj.id = ? AND caj.dihapus_at IS NULL)) AND `+bookingBerlaku,
		bookingID, accountID, accountID).Scan(&scheduleID, &peserta)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrTidakAda
	}
	return
}

// jamaahBooking: jamaah dewasa aktif sebuah booking beserta jawaban RSVP-nya (manasikID 0 = belum ada manasik).
// Pemesan melihat semua; peran jamaah hanya jamaah yang tertaut ke akunnya.
func jamaahBooking(ctx context.Context, q querier, accountID, bookingID, manasikID int64) ([]JamaahRSVP, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, r.hadir
		FROM booking_pax bp
		JOIN bookings b ON b.id = bp.booking_id
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN manasik_rsvp r ON r.manasik_id = ? AND r.jamaah_id = j.id
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant'
		  AND (b.customer_account_id = ? OR EXISTS (SELECT 1 FROM customer_accounts caj WHERE caj.id = ? AND caj.jamaah_id = bp.jamaah_id))
		ORDER BY bp.id`, manasikID, bookingID, accountID, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []JamaahRSVP{}
	for rows.Next() {
		var it JamaahRSVP
		var hadir sql.NullBool
		if err := rows.Scan(&it.JamaahID, &it.Nama, &hadir); err != nil {
			return nil, err
		}
		if hadir.Valid {
			v := hadir.Bool
			it.Hadir = &v
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// ManasikBooking: tampilan manasik untuk satu booking pelanggan.
type ManasikBooking struct {
	Manasik *Manasik     `json:"manasik"`
	Jamaah  []JamaahRSVP `json:"jamaah"`
	// BisaRSVP: manasik ada dan belum dimulai.
	BisaRSVP bool `json:"bisa_rsvp"`
}

// UntukBooking memuat manasik paket booking milik akun dan jawaban tiap jamaah dewasanya. Booking yang belum DP
// mendapat tampilan kosong (manasik null, tanpa jamaah) dengan struktur yang sama.
func UntukBooking(ctx context.Context, q querier, accountID, bookingID int64, now time.Time) (*ManasikBooking, error) {
	scheduleID, peserta, err := bookingAkun(ctx, q, accountID, bookingID)
	if err != nil {
		return nil, err
	}
	out := &ManasikBooking{Jamaah: []JamaahRSVP{}}
	if !peserta {
		return out, nil
	}
	var manasikID int64
	m, err := ambilUntukJadwal(ctx, q, scheduleID)
	switch {
	case errors.Is(err, ErrTidakAda):
	case err != nil:
		return nil, err
	default:
		out.Manasik, manasikID = m, m.ID
		out.BisaRSVP = now.Before(m.Mulai())
	}
	if out.Jamaah, err = jamaahBooking(ctx, q, accountID, bookingID, manasikID); err != nil {
		return nil, err
	}
	return out, nil
}

var (
	// ErrRSVPDitutup: manasik sudah dimulai.
	ErrRSVPDitutup = errors.New("Konfirmasi hadir sudah ditutup.")
	// ErrBelumAda: paket booking belum punya manasik.
	ErrBelumAda = errors.New("Jadwal manasik belum ada.")
	// ErrBelumDP: booking belum DP, jadi belum ikut manasik.
	ErrBelumDP = errors.New("Booking belum DP.")
	// ErrBukanJamaah: jamaah tidak aktif di booking ini, atau bayi.
	ErrBukanJamaah = errors.New("Jamaah tidak terdaftar di booking ini.")
)

// SimpanRSVP mencatat jawaban hadir atau tidak hadir satu jamaah dewasa booking milik akun.
func SimpanRSVP(ctx context.Context, q querier, accountID, bookingID, jamaahID int64, hadir bool, now time.Time) error {
	scheduleID, peserta, err := bookingAkun(ctx, q, accountID, bookingID)
	if err != nil {
		return err
	}
	if !peserta {
		return ErrBelumDP
	}
	m, err := ambilUntukJadwal(ctx, q, scheduleID)
	if errors.Is(err, ErrTidakAda) {
		return ErrBelumAda
	} else if err != nil {
		return err
	}
	if !now.Before(m.Mulai()) {
		return ErrRSVPDitutup
	}
	var x int
	// Peran jamaah (bukan pemesan) hanya menjawab untuk jamaah yang tertaut ke akunnya.
	err = q.QueryRowContext(ctx, `SELECT 1 FROM booking_pax bp JOIN bookings b ON b.id = bp.booking_id
		WHERE bp.booking_id = ? AND bp.jamaah_id = ? AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant'
		  AND (b.customer_account_id = ? OR EXISTS (SELECT 1 FROM customer_accounts caj WHERE caj.id = ? AND caj.jamaah_id = bp.jamaah_id))
		LIMIT 1`, bookingID, jamaahID, accountID, accountID).Scan(&x)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrBukanJamaah
	} else if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `
		INSERT INTO manasik_rsvp (manasik_id, jamaah_id, booking_id, hadir, dijawab_oleh) VALUES (?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE booking_id = VALUES(booking_id), hadir = VALUES(hadir), dijawab_oleh = VALUES(dijawab_oleh)`,
		m.ID, jamaahID, bookingID, hadir, accountID)
	return err
}
