package bankaccount

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	var brandID *int64
	if b := r.URL.Query().Get("brand_id"); b != "" {
		if id, err := strconv.ParseInt(b, 10, 64); err == nil && id > 0 {
			brandID = &id
		}
	}
	// Admin yang terikat satu brand hanya boleh melihat rekening brand-nya sendiri,
	// apa pun nilai brand_id yang dikirim.
	if scoped := identity.GetBrandID(r.Context()); scoped != nil {
		own := *scoped
		brandID = &own
	}
	activeOnly := r.URL.Query().Get("active") == "true"
	items, err := h.repo.List(r.Context(), brandID, activeOnly)
	if err != nil {
		writeError(w, 500, "gagal mengambil rekening")
		return
	}
	writeJSON(w, 200, items)
}

// PublicBankAccount hanya memuat field yang ditampilkan ke pelanggan (situs brand).
type PublicBankAccount struct {
	ID            int64   `json:"id"`
	BankName      string  `json:"bank_name"`
	LogoURL       *string `json:"logo_url"`
	AccountNumber string  `json:"account_number"`
	AccountHolder string  `json:"account_holder"`
}

// ListPublic GET /api/public/bank-accounts?brand_id=ID (atau ?brand=ID). Brand wajib; selalu
// hanya rekening aktif, tanpa field internal (brand, urutan, waktu dibuat). Handler admin (List)
// tidak berubah.
func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	raw := strings.TrimSpace(r.URL.Query().Get("brand_id"))
	if raw == "" {
		raw = strings.TrimSpace(r.URL.Query().Get("brand"))
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "parameter brand_id wajib diisi")
		return
	}
	items, err := h.repo.List(r.Context(), &id, true)
	if err != nil {
		writeError(w, 500, "gagal mengambil rekening")
		return
	}
	out := make([]PublicBankAccount, 0, len(items))
	for _, a := range items {
		if !a.IsActive || a.BrandID != id {
			continue
		}
		out = append(out, PublicBankAccount{ID: a.ID, BankName: a.BankName, LogoURL: a.LogoURL, AccountNumber: a.AccountNumber, AccountHolder: a.AccountHolder})
	}
	writeJSON(w, 200, out)
}

func validate(req *UpsertRequest) bool {
	req.BankName = strings.TrimSpace(req.BankName)
	req.AccountNumber = strings.TrimSpace(req.AccountNumber)
	req.AccountHolder = strings.TrimSpace(req.AccountHolder)
	return req.BrandID > 0 && req.BankName != "" && req.AccountNumber != "" && req.AccountHolder != ""
}
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req UpsertRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil || !validate(&req) {
		writeError(w, 400, "brand, bank, nomor rekening, dan atas nama wajib diisi")
		return
	}
	if scoped := identity.GetBrandID(r.Context()); scoped != nil && req.BrandID != *scoped {
		writeError(w, 403, "Anda hanya dapat mengelola rekening brand Anda sendiri")
		return
	}
	item, err := h.repo.Create(r.Context(), req)
	if err != nil {
		writeError(w, 400, "rekening gagal disimpan atau nomor sudah digunakan")
		return
	}
	writeJSON(w, 201, item)
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req UpsertRequest
	if id <= 0 || json.NewDecoder(r.Body).Decode(&req) != nil || !validate(&req) {
		writeError(w, 400, "data rekening tidak valid")
		return
	}
	if scoped := identity.GetBrandID(r.Context()); scoped != nil {
		existing, getErr := h.repo.Get(r.Context(), id)
		if getErr != nil || existing.BrandID != *scoped || req.BrandID != *scoped {
			writeError(w, 404, "rekening tidak ditemukan")
			return
		}
	}
	item, err := h.repo.Update(r.Context(), id, req)
	if errors.Is(err, ErrNotFound) {
		writeError(w, 404, "rekening tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, 400, "rekening gagal diperbarui")
		return
	}
	writeJSON(w, 200, item)
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if scoped := identity.GetBrandID(r.Context()); scoped != nil {
		existing, getErr := h.repo.Get(r.Context(), id)
		if getErr != nil || existing.BrandID != *scoped {
			writeError(w, 404, "rekening tidak ditemukan")
			return
		}
	}
	if err := h.repo.Delete(r.Context(), id); errors.Is(err, ErrNotFound) {
		writeError(w, 404, "rekening tidak ditemukan")
		return
	} else if err != nil {
		writeError(w, 409, "rekening masih dipakai transaksi")
		return
	}
	writeJSON(w, 200, map[string]string{"message": "rekening dihapus"})
}
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
