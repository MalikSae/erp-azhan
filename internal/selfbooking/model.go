package selfbooking

// PICInput adalah data pendaftar utama (kontak utama).
type PICInput struct {
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHP         string  `json:"no_hp"`
	Email        *string `json:"email,omitempty"`
	JenisKelamin string  `json:"jenis_kelamin"` // "L" atau "P"
	RoomType     string  `json:"room_type"`     // "Quad" | "Triple" | "Double"
	PortalPIN    string  `json:"portal_pin"`    // 6 digit PIN
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
	RequestKey    string   `json:"request_key"`
	TermsVersion  string   `json:"terms_version"`
	TermsAccepted bool     `json:"terms_accepted"`
	ExpectedTotal *float64 `json:"expected_total"`
	ExpectedDP    *float64 `json:"expected_dp"`
	BrandID       int64    `json:"brand_id"`
	ScheduleID    int64    `json:"schedule_id"`
	CaptchaToken  string   `json:"captcha_token"`
	// KodeReferral diisi route handler microsite dari cookie referral
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
	PortalToken  string            `json:"portal_token,omitempty"`
	BankAccounts []BankAccountInfo `json:"bank_accounts"`
}

// InvoiceBrandInfo berisi identitas travel/brand untuk kop invoice.
type InvoiceBrandInfo struct {
	ID             int64   `json:"id"`
	Name           string  `json:"name"`
	PTName         string  `json:"pt_name"`
	PPIUNumber     *string `json:"ppiu_number,omitempty"`
	PIHKNumber     *string `json:"pihk_number,omitempty"`
	Akreditasi     *string `json:"akreditasi,omitempty"`
	LogoURL        *string `json:"logo_url,omitempty"`
	PrimaryColor   string  `json:"primary_color"`
	Phone          *string `json:"phone,omitempty"`
	WhatsappNumber *string `json:"whatsapp_number,omitempty"`
	Alamat         *string `json:"alamat,omitempty"`
	City           *string `json:"city,omitempty"`
	Province       *string `json:"province,omitempty"`
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
	Status      string  `json:"status"`
	NamaLengkap string  `json:"nama_lengkap"`
	PaxType     string  `json:"pax_type"`
	RoomType    string  `json:"room_type"`
	Harga       float64 `json:"harga"`
}

// InvoiceFinancial berisi ringkasan finansial dan status tagihan.
type InvoiceFinancial struct {
	PendingPayment      float64 `json:"pending_payment"`
	Adjustments         float64 `json:"adjustments"`
	TotalHarga          float64 `json:"total_harga"`
	MinimalDP           float64 `json:"minimal_dp"`
	TotalDibayar        float64 `json:"total_dibayar"`
	SisaTagihan         float64 `json:"sisa_tagihan"`
	JatuhTempoPelunasan string  `json:"jatuh_tempo_pelunasan"`
	// JatuhTempoAt (RFC3339) dan FullPayment dipakai microsite untuk format
	// tanggal Indonesia dan memilih kalimat DP atau bayar penuh.
	JatuhTempoAt string `json:"jatuh_tempo_at"`
	FullPayment  bool   `json:"full_payment"`
}

// InvoiceResponse adalah response lengkap untuk GET /api/public/invoice/{code}.
type InvoiceResponse struct {
	ReservationStatus        string              `json:"reservation_status"`
	PortalActivationRequired bool                `json:"portal_activation_required"`
	BookingCode              string              `json:"booking_code"`
	Status                   string              `json:"status"`
	StatusLabel              string              `json:"status_label"`
	CreatedAt                string              `json:"created_at"`
	SeatHoldExpiresAt        string              `json:"seat_hold_expires_at"`
	Brand                    InvoiceBrandInfo    `json:"brand"`
	Schedule                 InvoiceScheduleInfo `json:"schedule"`
	PIC                      InvoicePICInfo      `json:"pic"`
	PaxItems                 []InvoicePaxItem    `json:"pax_items"`
	Financial                InvoiceFinancial    `json:"financial"`
	BankAccounts             []BankAccountInfo   `json:"bank_accounts"`
}
