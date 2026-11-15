package notifikasi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

// Handler: rute akun pelanggan untuk kotak masuk notifikasi app.
type Handler struct {
	db  *sql.DB
	now func() time.Time
}

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db, now: time.Now} }

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// Daftar: GET /api/account/notifikasi -> {belum_dibaca, notifikasi: [...]}.
func (h *Handler) Daftar(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	now := h.now()
	items, err := Daftar(r.Context(), h.db, akun, now)
	if err != nil {
		log.Printf("[ERROR] notifikasi daftar akun=%d: %v", akun, err)
		writeError(w, http.StatusInternalServerError, "Gagal memuat notifikasi")
		return
	}
	belum, err := JumlahBelumDibaca(r.Context(), h.db, akun, now)
	if err != nil {
		log.Printf("[ERROR] notifikasi jumlah akun=%d: %v", akun, err)
		writeError(w, http.StatusInternalServerError, "Gagal memuat notifikasi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"belum_dibaca": belum, "notifikasi": items})
}

// Jumlah: GET /api/account/notifikasi/jumlah -> {belum_dibaca}.
func (h *Handler) Jumlah(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	belum, err := JumlahBelumDibaca(r.Context(), h.db, akun, h.now())
	if err != nil {
		log.Printf("[ERROR] notifikasi jumlah akun=%d: %v", akun, err)
		writeError(w, http.StatusInternalServerError, "Gagal memuat notifikasi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"belum_dibaca": belum})
}

// Baca: POST /api/account/notifikasi/{id}/baca -> 204.
func (h *Handler) Baca(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return
	}
	if err := TandaiDibaca(r.Context(), h.db, akun, id); err != nil {
		if errors.Is(err, ErrTidakAda) {
			writeError(w, http.StatusNotFound, "Notifikasi tidak ditemukan")
			return
		}
		log.Printf("[ERROR] notifikasi baca akun=%d id=%d: %v", akun, id, err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// BacaSemua: POST /api/account/notifikasi/baca-semua -> 204.
func (h *Handler) BacaSemua(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	if err := TandaiSemuaDibaca(r.Context(), h.db, akun); err != nil {
		log.Printf("[ERROR] notifikasi baca semua akun=%d: %v", akun, err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Pengumuman: GET /api/account/notifikasi/pengumuman/{id} -> isi lengkap pengumuman.
func (h *Handler) Pengumuman(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid")
		return
	}
	p, err := AmbilPengumuman(r.Context(), h.db, akun, id)
	if err != nil {
		if errors.Is(err, ErrTidakAda) {
			writeError(w, http.StatusNotFound, "Pengumuman tidak ditemukan")
			return
		}
		log.Printf("[ERROR] notifikasi pengumuman akun=%d id=%d: %v", akun, id, err)
		writeError(w, http.StatusInternalServerError, "Gagal memuat pengumuman")
		return
	}
	writeJSON(w, http.StatusOK, p)
}
