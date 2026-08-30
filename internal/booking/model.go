package booking

import "time"

// ─── Response structs ─────────────────────────────────────────────────────────

// BookingAddon adalah detail biaya tambahan pada booking.
type BookingAddon struct {
	ID        int64     `json:"id"`
	BookingID int64     `json:"booking_id"`
	Nama      string    `json:"nama"`
	Nominal   float64   `json:"nominal"`
	CreatedAt time.Time `json:"created_at"`
}

// BookingPax adalah detail penumpang (pax) pada booking.
type BookingPax struct {
	ID                       int64     `json:"id"`
	BookingID                int64     `json:"booking_id"`
	JamaahID                 int64     `json:"jamaah_id"`
	NamaJamaah               string    `json:"nama_jamaah,omitempty"`
	PaxType                  string    `json:"pax_type"`
	RoomType                 *string   `json:"room_type"`
	HargaPax                 float64   `json:"harga_pax"`
	CountsForSeat            bool      `json:"counts_for_seat"`
	PaxStatus                string    `json:"pax_status"`
	ProgressPaspor           bool      `json:"progress_paspor"`
	ProgressVisa             bool      `json:"progress_visa"`
	ProgressSiskopatuh       bool      `json:"progress_siskopatuh"`
	ProgressManasik          bool      `json:"progress_manasik"`
	ProgressVaksinMeningitis bool      `json:"progress_vaksin_meningitis"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// Booking adalah response lengkap admin — JOIN nama jamaah + jadwal_nama schedule + primary pax.
type Booking struct {
	ID                int64    `json:"id"`
	IDBooking         *string  `json:"id_booking"`
	ScheduleID        int64    `json:"schedule_id"`
	BrandID           int64    `json:"brand_id"`
	JadwalNama        string   `json:"jadwal_nama"`
	BerangkatTanggal  *string  `json:"berangkat_tanggal"`
	PicJamaahID       *int64   `json:"pic_jamaah_id"`
	JamaahID          *int64   `json:"jamaah_id"` // alias untuk kompatibilitas endpoint/frontend
	NamaJamaah        *string  `json:"nama_jamaah"`
	RoomType          *string  `json:"room_type"`         // dari primary pax
	PaxCount          int      `json:"pax_count"`         // jumlah pax aktif total
	RegularPaxCount   int      `json:"regular_pax_count"` // jumlah pax reguler
	InfantPaxCount    int      `json:"infant_pax_count"`  // jumlah pax infant
	SeatCount         int      `json:"seat_count"`        // jumlah kursi yang dicadangkan
	HargaDasar        *float64 `json:"harga_dasar"`       // dari primary pax
	Status            string   `json:"status"`
	IsSeatBlocked     bool     `json:"is_seat_blocked"`
	SeatHoldExpiresAt *string  `json:"seat_hold_expires_at,omitempty"`
	TotalHarga        *float64 `json:"total_harga"`
	Diskon            float64  `json:"diskon"`
	DiskonKeterangan  *string  `json:"diskon_keterangan"`
	// ProgressPaspor dihitung dinamis dari dokumen_jamaah
	ProgressPaspor bool `json:"progress_paspor"`
	ProgressVisa   bool `json:"progress_visa"`
	// ProgressTiket dihitung dinamis dari schedules.is_ticket_confirmed
	ProgressTiket             bool           `json:"progress_tiket"`
	ProgressHotel             bool           `json:"progress_hotel"`
	ProgressLandArrangement   bool           `json:"progress_land_arrangement"`
	ProgressManasik           bool           `json:"progress_manasik"`
	ProgressSiskopatuh        bool           `json:"progress_siskopatuh"`
	ProgressVaksinMeningitis  bool           `json:"progress_vaksin_meningitis"`
	SiapBerangkat             bool           `json:"siap_berangkat"`
	PerlengkapanStatus        string         `json:"perlengkapan_status"`
	PerlengkapanTanggal       *string        `json:"perlengkapan_tanggal"`
	PerlengkapanDiberikanOleh *int64         `json:"perlengkapan_diberikan_oleh"`
	PerlengkapanJumlahPax     *int           `json:"perlengkapan_jumlah_pax"`
	Pax                       []BookingPax   `json:"pax"`
	Addons                    []BookingAddon `json:"addons"`
	CreatedBy                 *int64         `json:"created_by"`
	CreatedAt                 time.Time      `json:"created_at"`
}

// ─── Request structs ──────────────────────────────────────────────────────────

// CreateBookingPaxItem adalah item pax dalam request pembuatan booking.
type CreateBookingPaxItem struct {
	JamaahID int64   `json:"jamaah_id"`
	PaxType  string  `json:"pax_type"`  // "reguler" | "infant"
	RoomType *string `json:"room_type"` // "Quad" | "Triple" | "Double" | null
}

// CreateBookingRequest adalah payload untuk POST /api/admin/bookings.
type CreateBookingRequest struct {
	ScheduleID  int64                  `json:"schedule_id"`
	PicJamaahID int64                  `json:"pic_jamaah_id"`
	JamaahID    int64                  `json:"jamaah_id,omitempty"` // fallback backward-compatibility
	Pax         []CreateBookingPaxItem `json:"pax"`
}

// CreateDraftBookingRequest adalah payload untuk POST dan PUT /api/admin/bookings/draft.
type CreateDraftBookingRequest struct {
	ScheduleID  int64                  `json:"schedule_id"`
	PicJamaahID *int64                 `json:"pic_jamaah_id"`
	Pax         []CreateBookingPaxItem `json:"pax"`
}

// UpdateBookingStatusRequest adalah payload untuk PUT /api/admin/bookings/{id}/status.
type UpdateBookingStatusRequest struct {
	Status string `json:"status"`
}

type SeatBlockRequest struct {
	ExpiresAt string `json:"expires_at"`
}

// UpdatePaxRoomTypeRequest adalah payload untuk PUT /api/admin/bookings/{id}/pax/{pax_id}/room-type.
type UpdatePaxRoomTypeRequest struct {
	RoomType string `json:"room_type"`
}

// AddBookingAddonRequest adalah payload untuk POST /api/admin/bookings/{id}/addons.
type AddBookingAddonRequest struct {
	Nama    string  `json:"nama"`
	Nominal float64 `json:"nominal"`
}

// UpdateDiskonRequest adalah payload untuk PUT /api/admin/bookings/{id}/diskon.
type UpdateDiskonRequest struct {
	Diskon           float64 `json:"diskon"`
	DiskonKeterangan *string `json:"diskon_keterangan"`
}
