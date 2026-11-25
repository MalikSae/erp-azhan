package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"erp-azhan/api/internal/benefit"
	"erp-azhan/api/internal/itinerary"
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
		log.Printf("[ERROR] schedule: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil data jadwal.")
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
		writeError(w, http.StatusBadRequest, "Parameter brand wajib diisi.")
		return
	}

	brandID, err := strconv.ParseInt(brandRaw, 10, 64)
	if err != nil || brandID <= 0 {
		writeError(w, http.StatusBadRequest, "Parameter brand tidak valid.")
		return
	}

	items, err := h.repo.ListPublic(r.Context(), brandID)
	if err != nil {
		log.Printf("[ERROR] schedule.ListPublic: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil data jadwal.")
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
		writeError(w, http.StatusBadRequest, "Parameter brand wajib diisi dan harus valid.")
		return
	}
	s, err := h.repo.GetByID(r.Context(), id, &brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	// Paket yang berangkat kurang dari 30 hari lagi (termasuk yang sudah berangkat) tidak ditampilkan
	// dan tidak bisa dipesan lagi (MP-04); daftar publik memakai batas yang sama.
	if s.Status != StatusPublished || s.BerangkatTanggal < batasPublikWIB() {
		writeError(w, http.StatusNotFound, "Jadwal tidak ditemukan atau belum dipublikasikan.")
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
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	inp, ok := h.validateScheduleInput(r.Context(), w, &req, nil)
	if !ok {
		return
	}

	if inp.Status == "" {
		inp.Status = StatusDraft
	}
	if inp.Status == StatusPublished && inp.BerangkatTanggal < todayWIB() {
		writeError(w, http.StatusBadRequest, errPublishLampau)
		return
	}
	ids := []int64{}
	if req.BenefitIDs != nil {
		ids = *req.BenefitIDs
	}
	if !h.benefitSah(r.Context(), w, inp.BrandID, 0, ids) {
		return
	}
	if !benefitCukup(w, inp.Status, len(ids)) {
		return
	}
	inp.BenefitIDs = &ids
	lokasi := []int64{}
	if req.LokasiIDs != nil {
		lokasi = *req.LokasiIDs
	}
	if !h.lokasiSah(r.Context(), w, 0, lokasi) {
		return
	}
	inp.LokasiIDs = &lokasi
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
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
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
	// Sisa seat saat edit ditentukan repository dari data terkunci; nilai form diabaikan
	// agar kapasitas di bawah alokasi mendapat pesan yang tepat (MP-07).
	req.SeatSisa = nil
	// Status hanya berubah lewat PUT /schedules/{id}/status; nilai status di body diabaikan.
	req.Status = ""

	inp, ok := h.validateScheduleInput(r.Context(), w, &req, existing)
	if !ok {
		return
	}

	// Paket yang sedang tayang tetap tunduk pada aturan tayang saat isinya diubah.
	finalStatus := existing.Status
	if finalStatus == StatusPublished && inp.BerangkatTanggal < todayWIB() && inp.BerangkatTanggal != existing.BerangkatTanggal {
		writeError(w, http.StatusBadRequest, errPublishLampau)
		return
	}
	// Highlight benefit: daftar baru bila dikirim, selain itu yang tersimpan. Paket yang dijual wajib 4.
	jumlahBenefit := 0
	if req.BenefitIDs != nil {
		if !h.benefitSah(r.Context(), w, finalBrandID, id, *req.BenefitIDs) {
			return
		}
		inp.BenefitIDs = req.BenefitIDs
		jumlahBenefit = len(*req.BenefitIDs)
	} else if finalBrandID != existing.BrandID {
		// Benefit milik satu brand. Brand paket diganti tanpa daftar benefit baru: tautan lama dilepas,
		// supaya paket tidak membawa benefit brand lain.
		kosong := []int64{}
		inp.BenefitIDs = &kosong
	} else {
		n, err := h.repo.JumlahBenefit(r.Context(), id)
		if err != nil {
			handleRepoError(w, err)
			return
		}
		jumlahBenefit = n
	}
	if !benefitCukup(w, finalStatus, jumlahBenefit) {
		return
	}
	// Lokasi Dikunjungi: replaced only when sent.
	if req.LokasiIDs != nil {
		if !h.lokasiSah(r.Context(), w, id, *req.LokasiIDs) {
			return
		}
		inp.LokasiIDs = req.LokasiIDs
	}
	inp.AlasanUbahTanggal = strings.TrimSpace(req.AlasanUbahTanggal)
	if err := validateMaxLen("alasan_ubah_tanggal", inp.AlasanUbahTanggal, maxAlasanTanggal); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	inp.ActorID = identity.GetAdminUserID(r.Context())

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
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	validStatuses := map[string]bool{StatusDraft: true, StatusPublished: true, StatusArchived: true}
	if !validStatuses[req.Status] {
		writeError(w, http.StatusBadRequest, "Status tidak valid, harus draft/published/archived.")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	if req.Status == StatusPublished || req.Status == StatusArchived {
		s, err := h.repo.GetByID(r.Context(), id, brandID)
		if err != nil {
			handleRepoError(w, err)
			return
		}
		if req.Status == StatusPublished && s.BerangkatTanggal < todayWIB() {
			writeError(w, http.StatusBadRequest, errPublishLampau)
			return
		}
		if !benefitCukup(w, req.Status, len(s.Benefits)) {
			return
		}
	}
	// Syarat arsip (booking aktif) diperiksa repository dengan baris paket terkunci.
	resID, resStatus, err := h.repo.UpdateStatus(r.Context(), id, req.Status, brandID, todayWIB())
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": resID, "status": resStatus})
}

// ─── Admin: Samakan Seat ──────────────────────────────────────────────────────

// SamakanScheduleSeat godoc
// POST /api/admin/schedules/{id}/seat/samakan
// Mengisi seat_sisa = seat_total − seat dipegang booking. Tidak ada isian angka: semua seat
// terpakai wajib punya booking di ERP.
// Response 200: {"id": ..., "seat_sisa": ...}
func (h *Handler) SamakanScheduleSeat(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	sisa, err := h.repo.SamakanSeat(r.Context(), id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()))
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "seat_sisa": sisa})
}

