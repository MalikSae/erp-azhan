package schedule

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/shared"
)

// API publik v1 (SPEK-WEB-PUBLIK-CMS-2026-10-20.md §4.3): read-only dan stabil, dipakai situs publik
// dan CMS. Endpoint lama (/api/schedules...) tetap berjalan tanpa perubahan.

const (
	visibilityVisible = "visible"
	visibilityGone    = "gone"

	defaultPerPage = 50
	maxPerPage     = 200
)

// PackageV1 adalah satu paket pada API v1: field PublicSchedule yang sudah ada ditambah klasifikasi,
// durasi, harga mulai, dan status registrasi turunan.
type PackageV1 struct {
	*PublicSchedule
	Klasifikasi
	Visibility         string  `json:"visibility"`
	DurationDays       int     `json:"duration_days"`
	PriceFrom          float64 `json:"price_from"`
	RegistrationStatus string  `json:"registration_status"`
	// Lokasi: active Lokasi Dikunjungi in admin order; [] when none.
	Lokasi []LokasiPaket `json:"lokasi"`
}

// GonePackageV1 menggantikan detail paket yang tidak dijual lagi (diarsipkan, atau berangkat kurang dari
// 30 hari). Penerima memutuskan 301 ke hub relevan atau 410; klasifikasi disertakan untuk memilih hub.
type GonePackageV1 struct {
	ID            int64  `json:"id"`
	BrandID       int64  `json:"brand_id"`
	Name          string `json:"jadwal_nama"`
	Visibility    string `json:"visibility"`
	Reason        string `json:"reason"` // archived | closed
	DepartureDate string `json:"berangkat_tanggal"`
	Klasifikasi
}

// ListMetaV1 adalah metadata paginasi.
type ListMetaV1 struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	Total   int `json:"total"`
}

// ListPackagesV1Response adalah amplop daftar paket.
type ListPackagesV1Response struct {
	Data []PackageV1 `json:"data"`
	Meta ListMetaV1  `json:"meta"`
}

// SitemapEntryV1 adalah satu entri feed sitemap.
type SitemapEntryV1 struct {
	ID            int64     `json:"id"`
	UpdatedAt     time.Time `json:"updated_at"`
	DepartureDate string    `json:"berangkat_tanggal"`
	Klasifikasi
}

// filterV1 menampung filter daftar paket.
type filterV1 struct {
	Periode      string // YYYY-MM
	Tier         string // hemat|reguler|comfort|premium
	Durasi       int
	HotelBintang int
	HargaMaks    float64
	Registrasi   string
	UpdatedSince time.Time
	Page         int
	PerPage      int
}

var (
	tierValid       = map[string]bool{"hemat": true, "reguler": true, "comfort": true, "premium": true}
	registrasiValid = map[string]bool{RegOpen: true, RegAlmostFull: true, RegClosed: true, RegSoldOut: true}
)

// parseFilterV1 membaca dan memvalidasi query string daftar paket.
func parseFilterV1(q map[string][]string) (filterV1, error) {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	f := filterV1{Page: 1, PerPage: defaultPerPage}

	if v := get("periode"); v != "" {
		if _, err := time.Parse("2006-01", v); err != nil {
			return f, fmt.Errorf("periode harus berformat YYYY-MM")
		}
		f.Periode = v
	}
	if v := strings.ToLower(get("tier")); v != "" {
		if !tierValid[v] {
			return f, fmt.Errorf("tier harus salah satu dari hemat, reguler, comfort, premium")
		}
		f.Tier = v
	}
	if v := strings.ToUpper(get("registrasi")); v != "" {
		if !registrasiValid[v] {
			return f, fmt.Errorf("registrasi harus salah satu dari OPEN, ALMOST_FULL, CLOSED, SOLD_OUT")
		}
		f.Registrasi = v
	}
	if v := get("durasi"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return f, fmt.Errorf("durasi harus berupa angka hari lebih dari 0")
		}
		f.Durasi = n
	}
	if v := get("hotel_bintang"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 5 {
			return f, fmt.Errorf("hotel_bintang harus 1 sampai 5")
		}
		f.HotelBintang = n
	}
	// Spek §4.3 menamai filter ini harga_max; harga_maks tetap diterima untuk klien yang sudah memakainya.
	hargaMax := get("harga_max")
	if hargaMax == "" {
		hargaMax = get("harga_maks")
	}
	if v := hargaMax; v != "" {
		n, err := strconv.ParseFloat(v, 64)
		if err != nil || n <= 0 {
			return f, fmt.Errorf("harga_maks harus berupa angka lebih dari 0")
		}
		f.HargaMaks = n
	}
	if v := get("updated_since"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			t, err = time.Parse("2006-01-02", v)
		}
		if err != nil {
			return f, fmt.Errorf("updated_since harus berformat RFC3339 atau YYYY-MM-DD")
		}
		f.UpdatedSince = t
	}
	if v := get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return f, fmt.Errorf("page harus berupa angka mulai dari 1")
		}
		f.Page = n
	}
	if v := get("per_page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxPerPage {
			return f, fmt.Errorf("per_page harus 1 sampai %d", maxPerPage)
		}
		f.PerPage = n
	}
	return f, nil
}

