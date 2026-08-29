package airport

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
)

// Handler menyimpan dependency untuk semua HTTP handler airport.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru dengan repository yang diberikan.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListAirports godoc
// GET /api/admin/airports
// Query params: ?search=
// Response 200: array Airport ([] jika kosong)
func (h *Handler) ListAirports(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	airports, err := h.repo.List(r.Context(), search)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data bandara")
		return
	}
	writeJSON(w, http.StatusOK, airports)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateAirport godoc
// POST /api/admin/airports
// Response 201: Airport baru
func (h *Handler) CreateAirport(w http.ResponseWriter, r *http.Request) {
	var req CreateAirportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	name, code, city, ok := validateAirportInput(w, req.Name, req.Code, req.City)
	if !ok {
		return
	}

	airport, err := h.repo.Create(r.Context(), name, code, city)
	if err != nil {
		handleAirportRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, airport)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// UpdateAirport godoc
// PUT /api/admin/airports/{id}
// Response 200: Airport yang sudah diupdate
func (h *Handler) UpdateAirport(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateAirportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	name, code, city, ok := validateAirportInput(w, req.Name, req.Code, req.City)
	if !ok {
		return
	}

	airport, err := h.repo.Update(r.Context(), id, name, code, city)
	if err != nil {
		handleAirportRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, airport)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// DeleteAirport godoc
// DELETE /api/admin/airports/{id}
// Response 200: {"message": "berhasil dihapus"}
func (h *Handler) DeleteAirport(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		handleAirportRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation ───────────────────────────────────────────────────────────────

func validateAirportInput(w http.ResponseWriter, rawName string, rawCode string, rawCity string) (string, string, string, bool) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		writeError(w, http.StatusBadRequest, "nama bandara wajib diisi")
		return "", "", "", false
	}

	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		writeError(w, http.StatusBadRequest, "kode bandara wajib diisi")
		return "", "", "", false
	}
	if len(code) > 4 {
		writeError(w, http.StatusBadRequest, "kode bandara maksimal 4 karakter")
		return "", "", "", false
	}

	city := strings.TrimSpace(rawCity)
	if city == "" {
		writeError(w, http.StatusBadRequest, "kota bandara wajib diisi")
		return "", "", "", false
	}

	return name, code, city, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleAirportRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrDuplicateCode):
		writeError(w, http.StatusConflict, "Kode bandara sudah digunakan")
	default:
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) {
			if mysqlErr.Number == 1062 {
				writeError(w, http.StatusConflict, "Kode bandara sudah digunakan")
				return
			}
			if mysqlErr.Number == 1451 {
				writeError(w, http.StatusConflict, "tidak bisa dihapus, masih dipakai oleh data lain")
				return
			}
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
