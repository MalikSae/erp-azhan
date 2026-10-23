package media

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// uploadQuota membatasi jumlah berkas dan total byte per kunci dalam jendela waktu. reserve
// memeriksa dan mencatat secara atomik (tidak bisa diakali permintaan paralel).
type uploadQuota struct {
	mu        sync.Mutex
	files     int
	bytes     int64
	window    time.Duration
	pemakaian map[string]*quotaUse
}

type quotaUse struct {
	files int
	bytes int64
	start time.Time
}

func newUploadQuota(files int, bytes int64, window time.Duration) *uploadQuota {
	return &uploadQuota{files: files, bytes: bytes, window: window, pemakaian: map[string]*quotaUse{}}
}

func (q *uploadQuota) reserve(key string, size int64) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	now := time.Now()
	if len(q.pemakaian) > 10000 {
		for k, u := range q.pemakaian {
			if now.Sub(u.start) > q.window {
				delete(q.pemakaian, k)
			}
		}
	}
	u, ok := q.pemakaian[key]
	if !ok || now.Sub(u.start) > q.window {
		u = &quotaUse{start: now}
		q.pemakaian[key] = u
	}
	if u.files+1 > q.files || u.bytes+size > q.bytes {
		return false
	}
	u.files++
	u.bytes += size
	return true
}

var validCategoryRegex = regexp.MustCompile(`^[a-zA-Z0-9\-]+$`)

// Batas unggahan (tinjauan keamanan 2026-10-22). Satu tempat untuk semua jalur unggah.
const (
	// MaxUploadBytes: ukuran berkas maksimum (foto bukti transfer dari HP biasanya < 10 MB;
	// azhan-booking juga membatasi 10 MB di sisi klien).
	MaxUploadBytes = 10 << 20
	// multipartOverhead: ruang untuk boundary dan field lain di body multipart.
	multipartOverhead = 512 << 10

	customerUploadFilesPerDay = 20        // berkas per akun pelanggan per 24 jam
	customerUploadBytesPerDay = 100 << 20 // total byte per akun pelanggan per 24 jam
	customerUploadIPPerHour   = 60        // unggahan pelanggan per IP per jam
	adminUploadIPPerHour      = 300       // unggahan admin per IP per jam
)

// adminUploadCategoryPerms: kategori yang boleh diunggah lewat /api/admin/media/upload beserta
// izin yang dibutuhkan (salah satu cukup). Kategori di luar daftar ditolak. Disusun dari pemanggil
// nyata di dashboard (frontend/*/uploadMedia) dan route yang memakai URL-nya.
var adminUploadCategoryPerms = map[string][]string{
	"dokumen-jamaah":  {"dokumen.edit"},
	"payment-proofs":  {"payment.edit", "agen.approve", "agen.payout.approve"},
	"brand-logos":     {"brand.edit"},
	"brand-icons":     {"brand.edit"},
	"brand-og":        {"brand.edit"},
	"bank-logos":      {"bankaccount.edit"},
	"hotel-photos":    {"masterdata.edit"},
	"hotel-logos":     {"masterdata.edit"},
	"airline-logos":   {"masterdata.edit"},
	"perlengkapan":    {"masterdata.edit"},
	"schedule-brosur": {"schedule.edit"},
}

// protectedViewPerms: izin untuk GET /api/admin/media/{category}/{file} (berlaku juga untuk user
// holding tanpa brand). Bukti bayar pelanggan disimpan di kategori dokumen-jamaah (unggahan akun),
// jadi peran keuangan/agen boleh membuka berkas dokumen-jamaah yang BUKAN dokumen identitas
// (tidak tercatat di dokumen_jamaah), lihat canViewCategory.
var protectedViewPerms = map[string][]string{
	"dokumen-jamaah": {"dokumen.view"},
	"payment-proofs": {"payment.view", "agen.view", "agen.approve", "agen.payout.approve"},
}

var buktiViewPerms = []string{"payment.view", "agen.view", "agen.approve", "agen.payout.approve"}

// Handler struct
type Handler struct {
	db            *sql.DB
	customerQuota *uploadQuota
	customerIP    *shared.WindowLimiter
	adminIP       *shared.WindowLimiter
}

var publicCategories = map[string]struct{}{
	"brand-logos":     {},
	"brand-icons":     {},
	"hotel-logos":     {},
	"hotel-photos":    {},
	"airline-logos":   {},
	"bank-logos":      {},
	"schedule-brosur": {},
}

