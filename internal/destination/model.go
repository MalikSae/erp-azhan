package destination

import (
	"regexp"
	"time"
)

// Destination merepresentasikan satu baris di tabel destinations (master destinasi Umroh Plus).
type Destination struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	IsActive     bool      `json:"is_active"`
	PackageCount int       `json:"package_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SaveRequest adalah payload untuk POST dan PUT /api/admin/destinations.
type SaveRequest struct {
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	IsActive *bool  `json:"is_active"`
}

const maxSlugLen = 100

var slugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidSlug memeriksa slug huruf kecil, angka, dan tanda hubung tunggal (mis. "abu-dhabi").
func ValidSlug(s string) bool {
	return s != "" && len(s) <= maxSlugLen && slugRe.MatchString(s)
}
