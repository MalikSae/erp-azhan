package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

// Layar admin Akun Pelanggan: daftar dan detail akun yang bisa booking (hanya baca). Upline satu konsep:
// agen yang mengajak akun dan menerima bonus pembinaan darinya.

var ErrFilterAkun = errors.New("filter akun tidak valid")

type adminQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type FilterAkun struct {
	Cari       string
	StatusAgen string // tidak_aktif | pengajuan | aktif | nonaktif
	Upline     string // ada | tanpa
	Agen       string // agen (aktif atau nonaktif) | bukan
	Page       int
	PerPage    int
}

type AkunBaris struct {
	AkunID       int64      `json:"akun_id"`
	NamaLengkap  string     `json:"nama_lengkap"`
	Email        string     `json:"email"`
	NoHP         string     `json:"no_hp"`
	BrandID      int64      `json:"brand_id"`
	BrandName    string     `json:"brand_name"`
	StatusAgen   string     `json:"status_agen"`
	KodeReferral *string    `json:"kode_referral"`
	UplineID     *int64     `json:"upline_akun_id"`
	UplineNama   *string    `json:"upline_nama"`
	// JumlahBooking: semua booking yang dibuat akun ini, termasuk yang batal (sama dengan angka di detail akun).
	JumlahBooking int        `json:"jumlah_booking"`
	CreatedAt     time.Time  `json:"created_at"`
	LastLoginAt   *time.Time `json:"last_login_at"`
}

