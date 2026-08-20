package itinerary

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
)

// Handler menyimpan dependency untuk semua HTTP handler itinerary.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru dengan repository yang diberikan.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListItineraries godoc
// GET /api/admin/itineraries
// Response 200: array ItineraryListItem ([] jika kosong, tidak pernah null)
func (h *Handler) ListItineraries(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data itinerary")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Get Detail ───────────────────────────────────────────────────────────────

// GetItinerary godoc
// GET /api/admin/itineraries/{id}
// Response 200: Itinerary lengkap dengan days dan activities ter-unmarshal
func (h *Handler) GetItinerary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	it, err := h.repo.GetByID(r.Context(), id)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

// GetPublicItinerary godoc
// GET /api/itineraries/{id}
// Hanya mengembalikan itinerary yang terkait jadwal berstatus published.
func (h *Handler) GetPublicItinerary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	it, err := h.repo.GetPublishedByID(r.Context(), id)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateItinerary godoc
// POST /api/admin/itineraries
// Response 201: Itinerary baru dengan days
func (h *Handler) CreateItinerary(w http.ResponseWriter, r *http.Request) {
	var req CreateItineraryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if ok := validateItineraryInput(w, req.Title, req.Days); !ok {
		return
	}

	it, err := h.repo.Create(r.Context(), strings.TrimSpace(req.Title), req.Days)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, it)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// UpdateItinerary godoc
// PUT /api/admin/itineraries/{id}
// Response 200: Itinerary yang sudah diupdate
func (h *Handler) UpdateItinerary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateItineraryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if ok := validateItineraryInput(w, req.Title, req.Days); !ok {
		return
	}

	it, err := h.repo.Update(r.Context(), id, strings.TrimSpace(req.Title), req.Days)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, it)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// DeleteItinerary godoc
// DELETE /api/admin/itineraries/{id}
// Response 200: {"message": "berhasil dihapus"}
func (h *Handler) DeleteItinerary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation ───────────────────────────────────────────────────────────────

// validateItineraryInput menjalankan semua validasi sesuai urutan gate yang ditentukan.
// Mengembalikan false jika ada error — response sudah ditulis ke w.
func validateItineraryInput(w http.ResponseWriter, rawTitle string, days []DayRequest) bool {
	// Gate 1: title wajib
	if strings.TrimSpace(rawTitle) == "" {
		writeError(w, http.StatusBadRequest, "title wajib diisi")
		return false
	}

	// Gate 2: days boleh kosong/nil — tidak wajib ada
	// Gate 3-5: validasi per-hari (hanya jika ada)
	for i, d := range days {
		n := i + 1 // posisi 1-based untuk pesan error

		// Gate 3: title hari wajib
		if strings.TrimSpace(d.Title) == "" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("title hari ke-%d wajib diisi", n))
			return false
		}

		// Gate 4: activities minimal 1 elemen
		if len(d.Activities) == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("aktivitas hari ke-%d minimal 1", n))
			return false
		}

		// Gate 5: setiap activity harus punya text tidak kosong
		for _, act := range d.Activities {
			if strings.TrimSpace(act.Text) == "" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("aktivitas hari ke-%d memiliki teks kosong", n))
				return false
			}
		}
		// Gate 6: location BOLEH kosong — tidak divalidasi
	}

	return true
}

// ─── Error handling ───────────────────────────────────────────────────────────

// handleRepoError memetakan sentinel error ke respons HTTP yang tepat.
func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
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

// parseID mengekstrak URL param "id" dan mengonversi ke int64.
func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
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
