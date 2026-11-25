package manifest

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/identity"
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
		log.Printf("[ERROR] manifest: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal memproses manifest.")
	}
}

func idPaket(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID paket tidak valid.")
		return 0, false
	}
	return id, true
}

// Detail GET /api/admin/schedules/{id}/manifest
func (h *Handler) Detail(w http.ResponseWriter, r *http.Request) {
	id, ok := idPaket(w, r)
	if !ok {
		return
	}
	m, err := Ambil(r.Context(), h.db, id, identity.GetBrandID(r.Context()))
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// Simpan PUT /api/admin/schedules/{id}/manifest {pnr, detail_penerbangan}
func (h *Handler) Simpan(w http.ResponseWriter, r *http.Request) {
	id, ok := idPaket(w, r)
	if !ok {
		return
	}
	var in InputInfo
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	scope := identity.GetBrandID(r.Context())
	if err := SimpanInfo(r.Context(), h.db, id, scope, identity.GetAdminUserID(r.Context()), in); err != nil {
		tulisGalat(w, err)
		return
	}
	m, err := Ambil(r.Context(), h.db, id, scope)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, m)
}

// Unduh GET /api/admin/schedules/{id}/manifest/unduh: file .xlsx, lalu jejak unduhan dicatat.
func (h *Handler) Unduh(w http.ResponseWriter, r *http.Request) {
	id, ok := idPaket(w, r)
	if !ok {
		return
	}
	m, err := Ambil(r.Context(), h.db, id, identity.GetBrandID(r.Context()))
	if err != nil {
		tulisGalat(w, err)
		return
	}
	f, err := BuatFile(m)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	defer f.Close()
	buf, err := f.WriteToBuffer()
	if err != nil {
		tulisGalat(w, err)
		return
	}
	if err := CatatUnduhan(r.Context(), h.db, id, identity.GetAdminUserID(r.Context()), len(m.Pax)); err != nil {
		tulisGalat(w, err)
		return
	}
	nama := NamaFile(m.Paket)
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="`+nama+`"; filename*=UTF-8''`+url.PathEscape(nama))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Length", strconv.Itoa(buf.Len()))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(buf.Bytes())
}
