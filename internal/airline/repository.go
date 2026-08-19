package airline

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// ErrNotFound dikembalikan saat row tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// ErrDuplicate dikembalikan saat nama maskapai sudah ada (case-insensitive).
var ErrDuplicate = errors.New("maskapai dengan nama ini sudah ada")

// Repository mengelola semua query ke tabel airlines.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository dengan *sql.DB yang sudah ada.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List mengambil semua maskapai, diurutkan by name ASC.
func (r *Repository) List(ctx context.Context) ([]Airline, error) {
	const q = `SELECT id, name, logo_url, created_at FROM airlines ORDER BY name ASC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("airline.List: %w", err)
	}
	defer rows.Close()

	airlines := make([]Airline, 0) // pastikan tidak nil → serialisasi jadi []
	for rows.Next() {
		var a Airline
		if err := rows.Scan(&a.ID, &a.Name, &a.LogoURL, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("airline.List scan: %w", err)
		}
		airlines = append(airlines, a)
	}
	return airlines, rows.Err()
}

// Create menyisipkan maskapai baru. Mengembalikan Airline lengkap dengan ID & CreatedAt.
func (r *Repository) Create(ctx context.Context, name string, logoURL *string) (*Airline, error) {
	if exists, err := r.nameExists(ctx, name, 0); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicate
	}

	const q = `INSERT INTO airlines (name, logo_url) VALUES (?, ?)`
	res, err := r.db.ExecContext(ctx, q, name, logoURL)
	if err != nil {
		return nil, fmt.Errorf("airline.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("airline.Create LastInsertId: %w", err)
	}

	return r.getByID(ctx, uint64(id))
}

// Update memperbarui maskapai berdasarkan id.
func (r *Repository) Update(ctx context.Context, id uint64, name string, logoURL *string) (*Airline, error) {
	// Pastikan row ada
	if _, err := r.getByID(ctx, id); err != nil {
		return nil, err
	}

	// Cek duplikat, exclude id sendiri
	if exists, err := r.nameExists(ctx, name, id); err != nil {
		return nil, err
	} else if exists {
		return nil, ErrDuplicate
	}

	const q = `UPDATE airlines SET name=?, logo_url=? WHERE id=?`
	if _, err := r.db.ExecContext(ctx, q, name, logoURL, id); err != nil {
		return nil, fmt.Errorf("airline.Update: %w", err)
	}

	return r.getByID(ctx, id)
}

// Delete menghapus maskapai berdasarkan id.
// Error MySQL 1451 (FK constraint) harus ditangani di layer handler.
func (r *Repository) Delete(ctx context.Context, id uint64) error {
	if _, err := r.getByID(ctx, id); err != nil {
		return err
	}

	const q = `DELETE FROM airlines WHERE id=?`
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (r *Repository) getByID(ctx context.Context, id uint64) (*Airline, error) {
	const q = `SELECT id, name, logo_url, created_at FROM airlines WHERE id=?`
	var a Airline
	err := r.db.QueryRowContext(ctx, q, id).Scan(&a.ID, &a.Name, &a.LogoURL, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("airline.getByID: %w", err)
	}
	return &a, nil
}

// nameExists mengecek apakah nama maskapai (case-insensitive) sudah ada.
// excludeID = 0 berarti tidak ada yang di-exclude.
func (r *Repository) nameExists(ctx context.Context, name string, excludeID uint64) (bool, error) {
	const q = `SELECT COUNT(*) FROM airlines WHERE UPPER(name) = UPPER(?) AND id != ?`
	var count int
	err := r.db.QueryRowContext(ctx, q, name, excludeID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("airline.nameExists: %w", err)
	}
	return count > 0, nil
}
