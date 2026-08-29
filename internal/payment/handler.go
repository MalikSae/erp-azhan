package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

// Handler menyimpan dependency untuk semua HTTP handler payment.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ListDailyBrandTransactions mengembalikan transaksi terkonfirmasi 30 hari per brand.
// GET /api/admin/analytics/transactions-30-days
func (h *Handler) ListDailyBrandTransactions(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListDailyBrandTransactions(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data transaksi")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListPayments godoc
// GET /api/admin/bookings/{booking_id}/payments
func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	bookingID, ok := parseID(w, r, "booking_id")
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verify booking exists and belongs to brand
	exists, err := h.repo.BookingExistsForBrand(r.Context(), bookingID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "booking_id tidak ditemukan atau bukan milik brand Anda")
		return
	}

	items, err := h.repo.ListByBookingID(r.Context(), bookingID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data payment")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ListAllPayments menampilkan inbox pembayaran lintas brand untuk pusat,
// atau otomatis scoped untuk admin brand.
func (h *Handler) ListAllPayments(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "" && status != "pending" && status != "confirmed" && status != "rejected" {
		writeError(w, 400, "status tidak valid")
		return
	}
	items, err := h.repo.ListAll(r.Context(), identity.GetBrandID(r.Context()), status)
	if err != nil {
		writeError(w, 500, "gagal mengambil daftar pembayaran")
		return
	}
	writeJSON(w, 200, items)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreatePayment godoc
// POST /api/admin/bookings/{booking_id}/payments
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	bookingID, ok := parseID(w, r, "booking_id")
	if !ok {
		return
	}

	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if req.Jumlah <= 0 {
		writeError(w, http.StatusBadRequest, "jumlah pembayaran harus lebih dari 0")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verify booking
	exists, err := h.repo.BookingExistsForBrand(r.Context(), bookingID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !exists {
		writeError(w, http.StatusNotFound, "booking_id tidak valid atau bukan milik brand Anda")
		return
	}

	// Validasi nilai tidak boleh melebihi sisa tagihan
	totalHarga, totalPaid, err := h.repo.GetBookingTotalAndPaid(r.Context(), bookingID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	if totalHarga != nil {
		sisaTagihan := *totalHarga - totalPaid
		if sisaTagihan <= 0 {
			writeError(w, http.StatusBadRequest, "tagihan booking sudah lunas")
			return
		}
		if req.Jumlah > sisaTagihan {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("jumlah pembayaran tidak boleh melebihi sisa tagihan (sisa: Rp %.0f)", sisaTagihan))
			return
		}
	}

	p, err := h.repo.Create(r.Context(), bookingID, &req)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, p)
}

// ─── Update Status ────────────────────────────────────────────────────────────

// UpdatePaymentStatus godoc
// PUT /api/admin/payments/{id}/status
func (h *Handler) UpdatePaymentStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}

	var req UpdatePaymentStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if req.Status != "confirmed" && req.Status != "rejected" {
		writeError(w, http.StatusBadRequest, "status harus confirmed atau rejected")
		return
	}
	if req.Status == "rejected" && (req.RejectionReason == nil || len(*req.RejectionReason) < 3) {
		writeError(w, http.StatusBadRequest, "alasan penolakan wajib diisi")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Ambil payment untuk verifikasi brand
	p, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	// Idempotent check
	if p.Status != "pending" {
		writeJSON(w, http.StatusOK, p)
		return
	}

	updated, err := h.repo.UpdateStatus(r.Context(), id, req.Status, req.RejectionReason, identity.GetAdminUserID(r.Context()))
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// DeletePayment godoc
// DELETE /api/admin/payments/{id}
func (h *Handler) DeletePayment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verify brand via GetByID
	_, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrCannotDelete):
		writeError(w, http.StatusConflict, "tidak bisa menghapus pembayaran yang sudah dikonfirmasi")
	case errors.Is(err, ErrSeatUnavailable):
		writeError(w, http.StatusConflict, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("terjadi kesalahan internal: %v", err))
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, param+" tidak valid")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
