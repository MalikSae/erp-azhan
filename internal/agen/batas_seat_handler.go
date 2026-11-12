package agen

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"erp-azhan/api/internal/identity"
)

// GetBatasSeat GET /api/admin/agen/{akunID}/batas-seat: batas hold seat yang berlaku, seat yang sedang
// ditahan, dan riwayat perubahan batas khusus. Brand admin hanya untuk agen di brand-nya.
func (h *Handler) GetBatasSeat(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	b, err := h.repo.GetBatasSeatAgen(r.Context(), id, identity.GetBrandID(r.Context()))
	if err != nil {
		handleBatasSeatError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// UbahBatasSeat PUT /api/admin/agen/{akunID}/batas-seat {per_jadwal, total, alasan} (izin agen.edit).
// null pada per_jadwal atau total = kembali ke batas bawaan untuk kolom itu.
func (h *Handler) UbahBatasSeat(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	var req UbahBatasSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.UbahBatasSeatAgen(r.Context(), id, identity.GetBrandID(r.Context()), adminID, req); err != nil {
		handleBatasSeatError(w, err)
		return
	}
	log.Printf("[AUDIT] agen_batas_seat akun_id=%d per_jadwal=%v total=%v admin_id=%d", id, nullInt(req.PerJadwal), nullInt(req.Total), adminID)
	b, err := h.repo.GetBatasSeatAgen(r.Context(), id, identity.GetBrandID(r.Context()))
	if err != nil {
		handleBatasSeatError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func handleBatasSeatError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrBatasSeatTidakValid), errors.Is(err, ErrAlasanBatasWajib):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrBatasSeatSama):
		writeError(w, http.StatusConflict, err.Error())
	default:
		handleError(w, err)
	}
}
