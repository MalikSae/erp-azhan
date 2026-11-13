package benefit

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return id, true
}

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "benefit tidak ditemukan")
	case errors.Is(err, ErrDuplikat), errors.Is(err, ErrDuplikatNonaktif), errors.Is(err, ErrDipakai):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrBrandTidakSah):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		log.Printf("[ERROR] benefit: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal, silakan coba lagi")
	}
}

const pesanBrandWajib = "brand_id wajib diisi"

// brandTujuan menentukan brand yang dikerjakan. Admin yang terikat satu brand selalu memakai brand-nya;
// brand_id dari request diabaikan. Admin holding wajib mengirim brand_id (dariRequest).
func brandTujuan(w http.ResponseWriter, r *http.Request, dariRequest *int64) (int64, bool) {
	if scoped := identity.GetBrandID(r.Context()); scoped != nil {
		return *scoped, true
	}
	if dariRequest == nil || *dariRequest <= 0 {
		writeError(w, http.StatusBadRequest, pesanBrandWajib)
		return 0, false
	}
	return *dariRequest, true
}

// brandDariQuery membaca brand_id dari query string; nil bila kosong atau tidak valid.
func brandDariQuery(r *http.Request) *int64 {
	id, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64)
	if err != nil || id <= 0 {
		return nil
	}
	return &id
}

// ─── Master Benefit Paket (masterdata.view / masterdata.edit) ────────────────

// ListBenefits GET /api/admin/benefits?brand_id=&active=1
func (h *Handler) ListBenefits(w http.ResponseWriter, r *http.Request) {
	brandID, ok := brandTujuan(w, r, brandDariQuery(r))
	if !ok {
		return
	}
	items, err := h.repo.List(r.Context(), brandID, r.URL.Query().Get("active") == "1")
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ListIkon GET /api/admin/benefits/ikon
func (h *Handler) ListIkon(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, DaftarIkon)
}

// CreateBenefit POST /api/admin/benefits
// Dipakai juga oleh POST /api/admin/schedules/benefit-options (izin schedule.edit): admin paket boleh
// menambah benefit baru dari form paket, tetapi tidak mengubah atau menonaktifkan yang sudah ada.
func (h *Handler) CreateBenefit(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	teks, ikon, err := ValidasiIsi(req.Teks, req.Ikon)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	brandID, ok := brandTujuan(w, r, req.BrandID)
	if !ok {
		return
	}
	b, err := h.repo.Create(r.Context(), brandID, teks, ikon)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// UpdateBenefit PUT /api/admin/benefits/{id}
func (h *Handler) UpdateBenefit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	teks, ikon, err := ValidasiIsi(req.Teks, req.Ikon)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	b, err := h.repo.Update(r.Context(), id, identity.GetBrandID(r.Context()), teks, ikon, req.IsActive)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// DeleteBenefit DELETE /api/admin/benefits/{id}
func (h *Handler) DeleteBenefit(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	if err := h.repo.Delete(r.Context(), id, identity.GetBrandID(r.Context())); err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "benefit berhasil dihapus"})
}

// ─── Pilihan benefit di form paket (schedule.view / schedule.edit) ───────────

// ListOpsi GET /api/admin/schedules/benefit-options?brand_id=
// Daftar ikon dan benefit aktif brand, untuk admin paket yang tidak punya izin masterdata.
func (h *Handler) ListOpsi(w http.ResponseWriter, r *http.Request) {
	brandID, ok := brandTujuan(w, r, brandDariQuery(r))
	if !ok {
		return
	}
	items, err := h.repo.List(r.Context(), brandID, true)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, Opsi{Ikon: DaftarIkon, Benefits: items})
}
