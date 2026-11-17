package crmdeal

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/media"
	"github.com/google/uuid"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) ProcessDeal(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if _, err := uuid.Parse(idempotencyKey); err != nil {
		writeError(w, http.StatusBadRequest, "Idempotency-Key UUID wajib diisi.")
		return
	}

	var req DealRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	var finalBrandID int64
	if brandID != nil {
		finalBrandID = *brandID
	} else if req.BrandID != nil {
		finalBrandID = *req.BrandID
	}
	if finalBrandID <= 0 {
		writeError(w, http.StatusBadRequest, "Brand wajib dipilih.")
		return
	}
	if _, err := uuid.Parse(req.CRMLeadID); err != nil {
		writeError(w, http.StatusBadRequest, "Lead tidak valid.")
		return
	}
	if req.Pax == 0 {
		req.Pax = 1
	}
	if req.Pax > 1 {
		writeError(w, http.StatusBadRequest, "Deal CRM saat ini hanya mendukung 1 pax per transaksi. Untuk booking lebih dari 1 orang, gunakan menu Booking di dashboard admin.")
		return
	}
	if req.ScheduleID <= 0 || req.Pax < 1 {
		writeError(w, http.StatusBadRequest, "Paket dan pax wajib valid.")
		return
	}
	if req.RoomType != "Quad" && req.RoomType != "Triple" && req.RoomType != "Double" {
		writeError(w, http.StatusBadRequest, "Tipe kamar harus Quad, Triple, atau Double.")
		return
	}
	if req.Jamaah.ID == nil && (strings.TrimSpace(req.Jamaah.NamaLengkap) == "" || strings.TrimSpace(req.Jamaah.NoHP) == "") {
		writeError(w, http.StatusBadRequest, "Nama lengkap dan nomor HP jamaah wajib diisi.")
		return
	}
	if req.CommitmentType == "book_seat" {
		if req.SeatHoldExpiresAt == nil {
			writeError(w, http.StatusBadRequest, "Batas hold seat wajib diisi.")
			return
		}
		expiresAt, err := time.Parse(time.RFC3339, *req.SeatHoldExpiresAt)
		if err != nil || !expiresAt.After(time.Now().Add(time.Minute)) {
			writeError(w, http.StatusBadRequest, "Batas hold seat harus di masa depan.")
			return
		}
	} else if req.CommitmentType != "dp" && req.CommitmentType != "lunas" {
		writeError(w, http.StatusBadRequest, "Jenis komitmen tidak valid.")
		return
	}
	if req.CommitmentType != "book_seat" && req.PaymentAmount == nil {
		writeError(w, http.StatusBadRequest, "Nominal pembayaran wajib diisi.")
		return
	}
	// Bukti bayar harus unggahan admin resmi milik brand pemanggil, sama seperti alur pembayaran biasa.
	// URL bebas ditolak: payments.bukti_url menjadi dasar izin baca berkas lintas brand (media.canReadAdminFile).
	if req.PaymentProofURL != nil {
		if u := strings.TrimSpace(*req.PaymentProofURL); u == "" {
			req.PaymentProofURL = nil
		} else if err := media.ValidateAdminPaymentProof(r.Context(), h.repo.db, u, brandID); err != nil {
			var uploadErr *media.UploadError
			if errors.As(err, &uploadErr) {
				writeError(w, http.StatusBadRequest, uploadErr.Message)
				return
			}
			log.Printf("[ERROR] crmdeal validasi bukti: %v", err)
			writeError(w, http.StatusInternalServerError, "Proses Deal gagal.")
			return
		} else {
			req.PaymentProofURL = &u
		}
	}

	response, err := h.repo.Process(r.Context(), finalBrandID, identity.GetAdminUserID(r.Context()), idempotencyKey, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrMaxPaxExceeded):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrSeatUnavailable), errors.Is(err, ErrScheduleClosed), errors.Is(err, ErrAmbiguousJamaah), errors.Is(err, ErrIdempotencyConflict), errors.Is(err, ErrLeadAlreadyConverted):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrBrandCodeMissing), errors.Is(err, ErrInvalidPayment):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "Proses Deal gagal.")
		}
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
