// Package manasik: satu manasik per jadwal keberangkatan (judul, waktu, agenda, ustadz, lokasi) dan konfirmasi hadir
// (RSVP) per jamaah dari portal pelanggan. Lihat migrasi 140.
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
)

var (
	ErrTidakAda = errors.New("manasik tidak ditemukan")
	// ErrJadwalTidakAda: jadwal tidak ada atau milik brand lain.
	ErrJadwalTidakAda = errors.New("jadwal tidak ditemukan")
)

// ErrValidasi membawa pesan yang aman ditampilkan ke admin.
type ErrValidasi struct{ Pesan string }

func (e *ErrValidasi) Error() string { return e.Pesan }

// ZonaWIB: tanggal dan jam manasik dibaca sebagai waktu Indonesia Barat.
var ZonaWIB = time.FixedZone("WIB", 7*3600)

const maksAgenda = 30

var polaJam = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type Manasik struct {
	ID         int64    `json:"id"`
	ScheduleID int64    `json:"schedule_id"`
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

// validasi mengembalikan pesan galat pertama, atau "" bila isian sah. Panggil setelah rapikan.
func (in *Input) validasi() string {
	switch {
	case in.Judul == "":
		return "Judul manasik wajib diisi"
	case panjang(in.Judul) > 150:
		return "Judul manasik maksimal 150 karakter"
	}
	if _, err := time.ParseInLocation("2006-01-02", in.Tanggal, ZonaWIB); err != nil {
		return "Tanggal manasik tidak valid"
	}
	if !polaJam.MatchString(in.JamMulai) || !polaJam.MatchString(in.JamSelesai) {
		return "Jam manasik harus berformat JJ:MM"
	}
	if in.JamSelesai <= in.JamMulai {
		return "Jam selesai harus setelah jam mulai"
	}
	if len(in.Agenda) > maksAgenda {
		return fmt.Sprintf("Agenda maksimal %d butir", maksAgenda)
	}
	for _, a := range in.Agenda {
		if panjang(a) > 200 || strings.ContainsAny(a, "\r\n") {
			return "Setiap butir agenda maksimal 200 karakter dalam satu baris"
		}
	}
	switch {
	case panjang(in.Ustadz) > 150:
		return "Nama ustadz maksimal 150 karakter"
	case in.Tempat == "":
		return "Nama tempat wajib diisi"
	case panjang(in.Tempat) > 150:
		return "Nama tempat maksimal 150 karakter"
	case in.Alamat == "":
		return "Alamat wajib diisi"
	case panjang(in.Alamat) > 500:
		return "Alamat maksimal 500 karakter"
	case panjang(in.GmapsURL) > 500:
		return "Link Google Maps maksimal 500 karakter"
	}
	if in.GmapsURL != "" {
		u, err := url.Parse(in.GmapsURL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return "Link Google Maps harus diawali https://"
		}
	}
	return ""
}

// Mulai: waktu manasik dimulai (WIB). Konfirmasi hadir ditutup pada waktu ini.
func (m *Manasik) Mulai() time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", m.Tanggal+" "+m.JamMulai, ZonaWIB)
	if err != nil {
		return time.Time{}
	}
	return t
}

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

const kolomManasik = `m.id, m.schedule_id, m.judul, DATE_FORMAT(m.tanggal,'%Y-%m-%d'), TIME_FORMAT(m.jam_mulai,'%H:%i'),
	TIME_FORMAT(m.jam_selesai,'%H:%i'), COALESCE(m.agenda,''), m.ustadz, m.tempat, m.alamat, m.gmaps_url`

