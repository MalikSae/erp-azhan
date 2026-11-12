package agen

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/media"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

// validasiBuktiAdmin memastikan URL bukti yang diisi admin menunjuk unggahan admin resmi milik brand token
// (super admin: unggahan admin mana pun) dan berkasnya ada. false berarti jawaban galat sudah ditulis.
func validasiBuktiAdmin(w http.ResponseWriter, r *http.Request, db *sql.DB, url string, categories ...string) bool {
	err := media.ValidateAdminProof(r.Context(), db, url, identity.GetBrandID(r.Context()), categories...)
	if err == nil {
		return true
	}
	var uploadErr *media.UploadError
	if errors.As(err, &uploadErr) {
		writeError(w, http.StatusBadRequest, uploadErr.Message)
		return false
	}
	log.Printf("[ERROR] agen validasi bukti: %v", err)
	writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
	return false
}

// URL file hanya boleh hasil upload ke kategori media terproteksi, supaya
// pengguna tidak bisa mengisi URL sembarang (lihat media.processUpload).
var (
	nikAgenPattern = regexp.MustCompile(`^\d{16}$`)
	// FotoPortalURL: bentuk URL foto profil akun (hasil unggah ke /api/account/media/upload); dipakai paket customer.
	FotoPortalURL  = regexp.MustCompile(`^/api/admin/media/dokumen-jamaah/[0-9a-f-]{36}\.(jpg|png|webp|gif)$`)
	buktiPortalURL = regexp.MustCompile(`^/api/admin/media/dokumen-jamaah/[0-9a-f-]{36}\.(jpg|png|webp|gif|pdf)$`)
	buktiAdminURL  = regexp.MustCompile(`^/api/admin/media/(dokumen-jamaah|payment-proofs)/[0-9a-f-]{36}\.(jpg|png|webp|gif|pdf)$`)
)

// ─── Portal ───────────────────────────────────────────────────────────────────

// GetStatus GET /api/portal/agen
func (h *Handler) GetStatus(w http.ResponseWriter, r *http.Request) {
	s, err := h.repo.GetStatusPortal(r.Context(), identity.GetCustomerAccountID(r.Context()))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, s)
}

// Ajukan POST /api/portal/agen/pengajuan — screen A1a.
func (h *Handler) Ajukan(w http.ResponseWriter, r *http.Request) {
	var req AjukanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	req.Domisili = strings.TrimSpace(req.Domisili)
	req.NIK = strings.TrimSpace(req.NIK)

	if !nikAgenPattern.MatchString(req.NIK) {
		writeError(w, http.StatusBadRequest, ErrNIKAgenTidakValid.Error())
		return
	}
	if n := utf8.RuneCountInString(req.Domisili); n < 2 || n > 100 {
		writeError(w, http.StatusBadRequest, "domisili wajib diisi (2–100 karakter)")
		return
	}
	if msg := validasiBiodataAgen(&req, time.Now()); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if !req.SetujuSyaratKetentuan {
		writeError(w, http.StatusBadRequest, "anda wajib menyetujui Syarat & Ketentuan Agen")
		return
	}

	akunID := identity.GetCustomerAccountID(r.Context())
	if err := h.repo.Ajukan(r.Context(), akunID, req); err != nil {
		handleError(w, err)
		return
	}
	h.GetStatus(w, r)
}

// UploadBukti POST /api/portal/agen/pembayaran/bukti — screen A1b.
func (h *Handler) UploadBukti(w http.ResponseWriter, r *http.Request) {
	var req BuktiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	req.BuktiTransferURL = strings.TrimSpace(req.BuktiTransferURL)
	if !buktiPortalURL.MatchString(req.BuktiTransferURL) {
		writeError(w, http.StatusBadRequest, "bukti transfer wajib diunggah (gambar atau PDF)")
		return
	}
	akunID := identity.GetCustomerAccountID(r.Context())
	if ok, err := mediaMilikAkun(r.Context(), h.repo.db, akunID, req.BuktiTransferURL); err != nil || !ok {
		if err != nil {
			handleError(w, err)
			return
		}
		writeError(w, http.StatusBadRequest, "bukti transfer wajib diunggah (gambar atau PDF)")
		return
	}
	if err := h.repo.UploadBuktiAkun(r.Context(), akunID, req.BuktiTransferURL); err != nil {
		handleError(w, err)
		return
	}
	h.GetStatus(w, r)
}

// ─── Admin Travel ─────────────────────────────────────────────────────────────

// ListPengajuan GET /api/admin/agen/pengajuan — screen B1.
func (h *Handler) ListPengajuan(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListPengajuan(r.Context(), identity.GetBrandID(r.Context()))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ListAgenAktif GET /api/admin/agen/aktif?brand_id=&q= — sumber picker
// "Kaitkan ke Agen" (screen B4/C3). Admin Travel selalu brand-nya sendiri;
// Super Admin wajib menyebut brand_id jamaah yang sedang diinput.
func (h *Handler) ListAgenAktif(w http.ResponseWriter, r *http.Request) {
	brandID := identity.GetBrandID(r.Context())
	if brandID == nil {
		id, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusBadRequest, "brand_id wajib diisi")
			return
		}
		brandID = &id
	}
	items, err := ListAgenAktif(r.Context(), h.repo.db, *brandID, r.URL.Query().Get("q"))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// Setujui POST /api/admin/agen/{akunID}/setujui — selalu bisa, tanpa gate pembayaran.
