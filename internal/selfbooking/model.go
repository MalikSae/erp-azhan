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

// CheckPhoneRequest adalah payload untuk POST /api/public/jamaah/check.
type CheckPhoneRequest struct {
	BrandID *int64 `json:"brand_id"`
	NoHP    string `json:"no_hp"`
}

// CheckPhoneResponse adalah response untuk POST /api/public/jamaah/check.
type CheckPhoneResponse struct {
	Status string `json:"status"` // "baru" | "perlu_pin" | "tanpa_pin"
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
	PortalToken  string            `json:"portal_token,omitempty"`
	BankAccounts []BankAccountInfo `json:"bank_accounts"`
}

// InvoiceBrandInfo berisi identitas travel/brand untuk kop invoice.
type InvoiceBrandInfo struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	PTName         string  `json:"pt_name"`
	PPIUNumber     *string `json:"ppiu_number,omitempty"`
	Akreditasi     *string `json:"akreditasi,omitempty"`
	LogoURL      *string `json:"logo_url,omitempty"`
	PrimaryColor string  `json:"primary_color"`
	Phone        *string `json:"phone,omitempty"`
	WhatsappNumber *string `json:"whatsapp_number,omitempty"`
	Alamat       *string `json:"alamat,omitempty"`
	City         *string `json:"city,omitempty"`
	Province     *string `json:"province,omitempty"`
}

// InvoiceMaskapaiInfo berisi info maskapai penerbangan.
type InvoiceMaskapaiInfo struct {
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url,omitempty"`
}

// InvoiceScheduleInfo berisi info paket dan jadwal keberangkatan.
type InvoiceScheduleInfo struct {
	ID               int64                `json:"id"`
	JadwalNama       string               `json:"jadwal_nama"`
	BerangkatTanggal string               `json:"berangkat_tanggal"`
	PulangTanggal    string               `json:"pulang_tanggal"`
	Maskapai         *InvoiceMaskapaiInfo `json:"maskapai,omitempty"`
	HotelMekkah      string               `json:"hotel_mekkah"`
	HotelMadinah     string               `json:"hotel_madinah"`
}

// InvoicePICInfo berisi identitas pemesan dengan nomor telepon disensor.
type InvoicePICInfo struct {
	NamaLengkap string `json:"nama_lengkap"`
	NoHPMasked  string `json:"no_hp_masked"`
}

// InvoicePaxItem berisi rincian tiap jamaah dalam invoice.
type InvoicePaxItem struct {
	NamaLengkap string  `json:"nama_lengkap"`
	PaxType     string  `json:"pax_type"`
	RoomType    string  `json:"room_type"`
	Harga       float64 `json:"harga"`
}

// InvoiceFinancial berisi ringkasan finansial dan status tagihan.
type InvoiceFinancial struct {
	TotalHarga          float64 `json:"total_harga"`
	MinimalDP           float64 `json:"minimal_dp"`
	TotalDibayar        float64 `json:"total_dibayar"`
	SisaTagihan         float64 `json:"sisa_tagihan"`
	JatuhTempoPelunasan string  `json:"jatuh_tempo_pelunasan"`
}

// InvoiceResponse adalah response lengkap untuk GET /api/public/invoice/{code}.
type InvoiceResponse struct {
	BookingCode       string              `json:"booking_code"`
	Status            string              `json:"status"`
	StatusLabel       string              `json:"status_label"`
	CreatedAt         string              `json:"created_at"`
	SeatHoldExpiresAt string              `json:"seat_hold_expires_at"`
	Brand             InvoiceBrandInfo    `json:"brand"`
	Schedule          InvoiceScheduleInfo `json:"schedule"`
	PIC               InvoicePICInfo      `json:"pic"`
	PaxItems          []InvoicePaxItem    `json:"pax_items"`
	Financial         InvoiceFinancial    `json:"financial"`
	BankAccounts      []BankAccountInfo   `json:"bank_accounts"`
}