// UpdateScheduleSeatTampil godoc
// PUT /api/admin/schedules/{id}/seat-tampil
// Body: {"seat_tampil": int|null}; null mematikan scarcity. Hanya label publik, seat_sisa tidak berubah.
// Response 200: {"id": ..., "seat_tampil": ...}
func (h *Handler) UpdateScheduleSeatTampil(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateSeatTampilRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	seatTotal, err := h.repo.GetSeatTotal(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	if req.SeatTampil != nil && (*req.SeatTampil < 1 || *req.SeatTampil > seatTotal) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("Seat tampil harus antara 1 dan %d.", seatTotal))
		return
	}

	if err := h.repo.UpdateSeatTampil(r.Context(), id, req.SeatTampil, brandID, identity.GetAdminUserID(r.Context())); err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": id, "seat_tampil": req.SeatTampil})
}

// GetScheduleSeat godoc
// GET /api/admin/schedules/{id}/seat
// Response 200: SeatOverview (ringkasan seat + riwayat penyesuaian terakhir)
func (h *Handler) GetScheduleSeat(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	o, err := h.repo.SeatOverview(r.Context(), id, identity.GetBrandID(r.Context()))
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
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
// existing nil = paket baru; terisi = edit (sebagian gate hanya berlaku untuk nilai yang berubah).
func (h *Handler) validateScheduleInput(ctx context.Context, w http.ResponseWriter, req *CreateScheduleRequest, existing *Schedule) (*ScheduleInput, bool) {
	const dateLayout = "2006-01-02"

	ctxBrandID := identity.GetBrandID(ctx)
	var finalBrandID int64
	if ctxBrandID != nil {
		finalBrandID = *ctxBrandID
	} else {
		if req.BrandID == nil {
			writeError(w, http.StatusBadRequest, "Brand wajib dipilih.")
			return nil, false
		}
		finalBrandID = *req.BrandID
	}

	// Gate 1: jadwal_nama wajib
	if strings.TrimSpace(req.JadwalNama) == "" {
		writeError(w, http.StatusBadRequest, "Nama paket wajib diisi.")
		return nil, false
	}
	if len([]rune(strings.TrimSpace(req.JadwalNama))) > 255 {
		writeError(w, http.StatusBadRequest, "Nama paket maksimal 255 karakter.")
		return nil, false
	}

	// Gate 1.5: status (enum: draft, published, archived)
	reqStatus := strings.TrimSpace(req.Status)
	if reqStatus != "" {
		if reqStatus != StatusDraft && reqStatus != StatusPublished && reqStatus != StatusArchived {
			writeError(w, http.StatusBadRequest, "Status tidak valid, harus 'draft', 'published', atau 'archived'.")
			return nil, false
		}
	}

	// Gate 2: seat_total > 0
	if req.SeatTotal <= 0 {
		writeError(w, http.StatusBadRequest, "Total seat harus lebih dari 0.")
		return nil, false
	}

	// Gate 3 & 4: seat_sisa. Paket baru selalu mulai penuh; pada edit, nilai ini diabaikan
	// (repository mempertahankan sisa tersimpan, atau menghitungnya dari booking bila kuota berubah).
	seatSisa := req.SeatTotal
	if existing == nil {
		var err error
		seatSisa, err = seatSisaForCreate(req.SeatTotal, req.SeatSisa)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return nil, false
		}
	}

	// Gate 5: maskapai_id wajib
	if req.MaskapaiID == 0 {
		writeError(w, http.StatusBadRequest, "Maskapai wajib diisi.")
		return nil, false
	}

	// Gate 6: maskapai exist di DB
	if exists, err := h.repo.AirlineExists(ctx, req.MaskapaiID); err != nil {
		log.Printf("[ERROR] schedule: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return nil, false
	} else if !exists {
		writeError(w, http.StatusBadRequest, "Maskapai tidak valid.")
		return nil, false
	}

	// Gate 7: berangkat_tanggal wajib + format valid
	if strings.TrimSpace(req.BerangkatTanggal) == "" {
		writeError(w, http.StatusBadRequest, "Tanggal berangkat tidak valid.")
		return nil, false
	}
	berangkatTime, err := time.Parse(dateLayout, req.BerangkatTanggal)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Tanggal berangkat tidak valid.")
		return nil, false
	}

	// Gate 8: pulang_tanggal wajib + format valid
	if strings.TrimSpace(req.PulangTanggal) == "" {
		writeError(w, http.StatusBadRequest, "Tanggal pulang tidak valid.")
		return nil, false
	}
	pulangTime, err := time.Parse(dateLayout, req.PulangTanggal)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Tanggal pulang tidak valid.")
		return nil, false
	}

	// Gate 9: pulang_tanggal harus SETELAH berangkat_tanggal
	if !pulangTime.After(berangkatTime) {
		writeError(w, http.StatusBadRequest, "Tanggal pulang harus setelah tanggal berangkat.")
		return nil, false
	}

	// Gate 10: hotel_mekkah_id wajib
	if req.HotelMekkahID == 0 {
		writeError(w, http.StatusBadRequest, "Hotel Makkah wajib diisi.")
		return nil, false
	}

	// Gate 11: hotel_mekkah exist di DB dan berada di Makkah (MP-03)
	if ok := h.checkHotelCity(ctx, w, req.HotelMekkahID, "Makkah", "hotel_mekkah_id"); !ok {
		return nil, false
	}

	// Gate 12: hotel_madinah_id wajib
	if req.HotelMadinahID == 0 {
		writeError(w, http.StatusBadRequest, "Hotel Madinah wajib diisi.")
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
				writeError(w, http.StatusBadRequest, "Hotel transit tidak boleh sama dengan hotel Mekkah atau hotel Madinah.")
				return nil, false
			}
			if seenTransit[thID] {
				writeError(w, http.StatusBadRequest, "Hotel transit tidak boleh duplikat.")
				return nil, false
			}
			seenTransit[thID] = true

			if exists, err := h.repo.HotelExists(ctx, thID); err != nil {
				log.Printf("[ERROR] schedule: %v", err)
				writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
				return nil, false
			} else if !exists {
				writeError(w, http.StatusBadRequest, "Hotel transit tidak valid.")
				return nil, false
			}
			finalTransitHotelIDs = append(finalTransitHotelIDs, thID)
		}
	}

	if pesan := pesanNominalPaketBulat(
		nominalPaket{"harga_quad", &req.HargaQuad}, nominalPaket{"harga_triple", &req.HargaTriple}, nominalPaket{"harga_double", &req.HargaDouble},
		nominalPaket{"harga_infant", req.HargaInfant}, nominalPaket{"nominal_komisi_langsung", req.NominalKomisiLangsung},
		nominalPaket{"nominal_bonus_pembinaan", req.NominalBonusPembinaan},
	); pesan != "" {
		writeError(w, http.StatusBadRequest, pesan)
		return nil, false
	}

	// Gate 14: harga_quad, harga_triple, harga_double masing-masing > 0
	if req.HargaQuad <= 0 {
		writeError(w, http.StatusBadRequest, "Harga Quad harus lebih dari 0.")
		return nil, false
	}
	if req.HargaTriple <= 0 {
		writeError(w, http.StatusBadRequest, "Harga Triple harus lebih dari 0.")
		return nil, false
	}
	if req.HargaDouble <= 0 {
		writeError(w, http.StatusBadRequest, "Harga Double harus lebih dari 0.")
		return nil, false
	}
	// Gate 14.1: makin sedikit orang per kamar, harga per orang tidak boleh lebih murah (MP-09).
	if req.HargaTriple < req.HargaQuad {
		writeError(w, http.StatusBadRequest, "Harga Triple tidak boleh lebih murah dari harga Quad.")
		return nil, false
	}
	if req.HargaDouble < req.HargaTriple {
		writeError(w, http.StatusBadRequest, "Harga Double tidak boleh lebih murah dari harga Triple.")
		return nil, false
	}

	// Brand selalu diperiksa, bukan hanya saat DP kosong: brand_id tak dikenal akan
	// menabrak FK saat insert dan sampai ke klien sebagai 500 (TP-01).
	brandDP, err := h.repo.BrandMinimalDP(ctx, finalBrandID)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusBadRequest, "Brand tidak valid.")
		return nil, false
	}
	if err != nil {
		log.Printf("[ERROR] schedule: cek brand: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return nil, false
	}
	effectiveDP := req.MinimalDP
	if effectiveDP == nil {
		effectiveDP = &brandDP
	}
	if err := validateDP(effectiveDP, req.HargaQuad, req.HargaTriple, req.HargaDouble); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return nil, false
	}

	// Gate 14.2: Validasi harga_infant (opsional, jika diisi harus >= 0)
	var finalHargaInfant *float64
	if req.HargaInfant != nil {
		if *req.HargaInfant < 0 {
			writeError(w, http.StatusBadRequest, "Harga bayi tidak boleh kurang dari 0.")
			return nil, false
		}
		if *req.HargaInfant > req.HargaQuad {
			writeError(w, http.StatusBadRequest, "Harga bayi tidak boleh lebih mahal dari harga Quad.")
			return nil, false
		}
		finalHargaInfant = req.HargaInfant
	}

	// Gate 14.3: Nominal komisi Syiar (opsional; kosong = jadwal tidak ikut program)
	if req.NominalKomisiLangsung != nil && *req.NominalKomisiLangsung < 0 {
		writeError(w, http.StatusBadRequest, "Komisi langsung tidak boleh kurang dari 0.")
		return nil, false
	}
	if req.NominalBonusPembinaan != nil && *req.NominalBonusPembinaan < 0 {
		writeError(w, http.StatusBadRequest, "Bonus pembinaan tidak boleh kurang dari 0.")
		return nil, false
	}

	// Gate 15: itinerary_id opsional, tapi kalau dikirim harus ada di DB
	// Itinerary khusus (objek) menggantikan itinerary_id master dari body yang sama.
	if req.ItineraryKhusus != nil {
		req.ItineraryID = nil
		if err := itinerary.ValidateDays(req.ItineraryKhusus.Days); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return nil, false
		}
	}
	if req.ItineraryID != nil {
		var paketID int64
		if existing != nil {
			paketID = existing.ID
		}
		if exists, err := h.repo.ItineraryTersedia(ctx, *req.ItineraryID, paketID); err != nil {
			log.Printf("[ERROR] schedule: %v", err)
			writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return nil, false
		} else if !exists {
			writeError(w, http.StatusBadRequest, "Itinerary tidak valid.")
			return nil, false
		}
	}

	// Gate 15.2: category_id opsional, jika dikirim harus ada di DB
	if req.CategoryID != nil && *req.CategoryID > 0 {
		// Kategori nonaktif tidak bisa dipilih untuk paket baru atau saat mengganti kategori, tetapi paket yang
		// sudah memakainya tetap bisa disimpan (audit menu Paket S-09, S-10).
		tetap := existing != nil && existing.CategoryID != nil && *existing.CategoryID == *req.CategoryID
		if exists, err := h.repo.CategoryAvailable(ctx, *req.CategoryID, finalBrandID, tetap); err != nil {
			log.Printf("[ERROR] schedule: %v", err)
			writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return nil, false
		} else if !exists {
			writeError(w, http.StatusBadRequest, "Kategori tidak valid atau kategori tidak aktif.")
			return nil, false
		}
	}

	// Gate 15.3: klasifikasi web publik (journey_type).
	// Paket baru tanpa journey_type memakai default umroh; saat edit, tanpa journey_type klasifikasi tidak diubah.
	journey := req.JourneyType
	if journey == "" && existing == nil {
		journey = JourneyUmroh
	}
	klasifikasi, err := normalizeKlasifikasi(journey)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return nil, false
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
			writeError(w, http.StatusBadRequest, "Item termasuk tidak boleh berisi teks kosong.")
			return nil, false
		}
	}
	for _, item := range excludeItems {
		if strings.TrimSpace(item) == "" {
			writeError(w, http.StatusBadRequest, "Item tidak termasuk tidak boleh berisi teks kosong.")
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
			writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return nil, false
		}
		if !exists {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("%s %q tidak terdaftar, pilih kode dari daftar bandara.", ap.nama, code))
			return nil, false
		}
	}

	// Gate 16.6: format jam dan panjang teks penerbangan (TP-02, TP-03).
	for _, f := range []error{
		validateJam("berangkat_jam", req.BerangkatJam),
		validateJam("pulang_jam", req.PulangJam),
		validateMaxLen("berangkat_kode_penerbangan", strings.TrimSpace(req.BerangkatKodePenerbangan), maxKodePenerbangan),
		validateMaxLen("pulang_kode_penerbangan", strings.TrimSpace(req.PulangKodePenerbangan), maxKodePenerbangan),
		validateMaxLen("transit_bandara", strings.TrimSpace(req.TransitBandara), maxTransitBandara),
	} {
		if f != nil {
			writeError(w, http.StatusBadRequest, f.Error())
			return nil, false
		}
	}

	// Gate 17: brosur_url dan brosur_thumb_url harus path unggahan atau URL http(s) (MP-08)
	if !shared.ValidMediaURL(req.BrosurURL) || !shared.ValidMediaURL(req.BrosurThumbURL) {
		writeError(w, http.StatusBadRequest, "Brosur harus berupa berkas unggahan atau URL http(s).")
		return nil, false
	}

	// Gate 18: nama + tanggal berangkat tidak boleh kembar dalam satu brand (TP-06).
	// Hanya dicek untuk paket baru atau saat ketiganya berubah, agar duplikat lama tetap bisa diedit.
	namaFinal := strings.TrimSpace(req.JadwalNama)
	if existing == nil || namaFinal != existing.JadwalNama || req.BerangkatTanggal != existing.BerangkatTanggal || finalBrandID != existing.BrandID {
		var excludeID int64
		if existing != nil {
			excludeID = existing.ID
		}
		taken, err := h.repo.NameTaken(ctx, finalBrandID, namaFinal, req.BerangkatTanggal, excludeID)
		if err != nil {
			log.Printf("[ERROR] schedule: cek nama kembar: %v", err)
			writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return nil, false
		}
		if taken {
			writeError(w, http.StatusConflict, "Paket dengan nama dan tanggal berangkat yang sama sudah ada di brand ini.")
			return nil, false
		}
	}

	return &ScheduleInput{
		ExpectedSeatTotal:        req.ExpectedSeatTotal,
		BrandID:                  finalBrandID,
		CategoryID:               req.CategoryID,
		Klasifikasi:              klasifikasi,
		JadwalNama:               strings.TrimSpace(req.JadwalNama),
		Status:                   reqStatus,
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
		MinimalDP:                req.MinimalDP,
		NominalKomisiLangsung:    req.NominalKomisiLangsung,
		NominalBonusPembinaan:    req.NominalBonusPembinaan,
		ItineraryID:              req.ItineraryID,
		ItineraryKhusus:          req.ItineraryKhusus,
		IncludeItems:             includeItems,
		ExcludeItems:             excludeItems,
		BrosurURL:                req.BrosurURL,
		BrosurThumbURL:           req.BrosurThumbURL,
	}, true
}

