package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/itinerary"
	"erp-azhan/api/internal/shared"
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
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data jadwal.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
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
	shared.WriteJSON(w, http.StatusOK, s)
}

// ─── Public: List Published ───────────────────────────────────────────────────

// ─── Admin: Create ────────────────────────────────────────────────────────────

// CreateSchedule godoc
// POST /api/admin/schedules
// Response 201: Schedule baru dengan status="draft"
func (h *Handler) CreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req CreateScheduleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
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
		shared.WriteError(w, http.StatusBadRequest, errPublishLampau)
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
	shared.WriteJSON(w, http.StatusCreated, s)
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
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	ctxBrandID := identity.GetBrandID(r.Context())

	// 1. Ambil row schedule existing dulu untuk tahu brand_id lama-nya
	existing, err := h.repo.GetByID(r.Context(), id, ctxBrandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	// 2. Brand paket dikunci sejak dibuat: manasik, banner, dan data brand lain merujuk paket ini.
	finalBrandID := existing.BrandID
	if req.BrandID != nil && *req.BrandID != 0 && *req.BrandID != existing.BrandID {
		shared.WriteError(w, http.StatusConflict, errBrandTerkunci)
		return
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
		shared.WriteError(w, http.StatusBadRequest, errPublishLampau)
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
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	inp.ActorID = identity.GetAdminUserID(r.Context())

	// 3. Panggil repository Update dengan finalBrandID sebagai parameter
	s, err := h.repo.Update(r.Context(), id, *inp, ctxBrandID, finalBrandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, s)
}

// ─── Admin: Update Status ─────────────────────────────────────────────────────

// ─── Admin: Samakan Seat ──────────────────────────────────────────────────────

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
	shared.WriteJSON(w, http.StatusOK, map[string]string{"message": "berhasil dihapus"})
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
			shared.WriteError(w, http.StatusBadRequest, "Brand wajib dipilih.")
			return nil, false
		}
		finalBrandID = *req.BrandID
	}

	// Gate 1: jadwal_nama wajib
	if strings.TrimSpace(req.JadwalNama) == "" {
		shared.WriteError(w, http.StatusBadRequest, "Nama paket wajib diisi.")
		return nil, false
	}
	if len([]rune(strings.TrimSpace(req.JadwalNama))) > 255 {
		shared.WriteError(w, http.StatusBadRequest, "Nama paket maksimal 255 karakter.")
		return nil, false
	}

	// Gate 1.5: status (enum: draft, published, archived)
	reqStatus := strings.TrimSpace(req.Status)
	if reqStatus != "" {
		if reqStatus != StatusDraft && reqStatus != StatusPublished && reqStatus != StatusArchived {
			shared.WriteError(w, http.StatusBadRequest, "Status tidak valid, harus 'draft', 'published', atau 'archived'.")
			return nil, false
		}
	}

	// Gate 2: seat_total > 0
	if req.SeatTotal <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Total seat harus lebih dari 0.")
		return nil, false
	}

	// Gate 3 & 4: seat_sisa. Paket baru selalu mulai penuh; pada edit, nilai ini diabaikan
	// (repository mempertahankan sisa tersimpan, atau menghitungnya dari booking bila kuota berubah).
	seatSisa := req.SeatTotal
	if existing == nil {
		var err error
		seatSisa, err = seatSisaForCreate(req.SeatTotal, req.SeatSisa)
		if err != nil {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
			return nil, false
		}
	}

	// Gate 5: maskapai_id wajib
	if req.MaskapaiID == 0 {
		shared.WriteError(w, http.StatusBadRequest, "Maskapai wajib diisi.")
		return nil, false
	}

	// Gate 6: maskapai exist di DB
	if exists, err := h.repo.AirlineExists(ctx, req.MaskapaiID); err != nil {
		log.Printf("[ERROR] schedule: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return nil, false
	} else if !exists {
		shared.WriteError(w, http.StatusBadRequest, "Maskapai tidak valid.")
		return nil, false
	}

	// Gate 7: berangkat_tanggal wajib + format valid
	if strings.TrimSpace(req.BerangkatTanggal) == "" {
		shared.WriteError(w, http.StatusBadRequest, "Tanggal berangkat tidak valid.")
		return nil, false
	}
	berangkatTime, err := time.Parse(dateLayout, req.BerangkatTanggal)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Tanggal berangkat tidak valid.")
		return nil, false
	}

	// Gate 8: pulang_tanggal wajib + format valid
	if strings.TrimSpace(req.PulangTanggal) == "" {
		shared.WriteError(w, http.StatusBadRequest, "Tanggal pulang tidak valid.")
		return nil, false
	}
	pulangTime, err := time.Parse(dateLayout, req.PulangTanggal)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Tanggal pulang tidak valid.")
		return nil, false
	}

	// Gate 9: pulang_tanggal harus SETELAH berangkat_tanggal
	if !pulangTime.After(berangkatTime) {
		shared.WriteError(w, http.StatusBadRequest, "Tanggal pulang harus setelah tanggal berangkat.")
		return nil, false
	}

	// Gate 10: hotel_mekkah_id wajib
	if req.HotelMekkahID == 0 {
		shared.WriteError(w, http.StatusBadRequest, "Hotel Makkah wajib diisi.")
		return nil, false
	}

	// Gate 11: hotel_mekkah exist di DB dan berada di Makkah (MP-03)
	if ok := h.checkHotelCity(ctx, w, req.HotelMekkahID, "Makkah", "hotel_mekkah_id"); !ok {
		return nil, false
	}

	// Gate 12: hotel_madinah_id wajib
	if req.HotelMadinahID == 0 {
		shared.WriteError(w, http.StatusBadRequest, "Hotel Madinah wajib diisi.")
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
				shared.WriteError(w, http.StatusBadRequest, "Hotel transit tidak boleh sama dengan hotel Mekkah atau hotel Madinah.")
				return nil, false
			}
			if seenTransit[thID] {
				shared.WriteError(w, http.StatusBadRequest, "Hotel transit tidak boleh duplikat.")
				return nil, false
			}
			seenTransit[thID] = true

			if exists, err := h.repo.HotelExists(ctx, thID); err != nil {
				log.Printf("[ERROR] schedule: %v", err)
				shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
				return nil, false
			} else if !exists {
				shared.WriteError(w, http.StatusBadRequest, "Hotel transit tidak valid.")
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
		shared.WriteError(w, http.StatusBadRequest, pesan)
		return nil, false
	}

	// Gate 14: harga_quad, harga_triple, harga_double masing-masing > 0
	if req.HargaQuad <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Harga Quad harus lebih dari 0.")
		return nil, false
	}
	if req.HargaTriple <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Harga Triple harus lebih dari 0.")
		return nil, false
	}
	if req.HargaDouble <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Harga Double harus lebih dari 0.")
		return nil, false
	}
	// Gate 14.1: makin sedikit orang per kamar, harga per orang tidak boleh lebih murah (MP-09).
	if req.HargaTriple < req.HargaQuad {
		shared.WriteError(w, http.StatusBadRequest, "Harga Triple tidak boleh lebih murah dari harga Quad.")
		return nil, false
	}
	if req.HargaDouble < req.HargaTriple {
		shared.WriteError(w, http.StatusBadRequest, "Harga Double tidak boleh lebih murah dari harga Triple.")
		return nil, false
	}

	// Brand selalu diperiksa, bukan hanya saat DP kosong: brand_id tak dikenal akan
	// menabrak FK saat insert dan sampai ke klien sebagai 500 (TP-01).
	brandDP, err := h.repo.BrandMinimalDP(ctx, finalBrandID)
	if errors.Is(err, sql.ErrNoRows) {
		shared.WriteError(w, http.StatusBadRequest, "Brand tidak valid.")
		return nil, false
	}
	if err != nil {
		log.Printf("[ERROR] schedule: cek brand: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return nil, false
	}
	effectiveDP := req.MinimalDP
	if effectiveDP == nil {
		effectiveDP = &brandDP
	}
	if err := validateDP(effectiveDP, req.HargaQuad, req.HargaTriple, req.HargaDouble); err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return nil, false
	}

	// Gate 14.2: Validasi harga_infant (opsional, jika diisi harus >= 0)
	var finalHargaInfant *float64
	if req.HargaInfant != nil {
		if *req.HargaInfant < 0 {
			shared.WriteError(w, http.StatusBadRequest, "Harga bayi tidak boleh kurang dari 0.")
			return nil, false
		}
		if *req.HargaInfant > req.HargaQuad {
			shared.WriteError(w, http.StatusBadRequest, "Harga bayi tidak boleh lebih mahal dari harga Quad.")
			return nil, false
		}
		finalHargaInfant = req.HargaInfant
	}

	// Gate 14.3: Nominal komisi Syiar (opsional; kosong = jadwal tidak ikut program)
	if req.NominalKomisiLangsung != nil && *req.NominalKomisiLangsung < 0 {
		shared.WriteError(w, http.StatusBadRequest, "Komisi langsung tidak boleh kurang dari 0.")
		return nil, false
	}
	if req.NominalBonusPembinaan != nil && *req.NominalBonusPembinaan < 0 {
		shared.WriteError(w, http.StatusBadRequest, "Bonus pembinaan tidak boleh kurang dari 0.")
		return nil, false
	}

	// Gate 15: itinerary_id opsional, tapi kalau dikirim harus ada di DB
	// Itinerary khusus (objek) menggantikan itinerary_id master dari body yang sama.
	if req.ItineraryKhusus != nil {
		req.ItineraryID = nil
		if err := itinerary.ValidateDays(req.ItineraryKhusus.Days); err != nil {
			shared.WriteError(w, http.StatusBadRequest, err.Error())
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
			shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return nil, false
		} else if !exists {
			shared.WriteError(w, http.StatusBadRequest, "Itinerary tidak valid.")
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
			shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return nil, false
		} else if !exists {
			shared.WriteError(w, http.StatusBadRequest, "Kategori tidak valid atau kategori tidak aktif.")
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
		shared.WriteError(w, http.StatusBadRequest, err.Error())
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
			shared.WriteError(w, http.StatusBadRequest, "Item termasuk tidak boleh berisi teks kosong.")
			return nil, false
		}
	}
	for _, item := range excludeItems {
		if strings.TrimSpace(item) == "" {
			shared.WriteError(w, http.StatusBadRequest, "Item tidak termasuk tidak boleh berisi teks kosong.")
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
			shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return nil, false
		}
		if !exists {
			shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("%s %q tidak terdaftar, pilih kode dari daftar bandara.", ap.nama, code))
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
			shared.WriteError(w, http.StatusBadRequest, f.Error())
			return nil, false
		}
	}

	// Gate 17: brosur_url dan brosur_thumb_url harus path unggahan atau URL http(s) (MP-08)
	if !shared.ValidMediaURL(req.BrosurURL) || !shared.ValidMediaURL(req.BrosurThumbURL) {
		shared.WriteError(w, http.StatusBadRequest, "Brosur harus berupa berkas unggahan atau URL http(s).")
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
			shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return nil, false
		}
		if taken {
			shared.WriteError(w, http.StatusConflict, "Paket dengan nama dan tanggal berangkat yang sama sudah ada di brand ini.")
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
		shared.WriteError(w, http.StatusBadRequest, le.Error())
	case errors.Is(err, ErrConflict):
		shared.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrNotFound):
		shared.WriteError(w, http.StatusNotFound, "Data tidak ditemukan.")
	default:
		if shared.IsFKDipakai(err) {
			shared.WriteError(w, http.StatusConflict, "Tidak bisa dihapus, masih dipakai oleh booking.")
			return
		}
		log.Printf("[ERROR] schedule: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
	}
}

const errBrandTerkunci = "Brand paket tidak dapat diubah"

const errPublishLampau ="paket dengan tanggal berangkat yang sudah lewat tidak dapat diterbitkan"

// ─── Shared helpers ───────────────────────────────────────────────────────────

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "ID tidak valid.")
		return 0, false
	}
	return id, true
}
