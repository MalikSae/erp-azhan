package dokumen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"erp-azhan/api/internal/notifikasi"
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

// kolomDokumen dipakai bersama scanDokumen; butuh alias d (dokumen_jamaah) dan au (admin_users peninjau).
const kolomDokumen = `d.id, d.jamaah_id, d.jenis, d.file_url, d.status, d.updated_at, d.version, d.rejection_reason,
		d.verified_by, COALESCE(NULLIF(au.display_name, ''), au.email), d.verified_at`

type scanner interface{ Scan(dest ...any) error }

func scanDokumen(s scanner, d *DokumenJamaah) error {
	return s.Scan(&d.ID, &d.JamaahID, &d.Jenis, &d.FileURL, &d.Status, &d.UpdatedAt, &d.Version, &d.RejectionReason,
		&d.VerifiedBy, &d.VerifiedByNama, &d.VerifiedAt)
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListByJamaahID mengambil list dokumen untuk sebuah jamaah.
// Memverifikasi brand via 1-hop JOIN ke jamaah.
func (r *Repository) ListByJamaahID(ctx context.Context, jamaahID int64, brandID *int64) ([]DokumenJamaah, error) {
	q := `
		SELECT ` + kolomDokumen + `
		FROM dokumen_jamaah d
		JOIN jamaah j ON j.id = d.jamaah_id
		LEFT JOIN admin_users au ON au.id = d.verified_by
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
		if err := scanDokumen(rows, &d); err != nil {
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
		SELECT ` + kolomDokumen + `
		FROM dokumen_jamaah d
		JOIN jamaah j ON j.id = d.jamaah_id
		LEFT JOIN admin_users au ON au.id = d.verified_by
		WHERE d.id = ?`

	var args []interface{}
	args = append(args, id)

	if brandID != nil {
		q += " AND j.brand_id = ?"
		args = append(args, *brandID)
	}

	var d DokumenJamaah
	err := scanDokumen(r.db.QueryRowContext(ctx, q, args...), &d)
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
	id, err := UpsertTx(ctx, tx, jamaahID, req)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetByID(ctx, id, nil)
}

// UpsertTx: isi Upsert di dalam transaksi pemanggil, agar pemanggil bisa memeriksa status dokumen
// (SELECT ... FOR UPDATE) lalu menyimpan berkas pengganti secara atomik. Mengembalikan id dokumen;
// commit tanggung jawab pemanggil.
func UpsertTx(ctx context.Context, tx *sql.Tx, jamaahID int64, req *CreateDokumenRequest) (int64, error) {
	var owner int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM jamaah WHERE id=? FOR UPDATE`, jamaahID).Scan(&owner); err != nil {
		return 0, err
	}
	// Versi lama diarsipkan beserta peninjaunya sebelum baris utama ditimpa.
	if _, err := tx.ExecContext(ctx, `INSERT INTO dokumen_jamaah_versions(dokumen_id,version,file_url,status,rejection_reason,verified_by,verified_at) SELECT id,version,file_url,status,rejection_reason,verified_by,verified_at FROM dokumen_jamaah WHERE jamaah_id=? AND jenis=? ON DUPLICATE KEY UPDATE status=VALUES(status),rejection_reason=VALUES(rejection_reason),verified_by=VALUES(verified_by),verified_at=VALUES(verified_at)`, jamaahID, req.Jenis); err != nil {
		return 0, err
	}
	const q = `
		INSERT INTO dokumen_jamaah (jamaah_id, jenis, file_url, status)
		VALUES (?, ?, ?, 'submitted')
		ON DUPLICATE KEY UPDATE 
			file_url = VALUES(file_url),
			version = version + 1,
			rejection_reason = NULL,
			verified_by = NULL,
			verified_at = NULL,
			status = 'submitted',
			updated_at = CURRENT_TIMESTAMP`

	res, err := tx.ExecContext(ctx, q, jamaahID, req.Jenis, req.FileURL)
	if err != nil {
		return 0, fmt.Errorf("dokumen.Upsert: %w", err)
	}

	// LastInsertId() tidak akurat pada cabang UPDATE (ON DUPLICATE KEY), jadi id diambil ulang.
	var id int64
	err = tx.QueryRowContext(ctx, `SELECT id FROM dokumen_jamaah WHERE jamaah_id=? AND jenis=?`, jamaahID, req.Jenis).Scan(&id)
	if err != nil {
		if lastID, errLast := res.LastInsertId(); errLast == nil && lastID > 0 {
			return lastID, nil
		}
		return 0, fmt.Errorf("dokumen.Upsert get id: %w", err)
	}
	return id, nil
}

// ─── UpdateStatus ─────────────────────────────────────────────────────────────

// UpdateStatus mengubah status: approved/rejected (tinjau, hanya dari submitted), atau submitted untuk membatalkan
// persetujuan (hanya dari approved). Koreksi persetujuan: batalkan dulu, lalu tolak. Peninjau dan waktunya dicatat di
// verified_by/verified_at, termasuk saat pembatalan. Versi wajib cocok, jadi berkas pengganti tidak ikut tersetujui
// dengan versi lama. Status asal yang tidak cocok dilaporkan sebagai ErrVersionConflict.
func (r *Repository) UpdateStatus(ctx context.Context, id int64, newStatus string, version int64, reason string, adminID int64) (*DokumenJamaah, error) {
	q := `UPDATE dokumen_jamaah SET status=?,rejection_reason=NULLIF(?,''),verified_by=NULLIF(?,0),verified_at=NOW(),updated_at=CURRENT_TIMESTAMP WHERE id=? AND version=?`
	if newStatus == "submitted" {
		q += ` AND status='approved'`
	} else {
		q += ` AND status='submitted'`
	}
	res, err := r.db.ExecContext(ctx, q, newStatus, reason, adminID, id, version)
	if err != nil {
		return nil, fmt.Errorf("dokumen.UpdateStatus: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, ErrVersionConflict
	}
	d, err := r.GetByID(ctx, id, nil)
	if err == nil && (newStatus == "approved" || newStatus == "rejected") {
		notifikasi.Dokumen(ctx, r.db, d.JamaahID, d.Jenis, newStatus == "approved")
	}
	return d, err
}

// ─── Riwayat ──────────────────────────────────────────────────────────────────

// ListVersi mengambil versi lama sebuah dokumen beserta peninjaunya, terbaru dulu.
// Brand diverifikasi pemanggil lewat GetByID.
func (r *Repository) ListVersi(ctx context.Context, dokumenID int64) ([]VersiDokumen, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT v.version, v.status, v.rejection_reason, v.file_url, COALESCE(NULLIF(au.display_name, ''), au.email), v.verified_at, v.recorded_at
		FROM dokumen_jamaah_versions v
		LEFT JOIN admin_users au ON au.id = v.verified_by
		WHERE v.dokumen_id = ?
		ORDER BY v.version DESC`, dokumenID)
	if err != nil {
		return nil, fmt.Errorf("dokumen.ListVersi: %w", err)
	}
	defer rows.Close()
	items := make([]VersiDokumen, 0)
	for rows.Next() {
		var v VersiDokumen
		if err := rows.Scan(&v.Version, &v.Status, &v.RejectionReason, &v.FileURL, &v.VerifiedByNama, &v.VerifiedAt, &v.RecordedAt); err != nil {
			return nil, fmt.Errorf("dokumen.ListVersi scan: %w", err)
		}
		items = append(items, v)
	}
	return items, rows.Err()
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
