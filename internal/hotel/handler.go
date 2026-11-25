package hotel

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

	// MySQL driver error codes
	"erp-azhan/api/internal/shared"
)

// Handler menyimpan dependency untuk semua HTTP handler hotel.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru dengan repository yang diberikan.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// --- List -------------------------------------------------------------------

// ListHotels godoc
// GET /api/admin/hotels
// Response 200: array Hotel ([] jika kosong, tidak pernah null)
func (h *Handler) ListHotels(w http.ResponseWriter, r *http.Request) {
	hotels, err := h.repo.List(r.Context())
	if err != nil {
		log.Printf("[ERROR] hotel: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data hotel.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, hotels)
}

// ListCities godoc
// GET /api/admin/hotels/cities
// Response 200: array string ([] jika kosong, tidak pernah null)
func (h *Handler) ListCities(w http.ResponseWriter, r *http.Request) {
	cities, err := h.repo.ListCities(r.Context())
	if err != nil {
		log.Printf("[ERROR] hotel: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil daftar kota hotel.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, cities)
}

// --- Create -----------------------------------------------------------------

// CreateHotel godoc
// POST /api/admin/hotels
// Response 201: Hotel baru
func (h *Handler) CreateHotel(w http.ResponseWriter, r *http.Request) {
	var req CreateHotelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	name, city, starRating, distanceM, ok := validateHotelInput(w, req.Name, req.City, req.StarRating, req.DistanceM)
	if !ok {
		return
	}

	photo, video, ok := validateMedia(w, req.PhotoURL, req.VideoURL)
	if !ok {
		return
	}
	req.PhotoURL, req.VideoURL = photo, video

	hotel, err := h.repo.Create(r.Context(), name, city, starRating, distanceM, req.PhotoURL, req.VideoURL)
	if err != nil {
		handleHotelRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusCreated, hotel)
}

// --- Update -----------------------------------------------------------------

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
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	name, city, starRating, distanceM, ok := validateHotelInput(w, req.Name, req.City, req.StarRating, req.DistanceM)
	if !ok {
		return
	}

	photo, video, ok := validateMedia(w, req.PhotoURL, req.VideoURL)
	if !ok {
		return
	}
	req.PhotoURL, req.VideoURL = photo, video

	hotel, err := h.repo.Update(r.Context(), id, name, city, starRating, distanceM, req.PhotoURL, req.VideoURL)
	if err != nil {
		handleHotelRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, hotel)
}

// --- Delete -----------------------------------------------------------------

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
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// --- Validation -------------------------------------------------------------

// Panjang kolom hotels (migrasi 001, 005, 058).
const (
	maksNama     = 255
	maksKota     = 100
	maksMediaURL = 500
)

// validateMedia: foto dan video opsional; kosong menjadi nil. Jika ok=false, response sudah ditulis.
func validateMedia(w http.ResponseWriter, photo, video *string) (*string, *string, bool) {
	if photo != nil && strings.TrimSpace(*photo) == "" {
		photo = nil
	}
	if video != nil && strings.TrimSpace(*video) == "" {
		video = nil
	}
	if (photo != nil && utf8.RuneCountInString(*photo) > maksMediaURL) || (video != nil && utf8.RuneCountInString(*video) > maksMediaURL) {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("URL foto atau video maksimal %d karakter.", maksMediaURL))
		return nil, nil, false
	}
	if (photo != nil && !shared.ValidMediaURL(*photo)) || (video != nil && !shared.ValidMediaURL(*video)) {
		shared.WriteError(w, http.StatusBadRequest, "Foto atau video harus berkas unggahan atau URL http(s).")
		return nil, nil, false
	}
	return photo, video, true
}

// validateHotelInput menjalankan semua validasi hotel sesuai urutan yang ditentukan.
// Mengembalikan (normalizedName, normalizedCity, starRating, distanceM, ok).
// Jika ok=false, response error sudah ditulis ke w.
func validateHotelInput(w http.ResponseWriter, rawName, rawCity string, starRating *int, distanceM *int) (string, string, *int, *int, bool) {
	// Gate 1: nama wajib
	if strings.TrimSpace(rawName) == "" {
		shared.WriteError(w, http.StatusBadRequest, "Nama wajib diisi.")
		return "", "", nil, nil, false
	}

	// Gate 2: normalisasi dan panjang nama
	name := strings.ToUpper(strings.TrimSpace(rawName))
	if utf8.RuneCountInString(name) > maksNama {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Nama maksimal %d karakter.", maksNama))
		return "", "", nil, nil, false
	}

	// Gate 3: kota wajib
	if strings.TrimSpace(rawCity) == "" {
		shared.WriteError(w, http.StatusBadRequest, "Kota wajib diisi.")
		return "", "", nil, nil, false
	}
	city := strings.TrimSpace(rawCity)
	if utf8.RuneCountInString(city) > maksKota {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Kota maksimal %d karakter.", maksKota))
		return "", "", nil, nil, false
	}

	// Gate 4: star_rating range
	if starRating != nil && (*starRating < 1 || *starRating > 5) {
		shared.WriteError(w, http.StatusBadRequest, "Bintang hotel harus antara 1-5.")
		return "", "", nil, nil, false
	}

	// Gate 5: distance_m tidak boleh negatif jika dikirim
	if distanceM != nil && *distanceM < 0 {
		shared.WriteError(w, http.StatusBadRequest, "Jarak tidak boleh negatif.")
		return "", "", nil, nil, false
	}

	return name, city, starRating, distanceM, true
}

// --- Error handling ---------------------------------------------------------

// handleHotelRepoError memetakan sentinel error ke respons HTTP yang tepat.
func handleHotelRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		shared.WriteError(w, http.StatusNotFound, "Data tidak ditemukan.")
	case errors.Is(err, ErrDuplicate):
		shared.WriteError(w, http.StatusConflict, "Hotel dengan nama ini sudah ada di kota yang sama.")
	case errors.Is(err, ErrKotaDipakai):
		shared.WriteError(w, http.StatusConflict, ErrKotaDipakai.Error())
	default:
		if shared.IsFKDipakai(err) {
			shared.WriteError(w, http.StatusConflict, "Tidak bisa dihapus, masih dipakai oleh paket lain.")
			return
		}
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
	}
}

// --- Shared helpers ---------------------------------------------------------

// parseID mengekstrak URL param "id" dan mengonversi ke uint64.
func parseID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || id == 0 {
		shared.WriteError(w, http.StatusBadRequest, "ID tidak valid.")
		return 0, false
	}
	return id, true
}
