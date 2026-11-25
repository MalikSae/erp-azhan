package surat

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
)

// Handler melayani pengaturan surat per brand di dashboard (GET dan PUT /api/admin/brands/{id}/surat).
type Handler struct{ repo *Repository }

func NewHandler(repo *Repository) *Handler { return &Handler{repo: repo} }

var polaGambarBrand = regexp.MustCompile(`^/uploads/brand-logos/[A-Za-z0-9._-]+$`)

func brandID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	id, err := strconv.ParseUint(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id == 0 {
		shared.WriteError(w, http.StatusBadRequest, "ID tidak valid.")
		return 0, false
	}
	if scoped := identity.GetBrandID(r.Context()); scoped != nil && int64(id) != *scoped {
		shared.WriteError(w, http.StatusNotFound, "Brand tidak ditemukan.")
		return 0, false
	}
	return id, true
}

// Get GET /api/admin/brands/{id}/surat
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := brandID(w, r)
	if !ok {
		return
	}
	p, err := h.repo.Pengaturan(r.Context(), id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil pengaturan surat.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, p)
}

// Validasi merapikan dan memeriksa isian: panjang, kode surat (huruf dan angka), dan gambar hanya dari unggahan brand.
func (p *Pengaturan) Validasi() error {
	p.NamaPerusahaan = strings.TrimSpace(p.NamaPerusahaan)
	p.KodeSurat = strings.ToUpper(strings.TrimSpace(p.KodeSurat))
	p.PenandatanganNama = strings.TrimSpace(p.PenandatanganNama)
	p.PenandatanganJabatan = strings.TrimSpace(p.PenandatanganJabatan)
	p.KopURL, p.TandaTanganURL, p.StempelURL = strings.TrimSpace(p.KopURL), strings.TrimSpace(p.TandaTanganURL), strings.TrimSpace(p.StempelURL)
	if len(p.NamaPerusahaan) > 150 || len(p.PenandatanganNama) > 150 || len(p.PenandatanganJabatan) > 100 {
		return errors.New("nama perusahaan, nama penandatangan, atau jabatan terlalu panjang")
	}
	if p.KodeSurat != "" && !regexp.MustCompile(`^[A-Z0-9]{1,10}$`).MatchString(p.KodeSurat) {
		return errors.New("kode surat hanya boleh huruf dan angka (maksimal 10 karakter)")
	}
	for _, u := range []string{p.KopURL, p.TandaTanganURL, p.StempelURL} {
		if u != "" && !polaGambarBrand.MatchString(u) {
			return errors.New("gambar harus berasal dari unggahan brand")
		}
	}
	return nil
}

// Put PUT /api/admin/brands/{id}/surat
func (h *Handler) Put(w http.ResponseWriter, r *http.Request) {
	id, ok := brandID(w, r)
	if !ok {
		return
	}
	var p Pengaturan
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&p); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	if err := p.Validasi(); err != nil {
		shared.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	p.BrandID = id
	saatIni, err := h.repo.Pengaturan(r.Context(), id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil pengaturan surat.")
		return
	}
	if p.NomorTerakhir < saatIni.NomorTerakhir {
		shared.WriteError(w, http.StatusBadRequest, "Nomor terakhir tidak boleh diturunkan agar tidak ada nomor surat ganda.")
		return
	}
	if err := h.repo.SimpanPengaturan(r.Context(), p); err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Gagal menyimpan pengaturan surat.")
		return
	}
	hasil, err := h.repo.Pengaturan(r.Context(), id)
	if err != nil {
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil pengaturan surat.")
		return
	}
	shared.WriteJSON(w, http.StatusOK, hasil)
}
