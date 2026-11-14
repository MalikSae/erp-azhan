package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"erp-azhan/api/internal/mailer"
	"erp-azhan/api/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// Batas laju (di memori): daftar per IP, login per IP+email dan per IP, lupa/ kirim ulang per kunci.
const (
	registerIPLimit    = 10 // per jam
	loginKeyLimit      = 5  // gagal per 15 menit per IP+email
	loginIPLimit       = 30 // gagal per 15 menit per IP
	forgotKeyLimit     = 5  // per jam per IP+email
	profileUpdateLimit = 10 // ubah data akun per jam per akun
	passwordFailLimit  = 5  // password saat ini salah per 15 menit per akun
	// Lupa password tidak memakai captcha (frontend tidak mengirim token), jadi dibatasi lebih
	// ketat: per email (lintas IP) dan per IP (lintas email), selain per IP+email di atas.
	forgotEmailLimit = 3  // per jam per brand+email
	forgotIPLimit    = 20 // per jam per IP
)

type Handler struct {
	repo        *Repository
	mail        mailer.Sender
	captcha     *shared.Turnstile
	regIP       *shared.WindowLimiter
	loginKey    *shared.WindowLimiter
	loginIP     *shared.WindowLimiter
	forgot      *shared.WindowLimiter
	forgotEmail *shared.WindowLimiter
	forgotIP    *shared.WindowLimiter
	profile     *shared.WindowLimiter
	pwFail      *shared.WindowLimiter
	dummyHash   []byte
}

func NewHandler(repo *Repository, m mailer.Sender) *Handler {
	// Hash tiruan agar login untuk email yang tidak ada memakan waktu sama dengan yang ada.
	dummy, _ := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcryptCost)
	return &Handler{
		repo: repo, mail: m, captcha: shared.NewTurnstile(),
		regIP:       shared.NewWindowLimiter(registerIPLimit, time.Hour),
		loginKey:    shared.NewWindowLimiter(loginKeyLimit, 15*time.Minute),
		loginIP:     shared.NewWindowLimiter(loginIPLimit, 15*time.Minute),
		forgot:      shared.NewWindowLimiter(forgotKeyLimit, time.Hour),
		forgotEmail: shared.NewWindowLimiter(forgotEmailLimit, time.Hour),
		forgotIP:    shared.NewWindowLimiter(forgotIPLimit, time.Hour),
		profile:     shared.NewWindowLimiter(profileUpdateLimit, time.Hour),
		pwFail:      shared.NewWindowLimiter(passwordFailLimit, 15*time.Minute),
		dummyHash:   dummy,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 16<<10)
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return false
	}
	return true
}

type authResponse struct {
	Account      *Account `json:"account"`
	SessionToken string   `json:"session_token,omitempty"`
}

// Alamat halaman pelanggan di situs brand (azhan-booking), langsung di akar situs (tanpa awalan).
const (
	PathAturUlangPassword = "/atur-ulang-password"
	PathLupaPassword      = "/lupa-password"
)

// siteLink membentuk URL situs brand dari domain di database (bukan dari header request,
// supaya tautan di email tidak bisa dibelokkan). SITE_URL_TEMPLATE, mis. "http://{domain}:3100".
func siteLink(domain, path string, q url.Values) string {
	tpl := strings.TrimSpace(os.Getenv("SITE_URL_TEMPLATE"))
	if tpl == "" {
		tpl = "https://{domain}"
	}
	base := strings.TrimRight(strings.ReplaceAll(tpl, "{domain}", domain), "/")
	if len(q) > 0 {
		path += "?" + q.Encode()
	}
	return base + path
}

// SiteLink membentuk URL situs brand untuk paket lain (mis. email booking) dengan aturan yang sama
// seperti email reset password.
func SiteLink(domain, path string, q url.Values) string { return siteLink(domain, path, q) }

func (h *Handler) sendAsync(msg mailer.Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.mail.Send(ctx, msg); err != nil {
			log.Printf("[ERROR] mailer: %v", err)
		}
	}()
}

