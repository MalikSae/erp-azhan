package crmdeal

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
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
		writeError(w, http.StatusBadRequest, "Idempotency-Key UUID wajib diisi")
		return
	}

	var req DealRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
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
		writeError(w, http.StatusBadRequest, "brand_id wajib untuk Super Admin")
		return
	}
	if _, err := uuid.Parse(req.CRMLeadID); err != nil {
		writeError(w, http.StatusBadRequest, "crm_lead_id tidak valid")
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
		writeError(w, http.StatusBadRequest, "schedule_id dan pax wajib valid")
		return
	}
	if req.RoomType != "Quad" && req.RoomType != "Triple" && req.RoomType != "Double" {
		writeError(w, http.StatusBadRequest, "room_type harus Quad, Triple, atau Double")
		return
	}
	if req.Jamaah.ID == nil && (strings.TrimSpace(req.Jamaah.NamaLengkap) == "" || strings.TrimSpace(req.Jamaah.NoHP) == "") {
		writeError(w, http.StatusBadRequest, "nama_lengkap dan no_hp jamaah wajib diisi")
		return
	}
	if req.CommitmentType == "book_seat" {
		if req.SeatHoldExpiresAt == nil {
			writeError(w, http.StatusBadRequest, "seat_hold_expires_at wajib diisi")
			return
		}
		expiresAt, err := time.Parse(time.RFC3339, *req.SeatHoldExpiresAt)
		if err != nil || !expiresAt.After(time.Now().Add(time.Minute)) {
			writeError(w, http.StatusBadRequest, "seat_hold_expires_at harus RFC3339 dan berada di masa depan")
			return
		}
	} else if req.CommitmentType != "dp" && req.CommitmentType != "lunas" {
		writeError(w, http.StatusBadRequest, "commitment_type tidak valid")
		return
	}
	if req.CommitmentType != "book_seat" && req.PaymentAmount == nil {
		writeError(w, http.StatusBadRequest, "payment_amount wajib diisi")
		return
	}

	response, err := h.repo.Process(r.Context(), finalBrandID, identity.GetAdminUserID(r.Context()), idempotencyKey, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrMaxPaxExceeded):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, ErrNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrSeatUnavailable), errors.Is(err, ErrAmbiguousJamaah), errors.Is(err, ErrIdempotencyConflict), errors.Is(err, ErrLeadAlreadyConverted):
			writeError(w, http.StatusConflict, err.Error())
		case errors.Is(err, ErrBrandCodeMissing), errors.Is(err, ErrInvalidPayment):
			writeError(w, http.StatusUnprocessableEntity, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "proses Deal gagal")
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
