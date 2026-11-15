package manasik

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/customer"
	"erp-azhan/api/internal/identity"
)

type Handler struct {
	db  *sql.DB
	now func() time.Time
}

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db, now: time.Now} }

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func tulisGalat(w http.ResponseWriter, err error) {
	var val *ErrValidasi
	switch {
	case errors.As(err, &val):
		writeError(w, http.StatusBadRequest, val.Pesan)
	case errors.Is(err, ErrJadwalTidakAda), errors.Is(err, ErrTidakAda):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrBukanJamaah):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrRSVPDitutup), errors.Is(err, ErrBelumAda):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("[ERROR] manasik: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal memproses manasik")
	}
}

func idParam(w http.ResponseWriter, r *http.Request, nama, label string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, nama), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, label+" tidak valid")
		return 0, false
	}
	return id, true
}

// adminView: manasik jadwal (nil bila belum diisi) dan rekap konfirmasi hadir.
type adminView struct {
	Manasik *Manasik `json:"manasik"`
	Rekap   *Rekap   `json:"rekap"`
}

func (h *Handler) tampilAdmin(w http.ResponseWriter, r *http.Request, scheduleID int64) {
	m, err := AmbilAdmin(r.Context(), h.db, scheduleID, identity.GetBrandID(r.Context()))
	if err != nil {
		tulisGalat(w, err)
		return
	}
	out := adminView{Manasik: m}
	if m != nil {
		if out.Rekap, err = RekapJadwal(r.Context(), h.db, m); err != nil {
			tulisGalat(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, out)
}

// AmbilJadwal GET /api/admin/schedules/{id}/manasik
func (h *Handler) AmbilJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id jadwal")
	if !ok {
		return
	}
	h.tampilAdmin(w, r, id)
}

// SimpanJadwal PUT /api/admin/schedules/{id}/manasik
func (h *Handler) SimpanJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id jadwal")
	if !ok {
		return
	}
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}
	if _, err := Simpan(r.Context(), h.db, id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()), in); err != nil {
		tulisGalat(w, err)
		return
	}
	h.tampilAdmin(w, r, id)
}

// HapusJadwal DELETE /api/admin/schedules/{id}/manasik
func (h *Handler) HapusJadwal(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id jadwal")
	if !ok {
		return
	}
	if err := Hapus(r.Context(), h.db, id, identity.GetBrandID(r.Context())); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Booking GET /api/account/bookings/{id}/manasik
func (h *Handler) Booking(w http.ResponseWriter, r *http.Request) {
	acc := customer.FromContext(r.Context())
	id, ok := idParam(w, r, "id", "id booking")
	if !ok {
		return
	}
	out, err := UntukBooking(r.Context(), h.db, acc.ID, id, h.now())
	if errors.Is(err, ErrTidakAda) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	}
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type rsvpRequest struct {
	JamaahID int64 `json:"jamaah_id"`
	Hadir    *bool `json:"hadir"`
}

// RSVP PUT /api/account/bookings/{id}/manasik/rsvp: {jamaah_id, hadir}. Mengembalikan tampilan manasik booking terbaru.
func (h *Handler) RSVP(w http.ResponseWriter, r *http.Request) {
	acc := customer.FromContext(r.Context())
	id, ok := idParam(w, r, "id", "id booking")
	if !ok {
		return
	}
	var req rsvpRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.JamaahID <= 0 || req.Hadir == nil {
		writeError(w, http.StatusBadRequest, "jamaah_id dan hadir wajib diisi")
		return
	}
	now := h.now()
	err := SimpanRSVP(r.Context(), h.db, acc.ID, id, req.JamaahID, *req.Hadir, now)
	if errors.Is(err, ErrTidakAda) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	}
	if err != nil {
		tulisGalat(w, err)
		return
	}
	out, err := UntukBooking(r.Context(), h.db, acc.ID, id, now)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
