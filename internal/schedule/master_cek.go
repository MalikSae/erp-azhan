package schedule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"

	"erp-azhan/api/internal/shared"
)

// AirlineExists memeriksa apakah airline dengan id tersebut ada.
func (r *Repository) AirlineExists(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM airlines WHERE id=?`, id).Scan(&count)
	return count > 0, err
}

// HotelExists memeriksa apakah hotel dengan id tersebut ada.
func (r *Repository) HotelExists(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hotels WHERE id=?`, id).Scan(&count)
	return count > 0, err
}

// ItineraryExists memeriksa apakah itinerary dengan id tersebut ada.
func (r *Repository) ItineraryExists(ctx context.Context, id int64) (bool, error) {
	return r.ItineraryTersedia(ctx, id, 0)
}

// ItineraryTersedia: id itu master, atau salinan pribadi milik paket scheduleID (0 = paket baru).
func (r *Repository) ItineraryTersedia(ctx context.Context, id, scheduleID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM itineraries WHERE id=? AND (schedule_id IS NULL OR schedule_id=?)`, id, scheduleID).Scan(&count)
	return count > 0, err
}

// CategoryExists memeriksa apakah category dengan id tersebut ada.
// CategoryAvailable: kategori ada, berlaku untuk brand ini, dan aktif. bolehNonaktif melewati syarat aktif;
// dipakai saat paket tetap memakai kategori yang sudah dipilihnya sebelum kategori itu dinonaktifkan.
func (r *Repository) CategoryAvailable(ctx context.Context, id int64, brandID int64, bolehNonaktif bool) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM package_categories c WHERE c.id=? AND (c.is_active=TRUE OR ?) AND EXISTS (SELECT 1 FROM category_brands cb WHERE cb.category_id=c.id AND cb.brand_id=?)`, id, bolehNonaktif, brandID).Scan(&count)
	return count > 0, err
}

// HotelCity mengembalikan kota hotel; ErrNotFound bila hotel tidak ada (MP-03).
func (r *Repository) HotelCity(ctx context.Context, id int64) (string, error) {
	var city string
	err := r.db.QueryRowContext(ctx, `SELECT city FROM hotels WHERE id=?`, id).Scan(&city)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return city, err
}

// AirportCodeExists memeriksa kode bandara terdaftar di master (case-insensitive, MP-06).
func (r *Repository) AirportCodeExists(ctx context.Context, code string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM airports WHERE UPPER(code)=UPPER(?)`, code).Scan(&n)
	return n > 0, err
}

func (r *Repository) BrandMinimalDP(ctx context.Context, id int64) (float64, error) {
	var dp float64
	err := r.db.QueryRowContext(ctx, `SELECT minimal_dp FROM brands WHERE id=?`, id).Scan(&dp)
	return dp, err
}

// checkHotelCity memastikan hotel ada dan kotanya sesuai slot (Makkah/Madinah).
func (h *Handler) checkHotelCity(ctx context.Context, w http.ResponseWriter, id int64, kota, field string) bool {
	city, err := h.repo.HotelCity(ctx, id)
	if errors.Is(err, ErrNotFound) {
		shared.WriteError(w, http.StatusBadRequest, field+" Tidak valid.")
		return false
	}
	if err != nil {
		log.Printf("[ERROR] schedule: cek hotel: %v", err)
		shared.WriteError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return false
	}
	if normalizeKota(city) != normalizeKota(kota) {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("%s harus hotel di %s (hotel terpilih berada di %s).", field, kota, city))
		return false
	}
	return true
}
