package bankaccount

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

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
	activeOnly := r.URL.Query().Get("active") == "true"
	items, err := h.repo.List(r.Context(), brandID, activeOnly)
	if err != nil {
		writeError(w, 500, "gagal mengambil rekening")
		return
	}
	writeJSON(w, 200, items)
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
