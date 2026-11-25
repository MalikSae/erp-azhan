package airline

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"erp-azhan/api/internal/shared"
	"github.com/go-chi/chi/v5"
)

// Handler menyimpan dependency untuk semua HTTP handler airline.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru dengan repository yang diberikan.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListAirlines godoc
// GET /api/admin/airlines
// Response 200: array Airline ([] jika kosong)
func (h *Handler) ListAirlines(w http.ResponseWriter, r *http.Request) {
	airlines, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("[ERROR] airline: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data maskapai.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, airlines)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateAirline godoc
// POST /api/admin/airlines
// Response 201: Airline baru
func (h *Handler) CreateAirline(w http.ResponseWriter, r *http.Request) {
	var req CreateAirlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	name, code, logoURL, ok := validateAirlineInput(w, req.Name, req.Code, req.LogoURL)
	if !ok {
		return
	}

	airline, err := h.repo.Create(r.Context(), name, code, logoURL)
	if err != nil {
		handleAirlineRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, airline)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// UpdateAirline godoc
// PUT /api/admin/airlines/{id}
// Response 200: Airline yang sudah diupdate
func (h *Handler) UpdateAirline(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateAirlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	name, code, logoURL, ok := validateAirlineInput(w, req.Name, req.Code, req.LogoURL)
	if !ok {
		return
	}

	airline, err := h.repo.Update(r.Context(), id, name, code, logoURL)
	if err != nil {
		handleAirlineRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, airline)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// DeleteAirline godoc
// DELETE /api/admin/airlines/{id}
// Response 200: {"message": "berhasil dihapus"}
func (h *Handler) DeleteAirline(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		handleAirlineRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation ───────────────────────────────────────────────────────────────

// Panjang kolom airlines (migrasi 001, 004).
const (
	maksNama    = 150
	maksLogoURL = 500
)

// validateAirlineInput memvalidasi dan menormalisasi nama, kode IATA, dan logo maskapai, berhenti di gate
// pertama yang gagal. Kode dan logo kosong menjadi nil. Jika ok=false, response sudah ditulis.
func validateAirlineInput(w http.ResponseWriter, rawName string, rawCode, rawLogo *string) (string, *string, *string, bool) {
	// Gate 1: nama wajib
	name := strings.ToUpper(strings.TrimSpace(rawName))
	if name == "" {
		shared.WriteError(w, http.StatusBadRequest, "Nama wajib diisi.")
		return "", nil, nil, false
	}
	// Gate 2: panjang nama
	if utf8.RuneCountInString(name) > maksNama {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Nama maksimal %d karakter.", maksNama))
		return "", nil, nil, false
	}

	// Gate 3: kode IATA opsional, tepat 2 karakter
	var code *string
	if rawCode != nil && strings.TrimSpace(*rawCode) != "" {
		c := strings.ToUpper(strings.TrimSpace(*rawCode))
		if utf8.RuneCountInString(c) != 2 {
			shared.WriteError(w, http.StatusBadRequest, "Kode IATA harus 2 karakter.")
			return "", nil, nil, false
		}
		code = &c
	}

	// Gate 4: logo opsional
	var logoURL *string
	if rawLogo != nil && strings.TrimSpace(*rawLogo) != "" {
		val := strings.TrimSpace(*rawLogo)
		if utf8.RuneCountInString(val) > maksLogoURL {
			shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("URL logo maksimal %d karakter.", maksLogoURL))
			return "", nil, nil, false
		}
		if !shared.ValidMediaURL(val) {
			shared.WriteError(w, http.StatusBadRequest, "Logo harus berkas unggahan atau URL http(s).")
			return "", nil, nil, false
		}
		logoURL = &val
	}

	return name, code, logoURL, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleAirlineRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		shared.WriteError(w, http.StatusNotFound, "Data tidak ditemukan.")
	case errors.Is(err, ErrDuplicate):
		shared.WriteError(w, http.StatusConflict, "Maskapai dengan nama ini sudah ada.")
	case errors.Is(err, ErrCodeInUse):
		shared.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrCodeDuplicate):
		shared.WriteError(w, http.StatusConflict, "Kode maskapai sudah dipakai.")
	default:
		if shared.IsFKDipakai(err) {
			shared.WriteError(w, http.StatusConflict, "Tidak bisa dihapus, masih dipakai oleh paket lain.")
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		shared.WriteError(w, http.StatusBadRequest, "ID tidak valid.")
		return 0, false
	}
	return id, true
}
