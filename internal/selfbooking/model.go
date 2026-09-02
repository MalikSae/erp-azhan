package selfbooking

// PICInput adalah data pendaftar utama (kontak utama).
type PICInput struct {
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHP         string  `json:"no_hp"`
	Email        *string `json:"email,omitempty"`
	JenisKelamin string  `json:"jenis_kelamin"` // "L" atau "P"
	RoomType     string  `json:"room_type"`     // "Quad" | "Triple" | "Double"
	PortalPIN    string  `json:"portal_pin"`    // 6 digit PIN
}

// AnggotaInput adalah data anggota rombongan (minimal).
type AnggotaInput struct {
	PaxType      string  `json:"pax_type"`      // "reguler" | "infant"
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHP         *string `json:"no_hp,omitempty"`
	JenisKelamin string  `json:"jenis_kelamin"` // "L" atau "P"
	RoomType     *string `json:"room_type"`     // Quad/Triple/Double (null jika infant)
	TanggalLahir *string `json:"tanggal_lahir"` // Wajib jika pax_type = infant (YYYY-MM-DD)
}

// BookingRequest adalah payload untuk POST /api/public/book.
type BookingRequest struct {
	BrandID      int64          `json:"brand_id"`
	ScheduleID   int64          `json:"schedule_id"`
	CaptchaToken string         `json:"captcha_token"`
	PIC          PICInput       `json:"pic"`
	Anggota      []AnggotaInput `json:"anggota"`
}

// PaxSummary adalah ringkasan per pax di response.
type PaxSummary struct {
	Nama     string  `json:"nama"`
	PaxType  string  `json:"pax_type"`
	RoomType *string `json:"room_type"`
	Harga    float64 `json:"harga"`
}

// BankAccountInfo adalah info rekening untuk pembayaran.
type BankAccountInfo struct {
	ID            int64   `json:"id"`
	BankName      string  `json:"bank_name"`
	LogoURL       *string `json:"logo_url,omitempty"`
	AccountNumber string  `json:"account_number"`
	AccountHolder string  `json:"account_holder"`
	Instructions  *string `json:"instructions"`
}

// BookingSummary adalah ringkasan booking di response.
type BookingSummary struct {
	BookingCode       string       `json:"booking_code"`
	TotalHarga        float64      `json:"total_harga"`
	MinimalDP         float64      `json:"minimal_dp"`
	SeatHoldExpiresAt string       `json:"seat_hold_expires_at"`
	PaxSummary        []PaxSummary `json:"pax_summary"`
}

// JamaahInfo adalah info jamaah PIC di response.
type JamaahInfo struct {
	ID       int64  `json:"id"`
	IDJamaah string `json:"id_jamaah"`
}

// BookingResponse adalah response untuk POST /api/public/book.
type BookingResponse struct {
	Status       string            `json:"status"`
	Booking      BookingSummary    `json:"booking"`
	Jamaah       JamaahInfo        `json:"jamaah"`
	PortalToken  string            `json:"portal_token"`
	BankAccounts []BankAccountInfo `json:"bank_accounts"`
}
