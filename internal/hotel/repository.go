package hotel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound dikembalikan saat row tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// ErrDuplicate dikembalikan saat nama hotel sudah ada (case-insensitive).
var ErrDuplicate = errors.New("hotel dengan nama ini sudah ada")

// Repository mengelola semua query ke tabel hotels.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository dengan *sql.DB yang sudah ada.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List mengambil semua hotel, diurutkan by name ASC.
func (r *Repository) List(ctx context.Context) ([]Hotel, error) {
	const q = `
		SELECT id, name, city, star_rating, distance_m, photo_url, created_at
		FROM hotels
		ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("hotel.List: %w", err)
	}
	defer rows.Close()

	hotels := make([]Hotel, 0) // pastikan tidak nil → serialisasi jadi []
	for rows.Next() {
		var h Hotel
		if err := rows.Scan(&h.ID, &h.Name, &h.City, &h.StarRating, &h.DistanceM, &h.PhotoURL, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("hotel.List scan: %w", err)
		}
		hotels = append(hotels, h)
	}
	return hotels, rows.Err()
}

// ListCities mengambil daftar kota unik dari tabel hotels diurutkan ASC.
func (r *Repository) ListCities(ctx context.Context) ([]string, error) {
	const q = `SELECT DISTINCT city FROM hotels WHERE city IS NOT NULL AND city != '' ORDER BY city ASC`
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("hotel.ListCities: %w", err)
	}
	defer rows.Close()

	cities := make([]string, 0)
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("hotel.ListCities scan: %w", err)
		}
		if c != "" {
			cities = append(cities, c)
		}
	}
	return cities, rows.Err()
}

// normalizeCity mengecek apakah kota dengan nama yang sama (case-insensitive) sudah ada di DB.
// Jika ada, pakai ejaan yang sudah ada di DB. Jika tidak, pakai input user.
func (r *Repository) normalizeCity(ctx context.Context, city string) string {
	trimmed := strings.TrimSpace(city)
	if trimmed == "" {
		return trimmed
	}
	var existingCity string
	err := r.db.QueryRowContext(ctx, `SELECT city FROM hotels WHERE UPPER(TRIM(city)) = UPPER(TRIM(?)) LIMIT 1`, trimmed).Scan(&existingCity)
	if err == nil && existingCity != "" {
		return existingCity
	}
	return trimmed
}

// Create menyisipkan hotel baru. Mengembalikan Hotel lengkap dengan ID & CreatedAt.
// Cek duplikat case-insensitive dilakukan SEBELUM insert.
func (r *Repository) Create(ctx context.Context, name, city string, starRating *int, distanceM *int, photoURL *string) (*Hotel, error) {
	// Cek duplikat (exclude tidak diperlukan pada create)
	if exists, err := r.nameExists(ctx, name, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicate
	}

	city = r.normalizeCity(ctx, city)

	const q = `INSERT INTO hotels (name, city, star_rating, distance_m, photo_url) VALUES (?, ?, ?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, name, city, starRating, distanceM, photoURL)
	if err != nil {
		return nil, fmt.Errorf("hotel.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("hotel.Create LastInsertId: %w", err)
	}

	return r.getByID(ctx, uint64(id))
}

// Update memperbarui hotel berdasarkan id.
// Mengembalikan ErrNotFound jika id tidak ada, ErrDuplicate jika nama sudah dipakai hotel lain.
func (r *Repository) Update(ctx context.Context, id uint64, name, city string, starRating *int, distanceM *int, photoURL *string) (*Hotel, error) {
	// Pastikan row ada
	if _, err := r.getByID(ctx, id); err != nil {
		return nil, err
	}

	// Cek duplikat nama, exclude id sendiri
	if exists, err := r.nameExists(ctx, name, id); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicate
	}

	city = r.normalizeCity(ctx, city)

	const q = `UPDATE hotels SET name=?, city=?, star_rating=?, distance_m=?, photo_url=? WHERE id=?`
	if _, err := r.db.ExecContext(ctx, q, name, city, starRating, distanceM, photoURL, id); err != nil {
		return nil, fmt.Errorf("hotel.Update: %w", err)
	}

	return r.getByID(ctx, id)
}

// Delete menghapus hotel berdasarkan id.
// Mengembalikan ErrNotFound jika id tidak ada.
// Error MySQL 1451 (FK constraint) harus ditangani di layer handler.
func (r *Repository) Delete(ctx context.Context, id uint64) error {
	// Pastikan row ada terlebih dahulu
	if _, err := r.getByID(ctx, id); err != nil {
		return err
	}

	const q = `DELETE FROM hotels WHERE id=?`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// getByID mengambil satu hotel berdasarkan id. Mengembalikan ErrNotFound jika tidak ada.
func (r *Repository) getByID(ctx context.Context, id uint64) (*Hotel, error) {
	const q = `SELECT id, name, city, star_rating, distance_m, photo_url, created_at FROM hotels WHERE id=?`
	var h Hotel
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&h.ID, &h.Name, &h.City, &h.StarRating, &h.DistanceM, &h.PhotoURL, &h.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("hotel.getByID: %w", err)
	}
	return &h, nil
}

// nameExists mengecek apakah nama hotel (case-insensitive) sudah ada.
// excludeID = 0 berarti tidak ada yang di-exclude (dipakai saat Create).
func (r *Repository) nameExists(ctx context.Context, name string, excludeID uint64) (bool, error) {
	const q = `SELECT COUNT(*) FROM hotels WHERE UPPER(name) = UPPER(?) AND id != ?`
	var count int
	err := r.db.QueryRowContext(ctx, q, name, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("hotel.nameExists: %w", err)
	}
	return count > 0, nil
}
