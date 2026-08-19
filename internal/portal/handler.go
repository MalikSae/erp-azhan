package portal

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/dokumen"
	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/jamaah"
	"erp-azhan/api/internal/payment"
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
	BrandID     *int64 `json:"brand_id"`
	NamaLengkap string `json:"nama_lengkap"`
	IDJamaah    string `json:"id_jamaah"`
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

	req.NamaLengkap = strings.TrimSpace(req.NamaLengkap)
	req.IDJamaah = strings.TrimSpace(req.IDJamaah)

	if req.NamaLengkap == "" || req.IDJamaah == "" {
		writeError(w, http.StatusBadRequest, "nama_lengkap dan id_jamaah wajib diisi")
		return
	}

	// Query: SELECT jamaah WHERE brand_id = ? AND UPPER(id_jamaah) = UPPER(?) AND UPPER(nama_lengkap) = UPPER(?)
	var j PortalJamaahSummary
	query := `SELECT id, COALESCE(id_jamaah, ''), nama_lengkap, brand_id FROM jamaah 
		WHERE brand_id = ? AND UPPER(id_jamaah) = UPPER(?) AND UPPER(nama_lengkap) = UPPER(?)`

	err := h.db.QueryRowContext(r.Context(), query, *req.BrandID, req.IDJamaah, req.NamaLengkap).Scan(&j.ID, &j.IDJamaah, &j.NamaLengkap, &j.BrandID)
	if errors.Is(err, sql.ErrNoRows) {
		h.recordFailedLogin(clientIP)
		writeError(w, http.StatusUnauthorized, "nama atau id jamaah tidak cocok")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan pada server")
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

	bookings, err := h.bookingRepo.List(r.Context(), nil, &jamaahID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar booking")
		return
	}

	if bookings == nil {
		bookings = make([]booking.Booking, 0)
	}

	writeJSON(w, http.StatusOK, bookings)
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

	b, err := h.bookingRepo.GetByID(r.Context(), bookingID, nil)
	if errors.Is(err, booking.ErrNotFound) || (b != nil && b.JamaahID != jamaahID) {
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

	// Verify booking belongs to this jamaah
	b, err := h.bookingRepo.GetByID(r.Context(), bookingID, nil)
	if errors.Is(err, booking.ErrNotFound) || (b != nil && b.JamaahID != jamaahID) {
		writeError(w, http.StatusNotFound, "data tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memverifikasi booking")
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
		"foto":              true,
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