// ─── Error handling ───────────────────────────────────────────────────────────

func handleRepoError(w http.ResponseWriter, err error) {
	var le LokasiError
	switch {
	case errors.As(err, &le):
		writeError(w, http.StatusBadRequest, le.Error())
	case errors.Is(err, ErrConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "Data tidak ditemukan.")
	default:
		// Tangkap MySQL FK constraint violation (error 1451) — defensive
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			writeError(w, http.StatusConflict, "Tidak bisa dihapus, masih dipakai oleh booking.")
			return
		}
		log.Printf("[ERROR] schedule: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
	}
}

// checkHotelCity memastikan hotel ada dan kotanya sesuai slot (Makkah/Madinah).
func (h *Handler) checkHotelCity(ctx context.Context, w http.ResponseWriter, id int64, kota, field string) bool {
	city, err := h.repo.HotelCity(ctx, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusBadRequest, field+" Tidak valid.")
		return false
	}
	if err != nil {
		log.Printf("[ERROR] schedule: cek hotel: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return false
	}
	if normalizeKota(city) != normalizeKota(kota) {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("%s harus hotel di %s (hotel terpilih berada di %s).", field, kota, city))
		return false
	}
	return true
}

// todayWIB mengembalikan tanggal hari ini (YYYY-MM-DD) zona bisnis.
func todayWIB() string {
	return time.Now().In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02")
}

