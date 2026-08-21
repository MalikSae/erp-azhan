package payment

import "time"

// Payment adalah response lengkap.
type Payment struct {
	ID                       int64      `json:"id"`
	BookingID                int64      `json:"booking_id"`
	Jumlah                   float64    `json:"jumlah"`
	Metode                   *string    `json:"metode"`
	Tanggal                  *string    `json:"tanggal"`
	Status                   string     `json:"status"`
	BuktiURL                 *string    `json:"bukti_url"`
	BankAccountID            *int64     `json:"bank_account_id"`
	DestinationBankName      *string    `json:"destination_bank_name"`
	DestinationAccountNumber *string    `json:"destination_account_number"`
	DestinationAccountHolder *string    `json:"destination_account_holder"`
	SenderName               *string    `json:"sender_name"`
	SenderBank               *string    `json:"sender_bank"`
	Notes                    *string    `json:"notes"`
	Source                   string     `json:"source"`
	RejectionReason          *string    `json:"rejection_reason"`
	VerifiedBy               *int64     `json:"verified_by"`
	VerifiedAt               *time.Time `json:"verified_at"`
	JamaahName               string     `json:"jamaah_name,omitempty"`
	ScheduleName             string     `json:"schedule_name,omitempty"`
	BrandName                string     `json:"brand_name,omitempty"`
	BookingIDBooking         string     `json:"booking_id_booking,omitempty"`
	CreatedAt                time.Time  `json:"created_at"`
}

// CreatePaymentRequest adalah payload untuk POST.
type CreatePaymentRequest struct {
	Jumlah        float64 `json:"jumlah"`
	Metode        *string `json:"metode"`
	Tanggal       *string `json:"tanggal"`
	BuktiURL      *string `json:"bukti_url"`
	BankAccountID *int64  `json:"bank_account_id"`
	SenderName    *string `json:"sender_name"`
	SenderBank    *string `json:"sender_bank"`
	Notes         *string `json:"notes"`
	Source        string  `json:"source"`
}

// UpdatePaymentStatusRequest adalah payload untuk PUT status.
type UpdatePaymentStatusRequest struct {
	Status          string  `json:"status"`
	RejectionReason *string `json:"rejection_reason"`
}

// DailyBrandTransaction adalah agregasi pembayaran terkonfirmasi per brand dan hari.
type DailyBrandTransaction struct {
	Date        string  `json:"date"`
	BrandID     int64   `json:"brand_id"`
	TotalAmount float64 `json:"total_amount"`
	Count       int     `json:"count"`
}
