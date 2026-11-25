package airport

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/shared"
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
		log.Printf("[ERROR] airport: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data bandara.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, airports)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateAirport godoc
// POST /api/admin/airports
// Response 201: Airport baru
func (h *Handler) CreateAirport(w http.ResponseWriter, r *http.Request) {
	var req CreateAirportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
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
	shared.WriteJSON(w, http.StatusCreated, airport)
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
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
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
	shared.WriteJSON(w, http.StatusOK, airport)
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
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation ───────────────────────────────────────────────────────────────

// Panjang kolom airports (migrasi 037).
const (
	maksNama = 150
	maksKode = 4
	maksKota = 100
)

// validateAirportInput: berhenti di gate pertama yang gagal. Jika ok=false, response sudah ditulis.
func validateAirportInput(w http.ResponseWriter, rawName string, rawCode string, rawCity string) (string, string, string, bool) {
	name := strings.TrimSpace(rawName)
	if name == "" {
		shared.WriteError(w, http.StatusBadRequest, "Nama bandara wajib diisi.")
		return "", "", "", false
	}
	if utf8.RuneCountInString(name) > maksNama {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Nama bandara maksimal %d karakter.", maksNama))
		return "", "", "", false
	}

	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if code == "" {
		shared.WriteError(w, http.StatusBadRequest, "Kode bandara wajib diisi.")
		return "", "", "", false
	}
	if utf8.RuneCountInString(code) > maksKode {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Kode bandara maksimal %d karakter.", maksKode))
		return "", "", "", false
	}

	city := strings.TrimSpace(rawCity)
	if city == "" {
		shared.WriteError(w, http.StatusBadRequest, "Kota bandara wajib diisi.")
		return "", "", "", false
	}
	if utf8.RuneCountInString(city) > maksKota {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Kota bandara maksimal %d karakter.", maksKota))
		return "", "", "", false
	}

	return name, code, city, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleAirportRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		shared.WriteError(w, http.StatusNotFound, "Data tidak ditemukan.")
	case errors.Is(err, ErrDuplicateCode):
		shared.WriteError(w, http.StatusConflict, "Kode bandara sudah digunakan.")
	case errors.Is(err, ErrInUse), errors.Is(err, ErrCodeInUse):
		shared.WriteError(w, http.StatusConflict, err.Error())
	default:
		if shared.IsMySQLErr(err, 1062) {
			shared.WriteError(w, http.StatusConflict, "Kode bandara sudah digunakan.")
			return
		}
		if shared.IsFKDipakai(err) {
			shared.WriteError(w, http.StatusConflict, "Tidak bisa dihapus, masih dipakai oleh data lain.")
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
