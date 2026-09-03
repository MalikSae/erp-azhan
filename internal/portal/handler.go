package portal

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/dokumen"
	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/jamaah"
	"erp-azhan/api/internal/payment"
	"erp-azhan/api/internal/shared"
	"github.com/go-chi/chi/v5"
	"golang.org/x/crypto/bcrypt"
)

type loginAttempt struct {
	Count     int
	FirstFail time.Time
}

type Handler struct {
	db            *sql.DB
	jamaahRepo    *jamaah.Repository
	bookingRepo   *booking.Repository
	paymentRepo   *payment.Repository
	dokumenRepo   *dokumen.Repository
	failedLogins  map[string]*loginAttempt
	failedLoginMu sync.Mutex
}

func NewHandler(
	db *sql.DB,
	jamaahRepo *jamaah.Repository,
	bookingRepo *booking.Repository,
	paymentRepo *payment.Repository,
	dokumenRepo *dokumen.Repository,
) *Handler {
	return &Handler{
		db:           db,
		jamaahRepo:   jamaahRepo,
		bookingRepo:  bookingRepo,
		paymentRepo:  paymentRepo,
		dokumenRepo:  dokumenRepo,
		failedLogins: make(map[string]*loginAttempt),
	}
}

// ─── Rate Limiter Portal (5x per 15 menit per IP) ────────────────────────────

func (h *Handler) checkRateLimit(ip string) bool {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()

	attempt, exists := h.failedLogins[ip]
	if !exists {
		return true
	}

	if time.Since(attempt.FirstFail) > 15*time.Minute {
		delete(h.failedLogins, ip)
		return true
	}

	return attempt.Count < 5
}

func (h *Handler) recordFailedLogin(ip string) {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()

	attempt, exists := h.failedLogins[ip]
	if !exists || time.Since(attempt.FirstFail) > 15*time.Minute {
		h.failedLogins[ip] = &loginAttempt{
			Count:     1,
			FirstFail: time.Now(),
		}
		return
	}

	attempt.Count++
}

func (h *Handler) resetFailedLogin(ip string) {
	h.failedLoginMu.Lock()
	defer h.failedLoginMu.Unlock()
	delete(h.failedLogins, ip)
}

func getClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return xrip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// ─── Login Portal ─────────────────────────────────────────────────────────────

type PortalLoginRequest struct {
	BrandID    *int64 `json:"brand_id"`
	Identifier string `json:"identifier"`
	IDJamaah   string `json:"id_jamaah"`
	PortalPIN  string `json:"portal_pin"`
}

type PortalLoginResponse struct {
	AccessToken string              `json:"access_token"`
	Jamaah      PortalJamaahSummary `json:"jamaah"`
}

type PortalJamaahSummary struct {
	ID          int64  `json:"id"`
	IDJamaah    string `json:"id_jamaah"`
	NamaLengkap string `json:"nama_lengkap"`
	BrandID     int64  `json:"brand_id"`
}

