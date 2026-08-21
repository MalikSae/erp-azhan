package schedule

import "time"

// ─── Embedded refs ────────────────────────────────────────────────────────────

// MaskapaiRef adalah subset airlines untuk embed di response Schedule.
type MaskapaiRef struct {
	ID      int64   `json:"id"`
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
}

// HotelRef adalah subset hotels untuk embed di response Schedule.
type HotelRef struct {
	ID         int64  `json:"id"`
	Name       string `json:"name"`
	StarRating int    `json:"star_rating"`
	DistanceM  *int   `json:"distance_m"`
}

// AddOnRef merepresentasikan add-on yang dipasang ke schedule.
type AddOnRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// ─── Response structs ─────────────────────────────────────────────────────────

// Schedule adalah response lengkap admin — termasuk field status.
type Schedule struct {
	ID                       int64        `json:"id"`
	BrandID                  int64        `json:"brand_id"`
	JadwalNama               string       `json:"jadwal_nama"`
	Status                   string       `json:"status"`
	IsPromo                  bool         `json:"is_promo"`
	IsTicketConfirmed        bool         `json:"is_ticket_confirmed"`
	IsDirectFlight           bool         `json:"is_direct_flight"`
	SeatTotal                int          `json:"seat_total"`
	SeatSisa                 int          `json:"seat_sisa"`
	Maskapai                 *MaskapaiRef `json:"maskapai"`
	BerangkatTanggal         string       `json:"berangkat_tanggal"`
	BerangkatJam             string       `json:"berangkat_jam"`
	BerangkatKodePenerbangan string       `json:"berangkat_kode_penerbangan"`
	BerangkatBandaraAsal     string       `json:"berangkat_bandara_asal"`
	BerangkatBandaraTujuan   string       `json:"berangkat_bandara_tujuan"`
	PulangTanggal            string       `json:"pulang_tanggal"`
	PulangJam                string       `json:"pulang_jam"`
	PulangKodePenerbangan    string       `json:"pulang_kode_penerbangan"`
	PulangBandaraAsal        string       `json:"pulang_bandara_asal"`
	PulangBandaraTujuan      string       `json:"pulang_bandara_tujuan"`
	TransitBandara           string       `json:"transit_bandara"`
	HotelMekkah              HotelRef     `json:"hotel_mekkah"`
	HotelMadinah             HotelRef     `json:"hotel_madinah"`
	HargaQuad                float64      `json:"harga_quad"`
	HargaTriple              float64      `json:"harga_triple"`
	HargaDouble              float64      `json:"harga_double"`
	HargaCoret               *float64     `json:"harga_coret"`
	ItineraryID              *int64       `json:"itinerary_id"`
	IncludeItems             []string     `json:"include_items"`
	ExcludeItems             []string     `json:"exclude_items"`
	AddOns                   []AddOnRef   `json:"add_ons"`
	BrosurURL                string       `json:"brosur_url"`
	BrosurThumbURL           string       `json:"brosur_thumb_url"`
	CreatedAt                time.Time    `json:"created_at"`
	UpdatedAt                time.Time    `json:"updated_at"`
}

// PublicSchedule adalah response publik — field status TIDAK ditampilkan.
type PublicSchedule struct {
	ID                       int64        `json:"id"`
	BrandID                  int64        `json:"brand_id"`
	JadwalNama               string       `json:"jadwal_nama"`
	IsPromo                  bool         `json:"is_promo"`
	IsTicketConfirmed        bool         `json:"is_ticket_confirmed"`
	IsDirectFlight           bool         `json:"is_direct_flight"`
	SeatTotal                int          `json:"seat_total"`
	SeatSisa                 int          `json:"seat_sisa"`
	Maskapai                 *MaskapaiRef `json:"maskapai"`
	BerangkatTanggal         string       `json:"berangkat_tanggal"`
	BerangkatJam             string       `json:"berangkat_jam"`
	BerangkatKodePenerbangan string       `json:"berangkat_kode_penerbangan"`
	BerangkatBandaraAsal     string       `json:"berangkat_bandara_asal"`
	BerangkatBandaraTujuan   string       `json:"berangkat_bandara_tujuan"`
	PulangTanggal            string       `json:"pulang_tanggal"`
	PulangJam                string       `json:"pulang_jam"`
	PulangKodePenerbangan    string       `json:"pulang_kode_penerbangan"`
	PulangBandaraAsal        string       `json:"pulang_bandara_asal"`
	PulangBandaraTujuan      string       `json:"pulang_bandara_tujuan"`
	TransitBandara           string       `json:"transit_bandara"`
	HotelMekkah              HotelRef     `json:"hotel_mekkah"`
	HotelMadinah             HotelRef     `json:"hotel_madinah"`
	HargaQuad                float64      `json:"harga_quad"`
	HargaTriple              float64      `json:"harga_triple"`
	HargaDouble              float64      `json:"harga_double"`
	HargaCoret               *float64     `json:"harga_coret"`
	ItineraryID              *int64       `json:"itinerary_id"`
	IncludeItems             []string     `json:"include_items"`
	ExcludeItems             []string     `json:"exclude_items"`
	AddOns                   []AddOnRef   `json:"add_ons"`
	BrosurURL                string       `json:"brosur_url"`
	BrosurThumbURL           string       `json:"brosur_thumb_url"`
	CreatedAt                time.Time    `json:"created_at"`
	UpdatedAt                time.Time    `json:"updated_at"`
}

