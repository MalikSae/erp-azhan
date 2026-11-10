package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/media"
	"erp-azhan/api/internal/shared"
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
	// Admin yang terikat brand hanya melihat transaksi brand-nya sendiri.
	items, err := h.repo.ListDailyBrandTransactions(r.Context(), identity.GetBrandID(r.Context()))
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
	switch status {
	case "", StatusPending, StatusConfirmed, StatusRejected, StatusVoid:
	default:
		writeError(w, http.StatusBadRequest, "status tidak valid")
		return
	}
	items, err := h.repo.ListAll(r.Context(), identity.GetBrandID(r.Context()), status)
	if err != nil {
		log.Printf("[ERROR] payment: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar pembayaran")
		return
	}
	writeJSON(w, http.StatusOK, items)
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

	// Endpoint admin: sumber selalu "admin" (langsung terkonfirmasi); nilai dari klien diabaikan.
	req.Source = SourceAdmin

	if req.Jumlah <= 0 {
		writeError(w, http.StatusBadRequest, "jumlah pembayaran harus lebih dari 0")
		return
	}
	// Pembayaran admin langsung terkonfirmasi; tanggal masa depan merusak laporan (JB-10).
	// Aturan sama dengan konfirmasi transfer portal (JM-22).
	if req.Tanggal != nil && strings.TrimSpace(*req.Tanggal) != "" && !shared.TanggalTidakDiMasaDepan(*req.Tanggal) {
		writeError(w, http.StatusBadRequest, "tanggal pembayaran tidak valid atau berada di masa depan")
		return
	}

	// Aturan catat pembayaran oleh admin: transfer wajib menyebut rekening tujuan,
	// tunai wajib menyebut nama staff yang menerima uangnya.
	metode := "transfer"
	if req.Metode != nil && strings.TrimSpace(*req.Metode) != "" {
		metode = strings.ToLower(strings.TrimSpace(*req.Metode))
	}
	switch metode {
	case "transfer":
		if req.BankAccountID == nil || *req.BankAccountID <= 0 {
			writeError(w, http.StatusBadRequest, "rekening tujuan wajib dipilih untuk pembayaran transfer")
			return
		}
	case "cash":
		if req.ReceivedBy == nil || strings.TrimSpace(*req.ReceivedBy) == "" {
			writeError(w, http.StatusBadRequest, "nama staff penerima wajib diisi untuk pembayaran tunai")
			return
		}
		trimmed := strings.TrimSpace(*req.ReceivedBy)
		req.ReceivedBy = &trimmed
		req.BankAccountID = nil // tunai tidak melewati rekening
	default:
		writeError(w, http.StatusBadRequest, "metode pembayaran harus transfer atau cash")
		return
	}
	if metode != "cash" {
		req.ReceivedBy = nil
	}
	req.Metode = &metode // simpan nilai efektif (termasuk default transfer)

	brandID := identity.GetBrandID(r.Context())

	// Bukti pembayaran wajib unggahan admin resmi di brand ini, bukan URL bebas: URL luar yang dibuka di dashboard
	// akan menerima token admin saat berkas dimuat.
	if req.BuktiURL != nil {
		trimmed := strings.TrimSpace(*req.BuktiURL)
		if trimmed == "" {
			req.BuktiURL = nil
		} else if err := h.repo.ValidasiBuktiAdmin(r.Context(), trimmed, brandID); err != nil {
			var uploadErr *media.UploadError
			if errors.As(err, &uploadErr) {
				writeError(w, http.StatusBadRequest, uploadErr.Message)
				return
			}
			log.Printf("[ERROR] payment: %v", err)
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return
		} else {
			req.BuktiURL = &trimmed
		}
	}

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

	// Pemeriksaan awal tanpa kunci, hanya supaya admin mendapat pesan yang jelas (sudah lunas, sisa tagihan).
	// Penentu sebenarnya ada di repo.Create, yang mengulanginya setelah baris booking dikunci.
	status, totalHarga, totalPaid, err := h.repo.GetBookingTotalAndPaid(r.Context(), bookingID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	if status == bookingDraft {
		writeError(w, http.StatusBadRequest, "Booking ini masih berstatus draft, pembayaran belum bisa diproses. Selesaikan/finalisasi booking terlebih dahulu.")
		return
	}
	if totalHarga == nil {
		writeError(w, http.StatusBadRequest, "Total tagihan booking belum ditentukan, pembayaran tidak dapat diproses.")
		return
	}
	sisaTagihan := *totalHarga - totalPaid
	if sisaTagihan <= 0 {
		writeError(w, http.StatusBadRequest, "tagihan booking sudah lunas")
		return
	}
	if req.Jumlah > sisaTagihan {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("jumlah pembayaran tidak boleh melebihi sisa tagihan (sisa: Rp %.0f)", sisaTagihan))
		return
	}

	adminUserID := identity.GetAdminUserID(r.Context())
	p, err := h.repo.Create(r.Context(), bookingID, &req, &adminUserID)
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

	if req.Status != StatusConfirmed && req.Status != StatusRejected {
		writeError(w, http.StatusBadRequest, "status harus confirmed atau rejected")
		return
	}
	if req.Status == StatusRejected && (req.RejectionReason == nil || len(*req.RejectionReason) < 3) {
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

	// Sudah diputuskan: permintaan yang sama idempoten, yang berbeda ditolak agar klien tahu
	// tidak ada perubahan.
	if p.Status != "pending" {
		if p.Status != req.Status {
			writeError(w, http.StatusConflict, "pembayaran sudah berstatus "+p.Status+" dan tidak bisa diubah menjadi "+req.Status)
			return
		}
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

// ─── Void ─────────────────────────────────────────────────────────────────────

// validasiAlasanVoid merapikan alasan pembatalan pembayaran. Mengembalikan alasan yang sudah dirapikan
// dan pesan galat ("" bila valid). Batas atas mengikuti kolom payments.void_reason.
func validasiAlasanVoid(alasan string) (string, string) {
	alasan = strings.TrimSpace(alasan)
	n := utf8.RuneCountInString(alasan)
	if n < 5 {
		return "", "alasan pembatalan wajib diisi (minimal 5 karakter)"
	}
	if n > 500 {
		return "", "alasan pembatalan maksimal 500 karakter"
	}
	return alasan, ""
}

// VoidPayment godoc
// POST /api/admin/payments/{id}/void
func (h *Handler) VoidPayment(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req VoidPaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	alasan, msg := validasiAlasanVoid(req.Alasan)
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	// Verifikasi brand sebelum mengubah apa pun.
	if _, err := h.repo.GetByID(r.Context(), id, identity.GetBrandID(r.Context())); err != nil {
		handleRepoError(w, err)
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	updated, err := h.repo.Void(r.Context(), id, alasan, adminID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	log.Printf("[AUDIT] pembayaran %d (booking %d) dibatalkan oleh admin %d", id, updated.BookingID, adminID)
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
	default:
		// Pesan bisnis ditampilkan apa adanya, tanpa awalan teknis yang ditambahkan saat error dibungkus.
		for _, bisnis := range []error{ErrSeatUnavailable, ErrOverpayment, ErrBookingClosed, ErrNotVoidable, ErrVoidBelowRefund} {
			if errors.Is(err, bisnis) {
				writeError(w, http.StatusConflict, bisnis.Error())
				return
			}
		}
		log.Printf("[ERROR] payment: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal, silakan coba lagi")
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
