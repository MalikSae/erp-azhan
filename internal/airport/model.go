package airport

import "time"

// Airport merepresentasikan satu baris di tabel airports.
type Airport struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Code      string    `json:"code"`
	City      string    `json:"city"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateAirportRequest adalah payload untuk POST /api/admin/airports.
type CreateAirportRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
	City string `json:"city"`
}

// UpdateAirportRequest adalah payload untuk PUT /api/admin/airports/{id}.
type UpdateAirportRequest struct {
	Name string `json:"name"`
	Code string `json:"code"`
	City string `json:"city"`
}
