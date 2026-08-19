package airline

import "time"

// Airline merepresentasikan satu baris di tabel airlines.
type Airline struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	LogoURL   *string   `json:"logo_url"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateAirlineRequest adalah payload untuk POST /api/admin/airlines.
type CreateAirlineRequest struct {
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
}

// UpdateAirlineRequest adalah payload untuk PUT /api/admin/airlines/{id}.
type UpdateAirlineRequest struct {
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
}
