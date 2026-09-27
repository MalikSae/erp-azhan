package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"erp-azhan/api/internal/komisi"
)

// Tampilan kinerja agen: Dashboard Agen (A3), Riwayat Komisi (A6), Daftar
// Agen (B2), Detail Agen (B2a), dan Riwayat Komisi brand (B3).

var ErrBukanAgenAktif = errors.New("fitur ini hanya untuk agen aktif")

// KomisiItem adalah satu baris ledger transaksi_komisi.
type KomisiItem struct {
	ID               int64     `json:"id"`
	Jenis            string    `json:"jenis"`
	Nominal          float64   `json:"nominal"`
	BookingID        int64     `json:"booking_id"`
	BookingCode      string    `json:"booking_code"`
	PenerimaID       int64     `json:"penerima_id"`
	PenerimaNama     string    `json:"penerima_nama"`
	SumberNama       string    `json:"sumber_nama"`
	BerangkatTanggal time.Time `json:"berangkat_tanggal"`
	// Ketersediaan (D4, D6): "kredit" untuk cashback, "tersedia" bila booking
	// sumbernya sudah berangkat, selain itu "tertahan".
	Ketersediaan string    `json:"ketersediaan"`
	CreatedAt    time.Time `json:"created_at"`
}

// KomisiFilter untuk list ledger. BrandID nil = semua brand (super admin).
type KomisiFilter struct {
	BrandID    *int64
	PenerimaID int64
	Jenis      string
	Dari       string // YYYY-MM-DD, tanggal transaksi
	Sampai     string
	Limit      int
	Offset     int
}

var jenisKomisi = map[string]bool{"langsung": true, "pembinaan": true, "repeat_order": true, "cashback": true}