type HasilListAkun struct {
	Items   []AkunBaris `json:"items"`
	Total   int         `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
}

type JamaahRingkas struct {
	ID   int64  `json:"id"`
	Nama string `json:"nama_lengkap"`
}

// LogData: satu perubahan data akun oleh admin. NIK agen sudah disamarkan saat dicatat.
type LogData struct {
	Kolom      string    `json:"kolom"`
	NilaiLama  *string   `json:"nilai_lama"`
	NilaiBaru  *string   `json:"nilai_baru"`
	Alasan     string    `json:"alasan"`
	DiubahOleh string    `json:"diubah_oleh"`
	DiubahAt   time.Time `json:"diubah_at"`
}

// BookingAkun: satu booking yang dibuat akun ini, untuk riwayat booking di detail akun.
type BookingAkun struct {
	ID               int64   `json:"id"`
	IDBooking        *string `json:"id_booking"`
	JadwalNama       string  `json:"jadwal_nama"`
	BerangkatTanggal *string `json:"berangkat_tanggal"` // YYYY-MM-DD
	Status           string  `json:"status"`
	JumlahPax        int     `json:"jumlah_pax"`
}

type AkunDetail struct {
	AkunID          int64      `json:"akun_id"`
	BrandID         int64      `json:"brand_id"`
	BrandName       string     `json:"brand_name"`
	NamaLengkap     string     `json:"nama_lengkap"`
	Email           string     `json:"email"`
	NoHP            string     `json:"no_hp"`
	Domisili        *string    `json:"domisili"`
	TanggalLahir    *string    `json:"tanggal_lahir"` // YYYY-MM-DD, diisi pemilik akun atau saat mendaftar agen
	JenisKelamin    *string    `json:"jenis_kelamin"` // L atau P
	CreatedAt       time.Time  `json:"created_at"`
	LastLoginAt     *time.Time `json:"last_login_at"`
	FotoURL         *string    `json:"foto_url"` // privat; admin membukanya lewat media terlindungi
	// Upline = agen yang mengajak akun ini (direkrut_oleh_akun_id); untuk agen sama dengan upline_akun_id penerima bonus pembinaan.
	UplineID   *int64  `json:"upline_akun_id"`
	UplineNama *string `json:"upline_nama"`
	// Agen
	StatusAgen       string     `json:"status_agen"`
	KodeReferral     *string    `json:"kode_referral"`
	DiajukanAt       *time.Time `json:"diajukan_agen_at"`
	DisetujuiAt      *time.Time `json:"disetujui_agen_at"`
	NIKAgen          *string    `json:"nik_agen"` // disamarkan
	JumlahJamaahAgen int        `json:"jumlah_jamaah_agen"`
	JumlahDiajak   int        `json:"jumlah_diajak"`
	JumlahBooking    int        `json:"jumlah_booking"`
	// Booking: semua booking akun ini, terbaru dulu, termasuk yang batal.
	Booking []BookingAkun `json:"booking"`
	// Jamaah milik akun sendiri (yang dipakai sebagai PIC booking).
	Jamaah *JamaahRingkas        `json:"jamaah"`
	Log    []agen.AkunKaitanLog `json:"riwayat_upline"`
	// Perubahan nama, WhatsApp, domisili, dan NIK agen oleh admin (terbaru dulu).
	RiwayatData []LogData `json:"riwayat_data"`
}

// samarkanNIK menyisakan 4 angka awal dan 2 angka akhir.
func samarkanNIK(nik string) string {
	if len(nik) <= 6 {
		return strings.Repeat("*", len(nik))
	}
	return nik[:4] + strings.Repeat("*", len(nik)-6) + nik[len(nik)-2:]
}

func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}

func listAkun(ctx context.Context, q adminQuerier, brandID *int64, f FilterAkun) (HasilListAkun, error) {
	if f.StatusAgen != "" {
		switch f.StatusAgen {
		case "tidak_aktif", "pengajuan", "aktif", "nonaktif":
		default:
			return HasilListAkun{}, ErrFilterAkun
		}
	}
	if f.Upline != "" && f.Upline != "ada" && f.Upline != "tanpa" {
		return HasilListAkun{}, ErrFilterAkun
	}
	if f.Agen != "" && f.Agen != "agen" && f.Agen != "bukan" {
		return HasilListAkun{}, ErrFilterAkun
	}
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}

	where, args := []string{"1=1"}, []any{}
	if brandID != nil {
		where, args = append(where, "a.brand_id = ?"), append(args, *brandID)
	}
	if f.StatusAgen != "" {
		where, args = append(where, "a.status_agen = ?"), append(args, f.StatusAgen)
	}
	// Agen = akun yang punya kode referral, aktif maupun nonaktif. Calon agen (pengajuan) belum termasuk.
	switch f.Agen {
	case "agen":
		where = append(where, "a.status_agen IN ('aktif','nonaktif')")
	case "bukan":
		where = append(where, "a.status_agen NOT IN ('aktif','nonaktif')")
	}
	switch f.Upline {
	case "ada":
		where = append(where, "a.direkrut_oleh_akun_id IS NOT NULL")
	case "tanpa":
		where = append(where, "a.direkrut_oleh_akun_id IS NULL")
	}
	if cari := strings.TrimSpace(f.Cari); cari != "" {
		like := "%" + escapeLike(cari) + "%"
		where = append(where, "(a.nama_lengkap LIKE ? OR a.email LIKE ? OR a.no_hp LIKE ? OR a.kode_referral LIKE ?)")
		args = append(args, like, like, like, like)
	}
	cond := strings.Join(where, " AND ")

	out := HasilListAkun{Items: []AkunBaris{}, Page: f.Page, PerPage: f.PerPage}
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM customer_accounts a WHERE `+cond, args...).Scan(&out.Total); err != nil {
		return out, fmt.Errorf("customer: hitung akun: %w", err)
	}
	rows, err := q.QueryContext(ctx, `
		SELECT a.id, a.nama_lengkap, a.email, a.no_hp, a.brand_id, b.name, a.status_agen, a.kode_referral,
		       a.direkrut_oleh_akun_id, up.nama_lengkap,
		       (SELECT COUNT(*) FROM bookings WHERE customer_account_id = a.id), a.created_at, a.last_login_at
		FROM customer_accounts a
		JOIN brands b ON b.id = a.brand_id
		LEFT JOIN customer_accounts up ON up.id = a.direkrut_oleh_akun_id
		WHERE `+cond+` ORDER BY a.created_at DESC, a.id DESC LIMIT ? OFFSET ?`,
		append(args, f.PerPage, (f.Page-1)*f.PerPage)...)
	if err != nil {
		return out, fmt.Errorf("customer: daftar akun: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var a AkunBaris
		var upline sql.NullInt64
		if err := rows.Scan(&a.AkunID, &a.NamaLengkap, &a.Email, &a.NoHP, &a.BrandID, &a.BrandName, &a.StatusAgen, &a.KodeReferral,
			&upline, &a.UplineNama, &a.JumlahBooking, &a.CreatedAt, &a.LastLoginAt); err != nil {
			return out, fmt.Errorf("customer: scan akun: %w", err)
		}
		if upline.Valid {
			a.UplineID = &upline.Int64
		}
		out.Items = append(out.Items, a)
	}
	return out, rows.Err()
}

