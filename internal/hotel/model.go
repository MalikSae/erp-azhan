package hotel

import "time"

// Hotel merepresentasikan satu baris di tabel hotels.
type Hotel struct {
	ID         uint64     `json:"id"`
	Name       string     `json:"name"`
	City       string     `json:"city"`
	StarRating *int       `json:"star_rating"`
	DistanceM  *int       `json:"distance_m"` // pointer agar bisa null
	PhotoURL   *string    `json:"photo_url"`
	CreatedAt  time.Time  `json:"created_at"`
}

// CreateHotelRequest adalah payload untuk POST /api/admin/hotels.
type CreateHotelRequest struct {
	Name       string  `json:"name"`
	City       string  `json:"city"`
	StarRating *int    `json:"star_rating"`
	DistanceM  *int    `json:"distance_m"`
	PhotoURL   *string `json:"photo_url"`
}

// UpdateHotelRequest adalah payload untuk PUT /api/admin/hotels/{id}.
type UpdateHotelRequest struct {
	Name       string  `json:"name"`
	City       string  `json:"city"`
	StarRating *int    `json:"star_rating"`
	DistanceM  *int    `json:"distance_m"`
	PhotoURL   *string `json:"photo_url"`
}
