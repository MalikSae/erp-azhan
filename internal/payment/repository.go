package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/shared"
)

// Sentinel errors
var (
	ErrNotFound        = errors.New("data tidak ditemukan")
	ErrCannotDelete    = errors.New("tidak bisa menghapus pembayaran yang sudah dikonfirmasi")
	ErrInvalidStatus   = errors.New("status tidak valid")
	ErrSeatUnavailable = errors.New("kursi tidak mencukupi untuk mengonfirmasi pembayaran")
	ErrOverpayment     = errors.New("pembayaran melebihi sisa tagihan; periksa kemungkinan transfer ganda atau proses pengembalian dana")
	ErrBookingClosed   = errors.New("reservasi tidak aktif; hubungi admin untuk memastikan kursi sebelum pembayaran")
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
	// Nilai bersih: pembayaran terkonfirmasi dikurangi pengembalian dana (booking_refunds)
	// pada tanggal refund dicatat. Count tetap jumlah pembayaran.
	const q = `
		SELECT DATE_FORMAT(t.tgl, '%Y-%m-%d') AS payment_date, t.brand_id,
			COALESCE(SUM(t.jumlah), 0), SUM(t.is_payment)
		FROM (
			SELECT COALESCE(p.tanggal, DATE(p.created_at)) AS tgl, s.brand_id, p.jumlah, 1 AS is_payment
			FROM payments p
			JOIN bookings b ON b.id = p.booking_id
			JOIN schedules s ON s.id = b.schedule_id
			WHERE p.status = 'confirmed'
			UNION ALL
			SELECT rf.tanggal, s.brand_id, -rf.jumlah, 0
			FROM booking_refunds rf
			JOIN bookings b ON b.id = rf.booking_id
			JOIN schedules s ON s.id = b.schedule_id
		) t
		WHERE t.tgl BETWEEN CURDATE() - INTERVAL 29 DAY AND CURDATE()
		GROUP BY payment_date, t.brand_id
		ORDER BY payment_date ASC, t.brand_id ASC`

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
	q := `SELECT p.id,p.booking_id,p.jumlah,p.metode,DATE_FORMAT(p.tanggal,'%Y-%m-%d'),p.status,p.bukti_url,p.bank_account_id,p.destination_bank_name,p.destination_account_number,p.destination_account_holder,p.sender_name,p.sender_bank,p.notes,p.source,p.rejection_reason,p.verified_by,p.verified_at,p.created_at,j.nama_lengkap,s.jadwal_nama,br.name,COALESCE(b.id_booking, '') FROM payments p JOIN bookings b ON b.id=p.booking_id JOIN jamaah j ON j.id=b.pic_jamaah_id JOIN schedules s ON s.id=b.schedule_id JOIN brands br ON br.id=s.brand_id WHERE b.status != 'draft'`
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

// Create menambahkan payment baru. Jika source == 'admin', status otomatis 'confirmed', verified_by & verified_at terisi, dan booking status disinkronkan.
func (r *Repository) Create(ctx context.Context, bookingID int64, req *CreatePaymentRequest, verifiedBy *int64) (*Payment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("payment.Create tx: %w", err)
	}
	defer tx.Rollback()

	// Validasi nilai pembayaran tidak boleh lebih besar dari sisa tagihan
	var totalHarga sql.NullFloat64
	var totalPaid float64
	var bookingStatus string
	var blocked bool
	var expires sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT total_harga,status,is_seat_blocked,seat_hold_expires_at FROM bookings WHERE id=? FOR UPDATE`, bookingID).Scan(&totalHarga, &bookingStatus, &blocked, &expires)
	if err != nil {
		return nil, fmt.Errorf("payment.Create check booking: %w", err)
	}
	if bookingStatus == "batal" || bookingStatus == "draft" {
		return nil, ErrBookingClosed
	}
	if req.Source == "portal" && (!blocked || (expires.Valid && expires.Time.Before(time.Now()))) {
		return nil, ErrBookingClosed
	}
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah), 0) FROM payments WHERE booking_id=? AND status='confirmed'`, bookingID).Scan(&totalPaid)
	if err != nil {
		return nil, fmt.Errorf("payment.Create check paid: %w", err)
	}
	if totalHarga.Valid && totalHarga.Float64 > 0 {
		sisaTagihan := totalHarga.Float64 - totalPaid
		if sisaTagihan < 0 {
			sisaTagihan = 0
		}
		if req.Jumlah > sisaTagihan {
			return nil, ErrOverpayment
		}
	}

	var bankName, accountNumber, accountHolder *string
	if req.BankAccountID != nil {
		var n, no, h string
		err := tx.QueryRowContext(ctx, `SELECT bank_name,account_number,account_holder FROM bank_accounts WHERE id=? AND is_active=TRUE`, *req.BankAccountID).Scan(&n, &no, &h)
		if err != nil {
			return nil, ErrNotFound
		}
		bankName = &n
		accountNumber = &no
		accountHolder = &h
	}

	source := req.Source
	if source == "" {
		source = "admin"
	}

	status := "pending"
	var vBy *int64
	var verifiedAt any
	if source == "admin" {
		status = "confirmed"
		vBy = verifiedBy
		verifiedAt = time.Now().UTC()
	}

	const q = `INSERT INTO payments (booking_id,bank_account_id,destination_bank_name,destination_account_number,destination_account_holder,jumlah,metode,sender_name,sender_bank,tanggal,status,bukti_url,notes,source,verified_by,verified_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

	res, err := tx.ExecContext(ctx, q, bookingID, req.BankAccountID, bankName, accountNumber, accountHolder, req.Jumlah, req.Metode, req.SenderName, req.SenderBank, req.Tanggal, status, req.BuktiURL, req.Notes, source, vBy, verifiedAt)
	if err != nil {
		return nil, fmt.Errorf("payment.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("payment.Create LastInsertId: %w", err)
	}

	if status == "confirmed" {
		if err := r.syncBookingStatusTx(ctx, tx, bookingID); err != nil {
			return nil, fmt.Errorf("payment.Create sync booking: %w", err)
		}
	}
	if status == "pending" && expires.Valid && expires.Time.After(time.Now()) {
		if err := extendPendingReviewTx(ctx, tx, bookingID); err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("payment.Create commit: %w", err)
	}

	return r.GetByID(ctx, id, nil)
}

// GetBookingTotalAndPaid mengambil status, total_harga booking dan total pembayaran yang sudah dikonfirmasi.
func (r *Repository) GetBookingTotalAndPaid(ctx context.Context, bookingID int64) (string, *float64, float64, error) {
	q := `
		SELECT b.status, b.total_harga, COALESCE(SUM(p.jumlah), 0)
		FROM bookings b
		LEFT JOIN payments p ON p.booking_id = b.id AND p.status = 'confirmed'
		WHERE b.id = ?
		GROUP BY b.id, b.status, b.total_harga`
	var status string
	var totalHarga sql.NullFloat64
	var totalPaid float64
	err := r.db.QueryRowContext(ctx, q, bookingID).Scan(&status, &totalHarga, &totalPaid)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, 0, ErrNotFound
	}
	if err != nil {
		return "", nil, 0, fmt.Errorf("payment.GetBookingTotalAndPaid: %w", err)
	}
	if totalHarga.Valid {
		return status, &totalHarga.Float64, totalPaid, nil
	}
	return status, nil, totalPaid, nil
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
	// Lock the booking before the payment: every payment writer follows this order.
	var bookingStatus string
	var total float64
	if err = tx.QueryRowContext(ctx, `SELECT status,COALESCE(total_harga,0) FROM bookings WHERE id=? FOR UPDATE`, bookingID).Scan(&bookingStatus, &total); err != nil {
		return nil, err
	}
	if bookingStatus == "batal" || bookingStatus == "draft" {
		return nil, ErrBookingClosed
	}
	var amount float64
	var oldStatus string
	if err = tx.QueryRowContext(ctx, `SELECT jumlah,status FROM payments WHERE id=? FOR UPDATE`, id).Scan(&amount, &oldStatus); err != nil {
		return nil, err
	}
	// A concurrent verifier may have finished after the HTTP handler read pending.
	if oldStatus != "pending" {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return r.GetByID(ctx, id, nil)
	}
	if newStatus == "confirmed" {
		if err := checkConfirmationBalanceTx(ctx, tx, bookingID, id, amount, total); err != nil {
			return nil, err
		}
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

// Caller holds the booking lock, so concurrent confirmations cannot share a balance.
func checkConfirmationBalanceTx(ctx context.Context, tx *sql.Tx, bookingID, paymentID int64, amount, total float64) error {
	var otherPaid float64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah),0) FROM payments WHERE booking_id=? AND status='confirmed' AND id<>?`, bookingID, paymentID).Scan(&otherPaid); err != nil {
		return err
	}
	if amount+otherPaid > total {
		return ErrOverpayment
	}
	return nil
}