func (h *Handler) Setujui(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	if err := h.repo.Setujui(r.Context(), id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context())); err != nil {
		handleError(w, err)
		return
	}
	log.Printf("[AUDIT] agen_disetujui akun_id=%d admin_id=%d", id, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "pengajuan agen disetujui"})
}

// Tolak POST /api/admin/agen/{akunID}/tolak
func (h *Handler) Tolak(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	alasan, ok := readAlasan(w, r, "alasan penolakan keagenan wajib diisi (minimal 3 karakter)")
	if !ok {
		return
	}
	if err := h.repo.Tolak(r.Context(), id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()), alasan); err != nil {
		handleError(w, err)
		return
	}
	log.Printf("[AUDIT] agen_ditolak akun_id=%d admin_id=%d", id, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "pengajuan agen ditolak"})
}

// UbahStatus PUT /api/admin/agen/{akunID}/status — toggle aktif/nonaktif.
func (h *Handler) UbahStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	var req StatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	if err := h.repo.UbahStatus(r.Context(), id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()), strings.TrimSpace(req.Status), req.Alasan); err != nil {
		handleError(w, err)
		return
	}
	log.Printf("[AUDIT] agen_status akun_id=%d status=%s admin_id=%d", id, req.Status, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "status agen diperbarui"})
}

// VerifikasiPembayaran POST /api/admin/agen/pembayaran/{id}/verifikasi
func (h *Handler) VerifikasiPembayaran(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	if err := h.repo.VerifikasiPembayaran(r.Context(), id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context())); err != nil {
		handleError(w, err)
		return
	}
	log.Printf("[AUDIT] pembayaran_agen_diverifikasi akun_id=%d admin_id=%d", id, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "pembayaran pendaftaran terverifikasi"})
}

// TolakPembayaran POST /api/admin/agen/pembayaran/{id}/tolak
func (h *Handler) TolakPembayaran(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	catatan, ok := readAlasan(w, r, "catatan penolakan pembayaran wajib diisi (minimal 3 karakter)")
	if !ok {
		return
	}
	if err := h.repo.TolakPembayaran(r.Context(), id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()), catatan); err != nil {
		handleError(w, err)
		return
	}
	log.Printf("[AUDIT] pembayaran_agen_ditolak akun_id=%d admin_id=%d", id, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "pembayaran pendaftaran ditolak"})
}

// UploadBuktiAdmin POST /api/admin/agen/pembayaran/{id}/bukti
func (h *Handler) UploadBuktiAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "id")
	if !ok {
		return
	}
	var req BuktiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	req.BuktiTransferURL = strings.TrimSpace(req.BuktiTransferURL)
	if !buktiAdminURL.MatchString(req.BuktiTransferURL) {
		writeError(w, http.StatusBadRequest, "bukti transfer wajib diunggah (gambar atau PDF)")
		return
	}
	// Bentuk URL saja tidak cukup: berkas harus unggahan admin resmi milik brand ini.
	if !validasiBuktiAdmin(w, r, h.repo.db, req.BuktiTransferURL, "dokumen-jamaah", "payment-proofs") {
		return
	}
	if err := h.repo.UploadBuktiAdmin(r.Context(), id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()), req.BuktiTransferURL); err != nil {
		handleError(w, err)
		return
	}
	log.Printf("[AUDIT] bukti_pembayaran_agen_diunggah_admin akun_id=%d admin_id=%d", id, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "bukti transfer tersimpan"})
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func readAlasan(w http.ResponseWriter, r *http.Request, msg string) (string, bool) {
	var req AlasanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return "", false
	}
	alasan := strings.TrimSpace(req.Alasan)
	if utf8.RuneCountInString(alasan) < 3 {
		writeError(w, http.StatusBadRequest, msg)
		return "", false
	}
	return alasan, true
}

func parseID(w http.ResponseWriter, r *http.Request, param string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, param), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return id, true
}

func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrTidakAdaPengajuan), errors.Is(err, ErrAksiTidakValid), errors.Is(err, ErrCatatanWajib),
		errors.Is(err, ErrNIKAgenTidakValid), errors.Is(err, ErrNIKAgenTidakBisaDipakai):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrStatusTidakValid), errors.Is(err, ErrAlasanStatusWajib):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrSudahMengajukan), errors.Is(err, ErrBukanPengajuan),
		errors.Is(err, ErrPembayaranFinal), errors.Is(err, ErrPembayaranBukanMenunggu),
		errors.Is(err, ErrSiklusDitutup), errors.Is(err, ErrKeputusanTidakBisa), errors.Is(err, ErrBuktiBelumAda), errors.Is(err, ErrStatusSama):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("[ERROR] agen: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan internal")
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
