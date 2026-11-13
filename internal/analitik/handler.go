package analitik

import (
	"encoding/json"
	"log"
	"net/http"

	"erp-azhan/api/internal/identity"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// MarketingRingkas GET /api/admin/analytics/marketing-ringkas
// Admin yang terikat brand hanya melihat brand-nya sendiri.
func (h *Handler) MarketingRingkas(w http.ResponseWriter, r *http.Request) {
	hasil, err := h.repo.MarketingRingkas(r.Context(), identity.GetBrandID(r.Context()))
	if err != nil {
		log.Printf("[ERROR] analitik marketing ringkas: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "gagal mengambil ringkasan marketing"})
		return
	}
	writeJSON(w, http.StatusOK, hasil)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
