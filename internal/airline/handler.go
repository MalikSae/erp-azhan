package airline

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
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
		writeError(w, http.StatusInternalServerError, "gagal mengambil data maskapai")
		return
	}
	writeJSON(w, http.StatusOK, airlines)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateAirline godoc
// POST /api/admin/airlines
// Response 201: Airline baru
func (h *Handler) CreateAirline(w http.ResponseWriter, r *http.Request) {
	var req CreateAirlineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	name, ok := validateAirlineInput(w, req.Name)
	if !ok {
		return
	}

	var logoURL *string
	if req.LogoURL != nil && strings.TrimSpace(*req.LogoURL) != "" {
		val := strings.TrimSpace(*req.LogoURL)
		logoURL = &val
	}

	airline, err := h.repo.Create(r.Context(), name, logoURL)
	if err != nil {
		handleAirlineRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, airline)
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
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	name, ok := validateAirlineInput(w, req.Name)
	if !ok {
		return
	}

	var logoURL *string
	if req.LogoURL != nil && strings.TrimSpace(*req.LogoURL) != "" {
		val := strings.TrimSpace(*req.LogoURL)
		logoURL = &val
	}

	airline, err := h.repo.Update(r.Context(), id, name, logoURL)
	if err != nil {
		handleAirlineRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, airline)
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
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation ───────────────────────────────────────────────────────────────

// validateAirlineInput memvalidasi dan menormalisasi input nama maskapai.
// Mengembalikan (normalizedName, ok). Jika ok=false, response sudah ditulis.
func validateAirlineInput(w http.ResponseWriter, rawName string) (string, bool) {
	// Gate 1: name wajib
	if strings.TrimSpace(rawName) == "" {
		writeError(w, http.StatusBadRequest, "name wajib diisi")
		return "", false
	}

	// Gate 2: normalisasi
	name := strings.ToUpper(strings.TrimSpace(rawName))

	return name, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleAirlineRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrDuplicate):
		writeError(w, http.StatusConflict, "maskapai dengan nama ini sudah ada")
	default:
		// Tangkap MySQL FK constraint violation (error 1451)
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "tidak bisa dihapus, masih dipakai oleh paket lain")
			return
		}
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
