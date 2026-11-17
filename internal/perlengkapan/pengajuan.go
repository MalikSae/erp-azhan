package perlengkapan

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"erp-azhan/api/internal/identity"
)

// Antrean pengajuan perlengkapan dari app (migrasi 141): jamaah yang perlengkapannya sudah diajukan akun tetapi belum
// diserahkan petugas. Penyerahan tetap lewat endpoint distribusi booking (izin booking.edit).

type Pengajuan struct {
	PaxID            int64   `json:"pax_id"`
	BookingID        int64   `json:"booking_id"`
	BookingCode      string  `json:"booking_code"`
	JamaahID         int64   `json:"jamaah_id"`
	NamaJamaah       string  `json:"nama_jamaah"`
	JenisKelamin     string  `json:"jenis_kelamin"`
	ScheduleID       int64   `json:"schedule_id"`
	JadwalNama       string  `json:"jadwal_nama"`
	BerangkatTanggal string  `json:"berangkat_tanggal"`
	BrandID          int64   `json:"brand_id"`
	Metode           string  `json:"metode"`
	DiajukanAt       string  `json:"diajukan_at"`
	PengajuNama      *string `json:"pengaju_nama"`
	PengajuHP        *string `json:"pengaju_hp"`
}

// ListPengajuan: pengajuan yang belum diserahkan, keberangkatan terdekat lebih dulu. brandID nil = semua brand.
func (r *Repository) ListPengajuan(ctx context.Context, brandID *int64, scheduleID int64) ([]Pengajuan, error) {
	q := `
		SELECT bp.id, b.id, b.id_booking, j.id, j.nama_lengkap, COALESCE(j.jenis_kelamin,''),
		       s.id, s.jadwal_nama, DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d'), s.brand_id,
		       COALESCE(bp.perlengkapan_metode,''), DATE_FORMAT(bp.perlengkapan_diajukan_at,'%Y-%m-%d %H:%i'),
		       ca.nama_lengkap, ca.no_hp
		FROM booking_pax bp
		JOIN bookings b ON b.id = bp.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN customer_accounts ca ON ca.id = bp.perlengkapan_diajukan_oleh
		WHERE bp.perlengkapan_diajukan_at IS NOT NULL AND bp.perlengkapan_status <> 'sudah_diberikan'
		  AND bp.pax_status = 'aktif' AND b.status NOT IN ('draft','batal')`
	args := []any{}
	if brandID != nil {
		q += ` AND s.brand_id = ?`
		args = append(args, *brandID)
	}
	if scheduleID > 0 {
		q += ` AND s.id = ?`
		args = append(args, scheduleID)
	}
	q += ` ORDER BY s.berangkat_tanggal, bp.perlengkapan_diajukan_at, bp.id LIMIT 500`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Pengajuan{}
	for rows.Next() {
		var p Pengajuan
		if err := rows.Scan(&p.PaxID, &p.BookingID, &p.BookingCode, &p.JamaahID, &p.NamaJamaah, &p.JenisKelamin,
			&p.ScheduleID, &p.JadwalNama, &p.BerangkatTanggal, &p.BrandID, &p.Metode, &p.DiajukanAt,
			&p.PengajuNama, &p.PengajuHP); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListPengajuan GET /api/admin/perlengkapan-pengajuan?schedule_id=&brand_id=
func (h *Handler) ListPengajuan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	brandID := identity.GetBrandID(r.Context())
	if brandID == nil {
		if v, err := strconv.ParseInt(q.Get("brand_id"), 10, 64); err == nil && v > 0 {
			brandID = &v
		}
	}
	scheduleID, _ := strconv.ParseInt(q.Get("schedule_id"), 10, 64)
	items, err := h.repo.ListPengajuan(r.Context(), brandID, scheduleID)
	if err != nil {
		log.Printf("[ERROR] perlengkapan pengajuan: %v", err)
		h.sendError(w, http.StatusInternalServerError, "Gagal memuat pengajuan perlengkapan.")
		return
	}
	h.sendJSON(w, http.StatusOK, items)
}