// batasPublikWIB: tanggal berangkat paling awal yang masih tampil dan bisa dipesan di situs publik (hari ini + 30 hari).
func batasPublikWIB() string {
	return time.Now().In(time.FixedZone("WIB", 7*60*60)).AddDate(0, 0, 30).Format("2006-01-02")
}

// errPublishLampau: paket tidak boleh terbit bila keberangkatan sudah lewat (MP-04).
// errBenefitKurang: paket yang tampil ke publik selalu membawa empat highlight benefit (keputusan 13 November
// 2026). Draf boleh belum lengkap.
const errBenefitKurang = "paket yang dijual wajib punya 4 highlight benefit"

// benefitCukup menolak status published bila jumlah benefit bukan empat. Draf dan arsip tidak diperiksa.
func benefitCukup(w http.ResponseWriter, status string, jumlah int) bool {
	if status == StatusPublished && jumlah != benefit.JumlahPerPaket {
		writeError(w, http.StatusBadRequest, errBenefitKurang)
		return false
	}
	return true
}

// benefitSah memeriksa daftar benefit yang dipilih; pesan untuk pengguna ditulis sebagai 400.
func (h *Handler) benefitSah(ctx context.Context, w http.ResponseWriter, brandID, scheduleID int64, ids []int64) bool {
	err := h.repo.ValidasiBenefit(ctx, brandID, scheduleID, ids)
	if err == nil {
		return true
	}
	var pe benefit.PilihanError
	if errors.As(err, &pe) {
		writeError(w, http.StatusBadRequest, pe.Error())
		return false
	}
	log.Printf("[ERROR] schedule: validasi benefit: %v", err)
	writeError(w, http.StatusInternalServerError, "Gagal memeriksa highlight benefit.")
	return false
}

const errPublishLampau = "paket dengan tanggal berangkat yang sudah lewat tidak dapat diterbitkan"

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid.")
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