// priceFrom adalah harga terendah di antara tipe kamar yang berharga.
func priceFrom(s *PublicSchedule) float64 {
	var min float64
	for _, v := range []float64{s.HargaQuad, s.HargaTriple, s.HargaDouble} {
		if v > 0 && (min == 0 || v < min) {
			min = v
		}
	}
	return min
}

// newPackageV1 membentuk PackageV1 dari paket yang terlihat publik.
func newPackageV1(s *PublicSchedule, k Klasifikasi, batas string) PackageV1 {
	return PackageV1{
		PublicSchedule:     s,
		Klasifikasi:        k,
		Visibility:         visibilityVisible,
		DurationDays:       durationDays(s.BerangkatTanggal, s.PulangTanggal),
		PriceFrom:          priceFrom(s),
		RegistrationStatus: statusRegistrasiPublik(s, batas),
		Lokasi:             []LokasiPaket{},
	}
}

// match menerapkan filter ke satu paket.
func (f filterV1) match(p PackageV1) bool {
	if f.Periode != "" && !strings.HasPrefix(p.BerangkatTanggal, f.Periode) {
		return false
	}
	if f.Tier != "" && (p.Tier == nil || p.Tier.SEO == nil || *p.Tier.SEO != f.Tier) {
		return false
	}
	if f.Durasi > 0 && p.DurationDays != f.Durasi {
		return false
	}
	if f.HotelBintang > 0 && p.HotelMekkah.StarRating < f.HotelBintang {
		return false
	}
	if f.HargaMaks > 0 && (p.PriceFrom <= 0 || p.PriceFrom > f.HargaMaks) {
		return false
	}
	if f.Registrasi != "" && p.RegistrationStatus != f.Registrasi {
		return false
	}
	if !f.UpdatedSince.IsZero() && p.UpdatedAt.Before(f.UpdatedSince) {
		return false
	}
	return true
}

// paginate memotong hasil sesuai halaman dan mengembalikan irisan serta total.
func paginate(items []PackageV1, page, perPage int) ([]PackageV1, int) {
	total := len(items)
	start := (page - 1) * perPage
	if start >= total {
		return []PackageV1{}, total
	}
	end := start + perPage
	if end > total {
		end = total
	}
	return items[start:end], total
}

func brandParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	raw := strings.TrimSpace(r.URL.Query().Get("brand"))
	if raw == "" {
		shared.WriteError(w, http.StatusBadRequest, "Parameter brand wajib diisi.")
		return 0, false
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Parameter brand tidak valid.")
		return 0, false
	}
	return id, true
}

