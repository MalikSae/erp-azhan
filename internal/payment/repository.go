package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/media"
	"erp-azhan/api/internal/shared"
)

// Sentinel errors
var (
	ErrNotFound        = errors.New("data tidak ditemukan")
	ErrCannotDelete    = errors.New("tidak bisa menghapus pembayaran yang sudah dikonfirmasi")
	ErrInvalidStatus   = errors.New("status tidak valid")
	ErrSeatUnavailable = errors.New("seat tidak mencukupi untuk mengonfirmasi pembayaran")
	ErrOverpayment     = errors.New("pembayaran melebihi sisa tagihan; periksa kemungkinan transfer ganda atau proses pengembalian dana")
	ErrBookingClosed   = errors.New("booking tidak aktif; hubungi admin untuk memastikan seat sebelum pembayaran")
	ErrNotVoidable     = errors.New("hanya pembayaran yang sudah dikonfirmasi yang dapat dibatalkan")
	ErrVoidBelowRefund = errors.New("pembayaran tidak dapat dibatalkan karena dana booking ini sudah dikembalikan melebihi sisa pembayaran lainnya")
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
func (r *Repository) ListDailyBrandTransactions(ctx context.Context, brandID *int64) ([]DailyBrandTransaction, error) {
	// Nilai bersih: pembayaran terkonfirmasi dikurangi pengembalian dana (booking_refunds)
	// pada tanggal refund dicatat. Count tetap jumlah pembayaran.
	brandFilter := ""
	var args []any
	if brandID != nil {
		brandFilter = " AND s.brand_id = ?"
		args = []any{*brandID, *brandID}
	}
	q := `
		SELECT DATE_FORMAT(t.tgl, '%Y-%m-%d') AS payment_date, t.brand_id,
			COALESCE(SUM(t.jumlah), 0), SUM(t.is_payment)
		FROM (
			SELECT COALESCE(p.tanggal, DATE(p.created_at)) AS tgl, s.brand_id, p.jumlah, 1 AS is_payment
			FROM payments p
			JOIN bookings b ON b.id = p.booking_id
			JOIN schedules s ON s.id = b.schedule_id
			WHERE p.status = 'confirmed'` + brandFilter + `
			UNION ALL
			SELECT rf.tanggal, s.brand_id, -rf.jumlah, 0
			FROM booking_refunds rf
			JOIN bookings b ON b.id = rf.booking_id
			JOIN schedules s ON s.id = b.schedule_id
			WHERE 1=1` + brandFilter + `
		) t
		WHERE t.tgl BETWEEN CURDATE() - INTERVAL 29 DAY AND CURDATE()
		GROUP BY payment_date, t.brand_id
		ORDER BY payment_date ASC, t.brand_id ASC`

	rows, err := r.db.QueryContext(ctx, q, args...)
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

// kolomPayment: kolom tabel payments beralias p, urutannya sama dengan tujuanScan.
const kolomPayment = `p.id,p.booking_id,p.jumlah,p.metode,DATE_FORMAT(p.tanggal,'%Y-%m-%d'),p.status,p.bukti_url,p.bank_account_id,
	p.destination_bank_name,p.destination_account_number,p.destination_account_holder,p.sender_name,p.sender_bank,p.received_by,p.notes,p.source,p.rejection_reason,p.verified_by,p.verified_at,p.void_reason,p.voided_by,p.voided_at,p.created_at`

// tujuanScan mengembalikan tujuan Scan untuk kolomPayment, diikuti kolom tambahan bila ada.
func tujuanScan(p *Payment, tambahan ...any) []any {
	dasar := []any{&p.ID, &p.BookingID, &p.Jumlah, &p.Metode, &p.Tanggal, &p.Status, &p.BuktiURL, &p.BankAccountID,
		&p.DestinationBankName, &p.DestinationAccountNumber, &p.DestinationAccountHolder, &p.SenderName, &p.SenderBank,
		&p.ReceivedBy, &p.Notes, &p.Source, &p.RejectionReason, &p.VerifiedBy, &p.VerifiedAt, &p.VoidReason, &p.VoidedBy,
		&p.VoidedAt, &p.CreatedAt}
	return append(dasar, tambahan...)
}

// totalTerkonfirmasiTx menjumlahkan pembayaran terkonfirmasi sebuah booking. kecualiID > 0 mengeluarkan
// satu pembayaran dari hitungan (pembayaran yang sedang dikonfirmasi atau dibatalkan).
func totalTerkonfirmasiTx(ctx context.Context, tx *sql.Tx, bookingID, kecualiID int64) (float64, error) {
	var total float64
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah),0) FROM payments WHERE booking_id=? AND status=? AND id<>?`,
		bookingID, StatusConfirmed, kecualiID).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("payment: total terkonfirmasi booking %d: %w", bookingID, err)
	}
	return total, nil
}

// ─── List ─────────────────────────────────────────────────────────────────────

// ListByBookingID mengambil list payment untuk sebuah booking.
// Memverifikasi brand via 2-hop JOIN: payments -> bookings -> schedules.
func (r *Repository) ListByBookingID(ctx context.Context, bookingID int64, brandID *int64) ([]Payment, error) {
	q := `
		SELECT ` + kolomPayment + `
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
		return nil, fmt.Errorf("payment.ListByBookingID: %w", err)
	}
	defer rows.Close()

	items := make([]Payment, 0)
	for rows.Next() {
		var p Payment
		if err := rows.Scan(tujuanScan(&p)...); err != nil {
			return nil, fmt.Errorf("payment.ListByBookingID scan: %w", err)
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
		SELECT ` + kolomPayment + `
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
	err := r.db.QueryRowContext(ctx, q, args...).Scan(tujuanScan(&p)...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("payment.GetByID: %w", err)
	}
	return &p, nil
}

func (r *Repository) ListAll(ctx context.Context, brandID *int64, status string) ([]Payment, error) {
	q := `SELECT ` + kolomPayment + `,j.nama_lengkap,s.jadwal_nama,br.name,COALESCE(b.id_booking, '')
		FROM payments p JOIN bookings b ON b.id=p.booking_id JOIN jamaah j ON j.id=b.pic_jamaah_id
		JOIN schedules s ON s.id=b.schedule_id JOIN brands br ON br.id=s.brand_id WHERE b.status != 'draft'`
	args := []any{}
	if brandID != nil {
		q += " AND s.brand_id=?"
		args = append(args, *brandID)
	}
	if status != "" {
		q += " AND p.status=?"
		args = append(args, status)
	}
	q += " ORDER BY CASE p.status WHEN 'pending' THEN 0 ELSE 1 END,p.created_at DESC LIMIT ?"
	args = append(args, shared.DefaultListCap)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("payment.ListAll: %w", err)
	}
	defer rows.Close()
	items := []Payment{}
	for rows.Next() {
		var p Payment
		if err := rows.Scan(tujuanScan(&p, &p.JamaahName, &p.ScheduleName, &p.BrandName, &p.KodeBooking)...); err != nil {
			return nil, fmt.Errorf("payment.ListAll scan: %w", err)
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
	if bookingTertutup(bookingStatus) {
		return nil, ErrBookingClosed
	}
	if req.Source == SourcePortal && (!blocked || (expires.Valid && expires.Time.Before(time.Now()))) {
		return nil, ErrBookingClosed
	}
	if totalPaid, err = totalTerkonfirmasiTx(ctx, tx, bookingID, 0); err != nil {
		return nil, err
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
		// Rekening harus aktif dan milik brand paket booking ini (cegah memakai rekening brand lain).
		err := tx.QueryRowContext(ctx, `
			SELECT ba.bank_name, ba.account_number, ba.account_holder
			FROM bank_accounts ba
			JOIN bookings b ON b.id = ?
			JOIN schedules s ON s.id = b.schedule_id AND s.brand_id = ba.brand_id
			WHERE ba.id = ? AND ba.is_active = TRUE`, bookingID, *req.BankAccountID).Scan(&n, &no, &h)
		// Hanya "tidak ada baris" yang berarti rekening tidak ditemukan; gangguan database harus tetap
		// terlihat sebagai kesalahan, bukan sebagai data yang tidak ada.
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		if err != nil {
			return nil, fmt.Errorf("payment.Create cek rekening: %w", err)
		}
		bankName = &n
		accountNumber = &no
		accountHolder = &h
	}

	source := req.Source
	if source == "" {
		source = SourceAdmin
	}

	// Pembayaran yang dicatat admin langsung terkonfirmasi; dari portal menunggu verifikasi.
	status := StatusPending
	var diverifikasiOleh *int64
	var verifiedAt any
	if source == SourceAdmin {
		status = StatusConfirmed
		diverifikasiOleh = verifiedBy
		verifiedAt = time.Now().UTC()
	}

	const q = `INSERT INTO payments (booking_id,bank_account_id,destination_bank_name,destination_account_number,destination_account_holder,jumlah,metode,sender_name,sender_bank,received_by,tanggal,status,bukti_url,notes,source,verified_by,verified_at) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`

	res, err := tx.ExecContext(ctx, q, bookingID, req.BankAccountID, bankName, accountNumber, accountHolder, req.Jumlah, req.Metode, req.SenderName, req.SenderBank, req.ReceivedBy, req.Tanggal, status, req.BuktiURL, req.Notes, source, diverifikasiOleh, verifiedAt)
	if err != nil {
		return nil, fmt.Errorf("payment.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("payment.Create LastInsertId: %w", err)
	}

	if status == StatusConfirmed {
		if err := r.syncBookingStatusTx(ctx, tx, bookingID); err != nil {
			return nil, fmt.Errorf("payment.Create sync booking: %w", err)
		}
	}
	if status == StatusPending && expires.Valid && expires.Time.After(time.Now()) {
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
func (r *Repository) GetBookingTotalAndPaid(ctx context.Context, bookingID int64) (statusBooking string, totalHargaBooking *float64, terbayar float64, err error) {
	q := `
		SELECT b.status, b.total_harga, COALESCE(SUM(p.jumlah), 0)
		FROM bookings b
		LEFT JOIN payments p ON p.booking_id = b.id AND p.status = 'confirmed'
		WHERE b.id = ?
		GROUP BY b.id, b.status, b.total_harga`
	var status string
	var totalHarga sql.NullFloat64
	var totalPaid float64
	err = r.db.QueryRowContext(ctx, q, bookingID).Scan(&status, &totalHarga, &totalPaid)
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
		return nil, fmt.Errorf("payment.UpdateStatus lock booking: %w", err)
	}
	// Booking batal/draft tidak boleh menerima konfirmasi, tetapi pembayaran pending yang
	// tersisa harus tetap bisa ditolak agar tidak menggantung di inbox.
	if bookingTertutup(bookingStatus) && newStatus != StatusRejected {
		return nil, ErrBookingClosed
	}
	var amount float64
	var oldStatus string
	if err = tx.QueryRowContext(ctx, `SELECT jumlah,status FROM payments WHERE id=? FOR UPDATE`, id).Scan(&amount, &oldStatus); err != nil {
		return nil, fmt.Errorf("payment.UpdateStatus lock payment: %w", err)
	}
	// A concurrent verifier may have finished after the HTTP handler read pending.
	if oldStatus != StatusPending {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("payment.UpdateStatus commit: %w", err)
		}
		return r.GetByID(ctx, id, nil)
	}
	if newStatus == StatusConfirmed {
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
	otherPaid, err := totalTerkonfirmasiTx(ctx, tx, bookingID, paymentID)
	if err != nil {
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
		return fmt.Errorf("payment: tandai perpanjangan tinjauan: %w", err)
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("payment: tandai perpanjangan tinjauan: %w", err)
	}
	if changed == 0 {
		return nil
	}
	if _, err = tx.ExecContext(ctx, `UPDATE bookings b JOIN booking_checkout c ON c.booking_id=b.id SET b.seat_hold_expires_at=DATE_ADD(c.original_expires_at,INTERVAL 24 HOUR) WHERE b.id=?`, bookingID); err != nil {
		return fmt.Errorf("payment: perpanjang hold kursi: %w", err)
	}
	return nil
}

func (r *Repository) syncBookingStatusTx(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	return SyncBookingStatusTx(ctx, tx, bookingID)
}

// SyncBookingStatusTx menghitung ulang status booking (baru/dp/lunas) dari pembayaran terkonfirmasi, lalu
// menyesuaikan kursi dan komisi. Dipakai juga oleh perhitungan ulang harga, supaya pembayaran dan perubahan
// booking memakai satu mesin status. Booking draft dan batal tidak disentuh.
func SyncBookingStatusTx(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	var totalHarga sql.NullFloat64
	var currentStatus string
	var scheduleID int64
	var isSeatBlocked bool
	var seatCount int
	err := tx.QueryRowContext(ctx, `SELECT total_harga,status,schedule_id,is_seat_blocked,seat_count FROM bookings WHERE id=? FOR UPDATE`, bookingID).
		Scan(&totalHarga, &currentStatus, &scheduleID, &isSeatBlocked, &seatCount)
	if err != nil {
		return fmt.Errorf("payment: sinkron status, kunci booking %d: %w", bookingID, err)
	}
	if bookingTertutup(currentStatus) {
		return nil
	}

	totalPaid, err := totalTerkonfirmasiTx(ctx, tx, bookingID, 0)
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
		return fmt.Errorf("payment: sinkron status, DP wajib: %w", err)
	}
	targetStatus := statusBookingDari(targetHarga, totalPaid, requiredDP)
	if targetStatus == currentStatus {
		return nil
	}

	// Kursi diambil setiap kali booking akan ditandai memegang kursi (dp/lunas) tetapi
	// belum terblokir. Tidak cukup untuk status "baru": booking dp/lunas yang kursinya
	// dilepas manual lalu naik status juga harus mengambil kursi lagi (SM-01).
	if memegangKursi(targetStatus) && !isSeatBlocked {
		if err := ambilKursiTx(ctx, tx, bookingID, scheduleID, seatCount); err != nil {
			return err
		}
	}
	// Hold 24 jam hanya untuk reservasi tanpa pembayaran terkonfirmasi. Booking
	// yang sudah dibayar lalu turun ke 'baru' (mis. koreksi diskon/add-on) tetap
	// memegang kursi tanpa batas waktu sampai admin memutuskan (JB-03).
	pegang := memegangKursi(targetStatus)
	if _, err = tx.ExecContext(ctx,
		`UPDATE bookings SET status=?,is_seat_blocked=IF(?,TRUE,is_seat_blocked),
		 seat_hold_expires_at=IF(? OR ? > 0,NULL,IF(is_seat_blocked,DATE_ADD(NOW(),INTERVAL 24 HOUR),NULL)),
		 seat_hold_key=IF(?,NULL,seat_hold_key) WHERE id=?`,
		targetStatus, pegang, pegang, totalPaid, pegang, bookingID); err != nil {
		return fmt.Errorf("payment: sinkron status, simpan status %s: %w", targetStatus, err)
	}
	return sinkronKomisiTx(ctx, tx, bookingID, currentStatus, targetStatus)
}

// ambilKursiTx mengurangi sisa kursi jadwal sebanyak kursi booking. Jumlah kursi diambil dari pax aktif
// bila lebih besar dari seat_count yang tersimpan.
func ambilKursiTx(ctx context.Context, tx *sql.Tx, bookingID, scheduleID int64, seatCount int) error {
	var activeRegularPax int
	if err := tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM booking_pax
		WHERE booking_id=? AND counts_for_seat=TRUE AND pax_status='aktif'`, bookingID,
	).Scan(&activeRegularPax); err != nil {
		return fmt.Errorf("payment: ambil kursi, hitung pax: %w", err)
	}
	if activeRegularPax > seatCount {
		seatCount = activeRegularPax
	}
	var seatRemaining int
	if err := tx.QueryRowContext(ctx, `SELECT seat_sisa FROM schedules WHERE id=? FOR UPDATE`, scheduleID).Scan(&seatRemaining); err != nil {
		return fmt.Errorf("payment: ambil kursi, kunci jadwal: %w", err)
	}
	if seatRemaining < seatCount {
		return ErrSeatUnavailable
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa=seat_sisa-? WHERE id=?`, seatCount, scheduleID); err != nil {
		return fmt.Errorf("payment: ambil kursi, kurangi sisa: %w", err)
	}
	return nil
}

// sinkronKomisiTx: komisi mengikuti lunas terakhir (K25). Keluar dari lunas membatalkan komisi booking ini,
// masuk lunas mencatatnya lagi dengan penerima saat itu.
func sinkronKomisiTx(ctx context.Context, tx *sql.Tx, bookingID int64, statusLama, statusBaru string) error {
	if statusLama == bookingLunas {
		if _, err := komisi.OnBookingTurunDariLunas(ctx, tx, bookingID); err != nil {
			return err
		}
	}
	if statusBaru == bookingLunas {
		if err := komisi.OnBookingLunas(ctx, tx, bookingID); err != nil {
			return err
		}
	}
	return nil
}

// ─── Void ─────────────────────────────────────────────────────────────────────

// Void membatalkan pembayaran yang sudah dikonfirmasi (salah nominal, salah booking, atau salah
// konfirmasi). Baris tidak dihapus: status menjadi 'void' dengan alasan, pelaku, dan waktu, lalu status
// booking dihitung ulang. Bila booking turun dari lunas, komisinya batal dan dihitung ulang saat lunas lagi
// (K25). Permintaan ulang atas pembayaran yang sudah void tidak mengubah apa pun.
func (r *Repository) Void(ctx context.Context, id int64, alasan string, adminID int64) (*Payment, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("payment.Void tx: %w", err)
	}
	defer tx.Rollback()

	if err := voidTx(ctx, tx, id, alasan, adminID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("payment.Void commit: %w", err)
	}
	return r.GetByID(ctx, id, nil)
}

func voidTx(ctx context.Context, tx *sql.Tx, id int64, alasan string, adminID int64) error {
	var bookingID int64
	err := tx.QueryRowContext(ctx, `SELECT booking_id FROM payments WHERE id=?`, id).Scan(&bookingID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("payment.Void get booking_id: %w", err)
	}
	// Booking dikunci sebelum pembayaran, urutan yang sama dengan penulis pembayaran lain.
	var bookingStatus string
	var totalHarga float64
	if err = tx.QueryRowContext(ctx, `SELECT status, COALESCE(total_harga,0) FROM bookings WHERE id=? FOR UPDATE`, bookingID).Scan(&bookingStatus, &totalHarga); err != nil {
		return fmt.Errorf("payment.Void lock booking: %w", err)
	}
	var status string
	if err = tx.QueryRowContext(ctx, `SELECT status FROM payments WHERE id=? FOR UPDATE`, id).Scan(&status); err != nil {
		return fmt.Errorf("payment.Void lock payment: %w", err)
	}
	if status == StatusVoid {
		return nil
	}
	if status != StatusConfirmed {
		return ErrNotVoidable
	}
	// Dana yang sudah dikembalikan tidak boleh melebihi pembayaran terkonfirmasi yang tersisa.
	sisaTerkonfirmasi, err := totalTerkonfirmasiTx(ctx, tx, bookingID, id)
	if err != nil {
		return err
	}
	var refund float64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah),0) FROM booking_refunds WHERE booking_id=?`, bookingID).Scan(&refund); err != nil {
		return fmt.Errorf("payment.Void cek refund: %w", err)
	}
	if refund > sisaTerkonfirmasi {
		return ErrVoidBelowRefund
	}

	var pelaku any
	if adminID > 0 {
		pelaku = adminID
	}
	if _, err = tx.ExecContext(ctx, `UPDATE payments SET status=?, void_reason=?, voided_by=?, voided_at=NOW() WHERE id=?`, StatusVoid, alasan, pelaku, id); err != nil {
		return fmt.Errorf("payment.Void update: %w", err)
	}
	if err = SyncBookingStatusTx(komisi.DenganSebabBatal(ctx, komisi.SebabVoid), tx, bookingID); err != nil {
		return fmt.Errorf("payment.Void sync booking: %w", err)
	}
	// Booking yang sudah batal tidak disinkronkan lagi, jadi komisinya tidak ikut batal lewat jalur di atas.
	// K20 mempertahankan komisi booking batal karena uang jamaah sudah masuk; void menyatakan pembayaran
	// itu tidak pernah ada, sehingga dasar komisinya hilang. Refund tidak lewat sini dan tidak mengubah komisi.
	// Booking yang dibatalkan setelah dp atau lunas mempertahankan total harganya, jadi pembandingnya sama
	// dengan booking hidup: komisi batal bila sisa pembayaran terkonfirmasi tidak lagi menutup total harga.
	if bookingStatus == bookingBatal && sisaTerkonfirmasi < totalHarga {
		if _, err = komisi.BatalkanKomisiBookingTx(komisi.DenganSebabBatal(ctx, komisi.SebabVoid), tx, bookingID); err != nil {
			return fmt.Errorf("payment.Void batalkan komisi booking batal: %w", err)
		}
	}
	// Booking yang turun status karena pembatalan pembayaran tetap memegang kursinya tanpa batas waktu
	// sampai admin memutuskan; jangan sampai kursinya dilepas otomatis akibat koreksi admin.
	if _, err = tx.ExecContext(ctx, `UPDATE bookings SET seat_hold_expires_at=NULL WHERE id=? AND is_seat_blocked=TRUE AND status<>'batal'`, bookingID); err != nil {
		return fmt.Errorf("payment.Void keep seat: %w", err)
	}
	return nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Delete menghapus payment jika status masih 'pending'. Penghapusan bersyarat dalam satu
// statement agar tidak berlomba dengan konfirmasi yang berjalan bersamaan.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM payments WHERE id=? AND status='pending'`, id)
	if err != nil {
		return fmt.Errorf("payment.Delete: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("payment.Delete jumlah baris: %w", err)
	}
	if n > 0 {
		return nil
	}
	var status string
	err = r.db.QueryRowContext(ctx, `SELECT status FROM payments WHERE id=?`, id).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("payment.Delete cek status: %w", err)
	}
	return ErrCannotDelete
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

// ValidasiBuktiAdmin memastikan bukti pembayaran adalah unggahan admin resmi di brand ini, bukan URL bebas.
func (r *Repository) ValidasiBuktiAdmin(ctx context.Context, url string, brandID *int64) error {
	return media.ValidateAdminPaymentProof(ctx, r.db, url, brandID)
}