// validateProfile memeriksa nama, email, dan nomor WhatsApp (sudah di-trim); mengembalikan pesan galat, atau "" bila valid.
// validateDomisiliLahir memeriksa isian opsional profil: domisili maksimal 100 karakter; tanggal lahir berbentuk YYYY-MM-DD,
// bukan di masa depan, dan tidak sebelum 1900. nil atau string kosong lolos (kosong berarti dihapus).
func validateDomisiliLahir(domisili, tanggalLahir *string, sekarang time.Time) string {
	if domisili != nil && utf8.RuneCountInString(*domisili) > 100 {
		return "kota domisili maksimal 100 karakter"
	}
	if tanggalLahir != nil && *tanggalLahir != "" {
		t, err := time.Parse("2006-01-02", *tanggalLahir)
		if err != nil || t.After(sekarang) || t.Year() < 1900 {
			return "tanggal lahir tidak valid"
		}
	}
	return ""
}

// biodataDaftar merapikan dan memeriksa biodata opsional saat daftar. Mengembalikan biodata yang sudah dirapikan,
// atau pesan galat untuk pengguna.
func biodataDaftar(jenisKelamin, domisili, tanggalLahir string, sekarang time.Time) (Biodata, string) {
	bio := Biodata{
		JenisKelamin: strings.ToUpper(strings.TrimSpace(jenisKelamin)),
		Domisili:     strings.TrimSpace(domisili),
		TanggalLahir: strings.TrimSpace(tanggalLahir),
	}
	if bio.JenisKelamin != "" && bio.JenisKelamin != "L" && bio.JenisKelamin != "P" {
		return Biodata{}, "jenis kelamin harus L atau P"
	}
	if msg := validateDomisiliLahir(&bio.Domisili, &bio.TanggalLahir, sekarang); msg != "" {
		return Biodata{}, msg
	}
	return bio, ""
}

func validateProfile(nama, email, noHP string) string {
	switch {
	case !ValidName(nama):
		return "nama lengkap wajib diisi (2–100 karakter; hanya huruf, spasi, titik, koma, apostrof, dan tanda hubung)"
	case !ValidEmail(email):
		return "format email tidak valid"
	case !ValidPhone(noHP):
		return "nomor WhatsApp harus 10–15 digit"
	}
	return ""
}

type registerRequest struct {
	BrandID      int64  `json:"brand_id"`
	NamaLengkap  string `json:"nama_lengkap"`
	Email        string `json:"email"`
	NoHP         string `json:"no_hp"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token"`
	// KodeReferral diisi route handler web publik dari cookie azhan_ref (klik terakhir, 90 hari), bukan dari input
	// pengguna. Hanya dibaca di sini, saat akun dibuat: akun terikat ke agen pemilik kode (spek 4.1).
	KodeReferral string `json:"kode_referral"`
	// Biodata opsional (dipakai form daftar app, keputusan 14 November 2026): jenis kelamin L/P, domisili,
	// dan tanggal lahir YYYY-MM-DD. Kosong = tidak diisi.
	JenisKelamin string `json:"jenis_kelamin"`
	Domisili     string `json:"domisili"`
	TanggalLahir string `json:"tanggal_lahir"`
}

// modeDaftar membedakan pendaftaran akun biasa dari pendaftaran lewat form agen (K2): form agen tanpa captcha.
type modeDaftar struct {
	captcha bool
}

// Register POST /api/public/auth/register
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	h.daftar(w, r, modeDaftar{captcha: true})
}

// RegisterApp POST /api/public/auth/register-app: daftar dari app Hana/Alsha. App native tidak bisa menjalankan
// captcha web, jadi endpoint ini tanpa captcha dan hanya dibatasi pembatas per IP (keputusan 14 November 2026).
// Verifikasi perangkat (Play Integrity / App Attest) direncanakan sebelum rilis di toko aplikasi.
func (h *Handler) RegisterApp(w http.ResponseWriter, r *http.Request) {
	h.daftar(w, r, modeDaftar{})
}

// RegisterAgen POST /api/public/auth/register-agen: akun untuk calon agen. Pembatas per IP tetap berlaku.
func (h *Handler) RegisterAgen(w http.ResponseWriter, r *http.Request) {
	h.daftar(w, r, modeDaftar{})
}

