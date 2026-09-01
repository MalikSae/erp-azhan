package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
)

// Handler menyimpan dependency untuk semua HTTP handler schedule.
type Handler struct {
	repo *Repository
}

// NewHandler membuat Handler baru dengan repository yang diberikan.
func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// ─── Admin: List ──────────────────────────────────────────────────────────────

// ListSchedulesAdmin godoc
// GET /api/admin/schedules
// Response 200: array ScheduleListItem semua status, terbaru dulu
func (h *Handler) ListSchedulesAdmin(w http.ResponseWriter, r *http.Request) {
	brandID := identity.GetBrandID(r.Context())
	items, err := h.repo.List(r.Context(), brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jadwal")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ─── Admin: Get Detail ────────────────────────────────────────────────────────

// GetScheduleAdmin godoc
// GET /api/admin/schedules/{id}
// Response 200: Schedule lengkap dengan ref hotel/airline
func (h *Handler) GetScheduleAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	brandID := identity.GetBrandID(r.Context())
	s, err := h.repo.GetByID(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// ─── Public: List Published ───────────────────────────────────────────────────

// ListSchedulesPublic godoc
// GET /api/schedules
// Response 200: array PublicSchedule (hanya published, tanpa field status)
func (h *Handler) ListSchedulesPublic(w http.ResponseWriter, r *http.Request) {
	brandRaw := r.URL.Query().Get("brand")
	if strings.TrimSpace(brandRaw) == "" {
		writeError(w, http.StatusBadRequest, "parameter brand wajib diisi")
		return
	}

	brandID, err := strconv.ParseInt(brandRaw, 10, 64)
	if err != nil || brandID <= 0 {
		writeError(w, http.StatusBadRequest, "parameter brand tidak valid")
		return
	}

	items, err := h.repo.ListPublic(r.Context(), brandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jadwal")
		return
	}
	if items == nil {
		items = make([]*PublicSchedule, 0)
	}
	writeJSON(w, http.StatusOK, items)
}

// GetSchedulePublic godoc
// GET /api/schedules/{id}
// Response 200: PublicSchedule (hanya jika berstatus published)
func (h *Handler) GetSchedulePublic(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	s, err := h.repo.GetByID(r.Context(), id, nil)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	if s.Status != "published" {
		writeError(w, http.StatusNotFound, "jadwal tidak ditemukan atau belum dipublikasikan")
		return
	}

	writeJSON(w, http.StatusOK, s.ToPublic())
}

// ─── Admin: Create ────────────────────────────────────────────────────────────

// CreateSchedule godoc
// POST /api/admin/schedules
// Response 201: Schedule baru dengan status="draft"
func (h *Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	inp, ok := h.validateScheduleInput(r.Context(), w, &req)
	if !ok {
		return
	}

	s, err := h.repo.Create(r.Context(), *inp)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, s)
}

// ─── Admin: Update ────────────────────────────────────────────────────────────

// UpdateSchedule godoc
// PUT /api/admin/schedules/{id}
// Response 200: Schedule terupdate (status TIDAK berubah lewat endpoint ini)
func (h *Handler) UpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	ctxBrandID := identity.GetBrandID(r.Context())

	// 1. Ambil row schedule existing dulu untuk tahu brand_id lama-nya
	existing, err := h.repo.GetByID(r.Context(), id, ctxBrandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	// 2. Tentukan finalBrandID
	var finalBrandID int64
	if ctxBrandID != nil {
		// Scoped admin: tidak boleh ubah brand_id
		finalBrandID = existing.BrandID
	} else {
		// Super Admin
		if req.BrandID != nil && *req.BrandID != 0 {
			finalBrandID = *req.BrandID
		} else {
			finalBrandID = existing.BrandID
		}
	}

	// Override req.BrandID agar validateScheduleInput tidak error saat Super Admin tidak kirim brand_id
	req.BrandID = &finalBrandID

	inp, ok := h.validateScheduleInput(r.Context(), w, &req)
	if !ok {
		return
	}

	// 3. Panggil repository Update dengan finalBrandID sebagai parameter
	s, err := h.repo.Update(r.Context(), id, *inp, ctxBrandID, finalBrandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// ─── Admin: Update Status ─────────────────────────────────────────────────────

// UpdateScheduleStatus godoc
// PUT /api/admin/schedules/{id}/status
// Body: {"status": "draft"|"published"|"archived"}
// Response 200: {"id": ..., "status": "..."}
func (h *Handler) UpdateScheduleStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	validStatuses := map[string]bool{"draft": true, "published": true, "archived": true}
	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "status tidak valid, harus draft/published/archived")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	resID, resStatus, err := h.repo.UpdateStatus(r.Context(), id, req.Status, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": resID, "status": resStatus})
}

// ─── Admin: Update Seat ───────────────────────────────────────────────────────

// UpdateScheduleSeat godoc
// PUT /api/admin/schedules/{id}/seat
// Body: {"seat_sisa": int}
// Response 200: {"id": ..., "seat_sisa": ...}
func (h *Handler) UpdateScheduleSeat(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateSeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	// Ambil seat_total existing untuk validasi range
	seatTotal, err := h.repo.GetSeatTotal(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	if req.SeatSisa < 0 || req.SeatSisa > seatTotal {
		writeError(w, http.StatusBadRequest, "seat_sisa harus antara 0 dan seat_total")
		return
	}

	if err := h.repo.UpdateSeat(r.Context(), id, req.SeatSisa); err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "seat_sisa": req.SeatSisa})
}

// ─── Admin: Delete ────────────────────────────────────────────────────────────

// DeleteSchedule godoc
// DELETE /api/admin/schedules/{id}
// Response 200: {"message": "berhasil dihapus"}
func (h *Handler) DeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID := identity.GetBrandID(r.Context())
	if err := h.repo.Delete(r.Context(), id, brandID); err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
}

// ─── Validation (17 gate) ─────────────────────────────────────────────────────

// validateScheduleInput menjalankan semua 17 gate validasi sesuai urutan spesifikasi.
// Gate yang memerlukan DB query (6, 11, 13, 15) memanggil repo existence checks.
// Mengembalikan *ScheduleInput yang sudah siap pakai, atau false jika ada gate yang gagal.
func (h *Handler) validateScheduleInput(ctx context.Context, w http.ResponseWriter, req *CreateScheduleRequest) (*ScheduleInput, bool) {
	const dateLayout = "2006-01-02"

	ctxBrandID := identity.GetBrandID(ctx)
	var finalBrandID int64
	if ctxBrandID != nil {
		finalBrandID = *ctxBrandID
	} else {
		if req.BrandID == nil {
			writeError(w, http.StatusBadRequest, "brand_id wajib diisi oleh Super Admin")
			return nil, false
		}
		finalBrandID = *req.BrandID
	}

	// Gate 1: jadwal_nama wajib
	if strings.TrimSpace(req.JadwalNama) == "" {
		writeError(w, http.StatusBadRequest, "jadwal_nama wajib diisi")
		return nil, false
	}

	// Gate 1.5: status (enum: draft, published, archived)
	reqStatus := strings.TrimSpace(req.Status)
	if reqStatus != "" {
		if reqStatus != "draft" && reqStatus != "published" && reqStatus != "archived" {
			writeError(w, http.StatusBadRequest, "status tidak valid, harus 'draft', 'published', atau 'archived'")
			return nil, false
		}
	}

	// Gate 2: seat_total > 0
	if req.SeatTotal <= 0 {
		writeError(w, http.StatusBadRequest, "seat_total harus lebih dari 0")
		return nil, false
	}

	// Gate 3 & 4: seat_sisa resolution
	seatSisa := req.SeatTotal // default: sama dengan seat_total
	if req.SeatSisa != nil {
		if *req.SeatSisa < 0 || *req.SeatSisa > req.SeatTotal {
			writeError(w, http.StatusBadRequest, "seat_sisa harus antara 0 dan seat_total")
			return nil, false
		}
		seatSisa = *req.SeatSisa
	}

	// Gate 5: maskapai_id wajib
	if req.MaskapaiID == 0 {
		writeError(w, http.StatusBadRequest, "maskapai_id wajib diisi")
		return nil, false
	}

	// Gate 6: maskapai exist di DB
	if exists, err := h.repo.AirlineExists(ctx, req.MaskapaiID); err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return nil, false
	} else if !exists {
		writeError(w, http.StatusBadRequest, "maskapai_id tidak valid")
		return nil, false
	}

	// Gate 7: berangkat_tanggal wajib + format valid
	if strings.TrimSpace(req.BerangkatTanggal) == "" {
		writeError(w, http.StatusBadRequest, "berangkat_tanggal tidak valid")
		return nil, false
	}
	berangkatTime, err := time.Parse(dateLayout, req.BerangkatTanggal)
	if err != nil {
		writeError(w, http.StatusBadRequest, "berangkat_tanggal tidak valid")
		return nil, false
	}

	// Gate 8: pulang_tanggal wajib + format valid
	if strings.TrimSpace(req.PulangTanggal) == "" {
		writeError(w, http.StatusBadRequest, "pulang_tanggal tidak valid")
		return nil, false
	}
	pulangTime, err := time.Parse(dateLayout, req.PulangTanggal)
	if err != nil {
		writeError(w, http.StatusBadRequest, "pulang_tanggal tidak valid")
		return nil, false
	}

	// Gate 9: pulang_tanggal harus SETELAH berangkat_tanggal
	if !pulangTime.After(berangkatTime) {
		writeError(w, http.StatusBadRequest, "pulang_tanggal harus setelah berangkat_tanggal")
		return nil, false
	}

	// Gate 10: hotel_mekkah_id wajib
	if req.HotelMekkahID == 0 {
		writeError(w, http.StatusBadRequest, "hotel_mekkah_id wajib diisi")
		return nil, false
	}

	// Gate 11: hotel_mekkah exist di DB
	if exists, err := h.repo.HotelExists(ctx, req.HotelMekkahID); err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return nil, false
	} else if !exists {
		writeError(w, http.StatusBadRequest, "hotel_mekkah_id tidak valid")
		return nil, false
	}

	// Gate 12: hotel_madinah_id wajib
	if req.HotelMadinahID == 0 {
		writeError(w, http.StatusBadRequest, "hotel_madinah_id wajib diisi")
		return nil, false
	}

	// Gate 13: hotel_madinah exist di DB
	if exists, err := h.repo.HotelExists(ctx, req.HotelMadinahID); err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return nil, false
	} else if !exists {
		writeError(w, http.StatusBadRequest, "hotel_madinah_id tidak valid")
		return nil, false
	}

	// Gate 13.5: transit_hotel_ids opsional, tiap elemen harus ada di DB dan tidak boleh sama dengan hotel_mekkah_id atau hotel_madinah_id
	var finalTransitHotelIDs []int64
	if req.TransitHotelIDs != nil && len(req.TransitHotelIDs) > 0 {
		seenTransit := make(map[int64]bool)
		for _, thID := range req.TransitHotelIDs {
			if thID <= 0 {
				continue
			}
			if thID == req.HotelMekkahID || thID == req.HotelMadinahID {
				writeError(w, http.StatusBadRequest, "hotel transit tidak boleh sama dengan hotel Mekkah atau hotel Madinah")
				return nil, false
			}
			if seenTransit[thID] {
				writeError(w, http.StatusBadRequest, "hotel transit tidak boleh duplikat")
				return nil, false
			}
			seenTransit[thID] = true

			if exists, err := h.repo.HotelExists(ctx, thID); err != nil {
				writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
				return nil, false
			} else if !exists {
				writeError(w, http.StatusBadRequest, "hotel transit tidak valid")
				return nil, false
			}
			finalTransitHotelIDs = append(finalTransitHotelIDs, thID)
		}
	}

	// Gate 14: harga_quad, harga_triple, harga_double masing-masing > 0
	if req.HargaQuad <= 0 {
		writeError(w, http.StatusBadRequest, "harga_quad harus lebih dari 0")
		return nil, false
	}
	if req.HargaTriple <= 0 {
		writeError(w, http.StatusBadRequest, "harga_triple harus lebih dari 0")
		return nil, false
	}
	if req.HargaDouble <= 0 {
		writeError(w, http.StatusBadRequest, "harga_double harus lebih dari 0")
		return nil, false
	}

	// Gate 14.2: Validasi harga_infant (opsional, jika diisi harus >= 0)
	var finalHargaInfant *float64
	if req.HargaInfant != nil {
		if *req.HargaInfant < 0 {
			writeError(w, http.StatusBadRequest, "harga_infant tidak boleh kurang dari 0")
			return nil, false
		}
		if *req.HargaInfant > 0 {
			finalHargaInfant = req.HargaInfant
		}
	}

	// Gate 14.5: Validasi harga_coret
	var finalHargaCoret *float64
	if req.IsPromo {
		if req.HargaCoret != nil {
			if *req.HargaCoret <= req.HargaQuad {
				writeError(w, http.StatusBadRequest, "harga_coret harus lebih besar dari harga_quad")
				return nil, false
			}
			finalHargaCoret = req.HargaCoret
		}
	}

	// Gate 15: itinerary_id opsional, tapi kalau dikirim harus ada di DB
	if req.ItineraryID != nil {
		if exists, err := h.repo.ItineraryExists(ctx, *req.ItineraryID); err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return nil, false
		} else if !exists {
			writeError(w, http.StatusBadRequest, "itinerary_id tidak valid")
			return nil, false
		}
	}

	// Gate 15.2: category_id opsional, jika dikirim harus ada di DB
	if req.CategoryID != nil && *req.CategoryID > 0 {
		if exists, err := h.repo.CategoryExists(ctx, *req.CategoryID); err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return nil, false
		} else if !exists {
			writeError(w, http.StatusBadRequest, "category_id tidak valid")
			return nil, false
		}
	}

	// Gate 15.5: add_on_ids opsional, tiap elemen wajib ada di DB
	var finalAddOnIDs []int64
	if req.AddOnIDs != nil && len(req.AddOnIDs) > 0 {
		for _, addonID := range req.AddOnIDs {
			if exists, err := h.repo.AddOnExists(ctx, addonID); err != nil {
				writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
				return nil, false
			} else if !exists {
				writeError(w, http.StatusBadRequest, "add_on_id tidak valid")
				return nil, false
			}
			finalAddOnIDs = append(finalAddOnIDs, addonID)
		}
	}

	// Gate 16: include_items & exclude_items — tidak boleh berisi string kosong
	includeItems := req.IncludeItems
	if includeItems == nil {
		includeItems = []string{}
	}
	excludeItems := req.ExcludeItems
	if excludeItems == nil {
		excludeItems = []string{}
	}
	for _, item := range includeItems {
		if strings.TrimSpace(item) == "" {
			writeError(w, http.StatusBadRequest, "include_items tidak boleh berisi teks kosong")
			return nil, false
		}
	}
	for _, item := range excludeItems {
		if strings.TrimSpace(item) == "" {
			writeError(w, http.StatusBadRequest, "exclude_items tidak boleh berisi teks kosong")
			return nil, false
		}
	}

	// Gate 17: brosur_url dan brosur_thumb_url — tidak ada validasi format, terima apa adanya

	return &ScheduleInput{
		BrandID:                  finalBrandID,
		CategoryID:               req.CategoryID,
		JadwalNama:               strings.TrimSpace(req.JadwalNama),
		Status:                   reqStatus,
		IsPromo:                  req.IsPromo,
		IsTicketConfirmed:        req.IsTicketConfirmed,
		IsDirectFlight:           req.IsDirectFlight,
		SeatTotal:                req.SeatTotal,
		SeatSisa:                 seatSisa,
		MaskapaiID:               req.MaskapaiID,
		BerangkatTanggal:         req.BerangkatTanggal,
		BerangkatJam:             req.BerangkatJam,
		BerangkatKodePenerbangan: req.BerangkatKodePenerbangan,
		BerangkatBandaraAsal:     strings.TrimSpace(req.BerangkatBandaraAsal),
		BerangkatBandaraTujuan:   strings.TrimSpace(req.BerangkatBandaraTujuan),
		PulangTanggal:            req.PulangTanggal,
		PulangJam:                req.PulangJam,
		PulangKodePenerbangan:    req.PulangKodePenerbangan,
		PulangBandaraAsal:        strings.TrimSpace(req.PulangBandaraAsal),
		PulangBandaraTujuan:      strings.TrimSpace(req.PulangBandaraTujuan),
		TransitBandara:           strings.TrimSpace(req.TransitBandara),
		HotelMekkahID:            req.HotelMekkahID,
		HotelMadinahID:           req.HotelMadinahID,
		TransitHotelIDs:          finalTransitHotelIDs,
		HargaQuad:                req.HargaQuad,
		HargaTriple:              req.HargaTriple,
		HargaDouble:              req.HargaDouble,
		HargaInfant:              finalHargaInfant,
		HargaCoret:               finalHargaCoret,
		MinimalDP:                req.MinimalDP,
		ItineraryID:              req.ItineraryID,
		IncludeItems:             includeItems,
		ExcludeItems:             excludeItems,
		AddOnIDs:                 finalAddOnIDs,
		BrosurURL:                req.BrosurURL,
		BrosurThumbURL:           req.BrosurThumbURL,
	}, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	default:
		// Tangkap MySQL FK constraint violation (error 1451) — defensive
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "tidak bisa dihapus, masih ada jamaah yang booking paket ini")
			return
		}
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
