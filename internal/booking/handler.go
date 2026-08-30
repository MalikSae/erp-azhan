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
	"draft": true,
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

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	items, err := h.repo.List(r.Context(), brandID, jamaahID, status)
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

	// PIC Jamaah ID fallback
	picID := req.PicJamaahID
	if picID == 0 {
		picID = req.JamaahID
	}
	if picID == 0 && len(req.Pax) > 0 {
		picID = req.Pax[0].JamaahID
	}
	if picID == 0 {
		writeError(w, http.StatusBadRequest, "pic_jamaah_id wajib diisi")
		return
	}
	req.PicJamaahID = picID

	// Gate 3: PIC jamaah exist & brand cocok
	exists, err = h.repo.JamaahExistsForBrand(r.Context(), picID, brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return
	}
	if !exists {
		writeError(w, http.StatusBadRequest, "pic_jamaah_id tidak valid atau bukan milik brand Anda")
		return
	}

	// Gate 4: minimal 1 pax dalam array
	if len(req.Pax) == 0 {
		writeError(w, http.StatusBadRequest, "pax minimal 1 orang")
		return
	}

	// Gate 5: Validasi setiap item pax
	for i, p := range req.Pax {
		if p.JamaahID == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("jamaah_id pada pax urutan %d wajib diisi", i+1))
			return
		}
		exists, err := h.repo.JamaahExistsForBrand(r.Context(), p.JamaahID, brandID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return
		}
		if !exists {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("jamaah_id %d pada pax urutan %d tidak valid atau bukan milik brand Anda", p.JamaahID, i+1))
			return
		}

		if p.PaxType != "reguler" && p.PaxType != "infant" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("pax_type pada pax urutan %d tidak valid, harus 'reguler' atau 'infant'", i+1))
			return
		}

		if p.PaxType == "infant" {
			if p.RoomType != nil && strings.TrimSpace(*p.RoomType) != "" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("pax_type infant pada urutan %d tidak boleh memilih room_type (harus null)", i+1))
				return
			}
		} else if p.PaxType == "reguler" {
			if p.RoomType == nil || !validRoomTypes[*p.RoomType] {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("room_type untuk pax reguler pada urutan %d harus salah satu dari Quad, Triple, atau Double", i+1))
				return
			}
		}
	}

	createdBy := identity.GetAdminUserID(r.Context())

	b, err := h.repo.CreateBooking(r.Context(), &req, createdBy)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// ─── Draft Booking ────────────────────────────────────────────────────────────

// CreateDraftBooking godoc
// POST /api/admin/bookings/draft
func (h *Handler) CreateDraftBooking(w http.ResponseWriter, r *http.Request) {
	var req CreateDraftBookingRequest
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

	// PIC Jamaah ID validation (jika diisi)
	if req.PicJamaahID != nil && *req.PicJamaahID > 0 {
		exists, err := h.repo.JamaahExistsForBrand(r.Context(), *req.PicJamaahID, brandID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return
		}
		if !exists {
			writeError(w, http.StatusBadRequest, "pic_jamaah_id tidak valid atau bukan milik brand Anda")
			return
		}
	}

	// Validasi pax items yang jamaah_id-nya terisi
	for i, p := range req.Pax {
		if p.JamaahID == 0 {
			continue
		}
		exists, err := h.repo.JamaahExistsForBrand(r.Context(), p.JamaahID, brandID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return
		}
		if !exists {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("jamaah_id %d pada pax urutan %d tidak valid atau bukan milik brand Anda", p.JamaahID, i+1))
			return
		}

		if p.PaxType != "" && p.PaxType != "reguler" && p.PaxType != "infant" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("pax_type pada pax urutan %d tidak valid, harus 'reguler' atau 'infant'", i+1))
			return
		}

		if p.PaxType == "infant" {
			if p.RoomType != nil && strings.TrimSpace(*p.RoomType) != "" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("pax_type infant pada urutan %d tidak boleh memilih room_type (harus null)", i+1))
				return
			}
		} else if p.RoomType != nil && strings.TrimSpace(*p.RoomType) != "" {
			if !validRoomTypes[*p.RoomType] {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("room_type untuk pax reguler pada urutan %d harus salah satu dari Quad, Triple, atau Double", i+1))
				return
			}
		}
	}

	createdBy := identity.GetAdminUserID(r.Context())

	b, err := h.repo.CreateDraftBooking(r.Context(), &req, createdBy)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, b)
}

