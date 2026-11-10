package topbar

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

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

// aksesDari membaca brand dan izin dari token yang sudah divalidasi RequireAuth.
func aksesDari(r *http.Request) Akses {
	return Akses{BrandID: identity.GetBrandID(r.Context()), Izin: identity.GetPerms(r.Context())}
}

// Cari: GET /api/admin/cari?q=
func (h *Handler) Cari(w http.ResponseWriter, r *http.Request) {
	hasil, err := Cari(r.Context(), h.db, aksesDari(r), r.URL.Query().Get("q"))
	if err != nil {
		log.Printf("[ERROR] %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mencari")
		return
	}
	writeJSON(w, http.StatusOK, hasil)
}

// Notifikasi: GET /api/admin/notifikasi
func (h *Handler) Notifikasi(w http.ResponseWriter, r *http.Request) {
	n, err := HitungNotifikasi(r.Context(), h.db, aksesDari(r))
	if err != nil {
		log.Printf("[ERROR] %v", err)
		writeError(w, http.StatusInternalServerError, "gagal memuat notifikasi")
		return
	}
	writeJSON(w, http.StatusOK, n)
}
