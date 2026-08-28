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
		SELECT p.id,p.booking_id,p.jumlah,p.metode,DATE_FORMAT(p.tanggal,'%Y-%m-%d'),p.status,p.bukti_url,p.bank_account_id,
			p.destination_bank_name,p.destination_account_number,p.destination_account_holder,p.sender_name,p.sender_bank,p.notes,p.source,p.rejection_reason,p.verified_by,p.verified_at,p.created_at
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
		if err := rows.Scan(&p.ID, &p.BookingID, &p.Jumlah, &p.Metode, &p.Tanggal, &p.Status, &p.BuktiURL, &p.BankAccountID, &p.DestinationBankName, &p.DestinationAccountNumber, &p.DestinationAccountHolder, &p.SenderName, &p.SenderBank, &p.Notes, &p.Source, &p.RejectionReason, &p.VerifiedBy, &p.VerifiedAt, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("payment.List scan: %w", err)
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
		SELECT p.id,p.booking_id,p.jumlah,p.metode,DATE_FORMAT(p.tanggal,'%Y-%m-%d'),p.status,p.bukti_url,p.bank_account_id,
			p.destination_bank_name,p.destination_account_number,p.destination_account_holder,p.sender_name,p.sender_bank,p.notes,p.source,p.rejection_reason,p.verified_by,p.verified_at,p.created_at
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
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&p.ID, &p.BookingID, &p.Jumlah, &p.Metode, &p.Tanggal, &p.Status, &p.BuktiURL, &p.BankAccountID, &p.DestinationBankName, &p.DestinationAccountNumber, &p.DestinationAccountHolder, &p.SenderName, &p.SenderBank, &p.Notes, &p.Source, &p.RejectionReason, &p.VerifiedBy, &p.VerifiedAt, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("payment.GetByID: %w", err)
	}
	return &p, nil
}

func (r *Repository) ListAll(ctx context.Context, brandID *int64, status string) ([]Payment, error) {
	q := `SELECT p.id,p.booking_id,p.jumlah,p.metode,DATE_FORMAT(p.tanggal,'%Y-%m-%d'),p.status,p.bukti_url,p.bank_account_id,p.destination_bank_name,p.destination_account_number,p.destination_account_holder,p.sender_name,p.sender_bank,p.notes,p.source,p.rejection_reason,p.verified_by,p.verified_at,p.created_at,j.nama_lengkap,s.jadwal_nama,br.name,b.id_booking FROM payments p JOIN bookings b ON b.id=p.booking_id JOIN jamaah j ON j.id=b.jamaah_id JOIN schedules s ON s.id=b.schedule_id JOIN brands br ON br.id=s.brand_id WHERE 1=1`
	args := []any{}
	if brandID != nil {
		q += " AND s.brand_id=?"
		args = append(args, *brandID)
	}
	if status != "" {
		q += " AND p.status=?"
		args = append(args, status)
	}
	q += " ORDER BY CASE p.status WHEN 'pending' THEN 0 ELSE 1 END,p.created_at DESC"
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Payment{}
	for rows.Next() {
		var p Payment
		var bookingIDBooking sql.NullString
		if err := rows.Scan(&p.ID, &p.BookingID, &p.Jumlah, &p.Metode, &p.Tanggal, &p.Status, &p.BuktiURL, &p.BankAccountID, &p.DestinationBankName, &p.DestinationAccountNumber, &p.DestinationAccountHolder, &p.SenderName, &p.SenderBank, &p.Notes, &p.Source, &p.RejectionReason, &p.VerifiedBy, &p.VerifiedAt, &p.CreatedAt, &p.JamaahName, &p.ScheduleName, &p.BrandName, &bookingIDBooking); err != nil {
			return nil, err
		}
		if bookingIDBooking.Valid {
			p.BookingIDBooking = bookingIDBooking.String
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Create menambahkan payment baru. Status selalu 'pending'.
func (r *Repository) Create(ctx context.Context, bookingID int64, req *CreatePaymentRequest) (*Payment, error) {
	var bankName, accountNumber, accountHolder *string
	if req.BankAccountID != nil {
		var n, no, h string
		err := r.db.QueryRowContext(ctx, `SELECT bank_name,account_number,account_holder FROM bank_accounts WHERE id=? AND is_active=TRUE`, *req.BankAccountID).Scan(&n, &no, &h)
		if err != nil {
			return nil, ErrNotFound
		}
		bankName = &n
		accountNumber = &no
		accountHolder = &h
	}
	const q = `INSERT INTO payments (booking_id,bank_account_id,destination_bank_name,destination_account_number,destination_account_holder,jumlah,metode,sender_name,sender_bank,tanggal,status,bukti_url,notes,source) VALUES (?,?,?,?,?,?,?,?,?,?,'pending',?,?,?)`

	source := req.Source
	if source == "" {
		source = "admin"
	}
	res, err := r.db.ExecContext(ctx, q, bookingID, req.BankAccountID, bankName, accountNumber, accountHolder, req.Jumlah, req.Metode, req.SenderName, req.SenderBank, req.Tanggal, req.BuktiURL, req.Notes, source)
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
func (r *Repository) UpdateStatus(ctx context.Context, id int64, newStatus string, rejectionReason *string, verifiedBy int64) (*Payment, error) {
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

	_, err = tx.ExecContext(ctx, `UPDATE payments SET status=?,rejection_reason=?,verified_by=?,verified_at=NOW() WHERE id=?`, newStatus, rejectionReason, verifiedBy, id)
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
	err := tx.QueryRowContext(ctx, `SELECT total_harga, status FROM bookings WHERE id=? FOR UPDATE`, bookingID).
		Scan(&totalHarga, &currentStatus)
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
		// Kursi sudah direservasi saat CreateBooking (booking/repository.go), bukan di sini.
		// Payment murni mencatat uang masuk & mengubah status transaksi, tidak lagi
		// punya efek samping ke kuota kursi.
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
