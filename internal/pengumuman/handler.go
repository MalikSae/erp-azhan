package pengumuman

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

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
	default:
		log.Printf("[ERROR] pengumuman: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal memproses pengumuman")
	}
}

// brandAdmin: brand admin yang login, atau brand_id dari query (admin holding). nil bila tidak ada keduanya.
func brandAdmin(r *http.Request) *int64 {
	if b := identity.GetBrandID(r.Context()); b != nil {
		return b
	}
	if id, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64); err == nil && id > 0 {
		return &id
	}
	return nil
}

// DaftarAdmin GET /api/admin/pengumuman?brand_id=
func (h *Handler) DaftarAdmin(w http.ResponseWriter, r *http.Request) {
	items, err := Daftar(r.Context(), h.db, brandAdmin(r), time.Now())
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id pengumuman tidak valid")
		return 0, false
	}
	return id, true
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return false
	}
	return true
}

// Buat POST /api/admin/pengumuman. Admin satu brand memakai brand-nya; admin holding wajib mengirim brand_id.
func (h *Handler) Buat(w http.ResponseWriter, r *http.Request) {
	var in Input
	if !decode(w, r, &in) {
		return
	}
	brand := identity.GetBrandID(r.Context())
	if brand == nil {
		brand = in.BrandID
	}
	if brand == nil || *brand < 1 {
		writeError(w, http.StatusBadRequest, "brand wajib dipilih")
		return
	}
	id, err := Buat(r.Context(), h.db, *brand, identity.GetAdminUserID(r.Context()), in)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	log.Printf("[AUDIT] pengumuman_dibuat id=%d brand_id=%d admin_id=%d", id, *brand, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusCreated, map[string]int64{"id": id})
}

// Ubah PUT /api/admin/pengumuman/{id}
func (h *Handler) Ubah(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in Input
	if !decode(w, r, &in) {
		return
	}
	if err := Ubah(r.Context(), h.db, id, identity.GetBrandID(r.Context()), in); err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "pengumuman diperbarui"})
}

// Hapus DELETE /api/admin/pengumuman/{id}
func (h *Handler) Hapus(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := Hapus(r.Context(), h.db, id, identity.GetBrandID(r.Context())); err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "pengumuman dihapus"})
}

// Akun GET /api/account/pengumuman?limit=2: pengumuman yang tayang dan sesuai akun. {items, total}.
func (h *Handler) Akun(w http.ResponseWriter, r *http.Request) {
	akunID := identity.GetCustomerAccountID(r.Context())
	brandID := identity.GetCustomerBrandID(r.Context())
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 0 || limit > 100 {
		limit = 0
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	items, total, err := UntukAkun(ctx, h.db, brandID, akunID, limit, time.Now())
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
}