func scanManasik(row interface{ Scan(...any) error }) (*Manasik, error) {
	var m Manasik
	var agenda string
	var ustadz, gmaps sql.NullString
	if err := row.Scan(&m.ID, &m.ScheduleID, &m.Judul, &m.Tanggal, &m.JamMulai, &m.JamSelesai, &agenda, &ustadz, &m.Tempat, &m.Alamat, &gmaps); err != nil {
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

// cekJadwal memastikan jadwal ada dan (bila brandID terisi) milik brand itu.
func cekJadwal(ctx context.Context, q querier, scheduleID int64, brandID *int64) error {
	query, args := `SELECT 1 FROM schedules WHERE id = ?`, []any{scheduleID}
	if brandID != nil {
		query += ` AND brand_id = ?`
		args = append(args, *brandID)
	}
	var x int
	if err := q.QueryRowContext(ctx, query, args...).Scan(&x); errors.Is(err, sql.ErrNoRows) {
		return ErrJadwalTidakAda
	} else if err != nil {
		return err
	}
	return nil
}

// Ambil memuat manasik sebuah jadwal; ErrTidakAda bila belum diisi.
func Ambil(ctx context.Context, q querier, scheduleID int64) (*Manasik, error) {
	m, err := scanManasik(q.QueryRowContext(ctx, `SELECT `+kolomManasik+` FROM manasik m WHERE m.schedule_id = ?`, scheduleID))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTidakAda
	}
	return m, err
}

// AmbilAdmin memuat manasik jadwal milik brand admin (nil = semua brand). Manasik nil bila belum diisi.
func AmbilAdmin(ctx context.Context, q querier, scheduleID int64, brandID *int64) (*Manasik, error) {
	if err := cekJadwal(ctx, q, scheduleID, brandID); err != nil {
		return nil, err
	}
	m, err := Ambil(ctx, q, scheduleID)
	if errors.Is(err, ErrTidakAda) {
		return nil, nil
	}
	return m, err
}

func kosongJadiNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// Simpan membuat atau memperbarui manasik jadwal (satu per jadwal).
func Simpan(ctx context.Context, q querier, scheduleID int64, brandID *int64, adminID int64, in Input) (*Manasik, error) {
	in.rapikan()
	if msg := in.validasi(); msg != "" {
		return nil, &ErrValidasi{Pesan: msg}
	}
	if err := cekJadwal(ctx, q, scheduleID, brandID); err != nil {
		return nil, err
	}
	var admin any
	if adminID > 0 {
		admin = adminID
	}
	_, err := q.ExecContext(ctx, `
		INSERT INTO manasik (schedule_id, judul, tanggal, jam_mulai, jam_selesai, agenda, ustadz, tempat, alamat, gmaps_url, diubah_oleh)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE judul = VALUES(judul), tanggal = VALUES(tanggal), jam_mulai = VALUES(jam_mulai),
			jam_selesai = VALUES(jam_selesai), agenda = VALUES(agenda), ustadz = VALUES(ustadz), tempat = VALUES(tempat),
			alamat = VALUES(alamat), gmaps_url = VALUES(gmaps_url), diubah_oleh = VALUES(diubah_oleh)`,
		scheduleID, in.Judul, in.Tanggal, in.JamMulai, in.JamSelesai, kosongJadiNull(strings.Join(in.Agenda, "\n")),
		kosongJadiNull(in.Ustadz), in.Tempat, in.Alamat, kosongJadiNull(in.GmapsURL), admin)
	if err != nil {
		return nil, err
	}
	return Ambil(ctx, q, scheduleID)
}

// Hapus menghapus manasik jadwal beserta konfirmasi hadirnya.
func Hapus(ctx context.Context, q querier, scheduleID int64, brandID *int64) error {
	if err := cekJadwal(ctx, q, scheduleID, brandID); err != nil {
		return err
	}
	res, err := q.ExecContext(ctx, `DELETE FROM manasik WHERE schedule_id = ?`, scheduleID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTidakAda
	}
	return nil
}

// JamaahRSVP: satu jamaah aktif di jadwal beserta jawabannya (nil = belum menjawab).
type JamaahRSVP struct {
	JamaahID    int64  `json:"jamaah_id"`
	Nama        string `json:"nama"`
	BookingID   int64  `json:"booking_id,omitempty"`
	BookingCode string `json:"booking_code,omitempty"`
	Hadir       *bool  `json:"hadir"`
}

// Rekap: jumlah jawaban dan daftar jamaah aktif jadwal (untuk admin).
type Rekap struct {
	Hadir      int          `json:"hadir"`
	TidakHadir int          `json:"tidak_hadir"`
	Belum      int          `json:"belum"`
	Jamaah     []JamaahRSVP `json:"jamaah"`
}

// bookingBerlaku: booking tidak draft, tidak batal, dan bukan booking "baru" yang hold seat-nya sudah lewat (kedaluwarsa
// tanpa DP tetap berstatus baru di database). Sama dengan aturan tab "batal" di portal pelanggan (customer/orders.go).
const bookingBerlaku = `b.status NOT IN ('draft','batal')
	AND (b.status <> 'baru' OR (b.is_seat_blocked AND (b.seat_hold_expires_at IS NULL OR b.seat_hold_expires_at > NOW())))`

// paxAktif: jamaah aktif dari booking yang masih berlaku.
const paxAktif = `bp.pax_status = 'aktif' AND ` + bookingBerlaku

// RekapJadwal menyusun rekap konfirmasi hadir seluruh jamaah aktif jadwal.
func RekapJadwal(ctx context.Context, q querier, m *Manasik) (*Rekap, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, b.id, b.id_booking, r.hadir
		FROM booking_pax bp
		JOIN bookings b ON b.id = bp.booking_id
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN manasik_rsvp r ON r.manasik_id = ? AND r.jamaah_id = j.id
		WHERE b.schedule_id = ? AND `+paxAktif+`
		ORDER BY j.nama_lengkap, j.id`, m.ID, m.ScheduleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	rek := &Rekap{Jamaah: []JamaahRSVP{}}
	seen := map[int64]bool{}
	for rows.Next() {
		var it JamaahRSVP
		var hadir sql.NullBool
		if err := rows.Scan(&it.JamaahID, &it.Nama, &it.BookingID, &it.BookingCode, &hadir); err != nil {
			return nil, err
		}
		if seen[it.JamaahID] {
			continue
		}
		seen[it.JamaahID] = true
		switch {
		case !hadir.Valid:
			rek.Belum++
		case hadir.Bool:
			v := true
			it.Hadir = &v
			rek.Hadir++
		default:
			v := false
			it.Hadir = &v
			rek.TidakHadir++
		}
		rek.Jamaah = append(rek.Jamaah, it)
	}
	return rek, rows.Err()
}

// bookingAkun: jadwal sebuah booking milik akun pelanggan. ErrTidakAda bila bukan miliknya atau tidak berlaku lagi.
func bookingAkun(ctx context.Context, q querier, accountID, bookingID int64) (scheduleID int64, err error) {
	err = q.QueryRowContext(ctx, `SELECT b.schedule_id FROM bookings b WHERE b.id = ? AND b.customer_account_id = ? AND `+bookingBerlaku,
		bookingID, accountID).Scan(&scheduleID)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrTidakAda
	}
	return
}

// jamaahBooking: jamaah aktif sebuah booking beserta jawaban RSVP-nya (manasikID 0 = belum ada manasik).
func jamaahBooking(ctx context.Context, q querier, bookingID, manasikID int64) ([]JamaahRSVP, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, r.hadir
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN manasik_rsvp r ON r.manasik_id = ? AND r.jamaah_id = j.id
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'
		ORDER BY bp.id`, manasikID, bookingID)
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

// UntukBooking memuat manasik jadwal booking milik akun dan jawaban tiap jamaahnya.
func UntukBooking(ctx context.Context, q querier, accountID, bookingID int64, now time.Time) (*ManasikBooking, error) {
	scheduleID, err := bookingAkun(ctx, q, accountID, bookingID)
	if err != nil {
		return nil, err
	}
	out := &ManasikBooking{}
	var manasikID int64
	m, err := Ambil(ctx, q, scheduleID)
	switch {
	case errors.Is(err, ErrTidakAda):
	case err != nil:
		return nil, err
	default:
		out.Manasik, manasikID = m, m.ID
		out.BisaRSVP = now.Before(m.Mulai())
	}
	if out.Jamaah, err = jamaahBooking(ctx, q, bookingID, manasikID); err != nil {
		return nil, err
	}
	return out, nil
}

var (
	// ErrRSVPDitutup: manasik sudah dimulai.
	ErrRSVPDitutup = errors.New("konfirmasi kehadiran sudah ditutup karena manasik sudah dimulai")
	// ErrBelumAda: jadwal booking belum punya manasik.
	ErrBelumAda = errors.New("jadwal manasik belum diumumkan")
	// ErrBukanJamaah: jamaah tidak aktif di booking ini.
	ErrBukanJamaah = errors.New("jamaah tidak terdaftar di booking ini")
)

// SimpanRSVP mencatat jawaban hadir atau tidak hadir satu jamaah booking milik akun.
func SimpanRSVP(ctx context.Context, q querier, accountID, bookingID, jamaahID int64, hadir bool, now time.Time) error {
	scheduleID, err := bookingAkun(ctx, q, accountID, bookingID)
	if err != nil {
		return err
	}
	m, err := Ambil(ctx, q, scheduleID)
	if errors.Is(err, ErrTidakAda) {
		return ErrBelumAda
	} else if err != nil {
		return err
	}
	if !now.Before(m.Mulai()) {
		return ErrRSVPDitutup
	}
	var x int
	err = q.QueryRowContext(ctx, `SELECT 1 FROM booking_pax WHERE booking_id = ? AND jamaah_id = ? AND pax_status = 'aktif' LIMIT 1`, bookingID, jamaahID).Scan(&x)
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
