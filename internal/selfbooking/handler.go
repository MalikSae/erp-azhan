package selfbooking

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"erp-azhan/api/internal/customer"
	"erp-azhan/api/internal/mailer"
	"erp-azhan/api/internal/shared"

	"github.com/go-sql-driver/mysql"
)

// Batas endpoint publik. Setiap booking menahan kursi 24 jam, jadi pembuatan
// booking massal harus dibatasi.
const (
	// Percobaan booking dihitung per akun: app memanggil ERP langsung dari HP, jadi banyak pengguna seluler bisa
	// berbagi satu IP operator (CGNAT). Batas per IP tetap ada sebagai penahan banyak akun dari satu alamat.
	bookingAccountAttemptLimit = 20  // percobaan POST booking per akun per jam
	bookingIPAttemptLimit      = 120 // percobaan POST booking per IP per jam
	bookingAccountSuccessLimit = 3   // booking berhasil per akun per 24 jam (ditegakkan juga di DB, lihat cekBatasAkun)
	// Validasi NIK/nomor anggota yang gagal (data milik jamaah lain) per akun per jendela; menahan
	// penebakan NIK/nomor lewat pesan galat.
	identitasGagalAccountLimit  = 5
	identitasGagalAccountWindow = 10 * time.Minute
	// Token invoice acak 256-bit; miss tetap dibatasi untuk menahan scanning.
	invoiceMissIPLimit = 20 // kode invoice tidak ditemukan per IP per 15 menit
)

// Kode galat yang dikenali frontend (azhan-booking) di samping pesan "error".
const (
	KodeBatasSeatAkun          = "batas_seat_akun"
	KodeBatasBookingHarian     = "batas_booking_harian"
	KodeIdentitasTerlaluSering = "percobaan_data_jamaah_terlalu_sering"
)

type Handler struct {
	repo             *Repository
	captcha          *turnstileVerifier
	invoiceMissIP    *windowLimiter
	bookingByIP      *windowLimiter
	bookingAttempt   *windowLimiter // percobaan booking per akun per jam
	bookingByAccount *windowLimiter
	identitasGagal   *windowLimiter
	// Email konfirmasi "Booking berhasil dibuat" (hanya booking mandiri pelanggan).
	mail     mailer.Sender
	mailWG   sync.WaitGroup
	loadInfo func(ctx context.Context, brandID, scheduleID int64) (*mailInfo, error)
	// book dapat diganti di tes; bawaannya repo.ProcessBooking.
	book func(ctx context.Context, brandID, accountID int64, req BookingRequest) (*BookingResponse, error)
}

func NewHandler(repo *Repository, m mailer.Sender) *Handler {
	return &Handler{
		repo:             repo,
		mail:             m,
		captcha:          newTurnstileVerifier(),
		invoiceMissIP:    newWindowLimiter(invoiceMissIPLimit, 15*time.Minute),
		bookingByIP:      newWindowLimiter(bookingIPAttemptLimit, time.Hour),
		bookingAttempt:   newWindowLimiter(bookingAccountAttemptLimit, time.Hour),
		bookingByAccount: newWindowLimiter(bookingAccountSuccessLimit, 24*time.Hour),
		identitasGagal:   newWindowLimiter(identitasGagalAccountLimit, identitasGagalAccountWindow),
	}
}

func getClientIP(r *http.Request) string {
	return shared.ClientIP(r)
}

// jalurBooking: pintu HTTP booking. Logika inti sama untuk semuanya; yang beda hanya captcha (web) dan
// bawaan payload endpoint portal agen lama.
type jalurBooking int

const (
	jalurWeb jalurBooking = iota
	jalurApp
	jalurPortalAgen
)

// CreateBooking POST /api/account/bookings (butuh sesi akun pelanggan).
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	h.buatBooking(w, r, jalurWeb)
}

// CreateBookingApp POST /api/account/bookings-app: booking dari app Hana/Alsha (keputusan 14 November 2026).
// App native tidak bisa menjalankan captcha web, jadi endpoint ini tanpa captcha. Pengguna app selalu sudah
// masuk, dan pembatas yang sama tetap berlaku: percobaan per akun dan per IP, booking harian per akun, dan hold
// seat per akun.
func (h *Handler) CreateBookingApp(w http.ResponseWriter, r *http.Request) {
	h.buatBooking(w, r, jalurApp)
}

