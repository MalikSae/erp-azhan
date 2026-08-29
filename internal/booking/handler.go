package booking

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler menyimpan dependency untuk semua HTTP handler booking.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// validRoomTypes berisi enum room_type yang valid.
var validRoomTypes = map[string]bool{
	"Quad":   true,
	"Triple": true,
	"Double": true,
}

// validStatuses berisi enum status booking yang valid.
var validStatuses = map[string]bool{
	"baru":  true,
	"dp":    true,
	"lunas": true,
	"batal": true,
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListBookings godoc
// GET /api/admin/bookings
func (h *Handler) ListBookings(w http.ResponseWriter, r *http.Request) {
	brandID := identity.GetBrandID(r.Context())
	var jamaahID *int64
	if raw := r.URL.Query().Get("jamaah_id"); raw != "" {
		id, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "jamaah_id tidak valid")
			return
		}
		jamaahID = &id
	}

	items, err := h.repo.List(r.Context(), brandID, jamaahID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data booking")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Get Detail ───────────────────────────────────────────────────────────────

// GetBooking godoc
// GET /api/admin/bookings/{id}
func (h *Handler) GetBooking(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	brandID := identity.GetBrandID(r.Context())
	b, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateBooking godoc
// POST /api/admin/bookings
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req CreateBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Gate 1: schedule_id wajib
	if req.ScheduleID == 0 {
		writeError(w, http.StatusBadRequest, "schedule_id wajib diisi")
		return
	}

	// Gate 2: schedule exist & brand cocok
	exists, err := h.repo.ScheduleExistsForBrand(r.Context(), req.ScheduleID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !exists {
		writeError(w, http.StatusBadRequest, "schedule_id tidak valid atau bukan milik brand Anda")
		return
	}

	// Gate 3: jamaah_id wajib
	if req.JamaahID == 0 {
		writeError(w, http.StatusBadRequest, "jamaah_id wajib diisi")
		return
	}

	// Gate 4: jamaah exist & brand cocok
	exists, err = h.repo.JamaahExistsForBrand(r.Context(), req.JamaahID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !exists {
		writeError(w, http.StatusBadRequest, "jamaah_id tidak valid atau bukan milik brand Anda")
		return
	}

	// Gate 5: room_type valid enum
	if !validRoomTypes[req.RoomType] {
		writeError(w, http.StatusBadRequest, "room_type tidak valid, harus Quad/Triple/Double")
		return
	}

	// Auto-snapshot harga kalau total_harga tidak dikirim
	var autoHarga *float64
	if req.TotalHarga == nil {
		h, err := h.repo.GetScheduleHarga(r.Context(), req.ScheduleID, req.RoomType)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "gagal mengambil harga schedule")
			return
		}
		autoHarga = h
	}

	createdBy := identity.GetAdminUserID(r.Context())

	b, err := h.repo.CreateBooking(r.Context(), &req, createdBy, autoHarga)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// ─── Update Status ────────────────────────────────────────────────────────────

// UpdateBookingStatus godoc
// PUT /api/admin/bookings/{id}/status
func (h *Handler) UpdateBookingStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateBookingStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	// Validasi enum status
	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "status tidak valid, harus baru/dp/lunas/batal")
		return
	}

	// Verify brand access: ambil booking dulu, cek via schedule brand
	brandID := identity.GetBrandID(r.Context())
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}

	b, err := h.repo.UpdateBookingStatus(r.Context(), id, req.Status)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// CancelSeatBlock melepaskan blok kursi tanpa membatalkan booking.
// DELETE /api/admin/bookings/{id}/seat-block
func (h *Handler) CancelSeatBlock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	brandID := identity.GetBrandID(r.Context())
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}
	b, err := h.repo.CancelSeatBlock(r.Context(), id)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// BlockSeat menahan kursi tanpa membuat payment.
// PUT /api/admin/bookings/{id}/seat-block
func (h *Handler) BlockSeat(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if _, err := uuid.Parse(idempotencyKey); err != nil {
		writeError(w, http.StatusBadRequest, "Idempotency-Key UUID wajib diisi")
		return
	}
	var req SeatBlockRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	expiresAt, err := time.Parse(time.RFC3339, req.ExpiresAt)
	if err != nil || !expiresAt.After(time.Now().Add(time.Minute)) {
		writeError(w, http.StatusBadRequest, "expires_at harus RFC3339 dan berada di masa depan")
		return
	}
	if _, err := h.repo.GetByID(r.Context(), id, identity.GetBrandID(r.Context())); err != nil {
		handleRepoError(w, err)
		return
	}
	booking, err := h.repo.BlockSeat(r.Context(), id, expiresAt, idempotencyKey)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, booking)
}

// ─── Addons & Diskon ──────────────────────────────────────────────────────────

