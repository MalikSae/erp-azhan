package portal

import (
	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/komisi"
	"net/http"
)

func (h *Handler) GetCashback(w http.ResponseWriter, r *http.Request) {
	data, err := komisi.ReadCashback(r.Context(), h.db, identity.GetPortalJamaahID(r.Context()))
	if err != nil {
		writeError(w, 500, "gagal memuat kredit cashback")
		return
	}
	writeJSON(w, 200, data)
}
