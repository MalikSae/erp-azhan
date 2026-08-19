package itinerary

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNotFound dikembalikan saat itinerary tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// Repository mengelola semua query ke tabel itineraries dan itinerary_days.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository dengan *sql.DB yang sudah ada.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// List mengambil semua itinerary (ringkas, tanpa days) diurutkan by created_at DESC.
// DayCount didapat dari COUNT(*) JOIN ke itinerary_days.
func (r *Repository) List(ctx context.Context) ([]ItineraryListItem, error) {
	const q = `
		SELECT i.id, i.title, COUNT(d.id) AS day_count, i.created_at
		FROM itineraries i
		LEFT JOIN itinerary_days d ON d.itinerary_id = i.id
		GROUP BY i.id, i.title, i.created_at
		ORDER BY i.created_at DESC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("itinerary.List: %w", err)
	}
	defer rows.Close()

	items := make([]ItineraryListItem, 0)
	for rows.Next() {
		var it ItineraryListItem
		if err := rows.Scan(&it.ID, &it.Title, &it.DayCount, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("itinerary.List scan: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ─── Get Detail ───────────────────────────────────────────────────────────────

// GetByID mengambil itinerary lengkap beserta semua days-nya.
// Mengembalikan ErrNotFound jika itinerary tidak ada.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Itinerary, error) {
	return r.fetchFull(ctx, id)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Create menyisipkan itinerary baru beserta days-nya dalam satu transaction.
// Jika gagal di tengah, seluruh operasi di-ROLLBACK.
func (r *Repository) Create(ctx context.Context, title string, days []DayRequest) (*Itinerary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("itinerary.Create begin tx: %w", err)
	}
	defer tx.Rollback() // no-op jika sudah Commit

	// INSERT itinerary
	res, err := tx.ExecContext(ctx, `INSERT INTO itineraries (title) VALUES (?)`, title)
	if err != nil {
		return nil, fmt.Errorf("itinerary.Create insert itinerary: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("itinerary.Create last insert id: %w", err)
	}

	// INSERT days
	if err := insertDays(ctx, tx, id, days); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("itinerary.Create commit: %w", err)
	}

	return r.fetchFull(ctx, id)
}

// ─── Update ───────────────────────────────────────────────────────────────────

// Update memperbarui itinerary dan mengganti semua days-nya (delete+reinsert) dalam satu transaction.
func (r *Repository) Update(ctx context.Context, id int64, title string, days []DayRequest) (*Itinerary, error) {
	// Cek eksistensi dulu (di luar tx, read-only)
	if err := r.exists(ctx, id); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("itinerary.Update begin tx: %w", err)
	}
	defer tx.Rollback()

	// UPDATE title
	if _, err := tx.ExecContext(ctx, `UPDATE itineraries SET title=? WHERE id=?`, title, id); err != nil {
		return nil, fmt.Errorf("itinerary.Update update title: %w", err)
	}

	// DELETE semua days lama (ON DELETE CASCADE sudah handle itinerary_days,
	// tapi di sini kita delete manual karena hanya hapus days, bukan itinerary-nya)
	if _, err := tx.ExecContext(ctx, `DELETE FROM itinerary_days WHERE itinerary_id=?`, id); err != nil {
		return nil, fmt.Errorf("itinerary.Update delete days: %w", err)
	}

	// INSERT days baru
	if err := insertDays(ctx, tx, id, days); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("itinerary.Update commit: %w", err)
	}

	return r.fetchFull(ctx, id)
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Delete menghapus itinerary berdasarkan id.
// itinerary_days terhapus otomatis via ON DELETE CASCADE.
// Error MySQL 1451 (FK ke schedules) harus ditangani di layer handler.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	if err := r.exists(ctx, id); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM itineraries WHERE id=?`, id)
	return err
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// exists mengecek apakah itinerary dengan id tersebut ada. Mengembalikan ErrNotFound jika tidak.
func (r *Repository) exists(ctx context.Context, id int64) error {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM itineraries WHERE id=?`, id).Scan(&count)
	if err != nil {
		return fmt.Errorf("itinerary.exists: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

// fetchFull mengambil itinerary beserta semua days dan activities-nya.
func (r *Repository) fetchFull(ctx context.Context, id int64) (*Itinerary, error) {
	// Ambil itinerary header
	var it Itinerary
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, created_at FROM itineraries WHERE id=?`, id,
	).Scan(&it.ID, &it.Title, &it.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("itinerary.fetchFull header: %w", err)
	}

	// Ambil days
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, itinerary_id, day_number, title, location, activities
		 FROM itinerary_days WHERE itinerary_id=? ORDER BY day_number ASC`, id,
	)
	if err != nil {
		return nil, fmt.Errorf("itinerary.fetchFull days: %w", err)
	}
	defer rows.Close()

	it.Days = make([]ItineraryDay, 0)
	for rows.Next() {
		var d ItineraryDay
		var activitiesJSON []byte
		if err := rows.Scan(&d.ID, &d.ItineraryID, &d.DayNumber, &d.Title, &d.Location, &activitiesJSON); err != nil {
			return nil, fmt.Errorf("itinerary.fetchFull day scan: %w", err)
		}
		// Unmarshal JSON activities → []Activity
		if err := json.Unmarshal(activitiesJSON, &d.Activities); err != nil {
			return nil, fmt.Errorf("itinerary.fetchFull activities unmarshal: %w", err)
		}
		it.Days = append(it.Days, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("itinerary.fetchFull rows: %w", err)
	}

	return &it, nil
}

// insertDays menyisipkan slice DayRequest ke itinerary_days dalam konteks tx.
// day_number ditentukan dari index+1 (abaikan day_number dari client).
func insertDays(ctx context.Context, tx *sql.Tx, itineraryID int64, days []DayRequest) error {
	if len(days) == 0 {
		return nil
	}

	const q = `INSERT INTO itinerary_days (itinerary_id, day_number, title, location, activities) VALUES (?, ?, ?, ?, ?)`
	for i, d := range days {
		dayNumber := i + 1

		activitiesJSON, err := json.Marshal(d.Activities)
		if err != nil {
			return fmt.Errorf("itinerary.insertDays marshal activities day %d: %w", dayNumber, err)
		}

		if _, err := tx.ExecContext(ctx, q, itineraryID, dayNumber, d.Title, d.Location, activitiesJSON); err != nil {
			return fmt.Errorf("itinerary.insertDays insert day %d: %w", dayNumber, err)
		}
	}
	return nil
}
