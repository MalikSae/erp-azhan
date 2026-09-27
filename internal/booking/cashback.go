package booking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/komisi"
)

// Pakai Kredit Cashback (screen B5, agen-azhan.md 7.13, keputusan D4):
// kredit cashback jamaah dipakai sebagai potongan tagihan booking berikutnya
// di brand yang sama, lewat booking_discounts + pemakaian_cashback dalam satu
// transaksi. Menghapus diskonnya mengembalikan kredit (FK cascade, migrasi 063).

var (
	ErrCashbackMelebihiKredit  = errors.New("nominal melebihi kredit cashback jamaah")
	ErrCashbackMelebihiTagihan = errors.New("nominal melebihi sisa tagihan booking")
	ErrCashbackBukanPax        = errors.New("jamaah bukan pax aktif booking ini")
	ErrCashbackStatusBooking   = errors.New("kredit cashback hanya bisa dipakai pada booking yang belum lunas atau batal")
)

type KreditCashbackPax struct {
	JamaahID       int64   `json:"jamaah_id"`
	NamaLengkap    string  `json:"nama_lengkap"`
	KreditCashback float64 `json:"kredit_cashback"`
}

type KreditCashbackBooking struct {
	SisaTagihan float64             `json:"sisa_tagihan"`
	BisaDipakai bool                `json:"bisa_dipakai"`
	Pax         []KreditCashbackPax `json:"pax"` // hanya pax dengan kredit > 0
}

type PakaiCashbackRequest struct {
	JamaahID int64   `json:"jamaah_id"`
	Nominal  float64 `json:"nominal"`
}

type cashbackQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// bookingTagihan membaca status dan sisa tagihan booking dalam scope brand.
func bookingTagihan(ctx context.Context, q cashbackQuerier, bookingID int64, brandID *int64, lock bool) (status string, bookingBrand int64, sisa float64, err error) {
	query := `
		SELECT b.status, s.brand_id,
		       COALESCE(b.total_harga,0) - (SELECT COALESCE(SUM(jumlah),0) FROM payments WHERE booking_id = b.id AND status = 'confirmed')
		FROM bookings b JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ?`
	if lock {
		query += ` FOR UPDATE`
	}
	err = q.QueryRowContext(ctx, query, bookingID).Scan(&status, &bookingBrand, &sisa)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != bookingBrand) {
		return "", 0, 0, ErrNotFound
	}
	if err != nil {
		return "", 0, 0, fmt.Errorf("booking cashback: baca booking: %w", err)
	}
	return status, bookingBrand, sisa, nil
}

func cashbackBisaDipakai(status string) bool {
	return status == "baru" || status == "dp"
}

func kreditCashbackBooking(ctx context.Context, q cashbackQuerier, bookingID int64, brandID *int64) (*KreditCashbackBooking, error) {
	status, _, sisa, err := bookingTagihan(ctx, q, bookingID, brandID, false)
	if err != nil {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `
		SELECT DISTINCT j.id, j.nama_lengkap FROM booking_pax bp JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif' ORDER BY j.id`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking cashback: pax: %w", err)
	}
	var pax []KreditCashbackPax
	for rows.Next() {
		var p KreditCashbackPax
		if err := rows.Scan(&p.JamaahID, &p.NamaLengkap); err != nil {
			rows.Close()
			return nil, err
		}
		pax = append(pax, p)
	}
	rows.Close()
	res := &KreditCashbackBooking{SisaTagihan: sisa, BisaDipakai: cashbackBisaDipakai(status) && sisa > 0, Pax: []KreditCashbackPax{}}
	for _, p := range pax {
		saldo, err := komisi.HitungSaldo(ctx, q, p.JamaahID)
		if err != nil {
			return nil, err
		}
		if saldo.KreditCashback > 0 {
			p.KreditCashback = saldo.KreditCashback
			res.Pax = append(res.Pax, p)
		}
	}
	return res, nil
}