func isPhoneNumber(s string) bool {
	s = strings.TrimPrefix(s, "+")
	cleaned := strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '-' || r == '(' || r == ')' {
			return -1
		}
		return r
	}, s)
	if len(cleaned) == 0 {
		return false
	}
	for _, r := range cleaned {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	clientIP := getClientIP(r)
	if !h.checkRateLimit(clientIP) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan login yang gagal, coba lagi dalam 15 menit")
		return
	}

	var req PortalLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	if req.BrandID == nil || *req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "brand_id wajib diisi")
		return
	}

	identifier := strings.TrimSpace(req.Identifier)
	if identifier == "" {
		identifier = strings.TrimSpace(req.IDJamaah)
	}
	pin := strings.TrimSpace(req.PortalPIN)

	const invalidCredentialsMsg = "ID jamaah, nomor WhatsApp, atau PIN tidak cocok"

	if identifier == "" || pin == "" {
		h.recordFailedLogin(clientIP)
		writeError(w, http.StatusUnauthorized, invalidCredentialsMsg)
		return
	}

	var j PortalJamaahSummary
	var pinHash sql.NullString

	if isPhoneNumber(identifier) {
		canonical, local := shared.PhoneVariants(identifier)
		if canonical == "" || local == "" {
			h.recordFailedLogin(clientIP)
			writeError(w, http.StatusUnauthorized, invalidCredentialsMsg)
			return
		}

		phoneQuery := `SELECT id, COALESCE(id_jamaah, ''), nama_lengkap, brand_id, portal_pin_hash FROM jamaah 
			WHERE brand_id = ? 
			  AND no_hp IS NOT NULL 
			  AND no_hp != '' 
			  AND REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', '') IN (?, ?) 
			LIMIT 2`

		rows, err := h.db.QueryContext(r.Context(), phoneQuery, *req.BrandID, canonical, local)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
			return
		}
		defer rows.Close()

		type matchedJamaah struct {
			summary PortalJamaahSummary
			hash    sql.NullString
		}
		var matches []matchedJamaah
		for rows.Next() {
			var m matchedJamaah
			if err := rows.Scan(&m.summary.ID, &m.summary.IDJamaah, &m.summary.NamaLengkap, &m.summary.BrandID, &m.hash); err != nil {
				writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
				return
			}
			matches = append(matches, m)
		}
		if err := rows.Err(); err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
			return
		}

		if len(matches) != 1 {
			h.recordFailedLogin(clientIP)
			writeError(w, http.StatusUnauthorized, invalidCredentialsMsg)
			return
		}

		j = matches[0].summary
		pinHash = matches[0].hash
	} else {
		query := `SELECT id, COALESCE(id_jamaah, ''), nama_lengkap, brand_id, portal_pin_hash FROM jamaah 
			WHERE brand_id = ? AND UPPER(id_jamaah) = UPPER(?)`

		err := h.db.QueryRowContext(r.Context(), query, *req.BrandID, identifier).Scan(&j.ID, &j.IDJamaah, &j.NamaLengkap, &j.BrandID, &pinHash)
		if errors.Is(err, sql.ErrNoRows) {
			h.recordFailedLogin(clientIP)
			writeError(w, http.StatusUnauthorized, invalidCredentialsMsg)
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
			return
		}
	}

	if !pinHash.Valid || pinHash.String == "" {
		h.recordFailedLogin(clientIP)
		writeError(w, http.StatusUnauthorized, invalidCredentialsMsg)
		return
	}

	// Verifikasi bcrypt hash
	if err := bcrypt.CompareHashAndPassword([]byte(pinHash.String), []byte(pin)); err != nil {
		h.recordFailedLogin(clientIP)
		writeError(w, http.StatusUnauthorized, invalidCredentialsMsg)
		return
	}

	// Login sukses -> reset counter
	h.resetFailedLogin(clientIP)

	token, err := identity.GeneratePortalToken(j.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal membuat token portal")
		return
	}

	writeJSON(w, http.StatusOK, PortalLoginResponse{
		AccessToken: token,
		Jamaah:      j,
	})
}

// ─── GET /api/portal/me ───────────────────────────────────────────────────────

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	jamaahID := identity.GetPortalJamaahID(r.Context())
	if jamaahID <= 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	j, err := h.jamaahRepo.GetByID(r.Context(), jamaahID, nil)
	if errors.Is(err, jamaah.ErrNotFound) {
		writeError(w, http.StatusNotFound, "data jamaah tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}

	writeJSON(w, http.StatusOK, j)
}

// ─── GET /api/portal/bookings ─────────────────────────────────────────────────

func (h *Handler) ListBookings(w http.ResponseWriter, r *http.Request) {
	jamaahID := identity.GetPortalJamaahID(r.Context())
	if jamaahID <= 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bookings, err := h.bookingRepo.List(r.Context(), nil, &jamaahID, "non_draft")
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar booking")
		return
	}

	if bookings == nil {
		bookings = make([]booking.Booking, 0)
	}

	writeJSON(w, http.StatusOK, bookings)
}