// CreateBookingAgen POST /api/portal/agen/bookings: alias tipis. Payload lama diterjemahkan (pic.saya_sendiri
// menjadi pemesan_tidak_ikut=false; bawaan portal: pemesan tidak ikut), lalu diproses jalur inti. Bentuk
// respons tetap.
func (h *Handler) CreateBookingAgen(w http.ResponseWriter, r *http.Request) {
	h.buatBooking(w, r, jalurPortalAgen)
}

func (h *Handler) buatBooking(w http.ResponseWriter, r *http.Request, jalur jalurBooking) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	account := customer.FromContext(r.Context())
	if account == nil {
		writeError(w, http.StatusUnauthorized, "Sesi berakhir. Masuk kembali.")
		return
	}
	agenAktif := account.StatusAgen == "aktif"
	if jalur == jalurPortalAgen && !agenAktif {
		writeError(w, http.StatusForbidden, ErrBukanAgenAktif.Error())
		return
	}
	// Batas percobaan per akun dulu, lalu per IP. Ditolak di batas IP: jatah akun dikembalikan.
	clientIP := getClientIP(r)
	percobaanKey := strconv.FormatInt(account.ID, 10)
	if !h.bookingAttempt.Take(percobaanKey) {
		writeError(w, http.StatusTooManyRequests, "Terlalu banyak permintaan booking, coba lagi dalam 1 jam.")
		return
	}
	if !h.bookingByIP.Take(clientIP) {
		h.bookingAttempt.Refund(percobaanKey)
		writeError(w, http.StatusTooManyRequests, "Terlalu banyak permintaan booking, coba lagi dalam 1 jam.")
		return
	}
	// Verifikasi email tidak diwajibkan untuk booking (keputusan 2026-10-28); email tetap dikirim dan statusnya tampil di akun.
	var req BookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	if jalur == jalurPortalAgen {
		req.PemesanTidakIkut = !req.PIC.SayaSendiri
	}
	req.PIC.SayaSendiri = false
	// Brand dan identitas pemesan berasal dari akun, bukan dari payload.
	req.BrandID = account.BrandID
	if req.PemesanTidakIkut {
		// PIC = jamaah pertama yang diisi pemesan; kontak pemesan tetap di akun.
		req.PIC.Email = nil
	} else {
		if strings.TrimSpace(req.PIC.NamaLengkap) == "" {
			req.PIC.NamaLengkap = account.NamaLengkap
		}
		req.PIC.NoHP = account.NoHP
		req.PIC.Email = &account.Email
	}
	// Gate 1: Captcha (web saja). Opsional: tanpa TURNSTILE_SECRET_KEY gate ini dilewati, lihat turnstile.go.
	if jalur == jalurWeb && !h.lolosCaptcha(w, r.Context(), req.CaptchaToken, clientIP) {
		return
	}

	// Gate 2: Brand ID
	if req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "Brand wajib diisi.")
		return
	}

	// Gate 3: Schedule ID
	if req.ScheduleID <= 0 {
		writeError(w, http.StatusBadRequest, "Paket wajib diisi.")
		return
	}

	// Gate 3b: jamaah_id (Jamaah Saya) hanya untuk agen aktif.
	if !agenAktif && adaJamaahID(&req) {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	// Gate 4: PIC
	req.PIC.NamaLengkap = strings.TrimSpace(req.PIC.NamaLengkap)
	req.PIC.NoHP = strings.TrimSpace(req.PIC.NoHP)
	req.PIC.JenisKelamin = strings.TrimSpace(req.PIC.JenisKelamin)
	req.PIC.RoomType = strings.TrimSpace(req.PIC.RoomType)

	pintasPIC := agenAktif && req.PemesanTidakIkut && req.PIC.JamaahID != nil
	if (!pintasPIC && (req.PIC.NamaLengkap == "" || req.PIC.JenisKelamin == "")) ||
		(req.PIC.NoHP == "" && !req.PemesanTidakIkut) || req.PIC.RoomType == "" {
		writeError(w, http.StatusBadRequest, "Data jamaah utama tidak lengkap.")
		return
	}

	// Gate 5: Room Type
	if req.PIC.RoomType != "Quad" && req.PIC.RoomType != "Triple" && req.PIC.RoomType != "Double" {
		writeError(w, http.StatusBadRequest, "Tipe kamar PIC tidak valid.")
		return
	}

	// Gate 6: Anggota
	for i, a := range req.Anggota {
		req.Anggota[i].NamaLengkap = strings.TrimSpace(a.NamaLengkap)
		req.Anggota[i].JenisKelamin = strings.TrimSpace(a.JenisKelamin)
		req.Anggota[i].PaxType = strings.TrimSpace(a.PaxType)

		pintas := agenAktif && a.JamaahID != nil
		if (!pintas && (req.Anggota[i].NamaLengkap == "" || req.Anggota[i].JenisKelamin == "")) || req.Anggota[i].PaxType == "" {
			writeError(w, http.StatusBadRequest, "Data anggota tidak lengkap.")
			return
		}

		if req.Anggota[i].PaxType == "reguler" {
			if a.RoomType == nil || (*a.RoomType != "Quad" && *a.RoomType != "Triple" && *a.RoomType != "Double") {
				writeError(w, http.StatusBadRequest, "Tipe kamar reguler tidak valid.")
				return
			}
		} else if req.Anggota[i].PaxType == "infant" {
			if a.TanggalLahir == nil || *a.TanggalLahir == "" {
				writeError(w, http.StatusBadRequest, "Tanggal lahir wajib untuk infant.")
				return
			}
			// Umur divalidasi repository menggunakan identitas yang tersimpan.
		} else {
			writeError(w, http.StatusBadRequest, "Jenis pax anggota tidak valid.")
			return
		}
	}

	// Gate 7: Total Pax <= 9
	if 1+len(req.Anggota) > 9 {
		writeError(w, http.StatusBadRequest, "Maksimal jamaah dalam satu booking adalah 9.")
		return
	}

	// Gate 8: Batas booking per akun
	accountKey := strconv.FormatInt(account.ID, 10)
	// Slot dipesan atomik sebelum transaksi booking: request paralel tidak bisa melampaui batas sebelum
	// hasilnya tercatat. Slot booking dikembalikan bila booking tidak jadi dibuat (gagal atau jawaban ulang);
	// slot identitas dikembalikan kecuali percobaan gagal karena identitas jamaah. Batas harian hanya untuk
	// akun biasa: agen aktif dibatasi hold seat agen (cekBatasSeatAgen).
	if !agenAktif && !h.bookingByAccount.Take(accountKey) {
		writeErrorCode(w, http.StatusTooManyRequests, ErrBatasBookingHarian.Error(), KodeBatasBookingHarian)
		return
	}
	bookingBaru := false
	defer func() {
		if !agenAktif && !bookingBaru {
			h.bookingByAccount.Refund(accountKey)
		}
	}()
	identitasSalah := false
	if h.identitasGagal != nil {
		if !h.identitasGagal.Take(accountKey) {
			writeErrorCode(w, http.StatusTooManyRequests, "Terlalu banyak percobaan. Coba lagi dalam 10 menit.", KodeIdentitasTerlaluSering)
			return
		}
		defer func() {
			if !identitasSalah {
				h.identitasGagal.Refund(accountKey)
			}
		}()
	}

	// Process
	if msg := validateCheckout(&req, agenAktif); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	book := h.book
	if book == nil {
		book = h.repo.ProcessBooking
	}
	resp, err := book(r.Context(), req.BrandID, account.ID, req)
	if err != nil {
		if errors.Is(err, ErrNIKAtasNamaLain) || errors.Is(err, ErrJamaahBukanMilik) || errors.Is(err, ErrSayaSendiriBukanAnda) {
			identitasSalah = true
		}
		tulisGalatBooking(w, err, jalur == jalurPortalAgen)
		return
	}
	if !resp.Replayed {
		bookingBaru = true
		if agenAktif {
			log.Printf("[AUDIT] booking_agen agen_id=%d booking=%s", account.ID, resp.Booking.BookingCode)
		}
		// Email hanya untuk booking baru; jawaban ulang (retry idempoten) tidak mengirim ulang.
		h.notifyBookingCreated(account, req.ScheduleID, resp)
	}

	writeJSON(w, http.StatusOK, resp)
}

