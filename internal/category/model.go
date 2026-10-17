package category

import (
	"regexp"
	"time"
)

// BrandRef subset brand untuk embed di response Category.
type BrandRef struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	PrimaryColor *string `json:"primary_color,omitempty"`
}

var badgeColorRe = regexp.MustCompile(`^#[0-9a-f]{6}$`)

// ValidBadgeColor memeriksa warna badge berformat hex #rrggbb (huruf kecil); string kosong berarti "otomatis".
func ValidBadgeColor(c string) bool {
	return c == "" || badgeColorRe.MatchString(c)
}

// Category merepresentasikan response detail kategori paket.
type Category struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description *string    `json:"description"`
	BadgeColor  *string    `json:"badge_color"`
	IsActive    bool       `json:"is_active"`
	Brands      []BrandRef `json:"brands"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// CategoryListItem merepresentasikan data kategori pada tabel list.
type CategoryListItem struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Slug         string     `json:"slug"`
	Description  *string    `json:"description"`
	BadgeColor   *string    `json:"badge_color"`
	IsActive     bool       `json:"is_active"`
	Brands       []BrandRef `json:"brands"`
	PackageCount int        `json:"package_count"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CreateCategoryRequest payload untuk membuat kategori baru.
type CreateCategoryRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
	BadgeColor  *string `json:"badge_color"`
	IsActive    *bool   `json:"is_active"`
	BrandIDs    []int64 `json:"brand_ids"`
}

// UpdateCategoryRequest payload untuk mengubah kategori.
type UpdateCategoryRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
	BadgeColor  *string `json:"badge_color"`
	IsActive    *bool   `json:"is_active"`
	BrandIDs    []int64 `json:"brand_ids"`
}
