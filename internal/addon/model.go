package addon

import "time"

// AddOn merepresentasikan satu baris di tabel add_ons.
type AddOn struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateAddOnRequest adalah payload untuk POST /api/admin/addons.
type CreateAddOnRequest struct {
	Name string `json:"name"`
}

// UpdateAddOnRequest adalah payload untuk PUT /api/admin/addons/{id}.
type UpdateAddOnRequest struct {
	Name string `json:"name"`
}
