package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
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
		fmt.Println("ListPublic Error:", err)
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

	brandID, err := strconv.ParseInt(r.URL.Query().Get("brand"), 10, 64)
	if err != nil || brandID <= 0 {
		writeError(w, http.StatusBadRequest, "parameter brand wajib diisi dan harus valid")
		return
	}
	s, err := h.repo.GetByID(r.Context(), id, &brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	// Paket yang sudah berangkat tidak ditampilkan lagi (MP-04); daftar publik sudah memfilter.
	if s.Status != "published" || s.BerangkatTanggal < todayWIB() {
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

	if inp.Status == "" {
		inp.Status = "draft"
	}
	if inp.Status == "published" && inp.BerangkatTanggal < todayWIB() {
		writeError(w, http.StatusBadRequest, errPublishLampau)
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
	// Sisa kursi saat edit dihitung repository dari data terkunci; nilai form diabaikan
	// agar kapasitas di bawah alokasi mendapat pesan yang tepat (MP-07).
	req.SeatSisa = nil

	inp, ok := h.validateScheduleInput(r.Context(), w, &req)
	if !ok {
		return
	}

	finalStatus := inp.Status
	if finalStatus == "" {
		finalStatus = existing.Status
	}
	tanggalBerubah := inp.BerangkatTanggal != existing.BerangkatTanggal
	if finalStatus == "published" && inp.BerangkatTanggal < todayWIB() && (tanggalBerubah || existing.Status != "published") {
		writeError(w, http.StatusBadRequest, errPublishLampau)
		return
	}
	if finalStatus == "archived" && existing.Status != "archived" {
		if !h.bolehArsip(r.Context(), w, id, inp.BerangkatTanggal) {
			return
		}
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
	if req.Status == "published" || req.Status == "archived" {
		s, err := h.repo.GetByID(r.Context(), id, brandID)
		if err != nil {
			handleRepoError(w, err)
			return
		}
		if req.Status == "published" && s.BerangkatTanggal < todayWIB() {
			writeError(w, http.StatusBadRequest, errPublishLampau)
			return
		}
		if req.Status == "archived" && s.Status != "archived" {
			if !h.bolehArsip(r.Context(), w, id, s.BerangkatTanggal) {
				return
			}
		}
	}
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

	if err := h.repo.UpdateSeat(r.Context(), id, req.SeatSisa, req.ExpectedSeatSisa, brandID, req.Reason); err != nil {
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
	if len([]rune(strings.TrimSpace(req.JadwalNama))) > 255 {
		writeError(w, http.StatusBadRequest, "jadwal_nama maksimal 255 karakter")
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

	// Gate 11: hotel_mekkah exist di DB dan berada di Makkah (MP-03)
	if ok := h.checkHotelCity(ctx, w, req.HotelMekkahID, "Makkah", "hotel_mekkah_id"); !ok {
		return nil, false
	}

	// Gate 12: hotel_madinah_id wajib
	if req.HotelMadinahID == 0 {
		writeError(w, http.StatusBadRequest, "hotel_madinah_id wajib diisi")
		return nil, false
	}

	// Gate 13: hotel_madinah exist di DB dan berada di Madinah (MP-03)
	if ok := h.checkHotelCity(ctx, w, req.HotelMadinahID, "Madinah", "hotel_madinah_id"); !ok {
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
	// Gate 14.1: makin sedikit orang per kamar, harga per orang tidak boleh lebih murah (MP-09).
	if req.HargaTriple < req.HargaQuad {
		writeError(w, http.StatusBadRequest, "harga_triple tidak boleh lebih murah dari harga_quad")
		return nil, false
	}
	if req.HargaDouble < req.HargaTriple {
		writeError(w, http.StatusBadRequest, "harga_double tidak boleh lebih murah dari harga_triple")
		return nil, false
	}

	effectiveDP := req.MinimalDP
	if effectiveDP == nil {
		value, err := h.repo.BrandMinimalDP(ctx, finalBrandID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "brand_id tidak valid")
			return nil, false
		}
		effectiveDP = &value
	}
	if err := validateDP(effectiveDP, req.HargaQuad, req.HargaTriple, req.HargaDouble); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return nil, false
	}

	// Gate 14.2: Validasi harga_infant (opsional, jika diisi harus >= 0)
	var finalHargaInfant *float64
	if req.HargaInfant != nil {
		if *req.HargaInfant < 0 {
			writeError(w, http.StatusBadRequest, "harga_infant tidak boleh kurang dari 0")
			return nil, false
		}
		if *req.HargaInfant > req.HargaQuad {
			writeError(w, http.StatusBadRequest, "harga_infant tidak boleh lebih mahal dari harga_quad")
			return nil, false
		}
		finalHargaInfant = req.HargaInfant
	}

	// Gate 14.3: Nominal komisi Syiar (opsional; kosong = jadwal tidak ikut program)
	if req.NominalKomisiLangsung != nil && *req.NominalKomisiLangsung < 0 {
		writeError(w, http.StatusBadRequest, "nominal_komisi_langsung tidak boleh kurang dari 0")
		return nil, false
	}
	if req.NominalBonusPembinaan != nil && *req.NominalBonusPembinaan < 0 {
		writeError(w, http.StatusBadRequest, "nominal_bonus_pembinaan tidak boleh kurang dari 0")
		return nil, false
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
		if exists, err := h.repo.CategoryAvailable(ctx, *req.CategoryID, finalBrandID); err != nil {
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

	// Gate 16.5: kode bandara (jika diisi) wajib terdaftar di master bandara (MP-06).
	// Rute transit memakai format tersendiri dan tidak divalidasi di sini.
	for _, ap := range []struct {
		nama  string
		nilai *string
	}{
		{"bandara asal keberangkatan", &req.BerangkatBandaraAsal},
		{"bandara tujuan keberangkatan", &req.BerangkatBandaraTujuan},
		{"bandara asal kepulangan", &req.PulangBandaraAsal},
		{"bandara tujuan kepulangan", &req.PulangBandaraTujuan},
	} {
		code := strings.ToUpper(strings.TrimSpace(*ap.nilai))
		*ap.nilai = code
		if code == "" {
			continue
		}
		exists, err := h.repo.AirportCodeExists(ctx, code)
		if err != nil {
			log.Printf("[ERROR] schedule: cek bandara: %v", err)
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
			return nil, false
		}
		if !exists {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("%s %q tidak terdaftar, pilih kode dari daftar bandara", ap.nama, code))
			return nil, false
		}
	}

	// Gate 17: brosur_url dan brosur_thumb_url harus path unggahan atau URL http(s) (MP-08)
	if !shared.ValidMediaURL(req.BrosurURL) || !shared.ValidMediaURL(req.BrosurThumbURL) {
		writeError(w, http.StatusBadRequest, "brosur harus berupa berkas unggahan atau URL http(s)")
		return nil, false
	}

	var promoUntil *time.Time
	if req.IsPromo && req.PromoUntil != nil && req.PromoUntil.Valid {
		promoUntil = &req.PromoUntil.Time
	}

	return &ScheduleInput{
		ExpectedSeatTotal:        req.ExpectedSeatTotal,
		BrandID:                  finalBrandID,
		CategoryID:               req.CategoryID,
		JadwalNama:               strings.TrimSpace(req.JadwalNama),
		Status:                   reqStatus,
		IsPromo:                  req.IsPromo,
		PromoUntil:               promoUntil,
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
		NominalKomisiLangsung:    req.NominalKomisiLangsung,
		NominalBonusPembinaan:    req.NominalBonusPembinaan,
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
	case errors.Is(err, ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
	default:
		// Tangkap MySQL FK constraint violation (error 1451) — defensive
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "tidak bisa dihapus, masih ada jamaah yang booking paket ini")
			return
		}
		log.Printf("[ERROR] schedule: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal, silakan coba lagi")
	}
}

// checkHotelCity memastikan hotel ada dan kotanya sesuai slot (Makkah/Madinah).
func (h *Handler) checkHotelCity(ctx context.Context, w http.ResponseWriter, id int64, kota, field string) bool {
	city, err := h.repo.HotelCity(ctx, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusBadRequest, field+" tidak valid")
		return false
	}
	if err != nil {
		log.Printf("[ERROR] schedule: cek hotel: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(city), kota) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s harus hotel di %s (hotel terpilih berada di %s)", field, kota, city))
		return false
	}
	return true
}

// todayWIB mengembalikan tanggal hari ini (YYYY-MM-DD) zona bisnis.
func todayWIB() string {
	return time.Now().In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02")
}

// bolehArsip: sebelum tanggal berangkat lewat, paket dengan booking aktif tidak boleh
// diarsipkan; untuk menghentikan penjualan gunakan status draft (MP-09).
func (h *Handler) bolehArsip(ctx context.Context, w http.ResponseWriter, id int64, berangkat string) bool {
	if berangkat < todayWIB() {
		return true
	}
	aktif, err := h.repo.CountActiveBookings(ctx, id)
	if err != nil {
		log.Printf("[ERROR] schedule: hitung booking aktif: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
		return false
	}
	if aktif > 0 {
		writeError(w, http.StatusConflict, fmt.Sprintf("paket belum berangkat dan masih punya %d booking aktif; untuk menghentikan penjualan, ubah status ke Draft", aktif))
		return false
	}
	return true
}

// errPublishLampau: paket tidak boleh terbit bila keberangkatan sudah lewat (MP-04).
const errPublishLampau = "paket dengan tanggal berangkat yang sudah lewat tidak dapat diterbitkan"

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

func (h *Handler) IncrementViewsPublic(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	err := h.repo.IncrementViews(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "paket tidak ditemukan"})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "gagal mencatat view"})
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "success"})
}
