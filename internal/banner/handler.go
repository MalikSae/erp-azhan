package banner

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/identity"
)

type Handler struct {
	db        *sql.DB
	dirUnggah string
	now       func() time.Time
}

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db, dirUnggah: "uploads", now: time.Now} }

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
		log.Printf("[ERROR] banner: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal memproses banner.")
	}
}

func idParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid.")
		return 0, false
	}
	return id, true
}

// brandAdmin: brand admin brand, atau brand pilihan admin holding (query brand_id atau field body). 0 = belum dipilih.
func brandAdmin(r *http.Request, dariBody int64) int64 {
	if b := identity.GetBrandID(r.Context()); b != nil {
		return *b
	}
	if dariBody > 0 {
		return dariBody
	}
	id, _ := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64)
	return id
}

type daftarView struct {
	Banner []Banner `json:"banner"`
}

// Daftar GET /api/admin/banner-promosi?brand_id=
func (h *Handler) Daftar(w http.ResponseWriter, r *http.Request) {
	brand := brandAdmin(r, 0)
	if brand <= 0 {
		writeError(w, http.StatusBadRequest, "Pilih brand.")
		return
	}
	items, err := Daftar(r.Context(), h.db, brand, h.now())
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, daftarView{Banner: items})
}

// Buat POST /api/admin/banner-promosi
func (h *Handler) Buat(w http.ResponseWriter, r *http.Request) {
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	in.BrandID = brandAdmin(r, in.BrandID)
	if in.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "Pilih brand.")
		return
	}
	id, err := Buat(r.Context(), h.db, h.dirUnggah, identity.GetAdminUserID(r.Context()), in)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	b, err := ambil(r.Context(), h.db, id, nil)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	b.Status = statusBanner(b, hariIni(h.now()))
	writeJSON(w, http.StatusCreated, b)
}

// Ubah PUT /api/admin/banner-promosi/{id}
func (h *Handler) Ubah(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	scope := identity.GetBrandID(r.Context())
	if err := Ubah(r.Context(), h.db, h.dirUnggah, id, scope, identity.GetAdminUserID(r.Context()), in); err != nil {
		tulisGalat(w, err)
		return
	}
	b, err := ambil(r.Context(), h.db, id, scope)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	b.Status = statusBanner(b, hariIni(h.now()))
	writeJSON(w, http.StatusOK, b)
}

// Hapus DELETE /api/admin/banner-promosi/{id}
func (h *Handler) Hapus(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := Hapus(r.Context(), h.db, id, identity.GetBrandID(r.Context())); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Urutkan PUT /api/admin/banner-promosi/urutan: {brand_id, ids}
func (h *Handler) Urutkan(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BrandID int64   `json:"brand_id"`
		IDs     []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	brand := brandAdmin(r, req.BrandID)
	if brand <= 0 {
		writeError(w, http.StatusBadRequest, "Pilih brand.")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	if err := Urutkan(r.Context(), tx, brand, req.IDs); err != nil {
		tulisGalat(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Promosi GET /api/public/v1/promosi?brand_id=&platform=web|app (kosong = app) : banner tayang (maks 5) untuk Beranda app.
func (h *Handler) Promosi(w http.ResponseWriter, r *http.Request) {
	brand, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64)
	if err != nil || brand <= 0 {
		writeError(w, http.StatusBadRequest, "Brand wajib diisi.")
		return
	}
	p, err := AmbilPromosi(r.Context(), h.db, brand, r.URL.Query().Get("platform"), h.now())
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Klik POST /api/public/promosi/{id}/klik: selalu 204.
func (h *Handler) Klik(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r)
	if !ok {
		return
	}
	if err := CatatKlik(r.Context(), h.db, id, h.now()); err != nil {
		log.Printf("[ERROR] klik banner: %v", err)
	}
	w.WriteHeader(http.StatusNoContent)
}