func pakaiCashbackTx(ctx context.Context, tx *sql.Tx, r *Repository, bookingID int64, brandID *int64, adminID int64, req PakaiCashbackRequest) error {
	status, bookingBrand, sisa, err := bookingTagihan(ctx, tx, bookingID, brandID, true)
	if err != nil {
		return err
	}
	if !cashbackBisaDipakai(status) {
		return ErrCashbackStatusBooking
	}
	var nama string
	err = tx.QueryRowContext(ctx, `
		SELECT j.nama_lengkap FROM booking_pax bp JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE bp.booking_id = ? AND bp.jamaah_id = ? AND bp.pax_status = 'aktif' LIMIT 1`, bookingID, req.JamaahID).Scan(&nama)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrCashbackBukanPax
	}
	if err != nil {
		return fmt.Errorf("booking cashback: cek pax: %w", err)
	}
	// Kunci baris jamaah supaya kredit tidak terpakai dua kali secara bersamaan.
	if _, err := tx.ExecContext(ctx, `SELECT id FROM jamaah WHERE id = ? FOR UPDATE`, req.JamaahID); err != nil {
		return fmt.Errorf("booking cashback: kunci jamaah: %w", err)
	}
	saldo, err := komisi.HitungSaldo(ctx, tx, req.JamaahID)
	if err != nil {
		return err
	}
	if req.Nominal > saldo.KreditCashback {
		return ErrCashbackMelebihiKredit
	}
	if req.Nominal > sisa {
		return ErrCashbackMelebihiTagihan
	}
	res, err := tx.ExecContext(ctx, `INSERT INTO booking_discounts (booking_id, nama, nominal) VALUES (?, ?, ?)`,
		bookingID, "Kredit Cashback — "+nama, req.Nominal)
	if err != nil {
		return fmt.Errorf("booking cashback: diskon: %w", err)
	}
	discountID, _ := res.LastInsertId()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pemakaian_cashback (brand_id, jamaah_id, booking_id, booking_discount_id, nominal, dipakai_oleh)
		VALUES (?, ?, ?, ?, ?, ?)`, bookingBrand, req.JamaahID, bookingID, discountID, req.Nominal, adminID); err != nil {
		return fmt.Errorf("booking cashback: pemakaian: %w", err)
	}
	return r.recalculateTotalTx(ctx, tx, bookingID)
}

func (r *Repository) KreditCashback(ctx context.Context, bookingID int64, brandID *int64) (*KreditCashbackBooking, error) {
	return kreditCashbackBooking(ctx, r.db, bookingID, brandID)
}

func (r *Repository) PakaiCashback(ctx context.Context, bookingID int64, brandID *int64, adminID int64, req PakaiCashbackRequest) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := pakaiCashbackTx(ctx, tx, r, bookingID, brandID, adminID, req); err != nil {
		return err
	}
	return tx.Commit()
}

// GetKreditCashback GET /api/admin/bookings/{id}/cashback
func (h *Handler) GetKreditCashback(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	res, err := h.repo.KreditCashback(r.Context(), id, identity.GetBrandID(r.Context()))
	if err != nil {
		handleCashbackError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// PakaiKreditCashback POST /api/admin/bookings/{id}/cashback
func (h *Handler) PakaiKreditCashback(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req PakaiCashbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.JamaahID <= 0 {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	if req.Nominal <= 0 || req.Nominal != float64(int64(req.Nominal)) {
		writeError(w, http.StatusBadRequest, "nominal harus bilangan bulat lebih dari 0")
		return
	}
	brandID := identity.GetBrandID(r.Context())
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.PakaiCashback(r.Context(), id, brandID, adminID, req); err != nil {
		handleCashbackError(w, err)
		return
	}
	log.Printf("[AUDIT] cashback_dipakai booking_id=%d jamaah_id=%d nominal=%.0f admin_id=%d", id, req.JamaahID, req.Nominal, adminID)
	b, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func handleCashbackError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrCashbackMelebihiKredit), errors.Is(err, ErrCashbackMelebihiTagihan),
		errors.Is(err, ErrCashbackBukanPax), errors.Is(err, ErrCashbackStatusBooking):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		handleRepoError(w, err)
	}
}