var protectedCategories = map[string]struct{}{
	"dokumen-jamaah": {},
	"payment-proofs": {},
}

// NewHandler creates a new media handler
func NewHandler(db *sql.DB) *Handler {
	return &Handler{
		db:            db,
		customerQuota: newUploadQuota(customerUploadFilesPerDay, customerUploadBytesPerDay, 24*time.Hour),
		customerIP:    shared.NewWindowLimiter(customerUploadIPPerHour, time.Hour),
		adminIP:       shared.NewWindowLimiter(adminUploadIPPerHour, time.Hour),
	}
}

func hasAnyPermission(ctx context.Context, perms []string) bool {
	for _, p := range perms {
		if identity.HasPermission(ctx, p) {
			return true
		}
	}
	return false
}

// canViewCategory memeriksa izin baca berkas terlindungi sesuai kategorinya.
func (h *Handler) canViewCategory(r *http.Request, category, name string) bool {
	ctx := r.Context()
	if hasAnyPermission(ctx, protectedViewPerms[category]) {
		return true
	}
	if category != "dokumen-jamaah" || !hasAnyPermission(ctx, buktiViewPerms) {
		return false
	}
	// Tanpa dokumen.view: hanya berkas yang bukan dokumen identitas jamaah (mis. bukti transfer).
	var dokumen bool
	err := h.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM dokumen_jamaah WHERE file_url IN (?,?))`,
		"/api/admin/media/"+category+"/"+name, "/uploads/"+category+"/"+name).Scan(&dokumen)
	return err == nil && !dokumen
}

// ServePublic serves only non-sensitive branding media. It deliberately
// rejects directory requests so uploads can never be enumerated.
func (h *Handler) ServePublic(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	filename := chi.URLParam(r, "filename")
	if _, ok := publicCategories[category]; !ok || !safeFilename(filename) {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(".", "uploads", category, filename))
}

// ServeProtected serves sensitive media only after the admin middleware has
// authenticated the request. The file remains in the existing uploads folder;
// this changes exposure, not stored data.
func (h *Handler) ServeProtected(w http.ResponseWriter, r *http.Request) {
	category := chi.URLParam(r, "category")
	filename := chi.URLParam(r, "filename")
	if _, ok := protectedCategories[category]; !ok || !safeFilename(filename) {
		http.NotFound(w, r)
		return
	}
	if !h.canViewCategory(r, category, filename) || !h.canReadAdminFile(r, category, filename) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	path := filepath.Join(".", "uploads", category, filename)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func safeFilename(name string) bool {
	return name != "" && name != "." && name != ".." && filepath.Base(name) == name && !strings.ContainsAny(name, `/\\`)
}

// UploadMedia godoc
// POST /api/admin/media/upload
func (h *Handler) UploadMedia(w http.ResponseWriter, r *http.Request) {
	if ip := shared.ClientIP(r); h.adminIP != nil {
		if !h.adminIP.Allow(ip) {
			writeError(w, http.StatusTooManyRequests, "terlalu banyak unggahan, coba lagi nanti")
			return
		}
		h.adminIP.Record(ip)
	}
	// Batasi body SEBELUM FormValue mem-parse multipart.
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes+multipartOverhead)
	if err := r.ParseMultipartForm(MaxUploadBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "ukuran file terlalu besar (maksimal 10MB)")
		return
	}
	category := r.FormValue("category")
	if strings.TrimSpace(category) == "" {
		writeError(w, http.StatusBadRequest, "category wajib diisi")
		return
	}
	perms, known := adminUploadCategoryPerms[category]
	if !validCategoryRegex.MatchString(category) || !known {
		writeError(w, http.StatusBadRequest, "category tidak valid")
		return
	}
	if !hasAnyPermission(r.Context(), perms) {
		writeError(w, http.StatusForbidden, "anda tidak memiliki izin untuk mengunggah ke kategori ini")
		return
	}

	h.processUpload(w, r, category, "")
}

// UploadPortalMedia godoc
// POST /api/account/media/upload
// Category is strictly forced to "dokumen-jamaah" regardless of request payload.
// Dibatasi per IP dan kuota harian per akun pelanggan (jumlah berkas dan total byte).
func (h *Handler) UploadPortalMedia(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	if h.customerIP != nil {
		if !h.customerIP.Allow(ip) {
			writeError(w, http.StatusTooManyRequests, "terlalu banyak unggahan, coba lagi dalam 1 jam")
			return
		}
		h.customerIP.Record(ip)
	}
	key := "ip:" + ip
	if id := identity.GetCustomerAccountID(r.Context()); id > 0 {
		key = "akun:" + strconv.FormatInt(id, 10)
	}
	r.Body = http.MaxBytesReader(w, r.Body, MaxUploadBytes+multipartOverhead)
	if err := r.ParseMultipartForm(MaxUploadBytes); err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "ukuran file terlalu besar (maksimal 10MB)")
		return
	}
	h.processUpload(w, r, "dokumen-jamaah", key)
}

// processUpload menyimpan berkas dari form yang SUDAH di-parse pemanggil (dengan batas ukuran).
// quotaKey kosong = tanpa kuota harian (unggahan admin).
func (h *Handler) processUpload(w http.ResponseWriter, r *http.Request, category, quotaKey string) {
	// 2. Get the file
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file tidak ditemukan dalam request")
		return
	}
	defer file.Close()
	if header.Size > MaxUploadBytes {
		writeError(w, http.StatusRequestEntityTooLarge, "ukuran file terlalu besar (maksimal 10MB)")
		return
	}
	if quotaKey != "" && h.customerQuota != nil && !h.customerQuota.reserve(quotaKey, header.Size) {
		writeError(w, http.StatusTooManyRequests, "kuota unggahan harian akun Anda sudah habis, coba lagi besok atau hubungi admin")
		return
	}

	// Read first 512 bytes for magic bytes to detect content type
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		writeError(w, http.StatusInternalServerError, "gagal membaca file")
		return
	}

	contentType := http.DetectContentType(buffer[:n])
	extensions := map[string]string{
		"image/jpeg": ".jpg", "image/png": ".png", "image/webp": ".webp", "image/gif": ".gif",
	}
	ext, isImage := extensions[contentType]
	if contentType == "application/pdf" && category == "dokumen-jamaah" {
		ext = ".pdf"
	}
	if ext == "" {
		writeError(w, http.StatusBadRequest, "file harus berupa JPG, PNG, WEBP, GIF, atau PDF")
		return
	}

	// Rewind the file
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal memproses file")
		return
	}

	// 3. Generate unique name
	newFileName := uuid.New().String() + ext
	uploadDir := filepath.Join(".", "uploads", category)

	// 4. Create folder if not exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		fmt.Printf("Gagal membuat direktori upload: %v\n", err)
		writeError(w, http.StatusInternalServerError, "gagal memproses gambar")
		return
	}

	finalPath := filepath.Join(uploadDir, newFileName)
	destination, err := os.OpenFile(finalPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal menyimpan file")
		return
	}
	if _, err := io.Copy(destination, file); err != nil {
		destination.Close()
		_ = os.Remove(finalPath)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan file")
		return
	}
	if err := destination.Close(); err != nil {
		_ = os.Remove(finalPath)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan file")
		return
	}

	publicURL := fmt.Sprintf("/uploads/%s/%s", category, newFileName)
	if _, ok := protectedCategories[category]; ok {
		publicURL = fmt.Sprintf("/api/admin/media/%s/%s", category, newFileName)
		var brandID any = identity.GetBrandID(r.Context())
		var jamaahID, adminID, customerID any
		if id := identity.GetCustomerAccountID(r.Context()); id > 0 {
			customerID = id
			brandID = identity.GetCustomerBrandID(r.Context())
		} else if id := identity.GetPortalJamaahID(r.Context()); id > 0 {
			jamaahID = id
			var brand int64
			if err := h.db.QueryRowContext(r.Context(), `SELECT brand_id FROM jamaah WHERE id=?`, id).Scan(&brand); err != nil {
				_ = os.Remove(finalPath)
				writeError(w, 500, "gagal memeriksa pemilik berkas")
				return
			}
			brandID = brand
		} else {
			adminID = identity.GetAdminUserID(r.Context())
		}
		if _, err := h.db.ExecContext(r.Context(), `INSERT INTO media_uploads(path,brand_id,jamaah_id,admin_user_id,customer_account_id) VALUES(?,?,?,?,?)`, publicURL, brandID, jamaahID, adminID, customerID); err != nil {
			_ = os.Remove(finalPath)
			writeError(w, 500, "gagal mencatat pemilik berkas")
			return
		}
	}
	response := map[string]string{"url": publicURL}

	// Format asli dipertahankan agar upload tidak bergantung pada binary eksternal.
	_ = isImage

	// 8. Success
	writeJSON(w, http.StatusCreated, response)
}

// ─── Shared helpers ───────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
