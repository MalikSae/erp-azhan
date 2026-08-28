package airport

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// ErrNotFound dikembalikan saat row tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// ErrDuplicateCode dikembalikan saat kode bandara sudah ada (case-insensitive).
var ErrDuplicateCode = errors.New("kode bandara sudah digunakan")

// Repository mengelola semua query ke tabel airports.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository dengan *sql.DB yang sudah ada.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List mengambil semua bandara dengan filter search opsional (nama, kode, atau kota).
func (r *Repository) List(ctx context.Context, search string) ([]Airport, error) {
	var (
		rows *sql.Rows
		err  error
	)

	search = strings.TrimSpace(search)
	if search != "" {
		const q = `
			SELECT id, name, code, city, created_at, updated_at 
			FROM airports 
			WHERE name LIKE ? OR code LIKE ? OR city LIKE ? 
			ORDER BY name ASC
		`
		pattern := "%" + search + "%"
		rows, err = r.db.QueryContext(ctx, q, pattern, pattern, pattern)
	} else {
		const q = `
			SELECT id, name, code, city, created_at, updated_at 
			FROM airports 
			ORDER BY name ASC
		`
		rows, err = r.db.QueryContext(ctx, q)
	}

	if err != nil {
		return nil, fmt.Errorf("airport.List: %w", err)
	}
	defer rows.Close()

	airports := make([]Airport, 0)
	for rows.Next() {
		var a Airport
		if err := rows.Scan(&a.ID, &a.Name, &a.Code, &a.City, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, fmt.Errorf("airport.List scan: %w", err)
		}
		airports = append(airports, a)
	}
	return airports, rows.Err()
}

// Create menyisipkan bandara baru.
func (r *Repository) Create(ctx context.Context, name string, code string, city string) (*Airport, error) {
	if exists, err := r.codeExists(ctx, code, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicateCode
	}

	const q = `INSERT INTO airports (name, code, city) VALUES (?, ?, ?)`
	res, err := r.db.ExecContext(ctx, q, name, code, city)
	if err != nil {
		return nil, fmt.Errorf("airport.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("airport.Create LastInsertId: %w", err)
	}

	return r.getByID(ctx, uint64(id))
}

// Update memperbarui bandara berdasarkan id.
func (r *Repository) Update(ctx context.Context, id uint64, name string, code string, city string) (*Airport, error) {
	if _, err := r.getByID(ctx, id); err != nil {
		return nil, err
	}

	if exists, err := r.codeExists(ctx, code, id); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicateCode
	}

	const q = `UPDATE airports SET name=?, code=?, city=? WHERE id=?`
	if _, err := r.db.ExecContext(ctx, q, name, code, city, id); err != nil {
		return nil, fmt.Errorf("airport.Update: %w", err)
	}

	return r.getByID(ctx, id)
}

// Delete menghapus bandara berdasarkan id.
func (r *Repository) Delete(ctx context.Context, id uint64) error {
	if _, err := r.getByID(ctx, id); err != nil {
		return err
	}

	const q = `DELETE FROM airports WHERE id=?`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (r *Repository) getByID(ctx context.Context, id uint64) (*Airport, error) {
	const q = `SELECT id, name, code, city, created_at, updated_at FROM airports WHERE id=?`
	var a Airport
	err := r.db.QueryRowContext(ctx, q, id).Scan(&a.ID, &a.Name, &a.Code, &a.City, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("airport.getByID: %w", err)
	}
	return &a, nil
}

func (r *Repository) codeExists(ctx context.Context, code string, excludeID uint64) (bool, error) {
	const q = `SELECT COUNT(*) FROM airports WHERE UPPER(code) = UPPER(?) AND id != ?`
	var count int
	err := r.db.QueryRowContext(ctx, q, code, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("airport.codeExists: %w", err)
	}
	return count > 0, nil
}