// adaJamaahID: ada pax yang memilih jamaah lewat jamaah_id.
func adaJamaahID(req *BookingRequest) bool {
	if req.PIC.JamaahID != nil {
		return true
	}
	for _, a := range req.Anggota {
		if a.JamaahID != nil {
			return true
		}
	}
	return false
}

// tulisGalatBooking memetakan galat booking ke jawaban HTTP, sama untuk semua jalur. Galat khusus agen hanya
// muncul bila pemesannya agen aktif (repository yang menentukan).
func tulisGalatBooking(w http.ResponseWriter, err error, portal bool) {
	switch {
	case errors.Is(err, ErrQuoteChanged), errors.Is(err, ErrRequestConflict), errors.Is(err, ErrBookingKedaluwarsa):
		writeError(w, http.StatusConflict, err.Error())
	case adalahBentrokTransaksi(err):
		writeError(w, http.StatusConflict, "Booking bentrok dengan permintaan lain. Coba lagi.")
	case errors.Is(err, ErrBukanAgenAktif):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrBatasSeatAgen):
		var be *BatasSeatAgenError
		if errors.As(err, &be) {
			// Sisa seat dikirim sebagai data juga, bukan hanya di dalam pesan, agar app bisa menampilkannya.
			writeJSON(w, http.StatusTooManyRequests, map[string]any{
				"error": err.Error(), "code": KodeBatasSeatAgen,
				"batas_per_jadwal": be.BatasPerJadwal, "batas_total": be.BatasTotal,
				"sisa_per_jadwal": be.SisaPerJadwal, "sisa_total": be.SisaTotal,
			})
			return
		}
		writeErrorCode(w, http.StatusTooManyRequests, err.Error(), KodeBatasSeatAgen)
	case errors.Is(err, ErrBatasSeatAkun):
		writeErrorCode(w, http.StatusTooManyRequests, err.Error(), KodeBatasSeatAkun)
	case errors.Is(err, ErrBatasBookingHarian):
		writeErrorCode(w, http.StatusTooManyRequests, err.Error(), KodeBatasBookingHarian)
	case errors.Is(err, ErrSeatHabis), errors.Is(err, ErrDuplicate), errors.Is(err, ErrDuplicatePaxInBooking),
		errors.Is(err, ErrNIKAtasNamaLain), errors.Is(err, ErrJamaahBukanMilik), errors.Is(err, ErrSayaSendiriBukanAnda),
		errors.Is(err, ErrNIKAgenBelumAda), errors.Is(err, ErrInputTidakValid),
		errors.Is(err, ErrAkunTidakValid), errors.Is(err, ErrCutoffBooking),
		errors.Is(err, ErrTanggalLahirTidakValid), errors.Is(err, ErrUsiaInfant), errors.Is(err, ErrInfantUnavailable), errors.Is(err, ErrInvalidDP):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		if portal {
			writeError(w, http.StatusNotFound, "Jadwal tidak ditemukan.")
			return
		}
		writeError(w, http.StatusNotFound, "Jadwal atau brand tidak ditemukan.")
	default:
		log.Printf("[ERROR] ProcessBooking: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
	}
}