// ToPublic mengonversi Schedule admin ke PublicSchedule (tanpa field status).
func (s *Schedule) ToPublic() *PublicSchedule {
	return &PublicSchedule{
		ID:                       s.ID,
		BrandID:                  s.BrandID,
		JadwalNama:               s.JadwalNama,
		IsPromo:                  s.IsPromo,
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
		HargaQuad:                s.HargaQuad,
		HargaTriple:              s.HargaTriple,
		HargaDouble:              s.HargaDouble,
		HargaCoret:               s.HargaCoret,
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
	ID                int64        `json:"id"`
	BrandID           int64        `json:"brand_id"`
	JadwalNama        string       `json:"jadwal_nama"`
	Status            string       `json:"status"`
	IsPromo           bool         `json:"is_promo"`
	IsTicketConfirmed bool         `json:"is_ticket_confirmed"`
	IsDirectFlight    bool         `json:"is_direct_flight"`
	Maskapai          *MaskapaiRef `json:"maskapai"`
	BerangkatTanggal  string       `json:"berangkat_tanggal"`
	SeatTotal         int          `json:"seat_total"`
	SeatSisa          int          `json:"seat_sisa"`
	HargaQuad         float64      `json:"harga_quad"`
	HargaTriple       float64      `json:"harga_triple"`
	HargaDouble       float64      `json:"harga_double"`
	AddOns            []AddOnRef   `json:"add_ons"`
}

// ─── Internal input struct (setelah validasi) ─────────────────────────────────

// ScheduleInput adalah parameter yang sudah divalidasi, diteruskan ke repository.
type ScheduleInput struct {
	BrandID                  int64
	JadwalNama               string
	Status                   string // draft, published, archived
	IsPromo                  bool
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
	HargaQuad                float64
	HargaTriple              float64
	HargaDouble              float64
	HargaCoret               *float64
	ItineraryID              *int64
	IncludeItems             []string
	ExcludeItems             []string
	AddOnIDs                 []int64
	BrosurURL                string
	BrosurThumbURL           string
}

// ─── Request structs ──────────────────────────────────────────────────────────

// CreateScheduleRequest adalah payload untuk POST /api/admin/schedules.
// SeatSisa pakai pointer untuk membedakan "tidak dikirim" (nil) vs "dikirim sebagai 0".
type CreateScheduleRequest struct {
	BrandID                  *int64   `json:"brand_id"` // Opsional bagi admin biasa, Wajib bagi Super Admin
	JadwalNama               string   `json:"jadwal_nama"`
	Status                   string   `json:"status"`
	IsPromo                  bool     `json:"is_promo"`
	IsTicketConfirmed        bool     `json:"is_ticket_confirmed"`
	IsDirectFlight           bool     `json:"is_direct_flight"`
	SeatTotal                int      `json:"seat_total"`
	SeatSisa                 *int     `json:"seat_sisa"`
	MaskapaiID               int64    `json:"maskapai_id"`
	BerangkatTanggal         string   `json:"berangkat_tanggal"`
	BerangkatJam             string   `json:"berangkat_jam"`
	BerangkatKodePenerbangan string   `json:"berangkat_kode_penerbangan"`
	BerangkatBandaraAsal     string   `json:"berangkat_bandara_asal"`
	BerangkatBandaraTujuan   string   `json:"berangkat_bandara_tujuan"`
	PulangTanggal            string   `json:"pulang_tanggal"`
	PulangJam                string   `json:"pulang_jam"`
	PulangKodePenerbangan    string   `json:"pulang_kode_penerbangan"`
	PulangBandaraAsal        string   `json:"pulang_bandara_asal"`
	PulangBandaraTujuan      string   `json:"pulang_bandara_tujuan"`
	TransitBandara           string   `json:"transit_bandara"`
	HotelMekkahID            int64    `json:"hotel_mekkah_id"`
	HotelMadinahID           int64    `json:"hotel_madinah_id"`
	HargaQuad                float64  `json:"harga_quad"`
	HargaTriple              float64  `json:"harga_triple"`
	HargaDouble              float64  `json:"harga_double"`
	HargaCoret               *float64 `json:"harga_coret"`
	ItineraryID              *int64   `json:"itinerary_id"`
	IncludeItems             []string `json:"include_items"`
	ExcludeItems             []string `json:"exclude_items"`
	AddOnIDs                 []int64  `json:"add_on_ids"`
	BrosurURL                string   `json:"brosur_url"`
	BrosurThumbURL           string   `json:"brosur_thumb_url"`
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
