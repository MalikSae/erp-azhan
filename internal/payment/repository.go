package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Sentinel errors
var (
	ErrNotFound      = errors.New("data tidak ditemukan")
	ErrCannotDelete  = errors.New("tidak bisa menghapus pembayaran yang sudah dikonfirmasi")
	ErrInvalidStatus = errors.New("status tidak valid")
)

// Repository mengelola semua query ke tabel payments.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ListDailyBrandTransactions mengambil pembayaran confirmed selama 30 hari,
// termasuk hari ini, dan mengelompokkannya berdasarkan brand dan tanggal.
func (r *Repository) ListDailyBrandTransactions(ctx context.Context) ([]DailyBrandTransaction, error) {
	const q = `
		SELECT DATE_FORMAT(COALESCE(p.tanggal, DATE(p.created_at)), '%Y-%m-%d') AS payment_date,
			s.brand_id, COALESCE(SUM(p.jumlah), 0), COUNT(*)
		FROM payments p
		JOIN bookings b ON b.id = p.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE p.status = 'confirmed'
			AND COALESCE(p.tanggal, DATE(p.created_at)) BETWEEN CURDATE() - INTERVAL 29 DAY AND CURDATE()
		GROUP BY payment_date, s.brand_id
		ORDER BY payment_date ASC, s.brand_id ASC`

	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("payment.ListDailyBrandTransactions: %w", err)
	}
	defer rows.Close()

	items := make([]DailyBrandTransaction, 0)
	for rows.Next() {
		var item DailyBrandTransaction
		if err := rows.Scan(&item.Date, &item.BrandID, &item.TotalAmount, &item.Count); err != nil {
			return nil, fmt.Errorf("payment.ListDailyBrandTransactions scan: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListByBookingID mengambil list payment untuk sebuah booking.
// Memverifikasi brand via 2-hop JOIN: payments -> bookings -> schedules.
func (r *Repository) ListByBookingID(ctx context.Context, bookingID int64, brandID *int64) ([]Payment, error) {
	q := `
		SELECT p.id, p.booking_id, p.jumlah, p.metode, 
			DATE_FORMAT(p.tanggal, '%Y-%m-%d') AS tanggal, p.status, p.bukti_url, p.created_at
		FROM payments p
		JOIN bookings b ON b.id = p.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE p.booking_id = ?`

	var args []interface{}
	args = append(args, bookingID)

	if brandID != nil {
		q += " AND s.brand_id = ?"
		args = append(args, *brandID)
	}
	q += " ORDER BY p.created_at ASC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("payment.List: %w", err)
	}
	defer rows.Close()

	items := make([]Payment, 0)
	for rows.Next() {
		var p Payment
		var buktiURL sql.NullString
		if err := rows.Scan(&p.ID, &p.BookingID, &p.Jumlah, &p.Metode, &p.Tanggal, &p.Status, &buktiURL, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("payment.List scan: %w", err)
		}
		if buktiURL.Valid {
			p.BuktiURL = &buktiURL.String
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

// GetByID mengambil payment berdasarkan ID.
// Memverifikasi brand via 2-hop JOIN.
func (r *Repository) GetByID(ctx context.Context, id int64, brandID *int64) (*Payment, error) {
	q := `
		SELECT p.id, p.booking_id, p.jumlah, p.metode, 
			DATE_FORMAT(p.tanggal, '%Y-%m-%d') AS tanggal, p.status, p.bukti_url, p.created_at
		FROM payments p
		JOIN bookings b ON b.id = p.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE p.id = ?`

	var args []interface{}
	args = append(args, id)

	if brandID != nil {
		q += " AND s.brand_id = ?"
		args = append(args, *brandID)
	}

	var p Payment
	var buktiURL sql.NullString
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&p.ID, &p.BookingID, &p.Jumlah, &p.Metode, &p.Tanggal, &p.Status, &buktiURL, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("payment.GetByID: %w", err)
	}
	if buktiURL.Valid {
		p.BuktiURL = &buktiURL.String
	}
	return &p, nil
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Create menambahkan payment baru. Status selalu 'pending'.
func (r *Repository) Create(ctx context.Context, bookingID int64, req *CreatePaymentRequest) (*Payment, error) {
	const q = `INSERT INTO payments (booking_id, jumlah, metode, tanggal, status, bukti_url)
		VALUES (?, ?, ?, ?, 'pending', ?)`

	res, err := r.db.ExecContext(ctx, q, bookingID, req.Jumlah, req.Metode, req.Tanggal, req.BuktiURL)
	if err != nil {
		return nil, fmt.Errorf("payment.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("payment.Create LastInsertId: %w", err)
	}

	return r.GetByID(ctx, id, nil)
}

// GetBookingTotalAndPaid mengambil total_harga booking dan total pembayaran yang sudah dikonfirmasi.
func (r *Repository) GetBookingTotalAndPaid(ctx context.Context, bookingID int64) (*float64, float64, error) {
	q := `
		SELECT b.total_harga, COALESCE(SUM(p.jumlah), 0)
		FROM bookings b
		LEFT JOIN payments p ON p.booking_id = b.id AND p.status = 'confirmed'
		WHERE b.id = ?
		GROUP BY b.id, b.total_harga`
	var totalHarga sql.NullFloat64
	var totalPaid float64
	err := r.db.QueryRowContext(ctx, q, bookingID).Scan(&totalHarga, &totalPaid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("payment.GetBookingTotalAndPaid: %w", err)
	}
	if totalHarga.Valid {
		return &totalHarga.Float64, totalPaid, nil
	}
	return nil, totalPaid, nil
}

// ─── UpdateStatus ─────────────────────────────────────────────────────────────

// UpdateStatus mengubah status payment dan otomatis menyinkronkan status booking (Lunas/DP).
func (r *Repository) UpdateStatus(ctx context.Context, id int64, newStatus string) (*Payment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("payment.UpdateStatus tx: %w", err)
	}
	defer tx.Rollback()

	var bookingID int64
	err = tx.QueryRowContext(ctx, `SELECT booking_id FROM payments WHERE id=?`, id).Scan(&bookingID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("payment.UpdateStatus get booking_id: %w", err)
	}

	_, err = tx.ExecContext(ctx, `UPDATE payments SET status=? WHERE id=?`, newStatus, id)
	if err != nil {
		return nil, fmt.Errorf("payment.UpdateStatus update: %w", err)
	}

	if err := r.syncBookingStatusTx(ctx, tx, bookingID); err != nil {
		return nil, fmt.Errorf("payment.UpdateStatus sync booking: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("payment.UpdateStatus commit: %w", err)
	}

	return r.GetByID(ctx, id, nil)
}

func (r *Repository) syncBookingStatusTx(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	var totalHarga sql.NullFloat64
	var currentStatus string
	var scheduleID int64
	err := tx.QueryRowContext(ctx, `SELECT total_harga, status, schedule_id FROM bookings WHERE id=? FOR UPDATE`, bookingID).
		Scan(&totalHarga, &currentStatus, &scheduleID)
	if err != nil {
		return err
	}

	// Jangan ubah jika booking sudah dibatalkan
	if currentStatus == "batal" {
		return nil
	}

	var totalPaid float64
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah), 0) FROM payments WHERE booking_id=? AND status='confirmed'`, bookingID).
		Scan(&totalPaid)
	if err != nil {
		return err
	}

	targetStatus := currentStatus
	targetHarga := 0.0
	if totalHarga.Valid {
		targetHarga = totalHarga.Float64
	}

	if targetHarga > 0 && totalPaid >= targetHarga {
		targetStatus = "lunas"
	} else if totalPaid > 0 {
		if currentStatus == "baru" || currentStatus == "lunas" {
			targetStatus = "dp"
		}
	} else if totalPaid == 0 && currentStatus == "lunas" {
		targetStatus = "dp"
	}

	if targetStatus != currentStatus {
		if currentStatus == "baru" && (targetStatus == "dp" || targetStatus == "lunas") {
			_, err = tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa = GREATEST(0, seat_sisa - 1) WHERE id=?`, scheduleID)
			if err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE bookings SET status=? WHERE id=?`, targetStatus, bookingID)
		if err != nil {
			return err
		}
	}

	return nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Delete menghapus payment jika status masih 'pending'.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	var status string
	err := r.db.QueryRowContext(ctx, `SELECT status FROM payments WHERE id=?`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("payment.Delete cek status: %w", err)
	}

	if status != "pending" {
		return ErrCannotDelete
	}

	_, err = r.db.ExecContext(ctx, `DELETE FROM payments WHERE id=?`, id)
	return err
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

// BookingExistsForBrand memeriksa apakah booking_id ada dan sesuai brand.
func (r *Repository) BookingExistsForBrand(ctx context.Context, bookingID int64, brandID *int64) (bool, error) {
	q := `SELECT COUNT(*) FROM bookings b JOIN schedules s ON s.id = b.schedule_id WHERE b.id=?`
	var args []interface{}
	args = append(args, bookingID)
	if brandID != nil {
		q += " AND s.brand_id=?"
		args = append(args, *brandID)
	}
	var count int
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&count)
	return count > 0, err
}
