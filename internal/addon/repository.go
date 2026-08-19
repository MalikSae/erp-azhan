package addon

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound dikembalikan saat row tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// ErrDuplicate dikembalikan saat nama add-on sudah ada (case-insensitive).
var ErrDuplicate = errors.New("add-on dengan nama ini sudah ada")

// Repository mengelola semua query ke tabel add_ons.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository dengan *sql.DB yang sudah ada.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List mengambil semua add_ons, diurutkan by name ASC.
func (r *Repository) List(ctx context.Context) ([]AddOn, error) {
	const q = `SELECT id, name, created_at FROM add_ons ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("addon.List: %w", err)
	}
	defer rows.Close()

	addons := make([]AddOn, 0) // pastikan tidak nil
	for rows.Next() {
		var a AddOn
		if err := rows.Scan(&a.ID, &a.Name, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("addon.List scan: %w", err)
		}
		addons = append(addons, a)
	}
	return addons, rows.Err()
}

// Create menyisipkan add-on baru.
func (r *Repository) Create(ctx context.Context, name string) (*AddOn, error) {
	if exists, err := r.nameExists(ctx, name, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicate
	}

	const q = `INSERT INTO add_ons (name) VALUES (?)`
	res, err := r.db.ExecContext(ctx, q, name)
	if err != nil {
		return nil, fmt.Errorf("addon.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("addon.Create LastInsertId: %w", err)
	}

	return r.getByID(ctx, uint64(id))
}

// Update memperbarui add-on berdasarkan id.
func (r *Repository) Update(ctx context.Context, id uint64, name string) (*AddOn, error) {
	if _, err := r.getByID(ctx, id); err != nil {
		return nil, err
	}

	if exists, err := r.nameExists(ctx, name, id); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicate
	}

	const q = `UPDATE add_ons SET name=? WHERE id=?`
	if _, err := r.db.ExecContext(ctx, q, name, id); err != nil {
		return nil, fmt.Errorf("addon.Update: %w", err)
	}

	return r.getByID(ctx, id)
}

// Delete menghapus add-on berdasarkan id.
func (r *Repository) Delete(ctx context.Context, id uint64) error {
	if _, err := r.getByID(ctx, id); err != nil {
		return err
	}

	const q = `DELETE FROM add_ons WHERE id=?`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (r *Repository) getByID(ctx context.Context, id uint64) (*AddOn, error) {
	const q = `SELECT id, name, created_at FROM add_ons WHERE id=?`
	var a AddOn
	err := r.db.QueryRowContext(ctx, q, id).Scan(&a.ID, &a.Name, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("addon.getByID: %w", err)
	}
	return &a, nil
}

func (r *Repository) nameExists(ctx context.Context, name string, excludeID uint64) (bool, error) {
	const q = `SELECT COUNT(*) FROM add_ons WHERE UPPER(name) = UPPER(?) AND id != ?`
	var count int
	err := r.db.QueryRowContext(ctx, q, name, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("addon.nameExists: %w", err)
	}
	return count > 0, nil
}
