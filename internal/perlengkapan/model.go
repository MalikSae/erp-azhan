package perlengkapan

import "time"

// PerlengkapanItem merepresentasikan 1 item master perlengkapan (katalog global).
type PerlengkapanItem struct {
	ID        uint64    `json:"id"`
	Nama      string    `json:"nama"`
	QtyPerSet int       `json:"qty_per_set"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateItemRequest adalah payload untuk POST /api/admin/perlengkapan-items.
type CreateItemRequest struct {
	Nama      string `json:"nama"`
	QtyPerSet *int   `json:"qty_per_set"`
}

// UpdateItemRequest adalah payload untuk PUT /api/admin/perlengkapan-items/{id}.
type UpdateItemRequest struct {
	Nama      string `json:"nama"`
	QtyPerSet *int   `json:"qty_per_set"`
}

// PerlengkapanStokItem merepresentasikan stok item untuk 1 brand tertentu.
type PerlengkapanStokItem struct {
	PerlengkapanItemID uint64 `json:"perlengkapan_item_id"`
	Nama               string `json:"nama"`
	QtyPerSet          int    `json:"qty_per_set"`
	StokTersedia       int    `json:"stok_tersedia"`
}

// PerlengkapanStokAllRow merepresentasikan baris stok perlengkapan gabungan seluruh brand.
type PerlengkapanStokAllRow struct {
	BrandID            uint64  `json:"brand_id"`
	BrandName          string  `json:"brand_name"`
	BrandLogoURL       *string `json:"brand_logo_url"`
	BrandIconURL       *string `json:"brand_icon_url,omitempty"`
	PerlengkapanItemID uint64  `json:"perlengkapan_item_id"`
	NamaItem           string  `json:"nama_item"`
	QtyPerSet          int     `json:"qty_per_set"`
	StokTersedia       int     `json:"stok_tersedia"`
}

// UpdateStokRequest adalah payload untuk PUT /api/admin/perlengkapan-stok/{item_id}?brand_id={id}.
type UpdateStokRequest struct {
	StokTersedia *int `json:"stok_tersedia"`
}

// PerlengkapanSetTemplateItem merepresentasikan 1 item dalam set template global.
type PerlengkapanSetTemplateItem struct {
	PerlengkapanItemID uint64 `json:"perlengkapan_item_id"`
	Nama               string `json:"nama"`
	Qty                int    `json:"qty"`
}

// SetTemplateItemInput adalah payload elemen untuk PUT /api/admin/perlengkapan-set-template.
type SetTemplateItemInput struct {
	PerlengkapanItemID uint64 `json:"perlengkapan_item_id"`
	Qty                int    `json:"qty"`
}
