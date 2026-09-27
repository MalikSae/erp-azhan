package agen

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"erp-azhan/api/internal/identity"
)

// Bukti transfer keluar hanya boleh hasil upload admin ke kategori media
// terproteksi (sama dengan bukti pembayaran lain).
var buktiKeluarURL = regexp.MustCompile(`^/api/admin/media/payment-proofs/([0-9a-f-]{36}\.(jpg|png|webp|gif|pdf))$`)

func handleSprint5Error(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrBukanAgenAktif):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrPencairanMinimal), errors.Is(err, ErrPencairanMelebihi),
		errors.Is(err, ErrRekeningTidakLengkap), errors.Is(err, ErrAgenTidakValid),
		errors.Is(err, ErrKaitanTidakBerubah), errors.Is(err, ErrKaitanModeTidakSah):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrPencairanMasihPending), errors.Is(err, ErrPencairanBukanPending),
		errors.Is(err, ErrSaldoTidakCukup), errors.Is(err, ErrKaitanBukanJalur3),
		errors.Is(err, ErrKaitanSudahKomisi), errors.Is(err, ErrKaitanJamaahAgen):
		writeError(w, http.StatusConflict, err.Error())
	default:
		handleError(w, err)
	}
}

// ─── A7 (portal) ─────────────────────────────────────────────────────────────

// GetPencairan GET /api/portal/agen/pencairan
func (h *Handler) GetPencairan(w http.ResponseWriter, r *http.Request) {
	p, err := h.repo.GetPencairanPortal(r.Context(), identity.GetPortalJamaahID(r.Context()))
	if err != nil {
		handleSprint5Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// AjukanPencairan POST /api/portal/agen/pencairan
func (h *Handler) AjukanPencairan(w http.ResponseWriter, r *http.Request) {
	var req AjukanPencairanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	agenID := identity.GetPortalJamaahID(r.Context())
	id, err := h.repo.AjukanPencairan(r.Context(), agenID, req)
	if err != nil {
		handleSprint5Error(w, err)
		return
	}
	log.Printf("[AUDIT] pencairan_diajukan id=%d agen_id=%d nominal=%.0f", id, agenID, req.Nominal)
	writeJSON(w, http.StatusCreated, map[string]any{"id": id, "message": "pengajuan pencairan dikirim"})
}

// BuktiPencairanPortal GET /api/portal/agen/pencairan/{id}/bukti
func (h *Handler) BuktiPencairanPortal(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	url, err := h.repo.BuktiPencairan(r.Context(), id, identity.GetPortalJamaahID(r.Context()))
	if err != nil {
		handleSprint5Error(w, err)
		return
	}
	m := buktiKeluarURL.FindStringSubmatch(url)
	if m == nil {
		writeError(w, http.StatusNotFound, ErrNotFound.Error())
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	http.ServeFile(w, r, filepath.Join(".", "uploads", "payment-proofs", m[1]))
}

// ─── C1 (Admin Master) ───────────────────────────────────────────────────────

// ListPencairanAdmin GET /api/admin/pencairan?status=&brand_id=
func (h *Handler) ListPencairanAdmin(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListPencairanAdmin(r.Context(), r.URL.Query().Get("status"), adminBrandScope(r))
	if err != nil {
		handleSprint5Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// SetujuiPencairan POST /api/admin/pencairan/{id}/setujui {bukti_transfer_keluar_url}
func (h *Handler) SetujuiPencairan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		BuktiURL string `json:"bukti_transfer_keluar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	if !buktiKeluarURL.MatchString(strings.TrimSpace(req.BuktiURL)) {
		writeError(w, http.StatusBadRequest, "bukti transfer keluar wajib diunggah")
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.SetujuiPencairan(r.Context(), id, adminID, strings.TrimSpace(req.BuktiURL)); err != nil {
		handleSprint5Error(w, err)
		return
	}
	log.Printf("[AUDIT] pencairan_disetujui id=%d admin_id=%d", id, adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "pencairan disetujui"})
}

// TolakPencairan POST /api/admin/pencairan/{id}/tolak {alasan}
func (h *Handler) TolakPencairan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	catatan, ok := readAlasan(w, r, "catatan penolakan wajib diisi (minimal 3 karakter)")
	if !ok {
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.TolakPencairan(r.Context(), id, adminID, catatan); err != nil {
		handleSprint5Error(w, err)
		return
	}
	log.Printf("[AUDIT] pencairan_ditolak id=%d admin_id=%d", id, adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "pencairan ditolak"})
}

// ─── C4 (Admin Master) ───────────────────────────────────────────────────────

// CariJamaahKaitan GET /api/admin/kaitan-agen?q=&brand_id=
func (h *Handler) CariJamaahKaitan(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.CariJamaahKaitan(r.Context(), adminBrandScope(r), r.URL.Query().Get("q"))
	if err != nil {
		handleSprint5Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// LogKaitan GET /api/admin/kaitan-agen/{jamaahID}/log
func (h *Handler) LogKaitan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "jamaahID")
	if !ok {
		return
	}
	items, err := h.repo.ListKaitanLog(r.Context(), id)
	if err != nil {
		handleSprint5Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// GantiKaitan PUT /api/admin/kaitan-agen/{jamaahID}
func (h *Handler) GantiKaitan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "jamaahID")
	if !ok {
		return
	}
	var req GantiKaitanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	req.Alasan = strings.TrimSpace(req.Alasan)
	if n := utf8.RuneCountInString(req.Alasan); n < 5 || n > 1000 {
		writeError(w, http.StatusBadRequest, "alasan wajib diisi (5–1000 karakter)")
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.GantiKaitan(r.Context(), id, adminID, req); err != nil {
		handleSprint5Error(w, err)
		return
	}
	log.Printf("[AUDIT] kaitan_diganti jamaah_id=%d mode=%s admin_id=%d", id, req.Mode, adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "kaitan agen diganti"})
}
