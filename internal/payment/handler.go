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

// maxNominal adalah batas kolom jumlah DECIMAL(12,0) pada payments.
const maxNominal = 999_999_999_999

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
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data transaksi.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
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
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	if !exists {
		shared.WriteError(w, http.StatusNotFound, "Booking tidak ditemukan atau bukan milik brand Anda.")
		return
	}

	items, err := h.repo.ListByBookingID(r.Context(), bookingID, brandID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data payment.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
}

// ListAllPayments menampilkan inbox pembayaran lintas brand untuk pusat,
// atau otomatis scoped untuk admin brand.
func (h *Handler) ListAllPayments(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	switch status {
	case "", StatusPending, StatusConfirmed, StatusRejected, StatusVoid:
	default:
		shared.WriteError(w, http.StatusBadRequest, "Status tidak valid.")
		return
	}
	items, err := h.repo.ListAll(r.Context(), identity.GetBrandID(r.Context()), status)
	if err != nil {
		log.Printf("[ERROR] payment: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil daftar pembayaran.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
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
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	// Endpoint admin: sumber selalu "admin" (langsung terkonfirmasi); nilai dari klien diabaikan.
	req.Source = SourceAdmin

	if req.Jumlah <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Jumlah pembayaran harus lebih dari 0.")
		return
	}
	if !shared.RupiahBulat(req.Jumlah) {
		shared.WriteError(w, http.StatusBadRequest, "Nominal harus bilangan bulat rupiah.")
		return
	}
	if req.Jumlah > maxNominal {
		shared.WriteError(w, http.StatusBadRequest, "Jumlah pembayaran terlalu besar.")
		return
	}
	// Pembayaran admin langsung terkonfirmasi; tanggal masa depan merusak laporan (JB-10).
	// Aturan sama dengan konfirmasi transfer portal (JM-22).
	if req.Tanggal != nil && strings.TrimSpace(*req.Tanggal) != "" && !shared.TanggalTidakDiMasaDepan(*req.Tanggal) {
		shared.WriteError(w, http.StatusBadRequest, "Tanggal pembayaran tidak valid atau berada di masa depan.")
		return
	}

	// Aturan catat pembayaran oleh admin: transfer wajib menyebut rekening tujuan,
	// tunai wajib menyebut nama staff yang menerima uangnya.
	metode := metodeTransfer
	if req.Metode != nil && strings.TrimSpace(*req.Metode) != "" {
		metode = strings.ToLower(strings.TrimSpace(*req.Metode))
	}
	switch metode {
	case metodeTransfer:
		if req.BankAccountID == nil || *req.BankAccountID <= 0 {
			shared.WriteError(w, http.StatusBadRequest, "Rekening tujuan wajib dipilih untuk pembayaran transfer.")
			return
		}
	case metodeTunai:
		if req.ReceivedBy == nil || strings.TrimSpace(*req.ReceivedBy) == "" {
			shared.WriteError(w, http.StatusBadRequest, "Nama staff penerima wajib diisi untuk pembayaran tunai.")
			return
		}
		trimmed := strings.TrimSpace(*req.ReceivedBy)
		req.ReceivedBy = &trimmed
		req.BankAccountID = nil // tunai tidak melewati rekening
	default:
		shared.WriteError(w, http.StatusBadRequest, "Metode pembayaran harus transfer atau cash.")
		return
	}
	if metode != metodeTunai {
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
				shared.WriteError(w, http.StatusBadRequest, uploadErr.Message)
				return
			}
			log.Printf("[ERROR] payment: %v", err)
			shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return
		} else {
			req.BuktiURL = &trimmed
		}
	}

	// Transfer yang dicatat admin wajib berbukti: pencatat dan verifikatornya orang yang sama, jadi bukti
	// satu-satunya pegangan pihak lain (keputusan 11 November 2026). Tunai tidak punya bukti transfer.
	if pesan := pesanBuktiWajib(metode, req.BuktiURL); pesan != "" {
		shared.WriteError(w, http.StatusBadRequest, pesan)
		return
	}

	// Verify booking
	exists, err := h.repo.BookingExistsForBrand(r.Context(), bookingID, brandID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	if !exists {
		shared.WriteError(w, http.StatusNotFound, "Booking tidak valid atau bukan milik brand Anda.")
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
		shared.WriteError(w, http.StatusBadRequest, "Booking ini masih berstatus draft, pembayaran belum bisa diproses. Selesaikan/finalisasi booking terlebih dahulu.")
		return
	}
	if totalHarga == nil {
		shared.WriteError(w, http.StatusBadRequest, "Total tagihan booking belum ditentukan, pembayaran tidak dapat diproses.")
		return
	}
	sisaTagihan := *totalHarga - totalPaid
	if sisaTagihan <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Tagihan booking sudah lunas.")
		return
	}
	if req.Jumlah > sisaTagihan {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Jumlah pembayaran tidak boleh melebihi sisa tagihan (sisa: Rp %.0f).", sisaTagihan))
		return
	}

	adminUserID := identity.GetAdminUserID(r.Context())
	p, err := h.repo.Create(r.Context(), bookingID, &req, &adminUserID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, p)
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
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	if req.Status != StatusConfirmed && req.Status != StatusRejected {
		shared.WriteError(w, http.StatusBadRequest, "Status harus confirmed atau rejected.")
		return
	}
	if req.Status == StatusRejected && (req.RejectionReason == nil || len(*req.RejectionReason) < 3) {
		shared.WriteError(w, http.StatusBadRequest, "Alasan penolakan wajib diisi.")
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
			shared.WriteError(w, http.StatusConflict, "Pembayaran sudah berstatus "+p.Status+" dan tidak bisa diubah menjadi "+req.Status)
			return
		}
		shared.WriteJSON(w, http.StatusOK, p)
		return
	}

	updated, err := h.repo.UpdateStatus(r.Context(), id, req.Status, req.RejectionReason, identity.GetAdminUserID(r.Context()))
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, updated)
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
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	alasan, msg := validasiAlasanVoid(req.Alasan)
	if msg != "" {
		shared.WriteError(w, http.StatusBadRequest, msg)
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
	shared.WriteJSON(w, http.StatusOK, updated)
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
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		shared.WriteError(w, http.StatusNotFound, "Data tidak ditemukan.")
	case errors.Is(err, ErrCannotDelete):
		shared.WriteError(w, http.StatusConflict, "Tidak bisa menghapus pembayaran yang sudah dikonfirmasi.")
	default:
		// Pesan bisnis ditampilkan apa adanya, tanpa awalan teknis yang ditambahkan saat error dibungkus.
		for _, bisnis := range []error{ErrSeatUnavailable, ErrOverpayment, ErrBookingClosed, ErrNotVoidable, ErrVoidBelowRefund, ErrSudahDiputuskan} {
			if errors.Is(err, bisnis) {
				shared.WriteError(w, http.StatusConflict, bisnis.Error())
				return
			}
		}
		log.Printf("[ERROR] payment: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusBadRequest, param+" Tidak valid.")
		return 0, false
	}
	return id, true
}