func extendPendingReviewTx(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	result, err := tx.ExecContext(ctx, `UPDATE booking_checkout SET review_extended=TRUE WHERE booking_id=? AND review_extended=FALSE AND original_expires_at>=NOW()`, bookingID)
	if err != nil {
		return err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if changed > 0 {
		_, err = tx.ExecContext(ctx, `UPDATE bookings b JOIN booking_checkout c ON c.booking_id=b.id SET b.seat_hold_expires_at=DATE_ADD(c.original_expires_at,INTERVAL 24 HOUR) WHERE b.id=?`, bookingID)
	}
	return err
}

func (r *Repository) syncBookingStatusTx(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	return SyncBookingStatusTx(ctx, tx, bookingID)
}

// Shared with price recalculation so payment and booking edits use one state machine.
func SyncBookingStatusTx(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	var totalHarga sql.NullFloat64
	var currentStatus string
	var scheduleID int64
	var isSeatBlocked bool
	var seatCount int
	err := tx.QueryRowContext(ctx, `SELECT total_harga,status,schedule_id,is_seat_blocked,seat_count FROM bookings WHERE id=? FOR UPDATE`, bookingID).
		Scan(&totalHarga, &currentStatus, &scheduleID, &isSeatBlocked, &seatCount)
	if err != nil {
		return err
	}

	// Jangan ubah jika booking sudah dibatalkan
	if currentStatus == "batal" || currentStatus == "draft" {
		return nil
	}

	var totalPaid float64
	err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah), 0) FROM payments WHERE booking_id=? AND status='confirmed'`, bookingID).
		Scan(&totalPaid)
	if err != nil {
		return err
	}

	// Snapshot terms remain stable when the package or brand changes.
	targetHarga := 0.0
	if totalHarga.Valid {
		targetHarga = totalHarga.Float64
	}
	requiredDP, err := shared.RequiredBookingDP(ctx, tx, bookingID, targetHarga)
	if err != nil {
		return err
	}
	targetStatus := "baru"
	if targetHarga > 0 && totalPaid >= targetHarga {
		targetStatus = "lunas"
	} else if totalPaid > 0 && totalPaid >= requiredDP {
		targetStatus = "dp"
	}
	if targetStatus != currentStatus {
		if currentStatus == "baru" && (targetStatus == "dp" || targetStatus == "lunas") && !isSeatBlocked {
			var activeRegularPax int
			if err = tx.QueryRowContext(ctx, `
				SELECT COUNT(*) FROM booking_pax
				WHERE booking_id=? AND counts_for_seat=TRUE AND pax_status='aktif'`, bookingID,
			).Scan(&activeRegularPax); err != nil {
				return err
			}
			if activeRegularPax > seatCount {
				seatCount = activeRegularPax
			}
			var seatRemaining int
			if err = tx.QueryRowContext(ctx, `SELECT seat_sisa FROM schedules WHERE id=? FOR UPDATE`, scheduleID).Scan(&seatRemaining); err != nil {
				return err
			}
			if seatRemaining < seatCount {
				return ErrSeatUnavailable
			}
			_, err = tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa=seat_sisa-? WHERE id=?`, seatCount, scheduleID)
			if err != nil {
				return err
			}
		}
		// Hold 24 jam hanya untuk reservasi tanpa pembayaran terkonfirmasi. Booking
		// yang sudah dibayar lalu turun ke 'baru' (mis. koreksi diskon/add-on) tetap
		// memegang kursi tanpa batas waktu sampai admin memutuskan (JB-03).
		_, err = tx.ExecContext(ctx,
			`UPDATE bookings SET status=?,is_seat_blocked=IF(? IN ('dp','lunas'),TRUE,is_seat_blocked),
			 seat_hold_expires_at=IF(? IN ('dp','lunas') OR ? > 0,NULL,IF(is_seat_blocked,DATE_ADD(NOW(),INTERVAL 24 HOUR),NULL)),
			 seat_hold_key=IF(? IN ('dp','lunas'),NULL,seat_hold_key) WHERE id=?`,
			targetStatus, targetStatus, targetStatus, totalPaid, targetStatus, bookingID)
		if err != nil {
			return err
		}
		if targetStatus == "lunas" {
			if err := komisi.OnBookingLunas(ctx, tx, bookingID); err != nil {
				return err
			}
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
