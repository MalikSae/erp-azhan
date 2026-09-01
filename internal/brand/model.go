package brand

import "time"

// Brand merepresentasikan satu baris di tabel brands.
type Brand struct {
	ID                     uint64    `json:"id"`
	KodeBrand              *string   `json:"kode_brand"`
	JamaahCounter          uint32    `json:"jamaah_counter"`
	Name                   string    `json:"name"`
	Domain                 *string   `json:"domain"`
	WhatsappNumber         *string   `json:"whatsapp_number"`
	Email                  *string   `json:"email"`
	Phone                  *string   `json:"phone"`
	Address                *string   `json:"address"`
	City                   *string   `json:"city"`
	Province               *string   `json:"province"`
	GmapsURL               *string   `json:"gmaps_url"`
	Legalitas              *string   `json:"legalitas"`
	BankName               *string   `json:"bank_name"`
	BankAccountNumber      *string   `json:"bank_account_number"`
	BankAccountHolder      *string   `json:"bank_account_holder"`
	SocialFacebook         *string   `json:"social_facebook"`
	SocialInstagram        *string   `json:"social_instagram"`
	SocialTiktok           *string   `json:"social_tiktok"`
	SocialYoutube          *string   `json:"social_youtube"`
	LogoURL                *string   `json:"logo_url"`
	IconURL                *string   `json:"icon_url"`
	PrimaryColor           *string   `json:"primary_color"`
	MetaTitle              *string   `json:"meta_title"`
	MetaDescription        *string   `json:"meta_description"`
	OgImageURL             *string   `json:"og_image_url"`
	GoogleVerificationCode *string   `json:"google_verification_code"`
	MinimalDP              float64   `json:"minimal_dp"`
	CreatedAt              time.Time `json:"created_at"`
}

// CreateBrandRequest adalah payload untuk POST /api/admin/brands.
type CreateBrandRequest struct {
	KodeBrand              *string `json:"kode_brand"`
	Name                   string  `json:"name"`
	Domain                 *string `json:"domain"`
	WhatsappNumber         *string `json:"whatsapp_number"`
	Email                  *string `json:"email"`
	Phone                  *string `json:"phone"`
	Address                *string `json:"address"`
	City                   *string `json:"city"`
	Province               *string `json:"province"`
	GmapsURL               *string `json:"gmaps_url"`
	Legalitas              *string `json:"legalitas"`
	BankName               *string `json:"bank_name"`
	BankAccountNumber      *string `json:"bank_account_number"`
	BankAccountHolder      *string `json:"bank_account_holder"`
	SocialFacebook         *string `json:"social_facebook"`
	SocialInstagram        *string `json:"social_instagram"`
	SocialTiktok           *string `json:"social_tiktok"`
	SocialYoutube          *string `json:"social_youtube"`
	LogoURL                *string `json:"logo_url"`
	IconURL                *string `json:"icon_url"`
	PrimaryColor           *string `json:"primary_color"`
	MetaTitle              *string `json:"meta_title"`
	MetaDescription        *string `json:"meta_description"`
	OgImageURL             *string `json:"og_image_url"`
	GoogleVerificationCode *string `json:"google_verification_code"`
	MinimalDP              float64 `json:"minimal_dp"`
}

// UpdateBrandRequest adalah payload untuk PUT /api/admin/brands/{id}.
type UpdateBrandRequest struct {
	KodeBrand              *string `json:"kode_brand"`
	Name                   string  `json:"name"`
	Domain                 *string `json:"domain"`
	WhatsappNumber         *string `json:"whatsapp_number"`
	Email                  *string `json:"email"`
	Phone                  *string `json:"phone"`
	Address                *string `json:"address"`
	City                   *string `json:"city"`
	Province               *string `json:"province"`
	GmapsURL               *string `json:"gmaps_url"`
	Legalitas              *string `json:"legalitas"`
	BankName               *string `json:"bank_name"`
	BankAccountNumber      *string `json:"bank_account_number"`
	BankAccountHolder      *string `json:"bank_account_holder"`
	SocialFacebook         *string `json:"social_facebook"`
	SocialInstagram        *string `json:"social_instagram"`
	SocialTiktok           *string `json:"social_tiktok"`
	SocialYoutube          *string `json:"social_youtube"`
	LogoURL                *string `json:"logo_url"`
	IconURL                *string `json:"icon_url"`
	PrimaryColor           *string `json:"primary_color"`
	MetaTitle              *string `json:"meta_title"`
	MetaDescription        *string `json:"meta_description"`
	OgImageURL             *string `json:"og_image_url"`
	GoogleVerificationCode *string `json:"google_verification_code"`
	MinimalDP              float64 `json:"minimal_dp"`
}
