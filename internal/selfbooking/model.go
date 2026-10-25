package selfbooking

// PICInput adalah data pendaftar utama (kontak utama).
type PICInput struct {
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHP         string  `json:"no_hp"`
	Email        *string `json:"email,omitempty"`
	JenisKelamin string  `json:"jenis_kelamin"` // "L" atau "P"
	RoomType     string  `json:"room_type"`     // "Quad" | "Triple" | "Double"
	NIK          *string `json:"nik,omitempty"` // 16 digit; wajib di booking publik
	// JamaahID hanya dipakai booking agen (Jalur 1): memilih jamaah dari
	// daftar "Jamaah Saya". Booking publik mengabaikannya.
	JamaahID *int64 `json:"jamaah_id,omitempty"`
}

// AnggotaInput adalah data anggota rombongan (minimal).
type AnggotaInput struct {
	PaxType      string  `json:"pax_type"` // "reguler" | "infant"
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHP         *string `json:"no_hp,omitempty"`
	JenisKelamin string  `json:"jenis_kelamin"`       // "L" atau "P"
	RoomType     *string `json:"room_type"`           // Quad/Triple/Double (null jika infant)
	TanggalLahir *string `json:"tanggal_lahir"`       // Wajib jika pax_type = infant (YYYY-MM-DD)
	JamaahID     *int64  `json:"jamaah_id,omitempty"` // lihat PICInput.JamaahID
	NIK          *string `json:"nik,omitempty"`       // wajib 16 digit untuk reguler, opsional untuk infant
	Nomor        int     `json:"-"`                   // urutan jamaah di form (Jamaah 1, 2, ...), hanya untuk pesan error
}

// BookingRequest adalah payload untuk POST /api/public/book.
type BookingRequest struct {
	RequestKey    string   `json:"request_key"`
	TermsVersion  string   `json:"terms_version"`
	TermsAccepted bool     `json:"terms_accepted"`
	ExpectedTotal *float64 `json:"expected_total"`
	ExpectedDP    *float64 `json:"expected_dp"`
	BrandID       int64    `json:"brand_id"`
	ScheduleID    int64    `json:"schedule_id"`
	CaptchaToken  string   `json:"captcha_token"`
	// PemesanTidakIkut: pemesan (pemilik akun) memesankan untuk orang lain dan
	// tidak berangkat. Data "pic" lalu berisi jamaah pertama, bukan pemesan.
	PemesanTidakIkut bool `json:"pemesan_tidak_ikut"`
	// KodeReferral diisi route handler web publik dari cookie referral
	// (last-click 90 hari), bukan dari input pengguna.
	KodeReferral string         `json:"kode_referral"`
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
	BookingID         int64        `json:"booking_id"`
	InvoiceToken      string       `json:"invoice_token"`
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
	Replayed     bool              `json:"-"`
	Status       string            `json:"status"`
	Booking      BookingSummary    `json:"booking"`
	Jamaah       JamaahInfo        `json:"jamaah"`
	BankAccounts []BankAccountInfo `json:"bank_accounts"`
}

