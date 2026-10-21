package selfbooking

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/customer"
	"erp-azhan/api/internal/shared"
	"github.com/go-chi/chi/v5"
)

type checkAttempt struct {
	Count     int
	FirstFail time.Time
}

// Batas endpoint publik. Setiap booking menahan kursi 24 jam, jadi pembuatan
// booking massal harus dibatasi.
const (
	bookingIPAttemptLimit      = 20 // percobaan POST /api/account/bookings per IP per jam
	bookingAccountSuccessLimit = 3  // booking berhasil per akun per 24 jam
	// Token invoice acak 256-bit; miss tetap dibatasi untuk menahan scanning.
	invoiceMissIPLimit = 20 // kode invoice tidak ditemukan per IP per 15 menit
)

type Handler struct {
	repo             *Repository
	captcha          *turnstileVerifier
	invoiceMissIP    *windowLimiter
	bookingByIP      *windowLimiter
	bookingByAccount *windowLimiter
	bookingByAgen    *windowLimiter
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:             repo,
		captcha:          newTurnstileVerifier(),
		invoiceMissIP:    newWindowLimiter(invoiceMissIPLimit, 15*time.Minute),
		bookingByIP:      newWindowLimiter(bookingIPAttemptLimit, time.Hour),
		bookingByAccount: newWindowLimiter(bookingAccountSuccessLimit, 24*time.Hour),
		bookingByAgen:    newWindowLimiter(agenBookingLimit, time.Hour),
	}
}

func getClientIP(r *http.Request) string {
	return shared.ClientIP(r)
}

func (h *Handler) GetPublicInvoice(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.invoiceMissIP.allow(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}

	code := chi.URLParam(r, "code")
	if strings.TrimSpace(code) == "" {
		writeError(w, http.StatusBadRequest, "kode booking wajib diisi")
		return
	}

	brandID, parseErr := strconv.ParseInt(r.URL.Query().Get("brand"), 10, 64)
	if parseErr != nil || brandID <= 0 || len(code) != 64 {
		writeError(w, http.StatusNotFound, "invoice tidak ditemukan")
		return
	}
	bookingCode, err := resolveInvoiceToken(r.Context(), h.repo.db, code, brandID)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusServiceUnavailable, "invoice sementara tidak tersedia")
			return
		}
		h.invoiceMissIP.record(clientIP)
		writeError(w, http.StatusNotFound, "invoice tidak ditemukan")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
	w.Header().Set("Referrer-Policy", "no-referrer")
	invoice, err := h.repo.GetInvoiceByCode(r.Context(), bookingCode)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			h.invoiceMissIP.record(clientIP)
			writeError(w, http.StatusNotFound, "invoice booking tidak ditemukan")
			return
		}
		log.Printf("[ERROR] GetPublicInvoice: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal memuat data invoice")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(invoice)
}

