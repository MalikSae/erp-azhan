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
	"erp-azhan/api/internal/shared"
)

// Pakai Kredit Cashback (screen B5, agen-azhan.md 7.13, keputusan D4):
// kredit cashback akun pemesan dipakai sebagai potongan tagihan booking
// berikutnya di brand yang sama, lewat booking_discounts + pemakaian_cashback dalam satu
// transaksi. Menghapus diskonnya mengembalikan kredit (FK cascade, migrasi 063).

var (
	ErrCashbackMelebihiKredit  = errors.New("nominal melebihi kredit cashback akun pemesan")
	ErrCashbackMelebihiTagihan = errors.New("nominal melebihi sisa tagihan booking")
	ErrCashbackTanpaAkun       = errors.New("booking ini tidak punya akun pemesan, kredit cashback tidak bisa dipakai")
	ErrCashbackStatusBooking   = errors.New("kredit cashback hanya bisa dipakai pada booking yang belum lunas atau batal")
	// ErrCashbackNonaktif: fitur kredit cashback dimatikan di tahap awal (spek keagenan bagian 1).
	ErrCashbackNonaktif = errors.New("fitur kredit cashback belum diaktifkan")
)

type KreditCashbackAkun struct {
	AkunID         int64   `json:"akun_id"`
	NamaLengkap    string  `json:"nama_lengkap"`
	KreditCashback float64 `json:"kredit_cashback"`
}

type KreditCashbackBooking struct {
	SisaTagihan float64             `json:"sisa_tagihan"`
	BisaDipakai bool                `json:"bisa_dipakai"`
	Akun        *KreditCashbackAkun `json:"akun"` // nil bila tidak ada akun pemesan atau kreditnya 0
}

type PakaiCashbackRequest struct {
	Nominal float64 `json:"nominal"`
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

// akunPemesan mengembalikan akun pemesan booking. ok false bila booking tidak punya akun pemesan.
func akunPemesan(ctx context.Context, q cashbackQuerier, bookingID int64) (akunID int64, nama string, ok bool, err error) {
	var id sql.NullInt64
	var n sql.NullString
	err = q.QueryRowContext(ctx, `
		SELECT a.id, a.nama_lengkap FROM bookings b JOIN customer_accounts a ON a.id = b.customer_account_id
		WHERE b.id = ?`, bookingID).Scan(&id, &n)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, fmt.Errorf("booking cashback: akun pemesan: %w", err)
	}
	return id.Int64, n.String, id.Valid, nil
}

func kreditCashbackBooking(ctx context.Context, q cashbackQuerier, bookingID int64, brandID *int64) (*KreditCashbackBooking, error) {
	status, _, sisa, err := bookingTagihan(ctx, q, bookingID, brandID, false)
	if err != nil {
		return nil, err
	}
	res := &KreditCashbackBooking{SisaTagihan: sisa, BisaDipakai: cashbackBisaDipakai(status) && sisa > 0}
	akunID, nama, ada, err := akunPemesan(ctx, q, bookingID)
	if err != nil {
		return nil, err
	}
	if !ada {
		return res, nil
	}
	saldo, err := komisi.HitungSaldo(ctx, q, akunID)
	if err != nil {
		return nil, err
	}
	if saldo.KreditCashback > 0 {
		res.Akun = &KreditCashbackAkun{AkunID: akunID, NamaLengkap: nama, KreditCashback: saldo.KreditCashback}
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
	akunID, nama, ada, err := akunPemesan(ctx, tx, bookingID)
	if err != nil {
		return err
	}
	if !ada {
		return ErrCashbackTanpaAkun
	}
	// Kunci baris akun supaya kredit tidak terpakai dua kali secara bersamaan.
	var terkunci int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM customer_accounts WHERE id = ? FOR UPDATE`, akunID).Scan(&terkunci); err != nil {
		return fmt.Errorf("booking cashback: kunci akun: %w", err)
	}
	saldo, err := komisi.HitungSaldo(ctx, tx, akunID)
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
	discountID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("booking cashback: id diskon: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pemakaian_cashback (brand_id, akun_id, booking_id, booking_discount_id, nominal, dipakai_oleh)
		VALUES (?, ?, ?, ?, ?, ?)`, bookingBrand, akunID, bookingID, discountID, req.Nominal, adminID); err != nil {
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
		return fmt.Errorf("booking cashback: mulai transaksi: %w", err)
	}
	defer tx.Rollback()
	if err := pakaiCashbackTx(ctx, tx, r, bookingID, brandID, adminID, req); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("booking cashback: commit: %w", err)
	}
	return nil
}

// cashbackAktif menolak permintaan selama fitur kredit cashback dimatikan (komisi.KomisiRepeatCashbackAktif).
// Tanpa ini rute tetap bisa dipanggil langsung, dan pemakaian kredit menambah diskon tanpa melewati batas
// diskon 10% (audit Jamaah & Booking T-02). Saat fitur dinyalakan, putuskan dulu apakah kredit cashback
// ikut batas itu.
func cashbackAktif(w http.ResponseWriter) bool {
	if komisi.KomisiRepeatCashbackAktif {
		return true
	}
	writeError(w, http.StatusConflict, ErrCashbackNonaktif.Error())
	return false
}

// GetKreditCashback GET /api/admin/bookings/{id}/cashback
func (h *Handler) GetKreditCashback(w http.ResponseWriter, r *http.Request) {
	if !cashbackAktif(w) {
		return
	}
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
	if !cashbackAktif(w) {
		return
	}
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req PakaiCashbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	if req.Nominal <= 0 || !shared.RupiahBulat(req.Nominal) {
		writeError(w, http.StatusBadRequest, "nominal harus bilangan bulat lebih dari 0")
		return
	}
	brandID := identity.GetBrandID(r.Context())
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.PakaiCashback(r.Context(), id, brandID, adminID, req); err != nil {
		handleCashbackError(w, err)
		return
	}
	log.Printf("[AUDIT] cashback_dipakai booking_id=%d nominal=%.0f admin_id=%d", id, req.Nominal, adminID)
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
		errors.Is(err, ErrCashbackTanpaAkun), errors.Is(err, ErrCashbackStatusBooking):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		handleRepoError(w, err)
	}
}
