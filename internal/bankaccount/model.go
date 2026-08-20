package bankaccount

import "time"

type BankAccount struct {
	ID            int64     `json:"id"`
	BrandID       int64     `json:"brand_id"`
	BrandName     string    `json:"brand_name,omitempty"`
	BankName      string    `json:"bank_name"`
	AccountNumber string    `json:"account_number"`
	AccountHolder string    `json:"account_holder"`
	Instructions  *string   `json:"instructions"`
	IsActive      bool      `json:"is_active"`
	SortOrder     int       `json:"sort_order"`
	CreatedAt     time.Time `json:"created_at"`
}

type UpsertRequest struct {
	BrandID       int64   `json:"brand_id"`
	BankName      string  `json:"bank_name"`
	AccountNumber string  `json:"account_number"`
	AccountHolder string  `json:"account_holder"`
	Instructions  *string `json:"instructions"`
	IsActive      bool    `json:"is_active"`
	SortOrder     int     `json:"sort_order"`
}
