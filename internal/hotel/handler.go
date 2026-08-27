package hotel

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	// MySQL driver error codes
	"github.com/go-sql-driver/mysql"
)

// Handler menyimpan dependency untuk semua HTTP handler hotel.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru dengan repository yang diberikan.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListHotels godoc
// GET /api/admin/hotels
// Response 200: array Hotel ([] jika kosong, tidak pernah null)
func (h *Handler) ListHotels(w http.ResponseWriter, r *http.Request) {
	hotels, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data hotel")
		return
	}
	writeJSON(w, http.StatusOK, hotels)
}

// ListCities godoc
// GET /api/admin/hotels/cities
// Response 200: array string ([] jika kosong, tidak pernah null)
func (h *Handler) ListCities(w http.ResponseWriter, r *http.Request) {
	cities, err := h.repo.ListCities(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar kota hotel")
		return
	}
	writeJSON(w, http.StatusOK, cities)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateHotel godoc
// POST /api/admin/hotels
// Response 201: Hotel baru
func (h *Handler) CreateHotel(w http.ResponseWriter, r *http.Request) {
	var req CreateHotelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	name, city, starRating, distanceM, ok := validateHotelInput(w, req.Name, req.City, req.StarRating, req.DistanceM)
	if !ok {
		return
	}

	hotel, err := h.repo.Create(r.Context(), name, city, starRating, distanceM, req.PhotoURL)
	if err != nil {
		handleHotelRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, hotel)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// UpdateHotel godoc
// PUT /api/admin/hotels/{id}
// Response 200: Hotel yang sudah diupdate
func (h *Handler) UpdateHotel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateHotelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	name, city, starRating, distanceM, ok := validateHotelInput(w, req.Name, req.City, req.StarRating, req.DistanceM)
	if !ok {
		return
	}

	hotel, err := h.repo.Update(r.Context(), id, name, city, starRating, distanceM, req.PhotoURL)
	if err != nil {
		handleHotelRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, hotel)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// DeleteHotel godoc
// DELETE /api/admin/hotels/{id}
// Response 200: {"message": "berhasil dihapus"}
func (h *Handler) DeleteHotel(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		handleHotelRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation ───────────────────────────────────────────────────────────────

// validateHotelInput menjalankan semua validasi hotel sesuai urutan yang ditentukan.
// Mengembalikan (normalizedName, normalizedCity, starRating, distanceM, ok).
// Jika ok=false, response error sudah ditulis ke w.
func validateHotelInput(w http.ResponseWriter, rawName, rawCity string, starRating *int, distanceM *int) (string, string, *int, *int, bool) {
	// Gate 1: name wajib
	if strings.TrimSpace(rawName) == "" {
		writeError(w, http.StatusBadRequest, "name wajib diisi")
		return "", "", nil, nil, false
	}

	// Gate 2: normalisasi name
	name := strings.ToUpper(strings.TrimSpace(rawName))

	// Gate 3: city wajib
	if strings.TrimSpace(rawCity) == "" {
		writeError(w, http.StatusBadRequest, "city wajib diisi")
		return "", "", nil, nil, false
	}
	city := strings.TrimSpace(rawCity)

	// Gate 4: star_rating range
	if starRating != nil && (*starRating < 1 || *starRating > 5) {
		writeError(w, http.StatusBadRequest, "star_rating harus antara 1-5")
		return "", "", nil, nil, false
	}

	// Gate 5: distance_m tidak boleh negatif jika dikirim
	if distanceM != nil && *distanceM < 0 {
		writeError(w, http.StatusBadRequest, "distance_m tidak boleh negatif")
		return "", "", nil, nil, false
	}

	return name, city, starRating, distanceM, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

// handleHotelRepoError memetakan sentinel error ke respons HTTP yang tepat.
func handleHotelRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrDuplicate):
		writeError(w, http.StatusConflict, "hotel dengan nama ini sudah ada")
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

// parseID mengekstrak URL param "id" dan mengonversi ke uint64.
func parseID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusBadRequest, "id tidak valid")
		return 0, false
	}
	return id, true
}

// writeJSON menulis response JSON dengan Content-Type yang benar.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError menulis response JSON error standar.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
