package agen

import (
	"errors"
	"net/http"
	"strconv"

	"erp-azhan/api/internal/identity"
)

func komisiFilterFromQuery(r *http.Request) KomisiFilter {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	offset, _ := strconv.Atoi(q.Get("offset"))
	penerima, _ := strconv.ParseInt(q.Get("agen_id"), 10, 64)
	return KomisiFilter{
		PenerimaID: penerima,
		Jenis:      q.Get("jenis"),
		Dari:       q.Get("dari"),
		Sampai:     q.Get("sampai"),
		Limit:      limit,
		Offset:     offset,
	}
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
	d, err := h.repo.GetDashboardAgen(r.Context(), identity.GetPortalJamaahID(r.Context()))
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// RiwayatKomisiPortal GET /api/portal/agen/komisi (A6)
func (h *Handler) RiwayatKomisiPortal(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListKomisiAgen(r.Context(), identity.GetPortalJamaahID(r.Context()), komisiFilterFromQuery(r))
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

// DetailAgen GET /api/admin/agen/{jamaahID} (B2a)
func (h *Handler) DetailAgen(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "jamaahID")
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

// RiwayatKomisiAdmin GET /api/admin/agen/komisi?agen_id=&jenis=&dari=&sampai= (B3)
func (h *Handler) RiwayatKomisiAdmin(w http.ResponseWriter, r *http.Request) {
	f := komisiFilterFromQuery(r)
	f.BrandID = adminBrandScope(r)
	items, err := h.repo.ListKomisiBrand(r.Context(), f)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}
