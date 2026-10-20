package destination

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound dikembalikan saat destinasi tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// ErrDuplicateSlug dikembalikan saat slug destinasi sudah dipakai.
var ErrDuplicateSlug = errors.New("slug destinasi sudah digunakan")

// ErrInUse dikembalikan saat destinasi masih dirujuk paket.
var ErrInUse = errors.New("tidak bisa dihapus, masih dipakai oleh")

// Repository mengelola semua query ke tabel destinations.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository dengan *sql.DB yang sudah ada.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const selectDestination = `
	SELECT d.id, d.name, d.slug, d.is_active,
		(SELECT COUNT(*) FROM schedules s WHERE s.destination_id = d.id),
		d.created_at, d.updated_at
	FROM destinations d`

func scanDestination(row interface{ Scan(...any) error }) (*Destination, error) {
	var d Destination
	if err := row.Scan(&d.ID, &d.Name, &d.Slug, &d.IsActive, &d.PackageCount, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return nil, err
	}
	return &d, nil
}

// List mengambil semua destinasi diurutkan nama. activeOnly=true hanya yang aktif.
func (r *Repository) List(ctx context.Context, activeOnly bool) ([]Destination, error) {
	q := selectDestination
	if activeOnly {
		q += " WHERE d.is_active = TRUE"
	}
	q += " ORDER BY d.name ASC"
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("destination.List: %w", err)
	}
	defer rows.Close()

	items := make([]Destination, 0)
	for rows.Next() {
		d, err := scanDestination(rows)
		if err != nil {
			return nil, fmt.Errorf("destination.List scan: %w", err)
		}
		items = append(items, *d)
	}
	return items, rows.Err()
}

// GetByID mengambil satu destinasi.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Destination, error) {
	d, err := scanDestination(r.db.QueryRowContext(ctx, selectDestination+" WHERE d.id = ?", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("destination.GetByID: %w", err)
	}
	return d, nil
}

// Create menyisipkan destinasi baru.
func (r *Repository) Create(ctx context.Context, name, slug string, active bool) (*Destination, error) {
	if taken, err := r.slugTaken(ctx, slug, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrDuplicateSlug
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO destinations (name, slug, is_active) VALUES (?, ?, ?)`, name, slug, active)
	if err != nil {
		return nil, fmt.Errorf("destination.Create: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("destination.Create LastInsertId: %w", err)
	}
	return r.GetByID(ctx, id)
}

// Update memperbarui destinasi. Mengubah slug destinasi yang sudah dipakai paket diizinkan:
// slug hanya dipakai CMS untuk membentuk URL hub, bukan disalin ke paket.
func (r *Repository) Update(ctx context.Context, id int64, name, slug string, active bool) (*Destination, error) {
	if _, err := r.GetByID(ctx, id); err != nil {
		return nil, err
	}
	if taken, err := r.slugTaken(ctx, slug, id); err != nil {
		return nil, err
	} else if taken {
		return nil, ErrDuplicateSlug
	}
	if _, err := r.db.ExecContext(ctx, `UPDATE destinations SET name=?, slug=?, is_active=? WHERE id=?`, name, slug, active, id); err != nil {
		return nil, fmt.Errorf("destination.Update: %w", err)
	}
	return r.GetByID(ctx, id)
}

// Delete menghapus destinasi yang tidak dipakai paket mana pun.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	d, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if d.PackageCount > 0 {
		return fmt.Errorf("%w %d paket", ErrInUse, d.PackageCount)
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM destinations WHERE id=?`, id)
	return err
}

func (r *Repository) slugTaken(ctx context.Context, slug string, excludeID int64) (bool, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM destinations WHERE slug=? AND id<>?`, slug, excludeID).Scan(&n); err != nil {
		return false, fmt.Errorf("destination.slugTaken: %w", err)
	}
	return n > 0, nil
}