// adalahBentrokTransaksi mengenali deadlock (1213) dan lock wait timeout (1205) MySQL. Transaksi sudah di-rollback,
// jadi aman diulang. Level SERIALIZABLE membuat bentrok ini mungkin saat banyak booking ke jadwal yang sama.
func adalahBentrokTransaksi(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && (me.Number == 1213 || me.Number == 1205)
}

// lolosCaptcha memeriksa token captcha booking web dan menulis galat bila gagal.
func (h *Handler) lolosCaptcha(w http.ResponseWriter, ctx context.Context, token, clientIP string) bool {
	token = strings.TrimSpace(token)
	if h.captcha.required() && token == "" {
		writeError(w, http.StatusBadRequest, "Verifikasi keamanan gagal. Coba lagi.")
		return false
	}
	valid, err := h.captcha.verify(ctx, token, clientIP)
	if err != nil {
		log.Printf("[ERROR] selfbooking captcha: %v", err)
		writeError(w, http.StatusServiceUnavailable, "Verifikasi keamanan sedang tidak tersedia. Coba lagi.")
		return false
	}
	if !valid {
		writeError(w, http.StatusBadRequest, "Verifikasi keamanan gagal. Coba lagi.")
		return false
	}
	return true
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

// writeErrorCode menambahkan "code" yang stabil agar frontend dapat menampilkan aksi yang sesuai.
func writeErrorCode(w http.ResponseWriter, status int, msg, code string) {
	writeJSON(w, status, map[string]string{"error": msg, "code": code})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
