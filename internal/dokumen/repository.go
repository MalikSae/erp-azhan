package dokumen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Sentinel errors
var (
	ErrNotFound        = errors.New("data tidak ditemukan")
	ErrVersionConflict = errors.New("dokumen telah berubah, muat ulang dan tinjau berkas terbaru")
)

// Repository mengelola query ke tabel dokumen_jamaah.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListByJamaahID mengambil list dokumen untuk sebuah jamaah.
// Memverifikasi brand via 1-hop JOIN ke jamaah.
func (r *Repository) ListByJamaahID(ctx context.Context, jamaahID int64, brandID *int64) ([]DokumenJamaah, error) {
	q := `
		SELECT d.id, d.jamaah_id, d.jenis, d.file_url, d.status, d.updated_at, d.version, d.rejection_reason
		FROM dokumen_jamaah d
		JOIN jamaah j ON j.id = d.jamaah_id
		WHERE d.jamaah_id = ?`

	var args []interface{}
	args = append(args, jamaahID)

	if brandID != nil {
		q += " AND j.brand_id = ?"
		args = append(args, *brandID)
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("dokumen.List: %w", err)
	}
	defer rows.Close()

	items := make([]DokumenJamaah, 0)
	for rows.Next() {
		var d DokumenJamaah
		if err := rows.Scan(&d.ID, &d.JamaahID, &d.Jenis, &d.FileURL, &d.Status, &d.UpdatedAt, &d.Version, &d.RejectionReason); err != nil {
			return nil, fmt.Errorf("dokumen.List scan: %w", err)
		}
		items = append(items, d)
	}
	return items, rows.Err()
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

// GetByID mengambil dokumen berdasarkan ID, dengan verifikasi brand.
func (r *Repository) GetByID(ctx context.Context, id int64, brandID *int64) (*DokumenJamaah, error) {
	q := `
		SELECT d.id, d.jamaah_id, d.jenis, d.file_url, d.status, d.updated_at, d.version, d.rejection_reason
		FROM dokumen_jamaah d
		JOIN jamaah j ON j.id = d.jamaah_id
		WHERE d.id = ?`

	var args []interface{}
	args = append(args, id)

	if brandID != nil {
		q += " AND j.brand_id = ?"
		args = append(args, *brandID)
	}

	var d DokumenJamaah
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&d.ID, &d.JamaahID, &d.Jenis, &d.FileURL, &d.Status, &d.UpdatedAt, &d.Version, &d.RejectionReason)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("dokumen.GetByID: %w", err)
	}
	return &d, nil
}

// ─── Upsert ───────────────────────────────────────────────────────────────────

// Upsert menyisipkan dokumen baru atau mengupdate (replace) file_url jika sudah ada,
// memanfaatkan UNIQUE KEY (jamaah_id, jenis). Status di-reset ke 'submitted'.
func (r *Repository) Upsert(ctx context.Context, jamaahID int64, req *CreateDokumenRequest) (*DokumenJamaah, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	var owner int64
	if err = tx.QueryRowContext(ctx, `SELECT id FROM jamaah WHERE id=? FOR UPDATE`, jamaahID).Scan(&owner); err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO dokumen_jamaah_versions(dokumen_id,version,file_url,status,rejection_reason) SELECT id,version,file_url,status,rejection_reason FROM dokumen_jamaah WHERE jamaah_id=? AND jenis=? ON DUPLICATE KEY UPDATE status=VALUES(status),rejection_reason=VALUES(rejection_reason)`, jamaahID, req.Jenis)
	if err != nil {
		return nil, err
	}
	const q = `
		INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status)
		VALUES (?, ?, ?, 'submitted')
		ON DUPLICATE KEY UPDATE 
			file_url = VALUES(file_url),
			version = version + 1,
			rejection_reason = NULL,
			status = 'submitted',
			updated_at = CURRENT_TIMESTAMP`

	res, err := tx.ExecContext(ctx, q, jamaahID, req.Jenis, req.FileURL)
	if err != nil {
		return nil, fmt.Errorf("dokumen.Upsert: %w", err)
	}

	// Untuk mendapatkan ID setelah upsert, kita perlu SELECT ulang
	// karena LastInsertId() mungkin tidak akurat pada UPDATE (ON DUPLICATE KEY)
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM dokumen_jamaah WHERE jamaah_id=? AND jenis=?`, jamaahID, req.Jenis).Scan(&id)
	if err != nil {
		// Fallback ke LastInsertId jika query ini gagal, meski jarang terjadi
		lastID, errLast := res.LastInsertId()
		if errLast == nil && lastID > 0 {
			id = lastID
		} else {
			return nil, fmt.Errorf("dokumen.Upsert get id: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id, nil)
}

// ─── UpdateStatus ─────────────────────────────────────────────────────────────

// UpdateStatus mengubah status (approved/rejected).
func (r *Repository) UpdateStatus(ctx context.Context, id int64, newStatus string, version int64, reason string) (*DokumenJamaah, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE dokumen_jamaah SET status=?,rejection_reason=NULLIF(?,''),updated_at=CURRENT_TIMESTAMP WHERE id=? AND version=?`, newStatus, reason, id, version)
	if err != nil {
		return nil, fmt.Errorf("dokumen.UpdateStatus: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrVersionConflict
	}
	return r.GetByID(ctx, id, nil)
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// JamaahExistsForBrand memeriksa apakah jamaah_id ada dan sesuai brand.
func (r *Repository) JamaahExistsForBrand(ctx context.Context, jamaahID int64, brandID *int64) (bool, error) {
	q := `SELECT COUNT(*) FROM jamaah WHERE id=?`
	var args []interface{}
	args = append(args, jamaahID)
	if brandID != nil {
		q += " AND brand_id=?"
		args = append(args, *brandID)
	}
	var count int
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&count)
	return count > 0, err
}
