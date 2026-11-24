package customer

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/shared"
)

// Batas lapis depan di memori per IP (murah, tidak menggantikan penghitung atomik di database).
const aktivasiDepanLimit = 60 // permintaan per 15 menit per IP

// AktivasiHandler: pintu HTTP publik untuk Aktivasi. Flag AKTIVASI_JAMAAH_AKTIF mati = rute 404.
type AktivasiHandler struct {
	core  *Aktivasi
	repo  *Repository
	cfg   AktivasiConfig
	depan *shared.WindowLimiter
}

func NewAktivasiHandler(core *Aktivasi, repo *Repository, cfg AktivasiConfig) *AktivasiHandler {
	return &AktivasiHandler{core: core, repo: repo, cfg: cfg, depan: shared.NewWindowLimiter(aktivasiDepanLimit, 15*time.Minute)}
}

type aktivasiRequest struct {
	BrandID     int64  `json:"brand_id"`
	KodeBooking string `json:"kode_booking"`
	NIK         string `json:"nik"`
	Simpan      bool   `json:"simpan"`
	NoHP        string `json:"no_hp"`
	Email       string `json:"email"`
	Password    string `json:"password"`
}

// Publik POST /api/public/auth/aktivasi: cek (simpan=false) mengembalikan data jamaah; simpan=true membuat akun dan sesi.
func (h *AktivasiHandler) Publik(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Aktif {
		http.NotFound(w, r)
		return
	}
	var req aktivasiRequest
	if !decode(w, r, &req) {
		return
	}
	h.jalankan(w, r, AktivasiInput{
		BrandID: req.BrandID, KodeBooking: req.KodeBooking, NIK: req.NIK, Simpan: req.Simpan,
		NoHP: req.NoHP, Email: req.Email, Password: req.Password,
	})
}

func (h *AktivasiHandler) jalankan(w http.ResponseWriter, r *http.Request, in AktivasiInput) {
	in.IP = NormalisasiIPAktivasi(shared.ClientIP(r))
	if p := strings.TrimSpace(r.Header.Get("X-Perangkat-Id")); p != "" && len(p) <= 100 {
		in.Perangkat = p
	}
	if !h.depan.Take(in.IP) {
		h.tulis429(w)
		return
	}
	hasil, err := h.core.Jalankan(r.Context(), in)
	var sering *ErrAktivasiTerlaluSering
	var input *ErrAktivasiInput
	switch {
	case errors.As(err, &sering):
		h.tulis429(w)
	case errors.As(err, &input):
		writeError(w, http.StatusBadRequest, input.Pesan)
	case errors.Is(err, ErrAktivasiGagal):
		writeErrorCode(w, http.StatusBadRequest, PesanAktivasiGagal, KodeGalatAktivasiGagal)
	case errors.Is(err, ErrEmailTaken):
		writeError(w, http.StatusConflict, "Pendaftaran dengan email ini tidak dapat diproses. Bila Anda sudah punya akun, masuk atau gunakan Lupa kata sandi.")
	case errors.Is(err, ErrPhoneTaken):
		writeError(w, http.StatusConflict, "Nomor WhatsApp sudah terdaftar. Gunakan nomor WhatsApp lain.")
	case err != nil:
		log.Printf("[ERROR] customer aktivasi: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
	case !in.Simpan:
		writeJSON(w, http.StatusOK, hasil.Jamaah)
	default:
		akun, err := h.repo.GetByID(r.Context(), hasil.AkunID)
		if err != nil {
			log.Printf("[ERROR] customer aktivasi muat akun: %v", err)
			writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
			return
		}
		writeJSON(w, http.StatusCreated, authResponse{Account: akun, SessionToken: hasil.Token})
	}
}

// tulis429 memakai Retry-After tetap untuk semua jenis batas.
func (h *AktivasiHandler) tulis429(w http.ResponseWriter) {
	w.Header().Set("Retry-After", strconv.Itoa(int(RetryAfterAktivasi/time.Second)))
	writeErrorCode(w, http.StatusTooManyRequests, PesanAktivasiTerlaluSering, KodeGalatAktivasiTerlaluSering)
}
