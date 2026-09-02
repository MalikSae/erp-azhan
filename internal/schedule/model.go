package schedule

import (
	"strings"
	"time"
)

// CategoryRef menyimpan data relasi kategori paket.
type CategoryRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

// MaskapaiRef menyimpan data relasi maskapai.
type MaskapaiRef struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
}

// HotelRef menyimpan data relasi hotel.
type HotelRef struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	StarRating int     `json:"star_rating"`
	DistanceM  *int    `json:"distance_m"`
	PhotoURL   *string `json:"photo_url"`
}

// TransitHotel menyimpan hotel transit beserta urutannya.
type TransitHotel struct {
	HotelID    int64   `json:"hotel_id"`
	Urutan     int     `json:"urutan"`
	Nama       string  `json:"nama"`
	Kota       string  `json:"kota"`
	StarRating int     `json:"star_rating"`
	PhotoURL   *string `json:"photo_url"`
}

// AddOnRef menyimpan data add-on dari tabel relasi schedule_add_ons.
type AddOnRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Schedule merepresentasikan baris lengkap tabel schedules di database.
type Schedule struct {
	ID                       int64          `json:"id"`
	BrandID                  int64          `json:"brand_id"`
	CategoryID               *int64         `json:"category_id"`
	Category                 *CategoryRef   `json:"category"`
	JadwalNama               string         `json:"jadwal_nama"`
	Status                   string         `json:"status"` // draft, published, archived
	IsPromo                  bool           `json:"is_promo"`
	Views                    int            `json:"views"`
	PromoUntil               *time.Time     `json:"promo_until"`
	IsTicketConfirmed        bool           `json:"is_ticket_confirmed"`
	IsDirectFlight           bool           `json:"is_direct_flight"`
	SeatTotal                int            `json:"seat_total"`
	SeatSisa                 int            `json:"seat_sisa"`
	BookingCount             int            `json:"booking_count"`
	MaskapaiID               *int64         `json:"maskapai_id"`
	Maskapai                 *MaskapaiRef   `json:"maskapai"`
	BerangkatTanggal         string         `json:"berangkat_tanggal"`
	BerangkatJam             string         `json:"berangkat_jam"`
	BerangkatKodePenerbangan string         `json:"berangkat_kode_penerbangan"`
	BerangkatBandaraAsal     string         `json:"berangkat_bandara_asal"`
	BerangkatBandaraTujuan   string         `json:"berangkat_bandara_tujuan"`
	PulangTanggal            string         `json:"pulang_tanggal"`
	PulangJam                string         `json:"pulang_jam"`
	PulangKodePenerbangan    string         `json:"pulang_kode_penerbangan"`
	PulangBandaraAsal        string         `json:"pulang_bandara_asal"`
	PulangBandaraTujuan      string         `json:"pulang_bandara_tujuan"`
	TransitBandara           string         `json:"transit_bandara"`
	HotelMekkahID            *int64         `json:"hotel_mekkah_id"`
	HotelMekkah              HotelRef       `json:"hotel_mekkah"`
	HotelMadinahID           *int64         `json:"hotel_madinah_id"`
	HotelMadinah             HotelRef       `json:"hotel_madinah"`
	TransitHotels            []TransitHotel `json:"transit_hotels"`
	HargaQuad                float64        `json:"harga_quad"`
	HargaTriple              float64        `json:"harga_triple"`
	HargaDouble              float64        `json:"harga_double"`
	HargaInfant              *float64       `json:"harga_infant"`
	HargaCoret               *float64       `json:"harga_coret"`
	MinimalDP                *float64       `json:"minimal_dp"`
	ItineraryID              *int64         `json:"itinerary_id"`
	IncludeItems             []string       `json:"include_items"`
	ExcludeItems             []string       `json:"exclude_items"`
	AddOns                   []AddOnRef     `json:"add_ons"`
	BrosurURL                string         `json:"brosur_url"`
	BrosurThumbURL           string         `json:"brosur_thumb_url"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
}

// PublicSchedule adalah response publik — field status TIDAK ditampilkan.
type PublicSchedule struct {
	ID                       int64          `json:"id"`
	BrandID                  int64          `json:"brand_id"`
	CategoryID               *int64         `json:"category_id"`
	Category                 *CategoryRef   `json:"category"`
	JadwalNama               string         `json:"jadwal_nama"`
	IsPromo                  bool           `json:"is_promo"`
	Views                    int            `json:"views"`
	PromoUntil               *time.Time     `json:"promo_until"`
	IsTicketConfirmed        bool           `json:"is_ticket_confirmed"`
	IsDirectFlight           bool           `json:"is_direct_flight"`
	SeatTotal                int            `json:"seat_total"`
	SeatSisa                 int            `json:"seat_sisa"`
	Maskapai                 *MaskapaiRef   `json:"maskapai"`
	BerangkatTanggal         string         `json:"berangkat_tanggal"`
	BerangkatJam             string         `json:"berangkat_jam"`
	BerangkatKodePenerbangan string         `json:"berangkat_kode_penerbangan"`
	BerangkatBandaraAsal     string         `json:"berangkat_bandara_asal"`
	BerangkatBandaraTujuan   string         `json:"berangkat_bandara_tujuan"`
	PulangTanggal            string         `json:"pulang_tanggal"`
	PulangJam                string         `json:"pulang_jam"`
	PulangKodePenerbangan    string         `json:"pulang_kode_penerbangan"`
	PulangBandaraAsal        string         `json:"pulang_bandara_asal"`
	PulangBandaraTujuan      string         `json:"pulang_bandara_tujuan"`
	TransitBandara           string         `json:"transit_bandara"`
	HotelMekkah              HotelRef       `json:"hotel_mekkah"`
	HotelMadinah             HotelRef       `json:"hotel_madinah"`
	TransitHotels            []TransitHotel `json:"transit_hotels"`
	HargaQuad                float64        `json:"harga_quad"`
	HargaTriple              float64        `json:"harga_triple"`
	HargaDouble              float64        `json:"harga_double"`
	HargaInfant              *float64       `json:"harga_infant"`
	HargaCoret               *float64       `json:"harga_coret"`
	MinimalDP                *float64       `json:"minimal_dp"`
	ItineraryID              *int64         `json:"itinerary_id"`
	IncludeItems             []string       `json:"include_items"`
	ExcludeItems             []string       `json:"exclude_items"`
	AddOns                   []AddOnRef     `json:"add_ons"`
	BrosurURL                string         `json:"brosur_url"`
	BrosurThumbURL           string         `json:"brosur_thumb_url"`
	CreatedAt                time.Time      `json:"created_at"`
	UpdatedAt                time.Time      `json:"updated_at"`
}

// ToPublic mengonversi Schedule admin ke PublicSchedule (tanpa field status).
// Otomatis menonaktifkan status promo & harga coret jika promo_until sudah terlewat.
func (s *Schedule) ToPublic() *PublicSchedule {
	isPromo := s.IsPromo
	hargaCoret := s.HargaCoret

	// Jika memiliki batas promo dan tanggal batas promo sudah lewat, non-aktifkan promo untuk publik
	if isPromo && s.PromoUntil != nil {
		today := time.Now().Truncate(24 * time.Hour)
		promoEnd := s.PromoUntil.Truncate(24 * time.Hour)
		if promoEnd.Before(today) {
			isPromo = false
			hargaCoret = nil
		}
	}

	return &PublicSchedule{
		ID:                       s.ID,
		BrandID:                  s.BrandID,
		CategoryID:               s.CategoryID,
		Category:                 s.Category,
		JadwalNama:               s.JadwalNama,
		IsPromo:                  isPromo,
		Views:                    s.Views,
		PromoUntil:               s.PromoUntil,
		IsTicketConfirmed:        s.IsTicketConfirmed,
		IsDirectFlight:           s.IsDirectFlight,
		SeatTotal:                s.SeatTotal,
		SeatSisa:                 s.SeatSisa,
		Maskapai:                 s.Maskapai,
		BerangkatTanggal:         s.BerangkatTanggal,
		BerangkatJam:             s.BerangkatJam,
		BerangkatKodePenerbangan: s.BerangkatKodePenerbangan,
		BerangkatBandaraAsal:     s.BerangkatBandaraAsal,
		BerangkatBandaraTujuan:   s.BerangkatBandaraTujuan,
		PulangTanggal:            s.PulangTanggal,
		PulangJam:                s.PulangJam,
		PulangKodePenerbangan:    s.PulangKodePenerbangan,
		PulangBandaraAsal:        s.PulangBandaraAsal,
		PulangBandaraTujuan:      s.PulangBandaraTujuan,
		TransitBandara:           s.TransitBandara,
		HotelMekkah:              s.HotelMekkah,
		HotelMadinah:             s.HotelMadinah,
		TransitHotels:            s.TransitHotels,
		HargaQuad:                s.HargaQuad,
		HargaTriple:              s.HargaTriple,
		HargaDouble:              s.HargaDouble,
		HargaInfant:              s.HargaInfant,
		HargaCoret:               hargaCoret,
		MinimalDP:                s.MinimalDP,
		ItineraryID:              s.ItineraryID,
		IncludeItems:             s.IncludeItems,
		ExcludeItems:             s.ExcludeItems,
		AddOns:                   s.AddOns,
		BrosurURL:                s.BrosurURL,
		BrosurThumbURL:           s.BrosurThumbURL,
		CreatedAt:                s.CreatedAt,
		UpdatedAt:                s.UpdatedAt,
	}
}

// ScheduleListItem dipakai di ListSchedulesAdmin — ringkas.
type ScheduleListItem struct {
	ID                int64          `json:"id"`
	BrandID           int64          `json:"brand_id"`
	CategoryID        *int64         `json:"category_id"`
	Category          *CategoryRef   `json:"category"`
	JadwalNama        string         `json:"jadwal_nama"`
	Status            string         `json:"status"`
	IsPromo           bool           `json:"is_promo"`
	Views             int            `json:"views"`
	PromoUntil        *time.Time     `json:"promo_until"`
	IsTicketConfirmed bool           `json:"is_ticket_confirmed"`
	IsDirectFlight    bool           `json:"is_direct_flight"`
	Maskapai          *MaskapaiRef   `json:"maskapai"`
	BerangkatTanggal  string         `json:"berangkat_tanggal"`
	SeatTotal         int            `json:"seat_total"`
	SeatSisa          int            `json:"seat_sisa"`
	BookingCount      int            `json:"booking_count"`
	TransitHotels     []TransitHotel `json:"transit_hotels"`
	HargaQuad         float64        `json:"harga_quad"`
	HargaTriple       float64        `json:"harga_triple"`
	HargaDouble       float64        `json:"harga_double"`
	HargaInfant       *float64       `json:"harga_infant"`
	AddOns            []AddOnRef     `json:"add_ons"`
}

// ─── Internal input struct (setelah validasi) ────────────────────────────────

// ScheduleInput adalah parameter yang sudah divalidasi, diteruskan ke repository.
type ScheduleInput struct {
	BrandID                  int64
	CategoryID               *int64
	JadwalNama               string
	Status                   string // draft, published, archived
	IsPromo                  bool
	PromoUntil               *time.Time
	IsTicketConfirmed        bool
	IsDirectFlight           bool
	SeatTotal                int
	SeatSisa                 int // sudah di-resolve (absent = SeatTotal)
	MaskapaiID               int64
	BerangkatTanggal         string
	BerangkatJam             string
	BerangkatKodePenerbangan string
	BerangkatBandaraAsal     string
	BerangkatBandaraTujuan   string
	PulangTanggal            string
	PulangJam                string
	PulangKodePenerbangan    string
	PulangBandaraAsal        string
	PulangBandaraTujuan      string
	TransitBandara           string
	HotelMekkahID            int64
	HotelMadinahID           int64
	TransitHotelIDs          []int64
	HargaQuad                float64
	HargaTriple              float64
	HargaDouble              float64
	HargaInfant              *float64
	HargaCoret               *float64
	MinimalDP                *float64
	ItineraryID              *int64
	IncludeItems             []string
	ExcludeItems             []string
	AddOnIDs                 []int64
	BrosurURL                string
	BrosurThumbURL           string
}

// FlexibleTime menangani parsing tanggal/waktu yang fleksibel (RFC3339, YYYY-MM-DD, string kosong, null).
type FlexibleTime struct {
	Time  time.Time
	Valid bool
}

func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), "\"")
	if s == "" || s == "null" {
		ft.Valid = false
		return nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			ft.Time = t
			ft.Valid = true
			return nil
		}
	}
	ft.Valid = false
	return nil
}

// ─── Request structs ──────────────────────────────────────────────────────────

// CreateScheduleRequest adalah payload untuk POST /api/admin/schedules.
// SeatSisa pakai pointer untuk membedakan "tidak dikirim" (nil) vs "dikirim sebagai 0".
type CreateScheduleRequest struct {
	BrandID                  *int64        `json:"brand_id"` // Opsional bagi admin biasa, Wajib bagi Super Admin
	CategoryID               *int64        `json:"category_id"`
	JadwalNama               string        `json:"jadwal_nama"`
	Status                   string        `json:"status"`
	IsPromo                  bool          `json:"is_promo"`
	PromoUntil               *FlexibleTime `json:"promo_until"`
	IsTicketConfirmed        bool       `json:"is_ticket_confirmed"`
	IsDirectFlight           bool       `json:"is_direct_flight"`
	SeatTotal                int        `json:"seat_total"`
	SeatSisa                 *int       `json:"seat_sisa"`
	MaskapaiID               int64      `json:"maskapai_id"`
	BerangkatTanggal         string     `json:"berangkat_tanggal"`
	BerangkatJam             string     `json:"berangkat_jam"`
	BerangkatKodePenerbangan string     `json:"berangkat_kode_penerbangan"`
	BerangkatBandaraAsal     string     `json:"berangkat_bandara_asal"`
	BerangkatBandaraTujuan   string     `json:"berangkat_bandara_tujuan"`
	PulangTanggal            string     `json:"pulang_tanggal"`
	PulangJam                string     `json:"pulang_jam"`
	PulangKodePenerbangan    string     `json:"pulang_kode_penerbangan"`
	PulangBandaraAsal        string     `json:"pulang_bandara_asal"`
	PulangBandaraTujuan      string     `json:"pulang_bandara_tujuan"`
	TransitBandara           string     `json:"transit_bandara"`
	HotelMekkahID            int64      `json:"hotel_mekkah_id"`
	HotelMadinahID           int64      `json:"hotel_madinah_id"`
	TransitHotelIDs          []int64    `json:"transit_hotel_ids"`
	HargaQuad                float64    `json:"harga_quad"`
	HargaTriple              float64    `json:"harga_triple"`
	HargaDouble              float64    `json:"harga_double"`
	HargaInfant              *float64   `json:"harga_infant"`
	HargaCoret               *float64   `json:"harga_coret"`
	MinimalDP                *float64   `json:"minimal_dp"`
	ItineraryID              *int64     `json:"itinerary_id"`
	IncludeItems             []string   `json:"include_items"`
	ExcludeItems             []string   `json:"exclude_items"`
	AddOnIDs                 []int64    `json:"add_on_ids"`
	BrosurURL                string     `json:"brosur_url"`
	BrosurThumbURL           string     `json:"brosur_thumb_url"`
}

// UpdateScheduleRequest memiliki struktur yang sama persis dengan Create.
type UpdateScheduleRequest = CreateScheduleRequest

// UpdateStatusRequest adalah payload untuk PUT /api/admin/schedules/{id}/status.
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateSeatRequest adalah payload untuk PUT /api/admin/schedules/{id}/seat.
type UpdateSeatRequest struct {
	SeatSisa int `json:"seat_sisa"`
}