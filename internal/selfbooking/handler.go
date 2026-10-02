package selfbooking

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
	"github.com/go-chi/chi/v5"
)

type checkAttempt struct {
	Count     int
	FirstFail time.Time
}

// Batas endpoint publik. Setiap booking menahan kursi 24 jam, jadi pembuatan
// booking massal harus dibatasi. Cek nomor HP membocorkan apakah nomor
// terdaftar (baru / perlu_pin / tanpa_pin), jadi SETIAP pengecekan dihitung,
// bukan hanya yang hasilnya "baru".
const (
	bookingIPAttemptLimit    = 20 // percobaan POST /api/public/book per IP per jam
	bookingPhoneSuccessLimit = 3  // booking berhasil per nomor HP PIC per 24 jam
	phoneCheckIPLimit        = 10 // pengecekan POST /api/public/jamaah/check per IP per 15 menit
	// Token invoice acak 256-bit; miss tetap dibatasi untuk menahan scanning.
	invoiceMissIPLimit = 20 // kode invoice tidak ditemukan per IP per 15 menit
)

type Handler struct {
	repo           *Repository
	captcha        *turnstileVerifier
	phoneCheckByIP *windowLimiter
	invoiceMissIP  *windowLimiter
	bookingByIP    *windowLimiter
	bookingByPhone *windowLimiter
	daftarByIP     *windowLimiter
	bookingByAgen  *windowLimiter
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{
		repo:           repo,
		captcha:        newTurnstileVerifier(),
		phoneCheckByIP: newWindowLimiter(phoneCheckIPLimit, 15*time.Minute),
		invoiceMissIP:  newWindowLimiter(invoiceMissIPLimit, 15*time.Minute),
		bookingByIP:    newWindowLimiter(bookingIPAttemptLimit, time.Hour),
		bookingByPhone: newWindowLimiter(bookingPhoneSuccessLimit, 24*time.Hour),
		daftarByIP:     newWindowLimiter(daftarAgenIPLimit, time.Hour),
		bookingByAgen:  newWindowLimiter(agenBookingLimit, time.Hour),
	}
}

func getClientIP(r *http.Request) string {
	return shared.ClientIP(r)
}

// CheckPhone memeriksa status nomor HP jamaah (POST /api/public/jamaah/check).
func (h *Handler) CheckPhone(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.phoneCheckByIP.allow(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}
	h.phoneCheckByIP.record(clientIP)

	var req CheckPhoneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if req.BrandID == nil || *req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "brand_id wajib diisi")
		return
	}

	req.NoHP = strings.TrimSpace(req.NoHP)
	if !validPhone(req.NoHP) {
		writeError(w, http.StatusBadRequest, "nomor WhatsApp harus 10–15 digit")
		return
	}

	status, err := h.repo.CheckPhone(r.Context(), *req.BrandID, req.NoHP)
	if err != nil {
		log.Printf("[ERROR] CheckPhone: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(CheckPhoneResponse{Status: status})
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
			writeError(w, http.StatusNotFound, "invoice pendaftaran tidak ditemukan")
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

func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	clientIP := getClientIP(r)
	if !h.bookingByIP.allow(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak permintaan booking, coba lagi dalam 1 jam")
		return
	}
	h.bookingByIP.record(clientIP)

	// Optional Portal Auth Header
	var authenticatedJamaahID int64
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(authHeader) > 7 && strings.EqualFold(authHeader[:7], "Bearer ") {
		tokenStr := strings.TrimSpace(authHeader[7:])
		if jID, err := identity.ValidateAccountPortalToken(r.Context(), h.repo.db, tokenStr); err == nil && jID > 0 {
			authenticatedJamaahID = jID
		} else {
			writeError(w, 401, "sesi telah berakhir, silakan masuk kembali")
			return
		}
	}
	var req BookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
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
	req.PIC.PortalPIN = strings.TrimSpace(req.PIC.PortalPIN)

	if req.PIC.NamaLengkap == "" || req.PIC.NoHP == "" || req.PIC.JenisKelamin == "" || req.PIC.RoomType == "" {
		writeError(w, http.StatusBadRequest, "data pendaftar utama tidak lengkap")
		return
	}

	if authenticatedJamaahID == 0 && req.PIC.PortalPIN != "" && len(req.PIC.PortalPIN) != 6 {
		writeError(w, http.StatusBadRequest, "PIN portal harus 6 digit")
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

	// Gate 8: Batas booking per nomor HP PIC
	phoneKey, _ := shared.PhoneVariants(req.PIC.NoHP)
	if phoneKey == "" {
		phoneKey = req.PIC.NoHP
	}
	phoneKey = "brand:" + strconv.FormatInt(req.BrandID, 10) + "|" + phoneKey
	if !h.bookingByPhone.allow(phoneKey) {
		writeError(w, http.StatusTooManyRequests, "nomor ini sudah mencapai batas booking hari ini, silakan hubungi admin travel")
		return
	}

	// Process
	if msg := validateCheckout(&req, false); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	resp, err := h.repo.ProcessBooking(r.Context(), req.BrandID, req, authenticatedJamaahID)
	if err != nil {
		if errors.Is(err, ErrQuoteChanged) || errors.Is(err, ErrRequestConflict) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if errors.Is(err, ErrInvalidPin) {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
		if errors.Is(err, ErrSeatHabis) || errors.Is(err, ErrDuplicate) ||
			errors.Is(err, ErrAnggotaNameMismatch) || errors.Is(err, ErrDuplicatePaxInBooking) ||
			errors.Is(err, ErrPinRequired) || errors.Is(err, ErrCutoffBooking) ||
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
		h.bookingByPhone.record(phoneKey)
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
