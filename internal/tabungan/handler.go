package tabungan

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
	"github.com/go-chi/chi/v5"
)

// Handler: rute akun pelanggan untuk Tabungan Umroh dan token push.
type Handler struct {
	db  *sql.DB
	now func() time.Time
}

func NewHandler(db *sql.DB) *Handler { return &Handler{db: db, now: time.Now} }

func (h *Handler) gagal(w http.ResponseWriter, aksi string, akun int64, err error) {
	log.Printf("[ERROR] tabungan %s akun=%d: %v", aksi, akun, err)
	shared.WriteError(w, http.StatusInternalServerError, "Gagal menyimpan. Coba lagi.")
}

func (h *Handler) balas(w http.ResponseWriter, r *http.Request, status int, akun int64) {
	res, err := Ambil(r.Context(), h.db, akun, h.now())
	if err != nil {
		log.Printf("[ERROR] tabungan ambil akun=%d: %v", akun, err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal memuat target. Coba lagi.")
		return
	}
	shared.WriteJSON(w, status, res)
}

// Ambil: GET /api/account/tabungan.
func (h *Handler) Ambil(w http.ResponseWriter, r *http.Request) {
	h.balas(w, r, http.StatusOK, identity.GetCustomerAccountID(r.Context()))
}

// SimpanTarget: PUT /api/account/tabungan/target.
func (h *Handler) SimpanTarget(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	var t Target
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	now := h.now()
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.gagal(w, "simpan target", akun, err)
		return
	}
	defer tx.Rollback()
	lama, err := KunciTarget(r.Context(), tx, akun)
	if err != nil {
		h.gagal(w, "simpan target", akun, err)
		return
	}
	if pesan := ValidasiTarget(&t, lama.BulanBerangkat, now); pesan != "" {
		shared.WriteError(w, http.StatusBadRequest, pesan)
		return
	}
	if t.ScheduleID != nil {
		j, err := BacaJadwal(r.Context(), tx, *t.ScheduleID)
		if err != nil {
			h.gagal(w, "jadwal", akun, err)
			return
		}
		// Paket yang sama dengan yang tersimpan tetap diterima walau sudah tidak tayang.
		sama := lama.ScheduleID != nil && *lama.ScheduleID == *t.ScheduleID
		if !j.Ada || (!sama && !j.Tayang(now)) {
			shared.WriteError(w, http.StatusBadRequest, "Paket tidak ditemukan. Pilih paket lain.")
			return
		}
		if len(j.BerangkatTanggal) < 7 || j.BerangkatTanggal[:7] != t.BulanBerangkat {
			shared.WriteError(w, http.StatusBadRequest, "Bulan berangkat tidak sesuai paket.")
			return
		}
	}
	if err := SimpanTarget(r.Context(), tx, akun, t); err != nil {
		h.gagal(w, "simpan target", akun, err)
		return
	}
	if err := tx.Commit(); err != nil {
		h.gagal(w, "simpan target", akun, err)
		return
	}
	h.balas(w, r, http.StatusOK, akun)
}

// HapusTarget: DELETE /api/account/tabungan/target.
func (h *Handler) HapusTarget(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.gagal(w, "hapus target", akun, err)
		return
	}
	defer tx.Rollback()
	if err := HapusSemua(r.Context(), tx, akun); err != nil {
		h.gagal(w, "hapus target", akun, err)
		return
	}
	if err := tx.Commit(); err != nil {
		h.gagal(w, "hapus target", akun, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// TambahCatatan: POST /api/account/tabungan/catatan.
func (h *Handler) TambahCatatan(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	var req struct {
		Jumlah  int64  `json:"jumlah"`
		Tanggal string `json:"tanggal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	tanggal, pesan := ValidasiCatatan(req.Jumlah, req.Tanggal, h.now())
	if pesan != "" {
		shared.WriteError(w, http.StatusBadRequest, pesan)
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.gagal(w, "tambah catatan", akun, err)
		return
	}
	defer tx.Rollback()
	if err := TambahCatatan(r.Context(), tx, akun, req.Jumlah, tanggal); err != nil {
		switch {
		case errors.Is(err, ErrTanpaTarget):
			shared.WriteError(w, http.StatusConflict, "Belum ada target. Buat target dulu.")
		case errors.Is(err, ErrCatatanPenuh):
			shared.WriteError(w, http.StatusConflict, "Catatan sudah 200. Hapus catatan lama dulu.")
		default:
			h.gagal(w, "tambah catatan", akun, err)
		}
		return
	}
	if err := tx.Commit(); err != nil {
		h.gagal(w, "tambah catatan", akun, err)
		return
	}
	h.balas(w, r, http.StatusCreated, akun)
}

// HapusCatatan: DELETE /api/account/tabungan/catatan/{id}.
func (h *Handler) HapusCatatan(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		shared.WriteError(w, http.StatusNotFound, "Catatan tidak ditemukan.")
		return
	}
	if err := HapusCatatan(r.Context(), h.db, akun, id); err != nil {
		if errors.Is(err, ErrTidakAda) {
			shared.WriteError(w, http.StatusNotFound, "Catatan tidak ditemukan.")
			return
		}
		h.gagal(w, "hapus catatan", akun, err)
		return
	}
	h.balas(w, r, http.StatusOK, akun)
}

type perangkatReq struct {
	Token    string `json:"token"`
	Platform string `json:"platform"`
}

// SimpanPerangkat: PUT /api/account/perangkat-push.
func (h *Handler) SimpanPerangkat(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	var req perangkatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	if !TokenValid(req.Token) {
		shared.WriteError(w, http.StatusBadRequest, "Perangkat tidak dikenali. Buka ulang aplikasi.")
		return
	}
	if req.Platform != "ios" && req.Platform != "android" {
		shared.WriteError(w, http.StatusBadRequest, "Platform harus ios atau android.")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.gagal(w, "perangkat", akun, err)
		return
	}
	defer tx.Rollback()
	if err := SimpanPerangkat(r.Context(), tx, akun, req.Token, req.Platform); err != nil {
		h.gagal(w, "perangkat", akun, err)
		return
	}
	if err := tx.Commit(); err != nil {
		h.gagal(w, "perangkat", akun, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HapusPerangkat: DELETE /api/account/perangkat-push.
func (h *Handler) HapusPerangkat(w http.ResponseWriter, r *http.Request) {
	akun := identity.GetCustomerAccountID(r.Context())
	var req perangkatReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Token == "" {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	if err := HapusPerangkat(r.Context(), h.db, akun, req.Token); err != nil {
		h.gagal(w, "hapus perangkat", akun, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
