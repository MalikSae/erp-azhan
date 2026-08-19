package itinerary

import "time"

// Activity adalah satu item aktivitas dalam satu hari itinerary.
// Disimpan sebagai JSON array di kolom activities (itinerary_days.activities).
type Activity struct {
	Time string `json:"time"` // boleh kosong string
	Text string `json:"text"` // wajib tidak kosong (divalidasi di handler)
}

// ItineraryDay merepresentasikan satu hari dalam itinerary.
type ItineraryDay struct {
	ID          int64      `json:"id"`
	ItineraryID int64      `json:"itinerary_id"`
	DayNumber   int        `json:"day_number"`
	Title       string     `json:"title"`
	Location    string     `json:"location"`
	Activities  []Activity `json:"activities"`
}

// Itinerary adalah struct lengkap — dipakai di GetItinerary (Days terisi)
// dan di Create/Update response.
type Itinerary struct {
	ID        int64          `json:"id"`
	Title     string         `json:"title"`
	Days      []ItineraryDay `json:"days"`
	CreatedAt time.Time      `json:"created_at"`
}

// ItineraryListItem dipakai di ListItineraries — tanpa Days, hanya DayCount dari COUNT(*).
type ItineraryListItem struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	DayCount  int       `json:"day_count"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── Request structs ──────────────────────────────────────────────────────────

// DayRequest adalah satu hari yang dikirim client saat Create/Update.
// day_number TIDAK ada di sini — ditentukan otomatis server (index+1).
type DayRequest struct {
	Title      string     `json:"title"`
	Location   string     `json:"location"`
	Activities []Activity `json:"activities"`
}

// CreateItineraryRequest adalah payload untuk POST /api/admin/itineraries.
type CreateItineraryRequest struct {
	Title string       `json:"title"`
	Days  []DayRequest `json:"days"`
}

// UpdateItineraryRequest adalah payload untuk PUT /api/admin/itineraries/{id}.
type UpdateItineraryRequest struct {
	Title string       `json:"title"`
	Days  []DayRequest `json:"days"`
}