// CreateBooking POST /api/account/bookings (butuh sesi akun pelanggan).
func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	clientIP := getClientIP(r)
	if !h.bookingByIP.allow(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak permintaan booking, coba lagi dalam 1 jam")
		return
	}
	h.bookingByIP.record(clientIP)

	account := customer.FromContext(r.Context())
	if account == nil {
		writeError(w, http.StatusUnauthorized, "sesi berakhir, silakan masuk kembali")
		return
	}
	var req BookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	// Brand dan identitas pemesan berasal dari akun, bukan dari payload.
	req.BrandID = account.BrandID
	if strings.TrimSpace(req.PIC.NamaLengkap) == "" {
		req.PIC.NamaLengkap = account.NamaLengkap
	}
	if req.PemesanTidakIkut {
		// PIC = jamaah pertama yang diisi pemesan; kontak pemesan tetap di akun.
		req.PIC.Email = nil
	} else {
		req.PIC.NoHP = account.NoHP
		req.PIC.Email = &account.Email
	}
	// Gate 1: Captcha (hanya bila Turnstile aktif, lihat turnstile.go)
	if h.captcha.enabled() {
		if strings.TrimSpace(req.CaptchaToken) == "" {
			writeError(w, http.StatusBadRequest, "Verifikasi keamanan gagal, silakan coba lagi")
			return
		}
		valid, err := h.captcha.verify(r.Context(), strings.TrimSpace(req.CaptchaToken), clientIP)
		if err != nil {
			log.Printf("[ERROR] selfbooking captcha: %v", err)
			writeError(w, http.StatusServiceUnavailable, "verifikasi keamanan sedang tidak tersedia, silakan coba lagi")
			return
		}
		if !valid {
			writeError(w, http.StatusBadRequest, "Verifikasi keamanan gagal, silakan coba lagi")
			return
		}
	}

	// Gate 2: Brand ID
	if req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "brand_id wajib diisi")
		return
	}

	// Gate 3: Schedule ID
	if req.ScheduleID <= 0 {
		writeError(w, http.StatusBadRequest, "schedule_id wajib diisi")
		return
	}

	// Gate 4: PIC
	req.PIC.NamaLengkap = strings.TrimSpace(req.PIC.NamaLengkap)
	req.PIC.NoHP = strings.TrimSpace(req.PIC.NoHP)
	req.PIC.JenisKelamin = strings.TrimSpace(req.PIC.JenisKelamin)
	req.PIC.RoomType = strings.TrimSpace(req.PIC.RoomType)

	if req.PIC.NamaLengkap == "" || (req.PIC.NoHP == "" && !req.PemesanTidakIkut) || req.PIC.JenisKelamin == "" || req.PIC.RoomType == "" {
		writeError(w, http.StatusBadRequest, "data jamaah utama tidak lengkap")
		return
	}

	// Gate 5: Room Type
	if req.PIC.RoomType != "Quad" && req.PIC.RoomType != "Triple" && req.PIC.RoomType != "Double" {
		writeError(w, http.StatusBadRequest, "tipe kamar PIC tidak valid")
		return
	}

	// Gate 6: Anggota
	for i, a := range req.Anggota {
		req.Anggota[i].NamaLengkap = strings.TrimSpace(a.NamaLengkap)
		req.Anggota[i].JenisKelamin = strings.TrimSpace(a.JenisKelamin)
		req.Anggota[i].PaxType = strings.TrimSpace(a.PaxType)

		if req.Anggota[i].NamaLengkap == "" || req.Anggota[i].JenisKelamin == "" || req.Anggota[i].PaxType == "" {
			writeError(w, http.StatusBadRequest, "data anggota tidak lengkap")
			return
		}

		if req.Anggota[i].PaxType == "reguler" {
			if a.RoomType == nil || (*a.RoomType != "Quad" && *a.RoomType != "Triple" && *a.RoomType != "Double") {
				writeError(w, http.StatusBadRequest, "tipe kamar reguler tidak valid")
				return
			}
		} else if req.Anggota[i].PaxType == "infant" {
			if a.TanggalLahir == nil || *a.TanggalLahir == "" {
				writeError(w, http.StatusBadRequest, "tanggal lahir wajib untuk infant")
				return
			}
			// Umur divalidasi repository menggunakan identitas yang tersimpan.
		} else {
			writeError(w, http.StatusBadRequest, "pax_type anggota tidak valid")
			return
		}
	}

	// Gate 7: Total Pax <= 9
	if 1+len(req.Anggota) > 9 {
		writeError(w, http.StatusBadRequest, "maksimal jamaah dalam satu booking adalah 9")
		return
	}

	// Gate 8: Batas booking per akun
	accountKey := strconv.FormatInt(account.ID, 10)
	if !h.bookingByAccount.allow(accountKey) {
		writeError(w, http.StatusTooManyRequests, "akun ini sudah mencapai batas booking hari ini, silakan hubungi admin travel")
		return
	}

	// Process
	if msg := validateCheckout(&req, false); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	resp, err := h.repo.ProcessBooking(r.Context(), req.BrandID, account.ID, req)
	if err != nil {
		if errors.Is(err, ErrQuoteChanged) || errors.Is(err, ErrRequestConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, ErrSeatHabis) || errors.Is(err, ErrDuplicate) ||
			errors.Is(err, ErrAnggotaNameMismatch) || errors.Is(err, ErrDuplicatePaxInBooking) ||
			errors.Is(err, ErrNomorAtasNamaLain) || errors.Is(err, ErrNIKAtasNamaLain) || errors.Is(err, ErrAkunTidakValid) || errors.Is(err, ErrCutoffBooking) ||
			errors.Is(err, ErrTanggalLahirTidakValid) || errors.Is(err, ErrUsiaInfant) || errors.Is(err, ErrInfantUnavailable) || errors.Is(err, ErrInvalidDP) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "jadwal atau brand tidak ditemukan")
			return
		}
		log.Printf("[ERROR] ProcessBooking: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	if !resp.Replayed {
		h.bookingByAccount.record(accountKey)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