// canAccessBooking memeriksa apakah seorang jamaah berhak atas booking (sebagai PIC atau anggota pax).
func (h *Handler) canAccessBooking(ctx context.Context, bookingID int64, jamaahID int64) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (
		SELECT 1 FROM bookings b 
		WHERE b.id = ? 
		  AND (b.pic_jamaah_id = ? OR EXISTS (SELECT 1 FROM booking_pax bp2 WHERE bp2.booking_id = b.id AND bp2.jamaah_id = ?))
	)`
	err := h.db.QueryRowContext(ctx, query, bookingID, jamaahID, jamaahID).Scan(&exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// ─── GET /api/portal/bookings/{id} ───────────────────────────────────────────

func (h *Handler) GetBookingByID(w http.ResponseWriter, r *http.Request) {
	jamaahID := identity.GetPortalJamaahID(r.Context())
	if jamaahID <= 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	bookingID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || bookingID <= 0 {
		writeError(w, http.StatusBadRequest, "id booking tidak valid")
		return
	}

	allowed, err := h.canAccessBooking(r.Context(), bookingID, jamaahID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data booking")
		return
	}
	if !allowed {
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
		return
	}

	b, err := h.bookingRepo.GetByID(r.Context(), bookingID, nil)
	if errors.Is(err, booking.ErrNotFound) || (b != nil && b.Status == "draft") {
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil data booking")
		return
	}

	writeJSON(w, http.StatusOK, b)
}

// ─── GET /api/portal/bookings/{id}/payments ───────────────────────────────────

func (h *Handler) ListPayments(w http.ResponseWriter, r *http.Request) {
	jamaahID := identity.GetPortalJamaahID(r.Context())
	if jamaahID <= 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := chi.URLParam(r, "id")
	bookingID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || bookingID <= 0 {
		writeError(w, http.StatusBadRequest, "id booking tidak valid")
		return
	}

	allowed, err := h.canAccessBooking(r.Context(), bookingID, jamaahID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memverifikasi booking")
		return
	}
	if !allowed {
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
		return
	}

	payments, err := h.paymentRepo.ListByBookingID(r.Context(), bookingID, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil riwayat pembayaran")
		return
	}

	if payments == nil {
		payments = make([]payment.Payment, 0)
	}

	writeJSON(w, http.StatusOK, payments)
}

// ListBankAccounts mengembalikan rekening aktif milik brand jamaah.
func (h *Handler) ListBankAccounts(w http.ResponseWriter, r *http.Request) {
	jamaahID := identity.GetPortalJamaahID(r.Context())
	if jamaahID <= 0 {
		writeError(w, 401, "unauthorized")
		return
	}
	rows, err := h.db.QueryContext(r.Context(), `SELECT a.id,a.bank_name,a.account_number,a.account_holder,a.instructions FROM bank_accounts a JOIN jamaah j ON j.brand_id=a.brand_id WHERE j.id=? AND a.is_active=TRUE ORDER BY a.sort_order,a.id`, jamaahID)
	if err != nil {
		writeError(w, 500, "gagal mengambil rekening")
		return
	}
	defer rows.Close()
	type account struct {
		ID            int64   `json:"id"`
		BankName      string  `json:"bank_name"`
		AccountNumber string  `json:"account_number"`
		AccountHolder string  `json:"account_holder"`
		Instructions  *string `json:"instructions"`
	}
	items := []account{}
	for rows.Next() {
		var a account
		if rows.Scan(&a.ID, &a.BankName, &a.AccountNumber, &a.AccountHolder, &a.Instructions) != nil {
			writeError(w, 500, "gagal membaca rekening")
			return
		}
		items = append(items, a)
	}
	writeJSON(w, 200, items)
}

// CreatePayment menerima konfirmasi transfer manual dari jamaah.
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	jamaahID := identity.GetPortalJamaahID(r.Context())
	bookingID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if jamaahID <= 0 || err != nil || bookingID <= 0 {
		writeError(w, 400, "booking tidak valid")
		return
	}
	b, err := h.bookingRepo.GetByID(r.Context(), bookingID, nil)
	if err != nil || b.JamaahID == nil || *b.JamaahID != jamaahID {
		writeError(w, 404, "booking tidak ditemukan")
		return
	}
	var req payment.CreatePaymentRequest
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		writeError(w, 400, "request body tidak valid")
		return
	}
	req.Source = "portal"
	if req.Jumlah <= 0 || req.BankAccountID == nil || req.SenderName == nil || strings.TrimSpace(*req.SenderName) == "" || req.BuktiURL == nil || strings.TrimSpace(*req.BuktiURL) == "" {
		writeError(w, 400, "rekening tujuan, nominal, nama pengirim, dan bukti transfer wajib diisi")
		return
	}
	var accountBrand, bookingBrand int64
	err = h.db.QueryRowContext(r.Context(), `SELECT brand_id FROM bank_accounts WHERE id=? AND is_active=TRUE`, *req.BankAccountID).Scan(&accountBrand)
	if err == nil {
		err = h.db.QueryRowContext(r.Context(), `SELECT s.brand_id FROM bookings b JOIN schedules s ON s.id=b.schedule_id WHERE b.id=?`, bookingID).Scan(&bookingBrand)
	}
	if err != nil || accountBrand != bookingBrand {
		writeError(w, 400, "rekening tujuan tidak tersedia untuk brand ini")
		return
	}
	status, total, totalPaid, err := h.paymentRepo.GetBookingTotalAndPaid(r.Context(), bookingID)
	if err != nil {
		writeError(w, 500, "gagal memeriksa tagihan")
		return
	}
	if status == "draft" {
		writeError(w, 400, "Booking ini masih berstatus draft, pembayaran belum bisa diproses. Selesaikan/finalisasi booking terlebih dahulu.")
		return
	}
	if total == nil {
		writeError(w, 400, "Total tagihan booking belum ditentukan, pembayaran tidak dapat diproses.")
		return
	}
	sisa := *total - totalPaid
	if sisa <= 0 {
		writeError(w, 400, "tagihan booking sudah lunas")
		return
	}
	if req.Jumlah > sisa {
		writeError(w, 400, "nominal melebihi sisa tagihan")
		return
	}
	item, err := h.paymentRepo.Create(r.Context(), bookingID, &req, nil)
	if err != nil {
		writeError(w, 500, "gagal menyimpan konfirmasi pembayaran")
		return
	}
	writeJSON(w, 201, item)
}

// ─── GET /api/portal/dokumen ──────────────────────────────────────────────────

func (h *Handler) ListDokumen(w http.ResponseWriter, r *http.Request) {
	jamaahID := identity.GetPortalJamaahID(r.Context())
	if jamaahID <= 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	docs, err := h.dokumenRepo.ListByJamaahID(r.Context(), jamaahID, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar dokumen")
		return
	}

	if docs == nil {
		docs = make([]dokumen.DokumenJamaah, 0)
	}

	writeJSON(w, http.StatusOK, docs)
}

// ─── POST /api/portal/dokumen ─────────────────────────────────────────────────

func (h *Handler) UploadDokumen(w http.ResponseWriter, r *http.Request) {
	jamaahID := identity.GetPortalJamaahID(r.Context())
	if jamaahID <= 0 {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req dokumen.CreateDokumenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	req.Jenis = strings.TrimSpace(req.Jenis)
	req.FileURL = strings.TrimSpace(req.FileURL)

	if req.Jenis == "" || req.FileURL == "" {
		writeError(w, http.StatusBadRequest, "jenis dan file_url wajib diisi")
		return
	}

	validJenis := map[string]bool{
		"paspor":            true,
		"ktp":               true,
		"kk":                true,
		"buku_nikah":        true,
		"pas_foto":          true,
		"vaksin_meningitis": true,
	}
	if !validJenis[req.Jenis] {
		writeError(w, http.StatusBadRequest, "jenis dokumen tidak valid")
		return
	}

	doc, err := h.dokumenRepo.Upsert(r.Context(), jamaahID, &req)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan dokumen")
		return
	}

	writeJSON(w, http.StatusOK, doc)
}

// ─── Helper JSON response ────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