// AddBookingAddon godoc
// POST /api/admin/bookings/{id}/addons
func (h *Handler) AddBookingAddon(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}

	var req AddBookingAddonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}

	if req.Nama == "" {
		writeError(w, http.StatusBadRequest, "nama add-on wajib diisi")
		return
	}
	if req.Nominal <= 0 {
		writeError(w, http.StatusBadRequest, "nominal add-on harus lebih dari 0")
		return
	}

	if err := h.repo.AddAddon(r.Context(), id, req.Nama, req.Nominal); err != nil {
		handleRepoError(w, err)
		return
	}

	updated, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, updated)
}

// DeleteBookingAddon godoc
// DELETE /api/admin/bookings/{id}/addons/{addon_id}
func (h *Handler) DeleteBookingAddon(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	addonRaw := chi.URLParam(r, "addon_id")
	addonID, err := strconv.ParseInt(addonRaw, 10, 64)
	if err != nil || addonID <= 0 {
		writeError(w, http.StatusBadRequest, "addon_id tidak valid")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}

	if err := h.repo.DeleteAddon(r.Context(), id, addonID); err != nil {
		handleRepoError(w, err)
		return
	}

	updated, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// UpdateBookingDiskon godoc
// PUT /api/admin/bookings/{id}/diskon
func (h *Handler) UpdateBookingDiskon(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}

	var req UpdateDiskonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}

	if req.Diskon < 0 {
		writeError(w, http.StatusBadRequest, "nilai diskon tidak boleh negatif")
		return
	}

	if err := h.repo.UpdateDiskon(r.Context(), id, req.Diskon, req.DiskonKeterangan); err != nil {
		handleRepoError(w, err)
		return
	}

	updated, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// UpdateBookingProgress godoc
// PUT /api/admin/bookings/{id}/progress
func (h *Handler) UpdateBookingProgress(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}

	var rawMap map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&rawMap); err != nil {
		writeError(w, http.StatusBadRequest, "format JSON tidak valid")
		return
	}

	if len(rawMap) == 0 {
		writeError(w, http.StatusBadRequest, "request body tidak boleh kosong")
		return
	}

	if _, hasPaspor := rawMap["paspor"]; hasPaspor {
		writeError(w, http.StatusBadRequest, "status paspor otomatis mengikuti dokumen jamaah, tidak bisa diubah manual di sini")
		return
	}

	if _, hasTiket := rawMap["tiket"]; hasTiket {
		writeError(w, http.StatusBadRequest, "status tiket otomatis mengikuti status konfirmasi di master paket, tidak bisa diubah manual di sini")
		return
	}

	updates := make(map[string]bool)
	for key, rawVal := range rawMap {
		if _, ok := AllowedProgressFields[key]; !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("item progress tidak valid: %s", key))
			return
		}
		var b bool
		if err := json.Unmarshal(rawVal, &b); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("nilai untuk '%s' harus berupa boolean", key))
			return
		}
		updates[key] = b
	}

	updated, err := h.repo.UpdateProgress(r.Context(), id, brandID, updates)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// ─── Perlengkapan Distribusi ──────────────────────────────────────────────────

// PUT /api/admin/bookings/{id}/perlengkapan/distribusi
func (h *Handler) DistribusiPerlengkapan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	adminID := identity.GetAdminUserID(r.Context())
	brandID := identity.GetBrandID(r.Context())

	b, err := h.repo.MarkPerlengkapanDiberikan(r.Context(), id, adminID, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, b)
}

// DELETE /api/admin/bookings/{id}/perlengkapan/distribusi
func (h *Handler) BatalkanPerlengkapan(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	b, err := h.repo.BatalkanPerlengkapan(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, b)
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	var errStok *ErrStokKurang
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrSeatHabis):
		writeError(w, http.StatusConflict, "kursi sudah habis, tidak bisa konfirmasi DP")
	case errors.Is(err, ErrInvalidStatus):
		writeError(w, http.StatusBadRequest, "status tidak valid")
	case errors.Is(err, ErrSeatBelumDiblokir):
		writeError(w, http.StatusConflict, "kursi booking ini sudah tidak diblokir")
	case errors.Is(err, ErrSeatSudahDiblokir):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrTemplatePerlengkapanBelumDiatur):
		writeError(w, http.StatusBadRequest, "template set perlengkapan belum diatur untuk brand ini")
	case errors.Is(err, ErrPerlengkapanSudahDiberikan):
		writeError(w, http.StatusBadRequest, "perlengkapan untuk booking ini sudah pernah diberikan")
	case errors.Is(err, ErrPerlengkapanBelumDiberikan):
		writeError(w, http.StatusBadRequest, "perlengkapan belum pernah diberikan untuk booking ini")
	case errors.Is(err, ErrKodeBrandNotSet):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrGenerateIDBookingFailed):
		writeError(w, http.StatusInternalServerError, err.Error())
	case errors.As(err, &errStok):
		writeError(w, http.StatusConflict, errStok.Message)
	default:
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("terjadi kesalahan internal: %v", err))
	}
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
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
