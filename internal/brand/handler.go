package brand

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/identity"
)

var hexColorRegex = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)
var domainRegex = regexp.MustCompile(`(?i)^[a-z0-9.-]+$`)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func sanitizeString(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func (h *Handler) ListBrands(w http.ResponseWriter, r *http.Request) {
	brands, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data brands")
		return
	}
	writeJSON(w, http.StatusOK, brands)
}

func (h *Handler) GetBrand(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return
	}

	b, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "brand tidak ditemukan")
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal mengambil data brand")
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (h *Handler) CreateBrand(w http.ResponseWriter, r *http.Request) {
	var req CreateBrandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name wajib diisi")
		return
	}

	if req.KodeBrand != nil {
		kb := strings.ToUpper(strings.TrimSpace(*req.KodeBrand))
		if kb != "" {
			if len(kb) != 2 {
				writeError(w, http.StatusBadRequest, "kode_brand harus persis 2 karakter")
				return
			}
			existing, err := h.repo.GetByKodeBrand(r.Context(), kb)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "gagal mengecek kode brand")
				return
			}
			if existing != nil {
				writeError(w, http.StatusConflict, "kode brand sudah digunakan oleh brand lain")
				return
			}
			req.KodeBrand = &kb
		} else {
			req.KodeBrand = nil
		}
	}

	if req.PrimaryColor != nil && *req.PrimaryColor != "" {
		if !hexColorRegex.MatchString(*req.PrimaryColor) {
			writeError(w, http.StatusBadRequest, "primary_color harus format hex, contoh #1B3A6B")
			return
		}
	}

	if req.Domain != nil {
		domainVal := strings.TrimSpace(strings.ToLower(*req.Domain))
		if domainVal != "" {
			if !domainRegex.MatchString(domainVal) {
				writeError(w, http.StatusBadRequest, "format domain tidak valid")
				return
			}
			existing, err := h.repo.GetByDomain(r.Context(), domainVal)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "gagal mengecek domain")
				return
			}
			if existing != nil {
				writeError(w, http.StatusConflict, "domain sudah dipakai brand lain")
				return
			}
			req.Domain = &domainVal
		} else {
			req.Domain = nil
		}
	}

	req.WhatsappNumber = sanitizeString(req.WhatsappNumber)
	req.Address = sanitizeString(req.Address)
	req.GmapsURL = sanitizeString(req.GmapsURL)
	req.Legalitas = sanitizeString(req.Legalitas)
	req.BankName = sanitizeString(req.BankName)
	req.BankAccountNumber = sanitizeString(req.BankAccountNumber)
	req.BankAccountHolder = sanitizeString(req.BankAccountHolder)
	req.SocialFacebook = sanitizeString(req.SocialFacebook)
	req.SocialInstagram = sanitizeString(req.SocialInstagram)
	req.SocialTiktok = sanitizeString(req.SocialTiktok)
	req.SocialYoutube = sanitizeString(req.SocialYoutube)
	req.LogoURL = sanitizeString(req.LogoURL)
	req.IconURL = sanitizeString(req.IconURL)
	req.PrimaryColor = sanitizeString(req.PrimaryColor)

	b, err := h.repo.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal membuat brand")
		return
	}

	writeJSON(w, http.StatusCreated, b)
}

