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
		SELECT i.id, i.title, i.category_id, COALESCE(c.name, ''), COUNT(d.id) AS day_count, i.created_at
		FROM itineraries i
		LEFT JOIN package_categories c ON c.id = i.category_id
		LEFT JOIN itinerary_days d ON d.itinerary_id = i.id
		WHERE i.schedule_id IS NULL
		GROUP BY i.id, i.title, i.category_id, c.name, i.created_at
		ORDER BY i.created_at DESC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("itinerary.List: %w", err)
	}
	defer rows.Close()

	items := make([]ItineraryListItem, 0)
	for rows.Next() {
		var it ItineraryListItem
		var categoryID sql.NullInt64
		if err := rows.Scan(&it.ID, &it.Title, &categoryID, &it.CategoryName, &it.DayCount, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("itinerary.List scan: %w", err)
		}
		if categoryID.Valid {
			it.CategoryID = &categoryID.Int64
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

// GetPublishedByID mengambil itinerary hanya jika dipakai oleh minimal satu
// jadwal published. Ini mencegah itinerary paket draft bocor ke endpoint publik.
func (r *Repository) GetPublishedByID(ctx context.Context, id int64) (*Itinerary, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM schedules
			WHERE itinerary_id = ? AND status = 'published'
		)`, id).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("itinerary.GetPublishedByID: %w", err)
	}
	if !exists {
		return nil, ErrNotFound
	}
	return r.fetchFull(ctx, id)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Create menyisipkan itinerary baru beserta days-nya dalam satu transaction.
// Jika gagal di tengah, seluruh operasi di-ROLLBACK.
func (r *Repository) Create(ctx context.Context, title string, categoryID *int64, days []DayRequest) (*Itinerary, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("itinerary.Create begin tx: %w", err)
	}
	defer tx.Rollback() // no-op jika sudah Commit

	// INSERT itinerary
	res, err := tx.ExecContext(ctx, `INSERT INTO itineraries (title, category_id) VALUES (?, ?)`, title, categoryID)
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
func (r *Repository) Update(ctx context.Context, id int64, title string, categoryID *int64, days []DayRequest) (*Itinerary, error) {
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
	if _, err := tx.ExecContext(ctx, `UPDATE itineraries SET title=?, category_id=? WHERE id=?`, title, categoryID, id); err != nil {
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

// CategoryExists memeriksa apakah kategori paket dengan id tersebut ada.
func (r *Repository) CategoryExists(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM package_categories WHERE id=?`, id).Scan(&count)
	return count > 0, err
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Delete menghapus itinerary berdasarkan id.
// itinerary_days terhapus otomatis via ON DELETE CASCADE.
// Error MySQL 1451 (FK ke schedules) harus ditangani di layer handler.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	if err := r.exists(ctx, id); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM itineraries WHERE id=? AND schedule_id IS NULL`, id)
	return err
}

// ─── Itinerary khusus (salinan pribadi milik satu paket) ──────────────────────

// Execer dipenuhi *sql.DB dan *sql.Tx.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// KhususID mengembalikan id itinerary pribadi milik paket, atau nil bila belum ada.
func KhususID(ctx context.Context, q Execer, scheduleID int64) (*int64, error) {
	var id int64
	err := q.QueryRowContext(ctx, `SELECT id FROM itineraries WHERE schedule_id=?`, scheduleID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("itinerary.KhususID: %w", err)
	}
	return &id, nil
}

// SimpanKhusus membuat atau memperbarui itinerary pribadi paket (judul dan semua hari diganti)
// di dalam transaksi pemanggil, lalu mengembalikan id-nya.
func SimpanKhusus(ctx context.Context, tx *sql.Tx, scheduleID int64, title string, days []DayRequest) (int64, error) {
	id, err := KhususID(ctx, tx, scheduleID)
	if err != nil {
		return 0, err
	}
	if id == nil {
		res, err := tx.ExecContext(ctx, `INSERT INTO itineraries (title, schedule_id) VALUES (?, ?)`, title, scheduleID)
		if err != nil {
			return 0, fmt.Errorf("itinerary.SimpanKhusus insert: %w", err)
		}
		newID, err := res.LastInsertId()
		if err != nil {
			return 0, fmt.Errorf("itinerary.SimpanKhusus last insert id: %w", err)
		}
		id = &newID
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE itineraries SET title=? WHERE id=?`, title, *id); err != nil {
			return 0, fmt.Errorf("itinerary.SimpanKhusus update: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM itinerary_days WHERE itinerary_id=?`, *id); err != nil {
			return 0, fmt.Errorf("itinerary.SimpanKhusus hapus hari: %w", err)
		}
	}
	if err := insertDays(ctx, tx, *id, days); err != nil {
		return 0, err
	}
	return *id, nil
}

// HapusKhusus menghapus itinerary pribadi paket (hari ikut terhapus lewat CASCADE), kecuali yang id-nya
// sama dengan keep. Dipanggil setelah schedules.itinerary_id tidak lagi menunjuk baris yang dihapus.
func HapusKhusus(ctx context.Context, tx Execer, scheduleID int64, keep *int64) error {
	var err error
	if keep == nil {
		_, err = tx.ExecContext(ctx, `DELETE FROM itineraries WHERE schedule_id=?`, scheduleID)
	} else {
		_, err = tx.ExecContext(ctx, `DELETE FROM itineraries WHERE schedule_id=? AND id<>?`, scheduleID, *keep)
	}
	if err != nil {
		return fmt.Errorf("itinerary.HapusKhusus: %w", err)
	}
	return nil
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// exists mengecek apakah itinerary dengan id tersebut ada. Mengembalikan ErrNotFound jika tidak.
func (r *Repository) exists(ctx context.Context, id int64) error {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM itineraries WHERE id=? AND schedule_id IS NULL`, id).Scan(&count)
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
	var categoryID sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT id, title, category_id, created_at FROM itineraries WHERE id=?`, id,
	).Scan(&it.ID, &it.Title, &categoryID, &it.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("itinerary.fetchFull header: %w", err)
	}
	if categoryID.Valid {
		it.CategoryID = &categoryID.Int64
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
