package payment

import "time"

// Payment adalah response lengkap.
type Payment struct {
	ID        int64     `json:"id"`
	BookingID int64     `json:"booking_id"`
	Jumlah    float64   `json:"jumlah"`
	Metode    *string   `json:"metode"`
	Tanggal   *string   `json:"tanggal"`
	Status    string    `json:"status"`
	BuktiURL  *string   `json:"bukti_url"`
	CreatedAt time.Time `json:"created_at"`
}

// CreatePaymentRequest adalah payload untuk POST.
type CreatePaymentRequest struct {
	Jumlah   float64 `json:"jumlah"`
	Metode   *string `json:"metode"`
	Tanggal  *string `json:"tanggal"`
	BuktiURL *string `json:"bukti_url"`
}

// UpdatePaymentStatusRequest adalah payload untuk PUT status.
type UpdatePaymentStatusRequest struct {
	Status string `json:"status"`
}
