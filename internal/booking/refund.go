package booking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
)

// Catatan pengembalian dana (keputusan audit jamaah-booking 2026-09-30):
// booking batal yang sudah punya pembayaran terkonfirmasi ditandai perlu_refund
// sampai admin mencatat pengembalian sebesar dana yang diterima. Refund hanya
// dicatat (tidak ada update/hapus) agar jejak audit utuh.

type BookingRefund struct {
	ID        int64     `json:"id"`
	BookingID int64     `json:"booking_id"`
	Jumlah    float64   `json:"jumlah"`
	Tanggal   string    `json:"tanggal"`
	Metode    *string   `json:"metode"`
	Catatan   *string   `json:"catatan"`
	CreatedBy *int64    `json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateRefundRequest struct {
	Jumlah  float64 `json:"jumlah"`
	Tanggal string  `json:"tanggal"`
	Metode  *string `json:"metode"`
	Catatan *string `json:"catatan"`
}

// ListRefunds mengambil riwayat pengembalian dana sebuah booking.
func (r *Repository) ListRefunds(ctx context.Context, bookingID int64) ([]BookingRefund, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, booking_id, jumlah, DATE_FORMAT(tanggal, '%Y-%m-%d'), metode, catatan, created_by, created_at
		FROM booking_refunds WHERE booking_id = ? ORDER BY tanggal ASC, id ASC`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking.ListRefunds: %w", err)
	}
	defer rows.Close()

	items := make([]BookingRefund, 0)
	for rows.Next() {
		var rf BookingRefund
		var createdBy sql.NullInt64
		if err := rows.Scan(&rf.ID, &rf.BookingID, &rf.Jumlah, &rf.Tanggal, &rf.Metode, &rf.Catatan, &createdBy, &rf.CreatedAt); err != nil {
			return nil, fmt.Errorf("booking.ListRefunds scan: %w", err)
		}
		if createdBy.Valid {
			rf.CreatedBy = &createdBy.Int64
		}
		items = append(items, rf)
	}
	return items, rows.Err()
}

// CreateRefund mencatat pengembalian dana. Booking dikunci agar dua pencatatan
// bersamaan tidak melampaui dana yang diterima.
func (r *Repository) CreateRefund(ctx context.Context, bookingID int64, brandID *int64, adminID int64, req CreateRefundRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	var bookingBrand int64
	err = tx.QueryRowContext(ctx, `
		SELECT b.status, s.brand_id FROM bookings b JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ? FOR UPDATE`, bookingID).Scan(&status, &bookingBrand)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("booking.CreateRefund lock: %w", err)
	}
	if brandID != nil && *brandID != bookingBrand {
		return ErrNotFound
	}
	if status != "batal" {
		return bizErr("pengembalian dana hanya dapat dicatat untuk booking yang sudah dibatalkan")
	}

	var dibayar, sudahRefund float64
	err = tx.QueryRowContext(ctx, `
		SELECT (SELECT COALESCE(SUM(jumlah), 0) FROM payments WHERE booking_id = ? AND status = 'confirmed'),
		       (SELECT COALESCE(SUM(jumlah), 0) FROM booking_refunds WHERE booking_id = ?)`, bookingID, bookingID).
		Scan(&dibayar, &sudahRefund)
	if err != nil {
		return fmt.Errorf("booking.CreateRefund saldo: %w", err)
	}
	sisa := dibayar - sudahRefund
	if sisa <= 0 {
		return bizErr("tidak ada dana yang perlu dikembalikan untuk booking ini")
	}
	if req.Jumlah > sisa {
		return bizErr("jumlah pengembalian melebihi dana yang belum dikembalikan (%s)", formatRupiah(sisa))
	}

	var admin any
	if adminID > 0 {
		admin = adminID
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO booking_refunds (booking_id, jumlah, tanggal, metode, catatan, created_by)
		VALUES (?, ?, ?, ?, ?, ?)`, bookingID, req.Jumlah, req.Tanggal, req.Metode, req.Catatan, admin); err != nil {
		return fmt.Errorf("booking.CreateRefund insert: %w", err)
	}
	return tx.Commit()
}

// ListRefunds godoc
// GET /api/admin/bookings/{id}/refunds
func (h *Handler) ListRefunds(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if _, err := h.repo.GetByID(r.Context(), id, identity.GetBrandID(r.Context())); err != nil {
		handleRepoError(w, err)
		return
	}
	items, err := h.repo.ListRefunds(r.Context(), id)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreateRefund godoc
// POST /api/admin/bookings/{id}/refunds
func (h *Handler) CreateRefund(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req CreateRefundRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	if req.Jumlah <= 0 || req.Jumlah != float64(int64(req.Jumlah)) {
		writeError(w, http.StatusBadRequest, "jumlah harus bilangan bulat lebih dari 0")
		return
	}
	if req.Jumlah > maxNominal {
		writeError(w, http.StatusBadRequest, "jumlah terlalu besar")
		return
	}
	req.Tanggal = strings.TrimSpace(req.Tanggal)
	tgl, err := time.Parse("2006-01-02", req.Tanggal)
	today := time.Now().In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02")
	if err != nil || tgl.Format("2006-01-02") > today {
		writeError(w, http.StatusBadRequest, "tanggal pengembalian tidak valid atau berada di masa depan")
		return
	}
	req.Metode = trimOptional(req.Metode, 50)
	req.Catatan = trimOptional(req.Catatan, 500)

	brandID := identity.GetBrandID(r.Context())
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.CreateRefund(r.Context(), id, brandID, adminID, req); err != nil {
		handleRepoError(w, err)
		return
	}
	log.Printf("[AUDIT] refund_dicatat booking_id=%d jumlah=%.0f admin_id=%d", id, req.Jumlah, adminID)

	b, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// trimOptional merapikan teks opsional: kosong menjadi nil, dipotong ke batas kolom.
func trimOptional(v *string, max int) *string {
	if v == nil {
		return nil
	}
	s := strings.TrimSpace(*v)
	if s == "" {
		return nil
	}
	if r := []rune(s); len(r) > max {
		s = string(r[:max])
	}
	return &s
}