// UpdateDraftBooking godoc
// PUT /api/admin/bookings/{id}/draft
func (h *Handler) UpdateDraftBooking(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req CreateDraftBookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verifikasi booking exist, milik brand, dan status='draft'
	existing, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	if existing.Status != "draft" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("hanya booking berstatus draft yang dapat diubah melalui endpoint ini (status saat ini: %s)", existing.Status))
		return
	}

	if req.ScheduleID == 0 {
		req.ScheduleID = existing.ScheduleID
	} else {
		exists, err := h.repo.ScheduleExistsForBrand(r.Context(), req.ScheduleID, brandID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return
		}
		if !exists {
			writeError(w, http.StatusBadRequest, "schedule_id tidak valid atau bukan milik brand Anda")
			return
		}
	}

	if req.PicJamaahID != nil && *req.PicJamaahID > 0 {
		exists, err := h.repo.JamaahExistsForBrand(r.Context(), *req.PicJamaahID, brandID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return
		}
		if !exists {
			writeError(w, http.StatusBadRequest, "pic_jamaah_id tidak valid atau bukan milik brand Anda")
			return
		}
	}

	for i, p := range req.Pax {
		if p.JamaahID == 0 {
			continue
		}
		exists, err := h.repo.JamaahExistsForBrand(r.Context(), p.JamaahID, brandID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return
		}
		if !exists {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("jamaah_id %d pada pax urutan %d tidak valid atau bukan milik brand Anda", p.JamaahID, i+1))
			return
		}

		if p.PaxType != "" && p.PaxType != "reguler" && p.PaxType != "infant" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("pax_type pada pax urutan %d tidak valid, harus 'reguler' atau 'infant'", i+1))
			return
		}

		if p.PaxType == "infant" {
			if p.RoomType != nil && strings.TrimSpace(*p.RoomType) != "" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("pax_type infant pada urutan %d tidak boleh memilih room_type (harus null)", i+1))
				return
			}
		} else if p.RoomType != nil && strings.TrimSpace(*p.RoomType) != "" {
			if !validRoomTypes[*p.RoomType] {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("room_type untuk pax reguler pada urutan %d harus salah satu dari Quad, Triple, atau Double", i+1))
				return
			}
		}
	}

	b, err := h.repo.UpdateDraftBooking(r.Context(), id, &req)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// FinalizeBooking godoc
// POST /api/admin/bookings/{id}/finalize
func (h *Handler) FinalizeBooking(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	existing, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	if existing.Status != "draft" {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("booking bukan berstatus draft (status saat ini: %s)", existing.Status))
		return
	}

	// Validasi kelengkapan data sebelum finalisasi
	if existing.PicJamaahID == nil || *existing.PicJamaahID == 0 {
		writeError(w, http.StatusBadRequest, "Kontak Utama (PIC) wajib dipilih sebelum finalisasi booking")
		return
	}

	if len(existing.Pax) == 0 {
		writeError(w, http.StatusBadRequest, "Daftar pax kosong. Minimal 1 jamaah reguler wajib didaftarkan")
		return
	}

	var regularPaxCount int
	var picIsRegular bool
	for i, p := range existing.Pax {
		if p.JamaahID == 0 {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("Pax urutan %d belum memilih jamaah", i+1))
			return
		}
		if p.PaxType == "reguler" {
			regularPaxCount++
			if existing.PicJamaahID != nil && p.JamaahID == *existing.PicJamaahID {
				picIsRegular = true
			}
			if p.RoomType == nil || !validRoomTypes[*p.RoomType] {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Pax %s (urutan %d) belum memilih tipe kamar (Quad, Triple, atau Double)", p.NamaJamaah, i+1))
				return
			}
		} else if p.PaxType == "infant" {
			if p.RoomType != nil && *p.RoomType != "" {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("Pax infant %s tidak boleh memiliki tipe kamar", p.NamaJamaah))
				return
			}
		}
	}

	if regularPaxCount == 0 {
		writeError(w, http.StatusBadRequest, "Booking harus memiliki minimal 1 pax reguler (tidak boleh hanya infant)")
		return
	}

	if !picIsRegular {
		writeError(w, http.StatusBadRequest, "Kontak Utama (PIC) harus merupakan jamaah reguler (bukan infant)")
		return
	}

	b, err := h.repo.FinalizeBooking(r.Context(), id)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// ─── Cancel Pax ───────────────────────────────────────────────────────────────