func (h *Handler) daftar(w http.ResponseWriter, r *http.Request, mode modeDaftar) {
	ip := shared.ClientIP(r)
	var req registerRequest
	if !decode(w, r, &req) {
		return
	}
	if !h.regIP.Take(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak pendaftaran, coba lagi dalam 1 jam")
		return
	}

	// Captcha opsional (keputusan 2026-10-28): tanpa TURNSTILE_SECRET_KEY, NewTurnstile memakai bypass dan Verify
	// meloloskan semua request, sehingga pendaftaran hanya dibatasi rate limit. Bila secret diisi dan Cloudflare
	// tidak bisa dihubungi, Verify mengembalikan error dan pendaftaran ditolak (fail closed).
	if mode.captcha {
		ok, err := h.captcha.Verify(r.Context(), strings.TrimSpace(req.CaptchaToken), ip)
		if err != nil {
			if !errors.Is(err, shared.ErrCaptchaNotConfigured) {
				log.Printf("[ERROR] customer captcha: %v", err)
			}
			writeError(w, http.StatusServiceUnavailable, "verifikasi keamanan sedang tidak tersedia, coba lagi")
			return
		}
		if !ok {
			writeError(w, http.StatusBadRequest, "verifikasi keamanan gagal, silakan coba lagi")
			return
		}
	}
	if req.BrandID <= 0 {
		writeError(w, http.StatusBadRequest, "brand_id wajib diisi")
		return
	}
	req.NamaLengkap, req.Email, req.NoHP = strings.TrimSpace(req.NamaLengkap), strings.TrimSpace(req.Email), strings.TrimSpace(req.NoHP)
	if msg := validateProfile(req.NamaLengkap, req.Email, req.NoHP); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if ok, msg := ValidPassword(req.Password); !ok {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	bio, msg := biodataDaftar(req.JenisKelamin, req.Domisili, req.TanggalLahir, time.Now())
	if msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		log.Printf("[ERROR] customer hash: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	acc, err := h.repo.Create(r.Context(), req.BrandID, req.NamaLengkap, req.Email, req.NoHP, string(hash), req.KodeReferral, bio)
	if errors.Is(err, ErrEmailTaken) {
		// Status 409 dipertahankan untuk kontrak frontend; pesan tidak menegaskan data akun lain.
		// Penebakan email tetap dibatasi captcha + registerIPLimit.
		writeError(w, http.StatusConflict, "Pendaftaran dengan email ini tidak dapat diproses. Bila Anda sudah punya akun, silakan masuk atau gunakan Lupa kata sandi.")
		return
	}
	if errors.Is(err, ErrPhoneTaken) {
		writeError(w, http.StatusConflict, "Nomor WhatsApp sudah terdaftar. Gunakan nomor WhatsApp lain.")
		return
	}
	if err != nil {
		log.Printf("[ERROR] customer create: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	token, err := h.repo.CreateSession(r.Context(), acc.ID)
	if err != nil {
		log.Printf("[ERROR] customer session: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Account: acc, SessionToken: token})
}

type loginRequest struct {
	BrandID  int64  `json:"brand_id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login POST /api/public/auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req loginRequest
	if !decode(w, r, &req) {
		return
	}
	key := ip + "|" + strconv.FormatInt(req.BrandID, 10) + "|" + NormalizeEmail(req.Email)
	// Slot percobaan diambil SEBELUM bcrypt (Take atomik): request paralel tidak bisa lolos bersamaan
	// sebelum kegagalannya tercatat. Login berhasil mengembalikan slot itu.
	if !h.loginKey.Take(key) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan masuk, coba lagi dalam 15 menit")
		return
	}
	if !h.loginIP.Take(ip) {
		h.loginKey.Refund(key)
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan masuk, coba lagi dalam 15 menit")
		return
	}
	fail := func() {
		writeError(w, http.StatusUnauthorized, "email atau password salah")
	}
	if req.BrandID <= 0 || len(req.Password) > 72 {
		fail()
		return
	}
	acc, err := h.repo.GetByEmail(r.Context(), req.BrandID, req.Email)
	hash := h.dummyHash
	if err == nil {
		hash = []byte(acc.passwordHash)
	}
	cmp := bcrypt.CompareHashAndPassword(hash, []byte(req.Password))
	if err != nil || cmp != nil {
		if err != nil && !errors.Is(err, ErrNotFound) {
			log.Printf("[ERROR] customer login lookup: %v", err)
		}
		fail()
		return
	}
	// Password benar: percobaan ini tidak dihitung sebagai kegagalan.
	h.loginKey.Reset(key)
	h.loginIP.Refund(ip)
	token, err := h.repo.CreateSession(r.Context(), acc.ID)
	if err != nil {
		log.Printf("[ERROR] customer session: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	h.repo.TouchLogin(r.Context(), acc.ID)
	writeJSON(w, http.StatusOK, authResponse{Account: acc, SessionToken: token})
}

// Logout POST /api/account/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	_ = h.repo.RevokeSession(r.Context(), TokenFromContext(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// meView adalah isi "account" pada GET /api/account/me. NIK TIDAK pernah dikirim utuh (tinjauan
// keamanan 2026-10-22): hanya nik_masked (4 digit terakhir). Field "nik" lama dihapus; frontend
// memakai `account.nik || ”` sehingga isian NIK checkout cukup kosong dan diisi ulang pelanggan.
type meView struct {
	*Account
	JenisKelamin string `json:"jenis_kelamin"`
	NIKMasked    string `json:"nik_masked"`
	// NIKAgenMasked: NIK yang agen simpan saat mendaftar, tersamar; untuk petunjuk "terisi otomatis" di form booking agen.
	NIKAgenMasked string `json:"nik_agen_masked"`
}

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// MaskNIK menyisakan 4 digit terakhir, sisanya "•". Kosong tetap kosong.
func MaskNIK(nik string) string {
	nik = strings.TrimSpace(nik)
	n := utf8.RuneCountInString(nik)
	if n == 0 {
		return ""
	}
	if n <= 4 {
		return strings.Repeat("•", n)
	}
	r := []rune(nik)
	return strings.Repeat("•", n-4) + string(r[n-4:])
}

// buildMeView melengkapi akun dengan jenis kelamin dan NIK tersamar dari jamaah akun (bila ada).
func buildMeView(ctx context.Context, q rowQuerier, a *Account) meView {
	view := meView{Account: a}
	if a != nil {
		var nikAgen, jkAkun sql.NullString
		if err := q.QueryRowContext(ctx, `SELECT nik_agen, jenis_kelamin FROM customer_accounts WHERE id=?`, a.ID).Scan(&nikAgen, &jkAkun); err == nil {
			view.NIKAgenMasked = MaskNIK(nikAgen.String)
			view.JenisKelamin = jkAkun.String // jenis kelamin akun; diganti data jamaah di bawah bila ada
		}
	}
	if a != nil && a.JamaahID != nil {
		var jk, nik sql.NullString
		if err := q.QueryRowContext(ctx, `SELECT jenis_kelamin, nik FROM jamaah WHERE id=?`, *a.JamaahID).Scan(&jk, &nik); err == nil {
			if jk.String != "" {
				view.JenisKelamin = jk.String
			}
			view.NIKMasked = MaskNIK(nik.String)
		}
	}
	return view
}

// Me GET /api/account/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	a := FromContext(r.Context())
	// Jenis kelamin diambil dari data jamaah akun (bila sudah pernah memesan) untuk mengisi checkout otomatis.
	writeJSON(w, http.StatusOK, map[string]any{"account": buildMeView(r.Context(), h.repo.db, a)})
}

// Domisili dan TanggalLahir opsional: tidak dikirim = tidak berubah, string kosong = dihapus.
type updateProfileRequest struct {
	NamaLengkap  string  `json:"nama_lengkap"`
	Email        string  `json:"email"`
	NoHP         string  `json:"no_hp"`
	Domisili     *string `json:"domisili"`
	TanggalLahir *string `json:"tanggal_lahir"`
	// PasswordSaatIni wajib bila email diganti: sesi yang dicuri tidak cukup untuk memindahkan akun ke email lain.
	PasswordSaatIni string `json:"password_saat_ini"`
}

// cekPasswordSaatIni memeriksa password saat ini dengan batas percobaan per akun (pwFail). Slot percobaan diambil
// atomik sebelum bcrypt; hanya password salah yang tetap terhitung. ok=false berarti jawaban galat sudah ditulis.
func (h *Handler) cekPasswordSaatIni(w http.ResponseWriter, acc *Account, password string) bool {
	key := strconv.FormatInt(acc.ID, 10)
	if !h.pwFail.Take(key) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return false
	}
	if password == "" {
		h.pwFail.Refund(key)
		writeError(w, http.StatusBadRequest, "masukkan password saat ini")
		return false
	}
	if len(password) > 72 || bcrypt.CompareHashAndPassword([]byte(acc.passwordHash), []byte(password)) != nil {
		writeError(w, http.StatusBadRequest, "password saat ini salah")
		return false
	}
	h.pwFail.Reset(key)
	return true
}

// UpdateMe PATCH /api/account/me: ubah nama, email, nomor WhatsApp, domisili, dan tanggal lahir.
// Mengganti email wajib menyertakan password saat ini; sesudahnya sesi lain dan tautan reset lama dicabut dan
// alamat email lama diberi tahu.
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	var req updateProfileRequest
	if !decode(w, r, &req) {
		return
	}
	req.NamaLengkap, req.Email, req.NoHP = strings.TrimSpace(req.NamaLengkap), strings.TrimSpace(req.Email), strings.TrimSpace(req.NoHP)
	if msg := validateProfile(req.NamaLengkap, req.Email, req.NoHP); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if req.Domisili != nil {
		d := strings.TrimSpace(*req.Domisili)
		req.Domisili = &d
	}
	if req.TanggalLahir != nil {
		t := strings.TrimSpace(*req.TanggalLahir)
		req.TanggalLahir = &t
	}
	if msg := validateDomisiliLahir(req.Domisili, req.TanggalLahir, time.Now()); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	gantiEmail := NormalizeEmail(req.Email) != NormalizeEmail(acc.Email)
	if gantiEmail && !h.cekPasswordSaatIni(w, acc, req.PasswordSaatIni) {
		return
	}
	key := strconv.FormatInt(acc.ID, 10)
	if !h.profile.Take(key) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak perubahan data, coba lagi dalam 1 jam")
		return
	}

	err := h.repo.UpdateProfile(r.Context(), acc.ID, req.NamaLengkap, req.Email, req.NoHP, req.Domisili, req.TanggalLahir)
	if errors.Is(err, ErrEmailTaken) {
		writeError(w, http.StatusConflict, "email ini tidak dapat digunakan, coba email lain")
		return
	}
	if errors.Is(err, ErrPhoneTaken) {
		writeError(w, http.StatusConflict, "Nomor WhatsApp sudah dipakai akun lain. Gunakan nomor lain.")
		return
	}
	if err != nil {
		log.Printf("[ERROR] customer update profile: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	if gantiEmail {
		// Email sudah berganti: sesi lain dan tautan reset yang terkirim ke alamat lama tidak boleh berlaku lagi.
		if err := h.repo.RevokeOtherSessions(r.Context(), acc.ID, TokenFromContext(r.Context())); err != nil {
			log.Printf("[ERROR] customer ganti email revoke sesi akun=%d: %v", acc.ID, err)
		}
		if err := h.repo.InvalidateTokens(r.Context(), acc.ID, PurposeResetPass); err != nil {
			log.Printf("[ERROR] customer ganti email invalidate token akun=%d: %v", acc.ID, err)
		}
		if name, domain, err := h.repo.BrandSite(r.Context(), acc.BrandID); err == nil && domain != "" {
			subject, text := mailer.EmailChanged(name, acc.NamaLengkap, req.Email)
			h.sendAsync(mailer.Message{To: acc.Email, Subject: subject, Text: text})
		}
	}
	updated, err := h.repo.GetByID(r.Context(), acc.ID)
	if err != nil {
		log.Printf("[ERROR] customer reload profile: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"account": buildMeView(r.Context(), h.repo.db, updated)})
}

type changePasswordRequest struct {
	PasswordSaatIni string `json:"password_saat_ini"`
	PasswordBaru    string `json:"password_baru"`
}

// ChangePassword POST /api/account/password: ganti password dari dalam akun. Wajib menyebut password saat ini.
// Sesi lain dicabut, sesi yang dipakai tetap aktif; pemilik akun diberi tahu lewat email.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	var req changePasswordRequest
	if !decode(w, r, &req) {
		return
	}
	if !h.cekPasswordSaatIni(w, acc, req.PasswordSaatIni) {
		return
	}
	if ok, msg := ValidPassword(req.PasswordBaru); !ok {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if req.PasswordBaru == req.PasswordSaatIni {
		writeError(w, http.StatusBadRequest, "password baru harus berbeda dari password saat ini")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.PasswordBaru), bcryptCost)
	if err != nil {
		log.Printf("[ERROR] customer hash: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	// Sesi lain dan tautan reset lama dicabut SEBELUM password diganti: bila salah satunya gagal, password tetap lama
	// dan pelanggan bisa mencoba lagi; jawaban 200 berarti klaim "perangkat lain dikeluarkan" benar.
	if err = h.repo.RevokeOtherSessions(r.Context(), acc.ID, TokenFromContext(r.Context())); err == nil {
		err = h.repo.InvalidateTokens(r.Context(), acc.ID, PurposeResetPass)
	}
	if err == nil {
		err = h.repo.SetPassword(r.Context(), acc.ID, string(hash))
	}
	if err != nil {
		log.Printf("[ERROR] customer change password: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	if name, domain, err := h.repo.BrandSite(r.Context(), acc.BrandID); err == nil && domain != "" {
		subject, text := mailer.PasswordChanged(name, acc.NamaLengkap, siteLink(domain, PathLupaPassword, nil))
		h.sendAsync(mailer.Message{To: acc.Email, Subject: subject, Text: text})
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type forgotRequest struct {
	BrandID int64  `json:"brand_id"`
	Email   string `json:"email"`
}

// ForgotPassword POST /api/public/auth/forgot-password. Jawaban selalu sama agar
// tidak membocorkan email yang terdaftar.
func (h *Handler) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req forgotRequest
	if !decode(w, r, &req) {
		return
	}
	generic := map[string]string{"status": "ok", "message": "Jika email terdaftar, tautan atur ulang password sudah dikirim."}
	emailKey := strconv.FormatInt(req.BrandID, 10) + "|" + NormalizeEmail(req.Email)
	key := ip + "|" + emailKey
	if !h.forgot.Take(key) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak permintaan, coba lagi dalam 1 jam")
		return
	}
	if !h.forgotIP.Take(ip) {
		h.forgot.Refund(key)
		writeError(w, http.StatusTooManyRequests, "terlalu banyak permintaan, coba lagi dalam 1 jam")
		return
	}
	// Batas per email: jawaban tetap generik (200) agar tidak menjadi oracle, tetapi email tidak dikirim.
	if !h.forgotEmail.Take(emailKey) {
		writeJSON(w, http.StatusOK, generic)
		return
	}
	acc, err := h.repo.GetByEmail(r.Context(), req.BrandID, req.Email)
	if err != nil {
		writeJSON(w, http.StatusOK, generic)
		return
	}
	name, domain, err := h.repo.BrandSite(r.Context(), acc.BrandID)
	if err != nil || domain == "" {
		log.Printf("[WARN] customer: domain brand %d tidak ditemukan, email reset dilewati", acc.BrandID)
		writeJSON(w, http.StatusOK, generic)
		return
	}
	token, err := h.repo.IssueToken(r.Context(), acc.ID, PurposeResetPass, ResetTokenTTL)
	if err != nil {
		log.Printf("[ERROR] customer reset token: %v", err)
		writeJSON(w, http.StatusOK, generic)
		return
	}
	link := siteLink(domain, PathAturUlangPassword, url.Values{"token": {token}})
	subject, text := mailer.ResetPassword(name, acc.NamaLengkap, link)
	h.sendAsync(mailer.Message{To: acc.Email, Subject: subject, Text: text})
	writeJSON(w, http.StatusOK, generic)
}

type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ResetPassword POST /api/public/auth/reset-password. Semua sesi lama dicabut.
func (h *Handler) ResetPassword(w http.ResponseWriter, r *http.Request) {
	ip := shared.ClientIP(r)
	var req resetRequest
	if !decode(w, r, &req) {
		return
	}
	// Slot diambil atomik sebelum bcrypt; hanya token salah yang tetap terhitung sebagai kegagalan.
	if !h.loginIP.Take(ip) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak percobaan, coba lagi dalam 15 menit")
		return
	}
	tokenSalah := false
	defer func() {
		if !tokenSalah {
			h.loginIP.Refund(ip)
		}
	}()
	if ok, msg := ValidPassword(req.Password); !ok {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcryptCost)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	accountID, err := h.repo.ConsumeToken(r.Context(), strings.TrimSpace(req.Token), PurposeResetPass)
	if errors.Is(err, ErrTokenInvalid) {
		tokenSalah = true
		writeError(w, http.StatusBadRequest, ErrTokenInvalid.Error())
		return
	}
	if err != nil {
		log.Printf("[ERROR] customer consume reset: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	if err = h.repo.SetPassword(r.Context(), accountID, string(hash)); err != nil {
		log.Printf("[ERROR] customer set password: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	_ = h.repo.RevokeAllSessions(r.Context(), accountID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