// getAkunDetail: brandID nil = semua brand; admin brand hanya melihat akun brand-nya. Riwayat upline diisi pemanggil.
func getAkunDetail(ctx context.Context, q adminQuerier, brandID *int64, akunID int64) (*AkunDetail, error) {
	var d AkunDetail
	var upline, jamaahID sql.NullInt64
	var nik, jamaahNama sql.NullString
	err := q.QueryRowContext(ctx, `
		SELECT a.id, a.brand_id, b.name, a.nama_lengkap, a.email, a.no_hp, a.domisili, a.created_at, a.last_login_at,
		       a.direkrut_oleh_akun_id, up.nama_lengkap, a.status_agen, a.kode_referral, a.diajukan_agen_at, a.disetujui_agen_at, a.nik_agen,
		       a.jamaah_id, j.nama_lengkap,
		       (SELECT COUNT(*) FROM jamaah WHERE direkrut_oleh_akun_id = a.id),
		       (SELECT COUNT(*) FROM customer_accounts WHERE direkrut_oleh_akun_id = a.id),
		       (SELECT COUNT(*) FROM bookings WHERE customer_account_id = a.id), a.foto_url,
		       DATE_FORMAT(a.tanggal_lahir, '%Y-%m-%d'), a.jenis_kelamin
		FROM customer_accounts a
		JOIN brands b ON b.id = a.brand_id
		LEFT JOIN customer_accounts up ON up.id = a.direkrut_oleh_akun_id
		LEFT JOIN jamaah j ON j.id = a.jamaah_id
		WHERE a.id = ?`, akunID).
		Scan(&d.AkunID, &d.BrandID, &d.BrandName, &d.NamaLengkap, &d.Email, &d.NoHP, &d.Domisili, &d.CreatedAt, &d.LastLoginAt,
			&upline, &d.UplineNama, &d.StatusAgen, &d.KodeReferral, &d.DiajukanAt, &d.DisetujuiAt, &nik,
			&jamaahID, &jamaahNama, &d.JumlahJamaahAgen, &d.JumlahDiajak, &d.JumlahBooking, &d.FotoURL, &d.TanggalLahir, &d.JenisKelamin)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != d.BrandID) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("customer: detail akun: %w", err)
	}
	if upline.Valid {
		d.UplineID = &upline.Int64
	}
	if nik.Valid && nik.String != "" {
		m := samarkanNIK(nik.String)
		d.NIKAgen = &m
	}
	if jamaahID.Valid {
		d.Jamaah = &JamaahRingkas{ID: jamaahID.Int64, Nama: jamaahNama.String}
	}
	d.Log = []agen.AkunKaitanLog{}
	rows, err := q.QueryContext(ctx, `
		SELECT l.kolom, l.nilai_lama, l.nilai_baru, l.alasan, COALESCE(au.display_name, au.email), l.diubah_at
		FROM akun_data_log l JOIN admin_users au ON au.id = l.diubah_oleh
		WHERE l.akun_id = ? ORDER BY l.id DESC`, akunID)
	if err != nil {
		return nil, fmt.Errorf("customer: riwayat data akun: %w", err)
	}
	defer rows.Close()
	d.RiwayatData = []LogData{}
	for rows.Next() {
		var l LogData
		if err := rows.Scan(&l.Kolom, &l.NilaiLama, &l.NilaiBaru, &l.Alasan, &l.DiubahOleh, &l.DiubahAt); err != nil {
			return nil, fmt.Errorf("customer: scan riwayat data akun: %w", err)
		}
		d.RiwayatData = append(d.RiwayatData, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if d.Booking, err = bookingAkun(ctx, q, akunID); err != nil {
		return nil, err
	}
	return &d, nil
}

// bookingAkun: booking yang dibuat akun ini, terbaru dulu. JumlahPax menghitung pax aktif.
func bookingAkun(ctx context.Context, q adminQuerier, akunID int64) ([]BookingAkun, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT b.id, b.id_booking, s.jadwal_nama, DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d'), b.status,
		       (SELECT COUNT(*) FROM booking_pax bp WHERE bp.booking_id = b.id AND bp.pax_status = 'aktif')
		FROM bookings b JOIN schedules s ON s.id = b.schedule_id
		WHERE b.customer_account_id = ?
		ORDER BY b.created_at DESC, b.id DESC`, akunID)
	if err != nil {
		return nil, fmt.Errorf("customer: booking akun: %w", err)
	}
	defer rows.Close()
	out := []BookingAkun{}
	for rows.Next() {
		var b BookingAkun
		if err := rows.Scan(&b.ID, &b.IDBooking, &b.JadwalNama, &b.BerangkatTanggal, &b.Status, &b.JumlahPax); err != nil {
			return nil, fmt.Errorf("customer: scan booking akun: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// AdminAkun melayani layar Akun Pelanggan di dashboard admin.
type AdminAkun struct {
	db   *sql.DB
	agen *agen.Repository
}

func NewAdminAkun(db *sql.DB, agenRepo *agen.Repository) *AdminAkun {
	return &AdminAkun{db: db, agen: agenRepo}
}

func adminBrandScope(r *http.Request) *int64 {
	if b := identity.GetBrandID(r.Context()); b != nil {
		return b
	}
	if id, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64); err == nil && id > 0 {
		return &id
	}
	return nil
}

// List GET /api/admin/akun?page=&per_page=&q=&brand_id=&status_agen=&upline=ada|tanpa&agen=agen|bukan
func (h *AdminAkun) List(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	page, _ := strconv.Atoi(qs.Get("page"))
	perPage, _ := strconv.Atoi(qs.Get("per_page"))
	hasil, err := listAkun(r.Context(), h.db, adminBrandScope(r), FilterAkun{
		Cari: qs.Get("q"), StatusAgen: qs.Get("status_agen"), Upline: qs.Get("upline"), Agen: qs.Get("agen"), Page: page, PerPage: perPage,
	})
	if errors.Is(err, ErrFilterAkun) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar akun")
		return
	}
	writeJSON(w, http.StatusOK, hasil)
}

// Detail GET /api/admin/akun/{id}
func (h *AdminAkun) Detail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id akun tidak valid")
		return
	}
	brandID := adminBrandScope(r)
	d, err := getAkunDetail(r.Context(), h.db, brandID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "akun tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil detail akun")
		return
	}
	if d.Log, err = h.agen.ListAkunKaitanLog(r.Context(), brandID, id); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil riwayat upline")
		return
	}
	writeJSON(w, http.StatusOK, d)
}
