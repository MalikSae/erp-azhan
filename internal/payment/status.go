package payment

// Nilai kolom payments.status.
const (
	StatusPending   = "pending"
	StatusConfirmed = "confirmed"
	StatusRejected  = "rejected"
	StatusVoid      = "void"
)

// Nilai kolom payments.source.
const (
	SourceAdmin  = "admin"
	SourcePortal = "portal"
)

// Nilai kolom bookings.status yang dibaca atau ditulis paket ini.
const (
	bookingDraft = "draft"
	bookingBaru  = "baru"
	bookingDP    = "dp"
	bookingLunas = "lunas"
	bookingBatal = "batal"
)

// bookingTertutup: booking draft atau batal tidak menerima pembayaran baru dan tidak disinkronkan.
func bookingTertutup(status string) bool {
	return status == bookingBatal || status == bookingDraft
}

// memegangKursi: status booking yang wajib memegang kursi tanpa batas waktu.
func memegangKursi(status string) bool {
	return status == bookingDP || status == bookingLunas
}

// statusBookingDari menentukan status booking dari total harga, pembayaran terkonfirmasi, dan DP yang
// disyaratkan. Satu-satunya tempat aturan baru/dp/lunas dituliskan.
func statusBookingDari(totalHarga, terbayar, dpWajib float64) string {
	switch {
	case totalHarga > 0 && terbayar >= totalHarga:
		return bookingLunas
	case terbayar > 0 && terbayar >= dpWajib:
		return bookingDP
	default:
		return bookingBaru
	}
}

// Nilai kolom payments.metode yang diterima saat admin mencatat pembayaran.
const (
	metodeTransfer = "transfer"
	metodeTunai    = "cash"
)

// pesanBuktiWajib: pembayaran transfer yang dicatat admin wajib punya berkas bukti. Mengembalikan pesan
// untuk pengguna, atau teks kosong bila aturan terpenuhi.
func pesanBuktiWajib(metode string, buktiURL *string) string {
	if metode == metodeTransfer && (buktiURL == nil || *buktiURL == "") {
		return "bukti transfer wajib diunggah untuk pembayaran transfer"
	}
	return ""
}
