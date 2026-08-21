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

// Booking adalah response lengkap admin — JOIN nama jamaah + jadwal_nama schedule.
type Booking struct {
	ID               int64    `json:"id"`
	IDBooking        string   `json:"id_booking"`
	ScheduleID       int64    `json:"schedule_id"`
	JadwalNama       string   `json:"jadwal_nama"`
	BerangkatTanggal *string  `json:"berangkat_tanggal"`
	JamaahID         int64    `json:"jamaah_id"`
	NamaJamaah       string   `json:"nama_jamaah"`
	RoomType         string   `json:"room_type"`
	HargaDasar       *float64 `json:"harga_dasar"`
	Status           string   `json:"status"`
	IsSeatBlocked    bool     `json:"is_seat_blocked"`
	TotalHarga       *float64 `json:"total_harga"`
	Diskon           float64  `json:"diskon"`
	DiskonKeterangan *string  `json:"diskon_keterangan"`
	// ProgressPaspor dihitung dinamis dari dokumen_jamaah (kolom DB bookings.progress_paspor bersifat vestigial)
	ProgressPaspor            bool           `json:"progress_paspor"`
	ProgressVisa              bool           `json:"progress_visa"`
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
	Addons                    []BookingAddon `json:"addons"`
	CreatedBy                 *int64         `json:"created_by"`
	CreatedAt                 time.Time      `json:"created_at"`
}

// ─── Request structs ──────────────────────────────────────────────────────────

// CreateBookingRequest adalah payload untuk POST /api/admin/bookings.
type CreateBookingRequest struct {
	ScheduleID int64    `json:"schedule_id"`
	JamaahID   int64    `json:"jamaah_id"`
	RoomType   string   `json:"room_type"`
	TotalHarga *float64 `json:"total_harga"`
}

// UpdateBookingStatusRequest adalah payload untuk PUT /api/admin/bookings/{id}/status.
type UpdateBookingStatusRequest struct {
	Status string `json:"status"`
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
