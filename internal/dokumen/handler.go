package dokumen

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/media"
	"erp-azhan/api/internal/shared"
	"github.com/go-chi/chi/v5"
)

// Handler menyimpan dependency untuk semua HTTP handler dokumen.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// validJenis: jenis dokumen yang punya label di peta bersama (sama dengan enum dokumen_jamaah.jenis).
func validJenis(jenis string) bool { return shared.LabelDokumen[jenis] != "" }

// approved/rejected hanya dari submitted (menunggu verifikasi); submitted = batalkan persetujuan, hanya dari approved.
var validStatuses = map[string]bool{
	"approved":  true,
	"rejected":  true,
	"submitted": true,
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListDokumen godoc
// GET /api/admin/jamaah/{jamaah_id}/dokumen
func (h *Handler) ListDokumen(w http.ResponseWriter, r *http.Request) {
	jamaahID, ok := parseID(w, r, "jamaah_id")
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verify jamaah exists and belongs to brand
	exists, err := h.repo.JamaahExistsForBrand(r.Context(), jamaahID, brandID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	if !exists {
		shared.WriteError(w, http.StatusNotFound, "Jamaah tidak ditemukan atau bukan milik brand Anda.")
		return
	}

	items, err := h.repo.ListByJamaahID(r.Context(), jamaahID, brandID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data dokumen.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
}

// ─── Upsert ───────────────────────────────────────────────────────────────────

// UpsertDokumen godoc
// POST /api/admin/jamaah/{jamaah_id}/dokumen
func (h *Handler) UpsertDokumen(w http.ResponseWriter, r *http.Request) {
	jamaahID, ok := parseID(w, r, "jamaah_id")
	if !ok {
		return
	}

	var req CreateDokumenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	if !validJenis(req.Jenis) {
		shared.WriteError(w, http.StatusBadRequest, "Jenis dokumen tidak valid.")
		return
	}

	req.FileURL = strings.TrimSpace(req.FileURL)
	if req.FileURL == "" {
		shared.WriteError(w, http.StatusBadRequest, "File wajib diisi.")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verify jamaah
	exists, err := h.repo.JamaahExistsForBrand(r.Context(), jamaahID, brandID)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	if !exists {
		shared.WriteError(w, http.StatusNotFound, "Jamaah tidak valid atau bukan milik brand Anda.")
		return
	}

	// Berkas wajib unggahan resmi, bukan URL bebas (JB-11).
	if err := media.ValidateAdminUpload(r.Context(), h.repo.db, req.FileURL, jamaahID, identity.GetAdminUserID(r.Context())); err != nil {
		var uploadErr *media.UploadError
		if errors.As(err, &uploadErr) {
			shared.WriteError(w, http.StatusBadRequest, uploadErr.Message)
			return
		}
		log.Printf("[ERROR] dokumen: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}

	d, err := h.repo.Upsert(r.Context(), jamaahID, &req)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, d) // 200 OK karena bisa jadi update/replace
}

// ─── Update Status ────────────────────────────────────────────────────────────

// UpdateDokumenStatus godoc
// PUT /api/admin/dokumen/{id}/status
func (h *Handler) UpdateDokumenStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}

	var req UpdateDokumenStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	if !validStatuses[req.Status] {
		shared.WriteError(w, http.StatusBadRequest, "Status tidak valid.")
		return
	}
	req.RejectionReason = strings.TrimSpace(req.RejectionReason)
	if req.Version <= 0 {
		shared.WriteError(w, 400, "Versi dokumen wajib diisi.")
		return
	}
	if req.Status == "rejected" && (req.RejectionReason == "" || len(req.RejectionReason) > 500) {
		shared.WriteError(w, 400, "Alasan penolakan wajib diisi, maksimal 500 karakter.")
		return
	}
	if req.Status != "rejected" {
		req.RejectionReason = ""
	}

	brandID := identity.GetBrandID(r.Context())

	// Ambil dokumen untuk verifikasi brand
	current, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	if req.Status == "submitted" && current.Status != "approved" {
		shared.WriteError(w, http.StatusConflict, "Dokumen belum disetujui.")
		return
	}
	if req.Status != "submitted" && current.Status != "submitted" {
		shared.WriteError(w, http.StatusConflict, "Dokumen tidak menunggu verifikasi.")
		return
	}

	updated, err := h.repo.UpdateStatus(r.Context(), id, req.Status, req.Version, req.RejectionReason, identity.GetAdminUserID(r.Context()))
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, updated)
}

// ─── Riwayat ──────────────────────────────────────────────────────────────────

// RiwayatDokumen godoc
// GET /api/admin/dokumen/{id}/riwayat
// Versi lama dokumen (sebelum diganti) beserta peninjaunya, terbaru dulu.
func (h *Handler) RiwayatDokumen(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if _, err := h.repo.GetByID(r.Context(), id, identity.GetBrandID(r.Context())); err != nil {
		handleRepoError(w, err)
		return
	}
	items, err := h.repo.ListVersi(r.Context(), id)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrVersionConflict):
		shared.WriteError(w, http.StatusConflict, "Dokumen sudah berubah. Muat ulang halaman.")
	case errors.Is(err, ErrNotFound):
		shared.WriteError(w, http.StatusNotFound, "Data tidak ditemukan.")
	default:
		// Teks galat database tidak dikirim ke klien; cukup dicatat di log.
		log.Printf("[ERROR] dokumen: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	raw := chi.URLParam(r, param)
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "ID tidak valid.")
		return 0, false
	}
	return id, true
}