// CancelPax godoc
// PUT /api/admin/bookings/{id}/pax/{pax_id}/cancel
func (h *Handler) CancelPax(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	paxID, ok := parsePaxID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	b, err := h.repo.CancelPax(r.Context(), id, paxID, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

// ─── Update Pax Room Type ─────────────────────────────────────────────────────

// UpdatePaxRoomType godoc
// PUT /api/admin/bookings/{id}/pax/{pax_id}/room-type
func (h *Handler) UpdatePaxRoomType(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	paxID, ok := parsePaxID(w, r)
	if !ok {
		return
	}

	var req UpdatePaxRoomTypeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if !validRoomTypes[req.RoomType] {
		writeError(w, http.StatusBadRequest, "room_type tidak valid, harus salah satu dari Quad, Triple, atau Double")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	b, err := h.repo.UpdatePaxRoomType(r.Context(), id, paxID, req.RoomType, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, b)
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
		writeError(w, http.StatusBadRequest, "status tidak valid, harus salah satu dari: baru, dp, lunas, batal")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verifikasi booking exists dan milik brand (jika brand admin)
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

// ─── Cancel Seat Block ────────────────────────────────────────────────────────

// CancelSeatBlock godoc
// DELETE /api/admin/bookings/{id}/seat-block
func (h *Handler) CancelSeatBlock(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verifikasi booking exists dan milik brand
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

	var req AddBookingAddonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	req.Nama = strings.TrimSpace(req.Nama)
	if req.Nama == "" {
		writeError(w, http.StatusBadRequest, "nama addon wajib diisi")
		return
	}

	if req.Nominal <= 0 {
		writeError(w, http.StatusBadRequest, "nominal harus lebih besar dari 0")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verifikasi booking exists dan milik brand
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}

	if err := h.repo.AddAddon(r.Context(), id, req.Nama, req.Nominal); err != nil {
		handleRepoError(w, err)
		return
	}

	b, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, b)
}

// DeleteBookingAddon godoc
// DELETE /api/admin/bookings/{id}/addons/{addon_id}
func (h *Handler) DeleteBookingAddon(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	rawAddonID := chi.URLParam(r, "addon_id")
	addonID, err := strconv.ParseInt(rawAddonID, 10, 64)
	if err != nil || addonID <= 0 {
		writeError(w, http.StatusBadRequest, "addon_id tidak valid")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verifikasi booking exists dan milik brand
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}

	if err := h.repo.DeleteAddon(r.Context(), id, addonID); err != nil {
		handleRepoError(w, err)
		return
	}

	b, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, b)
}

// ─── Diskon ───────────────────────────────────────────────────────────────────

// UpdateBookingDiskon godoc
// PUT /api/admin/bookings/{id}/diskon
func (h *Handler) UpdateBookingDiskon(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateDiskonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if req.Diskon < 0 {
		writeError(w, http.StatusBadRequest, "diskon tidak boleh negatif")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	// Verifikasi booking exists dan milik brand
	if _, err := h.repo.GetByID(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}

	if err := h.repo.UpdateDiskon(r.Context(), id, req.Diskon, req.DiskonKeterangan); err != nil {
		handleRepoError(w, err)
		return
	}

	b, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, b)
}

// ─── Progress Checklists ──────────────────────────────────────────────────────

// UpdateBookingProgress godoc
// PUT /api/admin/bookings/{id}/progress
func (h *Handler) UpdateBookingProgress(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())

	var body map[string]bool
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid, format: {\"key\": bool}")
		return
	}

	for key := range body {
		if key == "paspor" {
			writeError(w, http.StatusBadRequest, "status paspor tidak bisa diubah manual, status otomatis tercentang jika dokumen paspor sudah diunggah")
			return
		}
		if key == "tiket" {
			writeError(w, http.StatusBadRequest, "status tiket tidak bisa diubah manual, status otomatis tercentang jika status tiket jadwal sudah confirmed")
			return
		}
		if _, ok := AllowedPaxProgressFields[key]; ok {
			writeError(w, http.StatusBadRequest, "Gunakan endpoint progress per-pax untuk item ini")
			return
		}
		if _, ok := AllowedHeaderProgressFields[key]; !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("field progress '%s' tidak valid", key))
			return
		}
	}

	updated, err := h.repo.UpdateProgress(r.Context(), id, brandID, body)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, updated)
}

// UpdatePaxProgress godoc
// PUT /api/admin/bookings/{id}/pax/{pax_id}/progress
func (h *Handler) UpdatePaxProgress(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	paxIDStr := chi.URLParam(r, "pax_id")
	paxID, err := strconv.ParseInt(paxIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "pax_id tidak valid")
		return
	}

	brandID := identity.GetBrandID(r.Context())

	var body map[string]bool
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid, format: {\"key\": bool}")
		return
	}

	for key := range body {
		if key == "paspor" {
			writeError(w, http.StatusBadRequest, "status paspor tidak bisa diubah manual, status otomatis tercentang jika dokumen paspor sudah diunggah")
			return
		}
		if _, ok := AllowedPaxProgressFields[key]; !ok {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("field progress pax '%s' tidak valid", key))
			return
		}
	}

	updated, err := h.repo.UpdatePaxProgress(r.Context(), id, paxID, brandID, body)
	if err != nil {
		if errors.Is(err, ErrPaxBatal) || errors.Is(err, ErrManasikInfant) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
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
	var errSeatNotEnough *ErrSeatNotEnough
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	case errors.Is(err, ErrSeatHabis):
		writeError(w, http.StatusConflict, "kursi sudah habis, tidak bisa konfirmasi DP")
	case errors.As(err, &errSeatNotEnough):
		writeError(w, http.StatusBadRequest, errSeatNotEnough.Message)
	case errors.Is(err, ErrPaxAlreadyCancelled):
		writeError(w, http.StatusBadRequest, "pax sudah dalam status batal")
	case errors.Is(err, ErrCannotChangeInfantRoom):
		writeError(w, http.StatusBadRequest, "tidak bisa mengubah tipe kamar untuk infant")
	case errors.Is(err, ErrCannotChangeCancelledPaxRoom):
		writeError(w, http.StatusBadRequest, "tidak bisa mengubah tipe kamar untuk pax yang sudah batal")
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
		writeError(w, http.StatusBadRequest, err.Error())
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

func parsePaxID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "pax_id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "pax_id tidak valid")
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
