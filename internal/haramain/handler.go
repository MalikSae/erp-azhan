package haramain

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/customer"
	"erp-azhan/api/internal/identity"
)

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

func tulisGalat(w http.ResponseWriter, err error) {
	var val *ErrValidasi
	switch {
	case errors.As(err, &val):
		writeError(w, http.StatusBadRequest, val.Pesan)
	case errors.Is(err, ErrTidakAda), errors.Is(err, ErrUlasanTidakAda):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrBelumBerangkat):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrDipakaiPaket):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("[ERROR] haramain: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal memproses data.")
	}
}

func idParam(w http.ResponseWriter, r *http.Request, nama string) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, nama), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "ID tidak valid.")
		return 0, false
	}
	return id, true
}

// ---- Publik ----

// DaftarPublik GET /api/public/v1/haramain: tempat aktif, urut tampil.
func (h *Handler) DaftarPublik(w http.ResponseWriter, r *http.Request) {
	items, err := Daftar(r.Context(), h.db, true)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type detailView struct {
	*Tempat
	Ulasan []Ulasan `json:"ulasan"`
}

// DetailPublik GET /api/public/v1/haramain/{id}: tempat dan ulasan yang tampil.
func (h *Handler) DetailPublik(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	t, err := Ambil(r.Context(), h.db, id, true)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	ulasan, err := UlasanTempat(r.Context(), h.db, id)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, detailView{Tempat: t, Ulasan: ulasan})
}

// ---- Akun ----

type ulasanSayaView struct {
	BolehUlas  bool    `json:"boleh_ulas"`
	UlasanSaya *Ulasan `json:"ulasan_saya"`
}

// UlasanSayaAkun GET /api/account/haramain/{id}/ulasan: boleh menulis dan ulasan akun di tempat itu.
func (h *Handler) UlasanSayaAkun(w http.ResponseWriter, r *http.Request) {
	acc := customer.FromContext(r.Context())
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	boleh, err := SudahBerangkat(r.Context(), h.db, acc.ID, h.now())
	if err != nil {
		tulisGalat(w, err)
		return
	}
	saya, err := UlasanSaya(r.Context(), h.db, id, acc.ID)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, ulasanSayaView{BolehUlas: boleh, UlasanSaya: saya})
}

// SimpanUlasanAkun PUT /api/account/haramain/{id}/ulasan: {teks, foto}.
func (h *Handler) SimpanUlasanAkun(w http.ResponseWriter, r *http.Request) {
	acc := customer.FromContext(r.Context())
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var in InputUlasan
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	if err := SimpanUlasan(r.Context(), h.db, id, acc.ID, in, h.now()); err != nil {
		tulisGalat(w, err)
		return
	}
	h.UlasanSayaAkun(w, r)
}

// HapusUlasanAkun DELETE /api/account/haramain/{id}/ulasan
func (h *Handler) HapusUlasanAkun(w http.ResponseWriter, r *http.Request) {
	acc := customer.FromContext(r.Context())
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	if err := HapusUlasan(r.Context(), h.db, id, acc.ID); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// LaporkanAkun POST /api/account/haramain/ulasan/{id}/laporkan
func (h *Handler) LaporkanAkun(w http.ResponseWriter, r *http.Request) {
	acc := customer.FromContext(r.Context())
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	if err := Laporkan(r.Context(), h.db, id, acc.ID); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Admin ----

// DaftarAdmin GET /api/admin/haramain
func (h *Handler) DaftarAdmin(w http.ResponseWriter, r *http.Request) {
	items, err := Daftar(r.Context(), h.db, false)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// DetailAdmin GET /api/admin/haramain/{id}
func (h *Handler) DetailAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	t, err := Ambil(r.Context(), h.db, id, false)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handler) bacaInput(w http.ResponseWriter, r *http.Request) (Input, bool) {
	var in Input
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return in, false
	}
	return in, true
}

// BuatAdmin POST /api/admin/haramain
func (h *Handler) BuatAdmin(w http.ResponseWriter, r *http.Request) {
	in, ok := h.bacaInput(w, r)
	if !ok {
		return
	}
	id, err := Buat(r.Context(), h.db, identity.GetAdminUserID(r.Context()), in)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	t, err := Ambil(r.Context(), h.db, id, false)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// UbahAdmin PUT /api/admin/haramain/{id}
func (h *Handler) UbahAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	in, ok := h.bacaInput(w, r)
	if !ok {
		return
	}
	if err := Ubah(r.Context(), h.db, id, identity.GetAdminUserID(r.Context()), in); err != nil {
		tulisGalat(w, err)
		return
	}
	h.DetailAdmin(w, r)
}

// HapusAdmin DELETE /api/admin/haramain/{id}
func (h *Handler) HapusAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	if err := Hapus(r.Context(), h.db, id); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UrutkanAdmin PUT /api/admin/haramain/urutan: {ids}
func (h *Handler) UrutkanAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		tulisGalat(w, err)
		return
	}
	defer func() { _ = tx.Rollback() }()
	if err := Urutkan(r.Context(), tx, req.IDs); err != nil {
		tulisGalat(w, err)
		return
	}
	if err := tx.Commit(); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// UlasanAdminList GET /api/admin/haramain-ulasan?tab=semua|dilaporkan
func (h *Handler) UlasanAdminList(w http.ResponseWriter, r *http.Request) {
	items, err := UlasanAdmin(r.Context(), h.db, r.URL.Query().Get("tab") == "dilaporkan")
	if err != nil {
		tulisGalat(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// AturTampilAdmin PUT /api/admin/haramain-ulasan/{id}/tampil: {disembunyikan}
func (h *Handler) AturTampilAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	var req struct {
		Disembunyikan bool `json:"disembunyikan"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	if err := AturTampil(r.Context(), h.db, id, identity.GetAdminUserID(r.Context()), req.Disembunyikan); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// AbaikanAdmin POST /api/admin/haramain-ulasan/{id}/abaikan
func (h *Handler) AbaikanAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(w, r, "id")
	if !ok {
		return
	}
	if err := Abaikan(r.Context(), h.db, id); err != nil {
		tulisGalat(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
