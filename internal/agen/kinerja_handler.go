package agen

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"erp-azhan/api/internal/identity"
)

// errFilterKomisi: parameter angka pada filter komisi bukan bilangan bulat tak negatif.
var errFilterKomisi = errors.New("parameter limit, offset, dan agen_id harus bilangan bulat tak negatif")

// angkaQuery membaca parameter angka opsional. Kosong berarti 0; isi yang bukan bilangan bulat tak negatif
// ditolak, supaya salah ketik tidak diam-diam menjadi "tanpa batas" atau "semua agen".
func angkaQuery(q url.Values, nama string) (int64, error) {
	v := strings.TrimSpace(q.Get(nama))
	if v == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return 0, errFilterKomisi
	}
	return n, nil
}

func komisiFilterFromQuery(r *http.Request) (KomisiFilter, error) {
	q := r.URL.Query()
	limit, err := angkaQuery(q, "limit")
	if err != nil {
		return KomisiFilter{}, err
	}
	offset, err := angkaQuery(q, "offset")
	if err != nil {
		return KomisiFilter{}, err
	}
	penerima, err := angkaQuery(q, "agen_id")
	if err != nil {
		return KomisiFilter{}, err
	}
	return KomisiFilter{
		PenerimaID: penerima,
		Jenis:      q.Get("jenis"),
		Dari:       q.Get("dari"),
		Sampai:     q.Get("sampai"),
		Status:     q.Get("status"),
		Cari:       q.Get("q"),
		Limit:      int(limit),
		Offset:     int(offset),
	}, nil
}

// adminBrandScope: Admin Travel selalu brand-nya sendiri; Super Admin boleh
// memfilter brand lewat ?brand_id= (kosong = semua brand).
func adminBrandScope(r *http.Request) *int64 {
	if b := identity.GetBrandID(r.Context()); b != nil {
		return b
	}
	if id, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64); err == nil && id > 0 {
		return &id
	}
	return nil
}

func handleKinerjaError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrBukanAgenAktif) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	handleError(w, err)
}

// Dashboard GET /api/portal/agen/dashboard (A3)
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := h.repo.GetDashboardAgen(r.Context(), identity.GetCustomerAccountID(r.Context()))
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// RiwayatKomisiPortal GET /api/portal/agen/komisi (A6)
func (h *Handler) RiwayatKomisiPortal(w http.ResponseWriter, r *http.Request) {
	f, err := komisiFilterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.repo.ListKomisiAgen(r.Context(), identity.GetCustomerAccountID(r.Context()), f)
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ListAgen GET /api/admin/agen?q= (B2)
func (h *Handler) ListAgen(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListAgen(r.Context(), adminBrandScope(r), r.URL.Query().Get("q"))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// DetailAgen GET /api/admin/agen/{akunID} (B2a)
func (h *Handler) DetailAgen(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	d, err := h.repo.GetDetailAgen(r.Context(), id, identity.GetBrandID(r.Context()))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// PeringkatAgen GET /api/admin/agen/peringkat?dari=&sampai=&brand_id=
// Periode wajib; tanggal mengacu ke waktu komisi tercatat (booking lunas).
func (h *Handler) PeringkatAgen(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.repo.PeringkatAgen(r.Context(), adminBrandScope(r), q.Get("dari"), q.Get("sampai"))
	if errors.Is(err, ErrPeriodeTidakValid) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// RiwayatKomisiAdmin GET /api/admin/agen/komisi?agen_id=&jenis=&dari=&sampai=&status=&limit=&offset= (B3)
// Jawaban: {items, ringkasan}. Ringkasan dihitung server atas seluruh hasil filter, bukan halaman yang tampil.
// status: kosong = komisi aktif, "batal" = yang dibatalkan, "semua" = keduanya.
func (h *Handler) RiwayatKomisiAdmin(w http.ResponseWriter, r *http.Request) {
	f, err := komisiFilterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	f.BrandID = adminBrandScope(r)
	halaman, err := h.repo.HalamanKomisiBrand(r.Context(), f)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, halaman)
}
