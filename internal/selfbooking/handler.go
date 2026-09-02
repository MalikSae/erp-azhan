package selfbooking

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository) *Handler {
	return &Handler{repo: repo}
}

func (h *Handler) GetPublicInvoice(w http.ResponseWriter, r *http.Request) {
	code := chi.URLParam(r, "code")
	if strings.TrimSpace(code) == "" {
		writeError(w, http.StatusBadRequest, "kode booking wajib diisi")
		return
	}

	invoice, err := h.repo.GetInvoiceByCode(r.Context(), code)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "invoice pendaftaran tidak ditemukan")
			return
		}
		log.Printf("[ERROR] GetPublicInvoice (%s): %v", code, err)
		writeError(w, http.StatusInternalServerError, "gagal memuat data invoice")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(invoice)
}

func (h *Handler) CreateBooking(w http.ResponseWriter, r *http.Request) {
	var req BookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}

	// Gate 1: Captcha
	if req.CaptchaToken == "" {
		writeError(w, http.StatusBadRequest, "Verifikasi keamanan gagal, silakan coba lagi")
		return
	}
	// TODO: verify turnstile token against cloudflare API here (skipped for now as per usual demo/MVP, or mocked)

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
	
	if len(req.PIC.PortalPIN) != 6 {
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
			// validasi umur < 2 tahun belum dilakukan secara strict (bisa ditambahkan di repo)
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

	// Rate Limiting could be added here (per IP/Phone)

	// Process
	resp, err := h.repo.ProcessBooking(r.Context(), req.BrandID, req)
	if err != nil {
		if errors.Is(err, ErrSeatHabis) || errors.Is(err, ErrDuplicate) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, "jadwal atau brand tidak ditemukan")
			return
		}
		log.Printf("[ERROR] ProcessBooking: %v", err)
		writeError(w, http.StatusInternalServerError, err.Error())
		return
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