// visiblePackages mengambil paket yang sedang dijual untuk satu brand beserta klasifikasinya.
func (h *Handler) visiblePackages(ctx context.Context, brandID int64) ([]PackageV1, error) {
	items, err := h.repo.ListPublic(ctx, brandID)
	if err != nil {
		return nil, err
	}
	ids := make([]int64, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	kl, err := h.repo.loadKlasifikasi(ctx, ids)
	if err != nil {
		return nil, err
	}
	// All pakets' locations in one query; a failure is an error, like the detail endpoint.
	lokasi, err := h.repo.LokasiPublik(ctx, ids)
	if err != nil {
		return nil, err
	}
	batas := batasPublikWIB()
	out := make([]PackageV1, 0, len(items))
	for _, it := range items {
		k, ok := kl[it.ID]
		if !ok {
			k = Klasifikasi{JourneyType: JourneyUmroh}
		}
		p := newPackageV1(it, k, batas)
		if l := lokasi[it.ID]; l != nil {
			p.Lokasi = l
		}
		out = append(out, p)
	}
	return out, nil
}

// ListPackagesV1 godoc
// GET /api/public/v1/packages?brand=ID
// Filter: periode (YYYY-MM), tier, durasi, hotel_bintang, harga_maks, registrasi,
// updated_since, page, per_page. Hanya paket published yang berangkat 30 hari atau lebih dari hari ini.
func (h *Handler) ListPackagesV1(w http.ResponseWriter, r *http.Request) {
	brandID, ok := brandParam(w, r)
	if !ok {
		return
	}
	f, err := parseFilterV1(r.URL.Query())
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	all, err := h.visiblePackages(r.Context(), brandID)
	if err != nil {
		log.Printf("[ERROR] schedule v1: list: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data paket.")
		return
	}
	filtered := make([]PackageV1, 0, len(all))
	for _, p := range all {
		if f.match(p) {
			filtered = append(filtered, p)
		}
	}
	page, total := paginate(filtered, f.Page, f.PerPage)
	shared.WriteJSON(w, http.StatusOK, ListPackagesV1Response{
		Data: page,
		Meta: ListMetaV1{Page: f.Page, PerPage: f.PerPage, Total: total},
	})
}

// GetPackageV1 godoc
// GET /api/public/v1/packages/{id}?brand=ID
// 200 dengan visibility "visible" untuk paket yang dijual; 200 dengan visibility "gone" untuk paket
// yang diarsipkan atau berangkat kurang dari 30 hari; 404 untuk draft atau paket tidak dikenal.
func (h *Handler) GetPackageV1(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	brandID, ok := brandParam(w, r)
	if !ok {
		return
	}
	s, err := h.repo.GetPublicByID(r.Context(), id, &brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	kl, err := h.repo.loadKlasifikasi(r.Context(), []int64{s.ID})
	if err != nil {
		log.Printf("[ERROR] schedule v1: klasifikasi: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	k, found := kl[s.ID]
	if !found {
		k = Klasifikasi{JourneyType: JourneyUmroh}
	}

	batas := batasPublikWIB()
	switch {
	case s.Status == StatusArchived:
		shared.WriteJSON(w, http.StatusOK, gonePackage(s, k, "archived"))
	case s.Status == StatusPublished && s.BerangkatTanggal < batas:
		shared.WriteJSON(w, http.StatusOK, gonePackage(s, k, "closed"))
	case s.Status == StatusPublished:
		p := newPackageV1(s.ToPublic(), k, batas)
		lokasi, err := h.repo.LokasiPublik(r.Context(), []int64{s.ID})
		if err != nil {
			log.Printf("[ERROR] schedule v1: lokasi: %v", err)
			shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return
		}
		if l := lokasi[s.ID]; l != nil {
			p.Lokasi = l
		}
		shared.WriteJSON(w, http.StatusOK, p)
	default:
		shared.WriteError(w, http.StatusNotFound, "Jadwal tidak ditemukan atau belum dipublikasikan.")
	}
}

func gonePackage(s *Schedule, k Klasifikasi, reason string) GonePackageV1 {
	return GonePackageV1{
		ID:            s.ID,
		BrandID:       s.BrandID,
		Name:          s.JadwalNama,
		Visibility:    visibilityGone,
		Reason:        reason,
		DepartureDate: s.BerangkatTanggal,
		Klasifikasi:   k,
	}
}

// SitemapFeedV1 godoc
// GET /api/public/v1/sitemap-feed?brand=ID
// Paket yang layak masuk sitemap (dijual saat ini) beserta waktu perubahan terakhir.
func (h *Handler) SitemapFeedV1(w http.ResponseWriter, r *http.Request) {
	brandID, ok := brandParam(w, r)
	if !ok {
		return
	}
	all, err := h.visiblePackages(r.Context(), brandID)
	if err != nil {
		log.Printf("[ERROR] schedule v1: sitemap: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data paket.")
		return
	}
	out := make([]SitemapEntryV1, 0, len(all))
	for _, p := range all {
		out = append(out, SitemapEntryV1{
			ID:            p.ID,
			UpdatedAt:     p.UpdatedAt,
			DepartureDate: p.BerangkatTanggal,
			Klasifikasi:   p.Klasifikasi,
		})
	}
	shared.WriteJSON(w, http.StatusOK, out)
}
