package analitik

import (
	"log"
	"net/http"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
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
		shared.WriteJSON(w, http.StatusInternalServerError, map[string]string{"error": "gagal mengambil ringkasan marketing"})
		return
	}
	shared.WriteJSON(w, http.StatusOK, hasil)
}