func listKomisi(ctx context.Context, q querier, f KomisiFilter) ([]KomisiItem, error) {
	where := []string{"1=1"}
	args := []any{}
	if f.BrandID != nil {
		where = append(where, "tk.brand_id = ?")
		args = append(args, *f.BrandID)
	}
	if f.PenerimaID > 0 {
		where = append(where, "tk.jamaah_penerima_id = ?")
		args = append(args, f.PenerimaID)
	}
	if jenisKomisi[f.Jenis] {
		where = append(where, "tk.jenis = ?")
		args = append(args, f.Jenis)
	}
	if _, err := time.Parse("2006-01-02", f.Dari); err == nil {
		where = append(where, "tk.created_at >= ?")
		args = append(args, f.Dari)
	}
	if _, err := time.Parse("2006-01-02", f.Sampai); err == nil {
		where = append(where, "tk.created_at < DATE_ADD(?, INTERVAL 1 DAY)")
		args = append(args, f.Sampai)
	}
	if f.Limit <= 0 || f.Limit > 200 {
		f.Limit = 50
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	args = append(args, f.Limit, f.Offset)

	rows, err := q.QueryContext(ctx, `
		SELECT tk.id, tk.jenis, tk.nominal, tk.booking_id, b.id_booking,
		       tk.jamaah_penerima_id, pn.nama_lengkap, sb.nama_lengkap, s.berangkat_tanggal,
		       s.berangkat_tanggal <= CURRENT_DATE, tk.created_at
		FROM transaksi_komisi tk
		JOIN bookings b ON b.id = tk.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		JOIN jamaah pn ON pn.id = tk.jamaah_penerima_id
		JOIN jamaah sb ON sb.id = tk.jamaah_sumber_id
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY tk.created_at DESC, tk.id DESC
		LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("agen: list komisi: %w", err)
	}
	defer rows.Close()
	items := []KomisiItem{}
	for rows.Next() {
		var k KomisiItem
		var sudahBerangkat bool
		if err := rows.Scan(&k.ID, &k.Jenis, &k.Nominal, &k.BookingID, &k.BookingCode,
			&k.PenerimaID, &k.PenerimaNama, &k.SumberNama, &k.BerangkatTanggal, &sudahBerangkat, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("agen: scan komisi: %w", err)
		}
		switch {
		case k.Jenis == "cashback":
			k.Ketersediaan = "kredit"
		case sudahBerangkat:
			k.Ketersediaan = "tersedia"
		default:
			k.Ketersediaan = "tertahan"
		}
		items = append(items, k)
	}
	return items, rows.Err()
}

// TotalPerJenis: total komisi sepanjang waktu per jenis.
type TotalPerJenis struct {
	Langsung    float64 `json:"langsung"`
	Pembinaan   float64 `json:"pembinaan"`
	RepeatOrder float64 `json:"repeat_order"`
	Cashback    float64 `json:"cashback"`
}

func totalPerJenis(ctx context.Context, q querier, jamaahID int64) (TotalPerJenis, error) {
	var t TotalPerJenis
	err := q.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN jenis='langsung' THEN nominal END),0),
		       COALESCE(SUM(CASE WHEN jenis='pembinaan' THEN nominal END),0),
		       COALESCE(SUM(CASE WHEN jenis='repeat_order' THEN nominal END),0),
		       COALESCE(SUM(CASE WHEN jenis='cashback' THEN nominal END),0)
		FROM transaksi_komisi WHERE jamaah_penerima_id = ?`, jamaahID).
		Scan(&t.Langsung, &t.Pembinaan, &t.RepeatOrder, &t.Cashback)
	if err != nil {
		return t, fmt.Errorf("agen: total komisi: %w", err)
	}
	return t, nil
}

// jaringan: jumlah downline (agen yang upline-nya agen ini) dan jumlah
// jamaah yang di-closing (direkrut_oleh = agen ini).
func jaringan(ctx context.Context, q querier, agenID int64) (downline, closing int, err error) {
	err = q.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM jamaah WHERE upline_jamaah_id = ?),
		       (SELECT COUNT(*) FROM jamaah WHERE direkrut_oleh_jamaah_id = ?)`, agenID, agenID).
		Scan(&downline, &closing)
	if err != nil {
		err = fmt.Errorf("agen: jaringan: %w", err)
	}
	return
}

// ─── Portal (A3, A6) ─────────────────────────────────────────────────────────

type DashboardAgen struct {
	komisi.Saldo
	KodeReferral   *string       `json:"kode_referral"`
	JumlahClosing  int           `json:"jumlah_closing"`
	JumlahDownline int           `json:"jumlah_downline"`
	Total          TotalPerJenis `json:"total_per_jenis"`
	KomisiTerbaru  []KomisiItem  `json:"komisi_terbaru"`
}

func requireAgenAktif(ctx context.Context, q querier, jamaahID int64) (*string, error) {
	var status string
	var kode sql.NullString
	err := q.QueryRowContext(ctx, `SELECT status_agen, kode_referral FROM jamaah WHERE id = ?`, jamaahID).Scan(&status, &kode)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("agen: cek status: %w", err)
	}
	if status != "aktif" {
		return nil, ErrBukanAgenAktif
	}
	if kode.Valid {
		return &kode.String, nil
	}
	return nil, nil
}

func getDashboardAgen(ctx context.Context, q querier, jamaahID int64) (*DashboardAgen, error) {
	kode, err := requireAgenAktif(ctx, q, jamaahID)
	if err != nil {
		return nil, err
	}
	d := DashboardAgen{KodeReferral: kode}
	if d.Saldo, err = komisi.HitungSaldo(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	if d.JumlahDownline, d.JumlahClosing, err = jaringan(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	if d.Total, err = totalPerJenis(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	if d.KomisiTerbaru, err = listKomisi(ctx, q, KomisiFilter{PenerimaID: jamaahID, Limit: 5}); err != nil {
		return nil, err
	}
	return &d, nil
}

func (r *Repository) GetDashboardAgen(ctx context.Context, jamaahID int64) (*DashboardAgen, error) {
	return getDashboardAgen(ctx, r.db, jamaahID)
}

func (r *Repository) ListKomisiAgen(ctx context.Context, jamaahID int64, f KomisiFilter) ([]KomisiItem, error) {
	if _, err := requireAgenAktif(ctx, r.db, jamaahID); err != nil {
		return nil, err
	}
	f.BrandID = nil
	f.PenerimaID = jamaahID
	return listKomisi(ctx, r.db, f)
}

// ─── Admin (B2, B2a, B3) ─────────────────────────────────────────────────────

type AgenListItem struct {
	JamaahID        int64      `json:"jamaah_id"`
	BrandID         int64      `json:"brand_id"`
	BrandName       string     `json:"brand_name"`
	NamaLengkap     string     `json:"nama_lengkap"`
	KodeReferral    *string    `json:"kode_referral"`
	StatusAgen      string     `json:"status_agen"`
	DisetujuiAgenAt *time.Time `json:"disetujui_agen_at"`
}

func listAgen(ctx context.Context, q querier, brandID *int64, cari string) ([]AgenListItem, error) {
	query := `
		SELECT j.id, j.brand_id, b.name, j.nama_lengkap, j.kode_referral, j.status_agen, j.disetujui_agen_at
		FROM jamaah j JOIN brands b ON b.id = j.brand_id
		WHERE j.status_agen IN ('aktif','nonaktif')`
	args := []any{}
	if brandID != nil {
		query += ` AND j.brand_id = ?`
		args = append(args, *brandID)
	}
	if c := strings.TrimSpace(cari); c != "" {
		query += ` AND (j.nama_lengkap LIKE ? OR j.kode_referral LIKE ?)`
		args = append(args, "%"+c+"%", "%"+c+"%")
	}
	query += ` ORDER BY j.status_agen ASC, j.nama_lengkap ASC`
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("agen: list agen: %w", err)
	}
	defer rows.Close()
	items := []AgenListItem{}
	for rows.Next() {
		var a AgenListItem
		if err := rows.Scan(&a.JamaahID, &a.BrandID, &a.BrandName, &a.NamaLengkap, &a.KodeReferral, &a.StatusAgen, &a.DisetujuiAgenAt); err != nil {
			return nil, fmt.Errorf("agen: scan agen: %w", err)
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

type Pencairan struct {
	ID                     int64      `json:"id"`
	NominalDiajukan        float64    `json:"nominal_diajukan"`
	Status                 string     `json:"status"`
	DiajukanAt             time.Time  `json:"diajukan_at"`
	DiprosesAt             *time.Time `json:"diproses_at"`
	CatatanPenolakan       *string    `json:"catatan_penolakan"`
	BuktiTransferKeluarURL *string    `json:"bukti_transfer_keluar_url"`
}

// DetailAgen berisi 8 kelompok data screen B2a (screen-agen.md).
type DetailAgen struct {
	// a. Identitas & data profil
	JamaahID    int64      `json:"jamaah_id"`
	BrandID     int64      `json:"brand_id"`
	BrandName   string     `json:"brand_name"`
	NamaLengkap string     `json:"nama_lengkap"`
	NoHP        *string    `json:"no_hp"`
	FotoAgenURL *string    `json:"foto_agen_url"`
	Domisili    *string    `json:"domisili"`
	SetujuSKAt  *time.Time `json:"menyetujui_syarat_ketentuan_agen_at"`
	// b. Status keagenan
	StatusAgen        string     `json:"status_agen"`
	KodeReferral      *string    `json:"kode_referral"`
	DiajukanAt        *time.Time `json:"diajukan_agen_at"`
	DisetujuiAt       *time.Time `json:"disetujui_agen_at"`
	DisetujuiOlehNama *string    `json:"disetujui_agen_oleh_nama"`
	// c. Jaringan (agen ke agen)
	UplineID       *int64  `json:"upline_jamaah_id"`
	UplineNama     *string `json:"upline_nama"`
	JumlahDownline int     `json:"jumlah_downline"`
	// d. Performa closing
	JumlahClosing int `json:"jumlah_closing"`
	// e. Ringkasan komisi & saldo
	Saldo komisi.Saldo  `json:"saldo"`
	Total TotalPerJenis `json:"total_per_jenis"`
	// f. Riwayat pencairan (read-only)
	Pencairan []Pencairan `json:"pencairan"`
	// g. Riwayat siklus pengajuan (terbaru dulu)
	Siklus []Pembayaran `json:"siklus"`
}

func getDetailAgen(ctx context.Context, q querier, jamaahID int64, brandID *int64) (*DetailAgen, error) {
	var d DetailAgen
	var upline sql.NullInt64
	err := q.QueryRowContext(ctx, `
		SELECT j.id, j.brand_id, b.name, j.nama_lengkap, j.no_hp, j.foto_agen_url, j.domisili,
		       j.menyetujui_syarat_ketentuan_agen_at, j.status_agen, j.kode_referral, j.diajukan_agen_at,
		       j.disetujui_agen_at, COALESCE(au.display_name, au.email), j.upline_jamaah_id, up.nama_lengkap
		FROM jamaah j
		JOIN brands b ON b.id = j.brand_id
		LEFT JOIN admin_users au ON au.id = j.disetujui_agen_oleh
		LEFT JOIN jamaah up ON up.id = j.upline_jamaah_id
		WHERE j.id = ? AND j.status_agen IN ('aktif','nonaktif')`, jamaahID).
		Scan(&d.JamaahID, &d.BrandID, &d.BrandName, &d.NamaLengkap, &d.NoHP, &d.FotoAgenURL, &d.Domisili,
			&d.SetujuSKAt, &d.StatusAgen, &d.KodeReferral, &d.DiajukanAt,
			&d.DisetujuiAt, &d.DisetujuiOlehNama, &upline, &d.UplineNama)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != d.BrandID) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("agen: detail: %w", err)
	}
	if upline.Valid {
		d.UplineID = &upline.Int64
	}
	if d.JumlahDownline, d.JumlahClosing, err = jaringan(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	if d.Saldo, err = komisi.HitungSaldo(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	if d.Total, err = totalPerJenis(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	if d.Pencairan, err = listPencairan(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	if d.Siklus, err = listSiklus(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	return &d, nil
}

func listPencairan(ctx context.Context, q querier, jamaahID int64) ([]Pencairan, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, nominal_diajukan, status, diajukan_at, diproses_at, catatan_penolakan, bukti_transfer_keluar_url
		FROM pengajuan_pencairan WHERE jamaah_agen_id = ? ORDER BY diajukan_at DESC, id DESC`, jamaahID)
	if err != nil {
		return nil, fmt.Errorf("agen: list pencairan: %w", err)
	}
	defer rows.Close()
	items := []Pencairan{}
	for rows.Next() {
		var p Pencairan
		if err := rows.Scan(&p.ID, &p.NominalDiajukan, &p.Status, &p.DiajukanAt, &p.DiprosesAt, &p.CatatanPenolakan, &p.BuktiTransferKeluarURL); err != nil {
			return nil, fmt.Errorf("agen: scan pencairan: %w", err)
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

// listSiklus: semua siklus pembayaran_pendaftaran_agen (tanpa riwayat upload;
// upload per siklus sudah tampil di B1).
func listSiklus(ctx context.Context, q querier, jamaahID int64) ([]Pembayaran, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, nominal_tagihan, status, bukti_transfer_url, catatan_penolakan, diverifikasi_at,
		       keputusan_agen, keputusan_agen_at, alasan_ditolak_agen, created_at
		FROM pembayaran_pendaftaran_agen WHERE jamaah_id = ? ORDER BY id DESC`, jamaahID)
	if err != nil {
		return nil, fmt.Errorf("agen: list siklus: %w", err)
	}
	defer rows.Close()
	items := []Pembayaran{}
	for rows.Next() {
		var p Pembayaran
		if err := rows.Scan(&p.ID, &p.NominalTagihan, &p.Status, &p.BuktiTransferURL, &p.CatatanPenolakan, &p.DiverifikasiAt,
			&p.KeputusanAgen, &p.KeputusanAgenAt, &p.AlasanDitolakAgen, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("agen: scan siklus: %w", err)
		}
		p.Uploads = []Upload{}
		items = append(items, p)
	}
	return items, rows.Err()
}

func (r *Repository) ListAgen(ctx context.Context, brandID *int64, cari string) ([]AgenListItem, error) {
	return listAgen(ctx, r.db, brandID, cari)
}

func (r *Repository) GetDetailAgen(ctx context.Context, jamaahID int64, brandID *int64) (*DetailAgen, error) {
	return getDetailAgen(ctx, r.db, jamaahID, brandID)
}

func (r *Repository) ListKomisiBrand(ctx context.Context, f KomisiFilter) ([]KomisiItem, error) {
	return listKomisi(ctx, r.db, f)
}
