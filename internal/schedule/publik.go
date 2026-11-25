package schedule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"erp-azhan/api/internal/benefit"
	"erp-azhan/api/internal/shared"
)

// ListSchedulesPublic godoc
// GET /api/schedules
// Response 200: array PublicSchedule (hanya published, tanpa field status)
func (h *Handler) ListSchedulesPublic(w http.ResponseWriter, r *http.Request) {
	brandRaw := r.URL.Query().Get("brand")
	if strings.TrimSpace(brandRaw) == "" {
		shared.WriteError(w, http.StatusBadRequest, "Parameter brand wajib diisi.")
		return
	}

	brandID, err := strconv.ParseInt(brandRaw, 10, 64)
	if err != nil || brandID <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Parameter brand tidak valid.")
		return
	}

	items, err := h.repo.ListPublic(r.Context(), brandID)
	if err != nil {
		log.Printf("[ERROR] schedule.ListPublic: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Gagal mengambil data jadwal.")
		return
	}
	if items == nil {
		items = make([]*PublicSchedule, 0)
	}
	shared.WriteJSON(w, http.StatusOK, items)
}

// GetSchedulePublic godoc
// GET /api/schedules/{id}
// Response 200: PublicSchedule (hanya jika berstatus published)
func (h *Handler) GetSchedulePublic(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	brandID, err := strconv.ParseInt(r.URL.Query().Get("brand"), 10, 64)
	if err != nil || brandID <= 0 {
		shared.WriteError(w, http.StatusBadRequest, "Parameter brand wajib diisi dan harus valid.")
		return
	}
	s, err := h.repo.GetByID(r.Context(), id, &brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}

	// Paket yang berangkat kurang dari 30 hari lagi (termasuk yang sudah berangkat) tidak ditampilkan
	// dan tidak bisa dipesan lagi (MP-04); daftar publik memakai batas yang sama.
	if s.Status != StatusPublished || s.BerangkatTanggal < batasPublikWIB() {
		shared.WriteError(w, http.StatusNotFound, "Jadwal tidak ditemukan atau belum dipublikasikan.")
		return
	}

	shared.WriteJSON(w, http.StatusOK, s.ToPublic())
}

func (h *Handler) IncrementViewsPublic(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	err := h.repo.IncrementViews(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "paket tidak ditemukan"})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "gagal mencatat view"})
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "success"})
}

// ListPublic mengambil schedule yang published saja untuk brand tertentu, diurutkan berangkat_tanggal ASC.
// Paket yang berangkat kurang dari 30 hari lagi tidak ditampilkan (pendaftaran online sudah ditutup).
func (r *Repository) ListPublic(ctx context.Context, brandID int64) ([]*PublicSchedule, error) {
	q := selectFull + " WHERE s.status = 'published' AND s.brand_id = ? AND s.berangkat_tanggal >= DATE_ADD(CURDATE(), INTERVAL 30 DAY) ORDER BY s.berangkat_tanggal ASC"
	rows, err := r.db.QueryContext(ctx, q, brandID)
	if err != nil {
		return nil, fmt.Errorf("schedule.ListPublic: %w", err)
	}
	defer rows.Close()

	var list []*Schedule
	for rows.Next() {
		s, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()
	// Hotel transit seluruh paket diambil dalam satu query (sebelumnya satu query per paket).
	ids := make([]int64, len(list))
	for i, s := range list {
		ids[i] = s.ID
	}
	transit, err := transitHotelsBySchedule(ctx, r.db, ids)
	if err != nil {
		// Sama dengan perilaku lama: kegagalan hotel transit tidak menggagalkan daftar paket.
		log.Printf("[WARN] schedule.ListPublic hotel transit: %v", err)
		transit = map[int64][]TransitHotel{}
	}
	// Benefit seluruh paket diambil dalam satu query. Kegagalannya tidak menggagalkan daftar paket:
	// paket tetap tayang tanpa benefit.
	benefits, err := benefit.UntukJadwal(ctx, r.db, ids)
	if err != nil {
		log.Printf("[WARN] schedule.ListPublic benefit: %v", err)
		benefits = map[int64][]benefit.Ref{}
	}
	items := make([]*PublicSchedule, 0, len(list))
	for _, s := range list {
		s.Benefits = benefits[s.ID]
		s.TransitHotels = transit[s.ID]
		if s.TransitHotels == nil {
			s.TransitHotels = []TransitHotel{}
		}
		items = append(items, s.ToPublic())
	}
	return items, nil
}

func (r *Repository) IncrementViews(ctx context.Context, id int64) error {
	query := "UPDATE schedules SET views = views + 1 WHERE id = ?"
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}