func (h *Handler) UpdateBrand(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return
	}

	var req UpdateBrandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "name wajib diisi")
		return
	}

	if req.KodeBrand != nil {
		kb := strings.ToUpper(strings.TrimSpace(*req.KodeBrand))
		if kb != "" {
			if len(kb) != 2 {
				writeError(w, http.StatusBadRequest, "kode_brand harus persis 2 karakter")
				return
			}
			existing, err := h.repo.GetByKodeBrand(r.Context(), kb)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "gagal mengecek kode brand")
				return
			}
			if existing != nil && existing.ID != id {
				writeError(w, http.StatusConflict, "kode brand sudah digunakan oleh brand lain")
				return
			}
			req.KodeBrand = &kb
		} else {
			req.KodeBrand = nil
		}
	}

	if req.PrimaryColor != nil && *req.PrimaryColor != "" {
		if !hexColorRegex.MatchString(*req.PrimaryColor) {
			writeError(w, http.StatusBadRequest, "primary_color harus format hex, contoh #1B3A6B")
			return
		}
	}

	if req.Domain != nil {
		domainVal := strings.TrimSpace(strings.ToLower(*req.Domain))
		if domainVal != "" {
			if !domainRegex.MatchString(domainVal) {
				writeError(w, http.StatusBadRequest, "format domain tidak valid")
				return
			}
			existing, err := h.repo.GetByDomain(r.Context(), domainVal)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "gagal mengecek domain")
				return
			}
			if existing != nil && existing.ID != id {
				writeError(w, http.StatusConflict, "domain sudah dipakai brand lain")
				return
			}
			req.Domain = &domainVal
		} else {
			req.Domain = nil
		}
	}

	req.WhatsappNumber = sanitizeString(req.WhatsappNumber)
	req.Address = sanitizeString(req.Address)
	req.GmapsURL = sanitizeString(req.GmapsURL)
	req.Legalitas = sanitizeString(req.Legalitas)
	req.BankName = sanitizeString(req.BankName)
	req.BankAccountNumber = sanitizeString(req.BankAccountNumber)
	req.BankAccountHolder = sanitizeString(req.BankAccountHolder)
	req.SocialFacebook = sanitizeString(req.SocialFacebook)
	req.SocialInstagram = sanitizeString(req.SocialInstagram)
	req.SocialTiktok = sanitizeString(req.SocialTiktok)
	req.SocialYoutube = sanitizeString(req.SocialYoutube)
	req.LogoURL = sanitizeString(req.LogoURL)
	req.IconURL = sanitizeString(req.IconURL)
	req.PrimaryColor = sanitizeString(req.PrimaryColor)

	b, err := h.repo.Update(r.Context(), id, req)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "brand tidak ditemukan")
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal mengupdate brand")
		return
	}

	writeJSON(w, http.StatusOK, b)
}

func (h *Handler) DeleteBrand(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return
	}

	err = h.repo.Delete(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "brand tidak ditemukan")
			return
		}
		if errors.Is(err, ErrInUse) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "gagal menghapus brand")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "brand berhasil dihapus"})
}

func (h *Handler) ResolveDomain(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if strings.TrimSpace(domain) == "" {
		writeError(w, http.StatusBadRequest, "parameter domain wajib diisi")
		return
	}

	b, err := h.repo.GetByDomain(r.Context(), domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data brand")
		return
	}
	if b == nil {
		writeError(w, http.StatusNotFound, "brand tidak ditemukan untuk domain ini")
		return
	}

	resp := map[string]any{
		"id":              b.ID,
		"name":            b.Name,
		"whatsapp_number": b.WhatsappNumber,
		"logo_url":        b.LogoURL,
		"icon_url":        b.IconURL,
		"primary_color":   b.PrimaryColor,
	}

	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// ─── My Brand (Travel Admin) ───────────────────────────────────────────────────

func (h *Handler) GetMyBrand(w http.ResponseWriter, r *http.Request) {
	brandID := identity.GetBrandID(r.Context())
	var b *Brand
	var err error

	if brandID == nil {
		// Fallback untuk super admin atau testing: ambil brand pertama yang ada
		brands, listErr := h.repo.List(r.Context())
		if listErr != nil || len(brands) == 0 {
			writeError(w, http.StatusNotFound, "belum ada data brand")
			return
		}
		b = &brands[0]
	} else {
		b, err = h.repo.GetByID(r.Context(), uint64(*brandID))
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				writeError(w, http.StatusNotFound, "brand tidak ditemukan")
				return
			}
			writeError(w, http.StatusInternalServerError, "gagal mengambil data brand")
			return
		}
	}

	// Hanya kembalikan field publik/esensial
	resp := map[string]any{
		"id":            b.ID,
		"name":          b.Name,
		"logo_url":      b.LogoURL,
		"primary_color": b.PrimaryColor,
	}

	writeJSON(w, http.StatusOK, resp)
}

