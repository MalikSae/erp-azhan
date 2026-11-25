package topbar

import (
	"database/sql"
	"log"
	"net/http"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
)

type Handler struct{ db *sql.DB }

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db} }

// aksesDari membaca brand dan izin dari token yang sudah divalidasi RequireAuth.
func aksesDari(r *http.Request) Akses {
	return Akses{BrandID: identity.GetBrandID(r.Context()), Izin: identity.GetPerms(r.Context())}
}

// Cari: GET /api/admin/cari?q=
func (h *Handler) Cari(w http.ResponseWriter, r *http.Request) {
	hasil, err := Cari(r.Context(), h.db, aksesDari(r), r.URL.Query().Get("q"))
	if err != nil {
		log.Printf("[ERROR] %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mencari.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, hasil)
}

// Notifikasi: GET /api/admin/notifikasi
func (h *Handler) Notifikasi(w http.ResponseWriter, r *http.Request) {
	n, err := HitungNotifikasi(r.Context(), h.db, aksesDari(r))
	if err != nil {
		log.Printf("[ERROR] %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal memuat notifikasi.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, n)
}
