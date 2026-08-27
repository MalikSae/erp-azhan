package category

import "time"

// BrandRef subset brand untuk embed di response Category.
type BrandRef struct {
	ID           int64   `json:"id"`
	Name         string  `json:"name"`
	PrimaryColor *string `json:"primary_color,omitempty"`
}

// Category merepresentasikan response detail kategori paket.
type Category struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description *string    `json:"description"`
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
	IsActive    *bool   `json:"is_active"`
	BrandIDs    []int64 `json:"brand_ids"`
}

// UpdateCategoryRequest payload untuk mengubah kategori.
type UpdateCategoryRequest struct {
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Description *string `json:"description"`
	IsActive    *bool   `json:"is_active"`
	BrandIDs    []int64 `json:"brand_ids"`
}
