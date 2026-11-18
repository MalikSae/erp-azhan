package manasik

import (
	"context"
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
	case errors.Is(err, ErrTidakAda):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrBukanJamaah):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrRSVPDitutup), errors.Is(err, ErrBelumAda), errors.Is(err, ErrSudahDimulai),
		errors.Is(err, ErrAdaAbsensi), errors.Is(err, ErrAbsensiBelumDibuka):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("[ERROR] manasik: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal memproses manasik.")
	}
}

func idParam(w http.ResponseWriter, r *http.Request, nama, label string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, nama), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, label+" Tidak valid.")
		return 0, false
	}
	return id, true
}

// dalamTx menjalankan fn dalam satu transaksi; rollback bila fn gagal.
func (h *Handler) dalamTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// lingkupBrand: brand admin brand, atau filter ?brand_id untuk admin holding (nil = semua brand).
func lingkupBrand(r *http.Request) *int64 {
	if b := identity.GetBrandID(r.Context()); b != nil {
		return b
	}
	if v, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64); err == nil && v > 0 {
		return &v
	}
	return nil
}

func (h *Handler) tulisDetail(w http.ResponseWriter, r *http.Request, id int64, status int) {
	d, err := AmbilDetail(r.Context(), h.db, id, identity.GetBrandID(r.Context()), h.now())
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, status, d)
}

// Daftar GET /api/admin/manasik?tab=akan_datang|selesai&brand_id=
func (h *Handler) Daftar(w http.ResponseWriter, r *http.Request) {
	out, err := Daftar(r.Context(), h.db, lingkupBrand(r), r.URL.Query().Get("tab") == "selesai", h.now())
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// PaketTersedia GET /api/admin/manasik/paket-tersedia?brand_id=
func (h *Handler) PaketTersedia(w http.ResponseWriter, r *http.Request) {
	out, err := PaketTersedia(r.Context(), h.db, lingkupBrand(r), h.now())
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

type buatRequest struct {
	Input
	ScheduleIDs []int64 `json:"schedule_ids"`
}

// Buat POST /api/admin/manasik: isi manasik + schedule_ids.
func (h *Handler) Buat(w http.ResponseWriter, r *http.Request) {
	var req buatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	var id int64
	err := h.dalamTx(r.Context(), func(tx *sql.Tx) error {
		var err error
		id, err = Buat(r.Context(), tx, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()), req.Input, req.ScheduleIDs, h.now())
		return err
	})
	if err != nil {
		tulisGalat(w, err)
		return
	}
	h.tulisDetail(w, r, id, http.StatusCreated)
}

// Detail GET /api/admin/manasik/{id}
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id manasik")
	if !ok {
		return
	}
	h.tulisDetail(w, r, id, http.StatusOK)
}

// Ubah PUT /api/admin/manasik/{id}
func (h *Handler) Ubah(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id manasik")
	if !ok {
		return
	}
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	if err := Ubah(r.Context(), h.db, id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()), in); err != nil {
		tulisGalat(w, err)
		return
	}
	h.tulisDetail(w, r, id, http.StatusOK)
}

// Hapus DELETE /api/admin/manasik/{id}
func (h *Handler) Hapus(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id manasik")
	if !ok {
		return
	}
	if err := Hapus(r.Context(), h.db, id, identity.GetBrandID(r.Context()), h.now()); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TambahPaket POST /api/admin/manasik/{id}/paket: {schedule_ids}
func (h *Handler) TambahPaket(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id manasik")
	if !ok {
		return
	}
	var req struct {
		ScheduleIDs []int64 `json:"schedule_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	err := h.dalamTx(r.Context(), func(tx *sql.Tx) error {
		return TambahPaket(r.Context(), tx, id, identity.GetBrandID(r.Context()), req.ScheduleIDs, h.now())
	})
	if err != nil {
		tulisGalat(w, err)
		return
	}
	h.tulisDetail(w, r, id, http.StatusOK)
}

// LepasPaket DELETE /api/admin/manasik/{id}/paket/{schedule_id}
func (h *Handler) LepasPaket(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id manasik")
	if !ok {
		return
	}
	sid, ok := idParam(w, r, "schedule_id", "id paket")
	if !ok {
		return
	}
	err := h.dalamTx(r.Context(), func(tx *sql.Tx) error {
		return LepasPaket(r.Context(), tx, id, identity.GetBrandID(r.Context()), sid, h.now())
	})
	if err != nil {
		tulisGalat(w, err)
		return
	}
	h.tulisDetail(w, r, id, http.StatusOK)
}

// Absensi PUT /api/admin/manasik/{id}/absensi: {pax_id, hadir}
func (h *Handler) Absensi(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id manasik")
	if !ok {
		return
	}
	var req struct {
		PaxID int64 `json:"pax_id"`
		Hadir *bool `json:"hadir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PaxID <= 0 || req.Hadir == nil {
		writeError(w, http.StatusBadRequest, "Pax dan hadir wajib diisi.")
		return
	}
	if err := CatatHadir(r.Context(), h.db, id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()), req.PaxID, *req.Hadir, h.now()); err != nil {
		tulisGalat(w, err)
		return
	}
	h.tulisDetail(w, r, id, http.StatusOK)
}

// UntukPaket GET /api/admin/schedules/{id}/manasik: ringkasan manasik paket ({manasik: null} bila belum ada).
func (h *Handler) UntukPaket(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id", "id paket")
	if !ok {
		return
	}
	m, err := UntukPaket(r.Context(), h.db, id, identity.GetBrandID(r.Context()))
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"manasik": m})
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
		writeError(w, http.StatusNotFound, "Booking tidak ditemukan.")
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
		writeError(w, http.StatusBadRequest, "Jamaah dan hadir wajib diisi.")
		return
	}
	now := h.now()
	err := SimpanRSVP(r.Context(), h.db, acc.ID, id, req.JamaahID, *req.Hadir, now)
	if errors.Is(err, ErrTidakAda) {
		writeError(w, http.StatusNotFound, "Booking tidak ditemukan.")
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
