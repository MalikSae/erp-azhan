package crmdeal

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"erp-azhan/api/internal/shared"
	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound             = errors.New("jadwal atau jamaah tidak ditemukan pada brand aktif")
	ErrSeatUnavailable      = errors.New("kursi pada jadwal tidak mencukupi")
	ErrAmbiguousJamaah      = errors.New("lebih dari satu jamaah memiliki nomor telepon tersebut")
	ErrIdempotencyConflict  = errors.New("idempotency key pernah digunakan untuk payload berbeda")
	ErrLeadAlreadyConverted = errors.New("lead ini sudah pernah dikonversi")
	ErrBrandCodeMissing     = errors.New("kode_brand belum diatur")
	ErrInvalidPayment       = errors.New("nominal pembayaran tidak sesuai")
	ErrMaxPaxExceeded       = errors.New("Deal CRM saat ini hanya mendukung 1 pax per transaksi. Untuk booking lebih dari 1 orang, gunakan menu Booking di dashboard admin.")
)

const codeCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

var nonDigit = regexp.MustCompile(`[^0-9]+`)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Process(ctx context.Context, brandID, createdBy int64, idempotencyKey string, req DealRequest) (*DealResponse, error) {
	if req.Pax == 0 {
		req.Pax = 1
	}
	if req.Pax > 1 {
		return nil, ErrMaxPaxExceeded
	}

	req.BrandID = &brandID
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}
	hashBytes := sha256.Sum256(payload)
	requestHash := hex.EncodeToString(hashBytes[:])

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin deal transaction: %w", err)
	}
	defer tx.Rollback()

	var existingHash string
	var existingPayload []byte
	err = tx.QueryRowContext(ctx,
		`SELECT request_hash, response_payload FROM crm_deal_requests WHERE idempotency_key=? FOR UPDATE`,
		idempotencyKey,
	).Scan(&existingHash, &existingPayload)
	if err == nil {
		if existingHash != requestHash {
			return nil, ErrIdempotencyConflict
		}
		var response DealResponse
		if err := json.Unmarshal(existingPayload, &response); err != nil {
			return nil, fmt.Errorf("decode stored deal response: %w", err)
		}
		return &response, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup idempotency key: %w", err)
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO crm_deal_requests (brand_id,crm_lead_id,idempotency_key,request_hash,created_by) VALUES (?,?,?,?,?)`,
		brandID, req.CRMLeadID, idempotencyKey, requestHash, createdBy,
	)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrLeadAlreadyConverted
		}
		return nil, fmt.Errorf("reserve deal idempotency: %w", err)
	}

	var scheduleBrandID int64
	var seatRemaining int
	var quad, triple, double float64
	err = tx.QueryRowContext(ctx,
		`SELECT brand_id,seat_sisa,harga_quad,harga_triple,harga_double FROM schedules WHERE id=? FOR UPDATE`,
		req.ScheduleID,
	).Scan(&scheduleBrandID, &seatRemaining, &quad, &triple, &double)
	if errors.Is(err, sql.ErrNoRows) || scheduleBrandID != brandID {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock schedule: %w", err)
	}

	unitPrice := map[string]float64{"Quad": quad, "Triple": triple, "Double": double}[req.RoomType]
	totalPrice := unitPrice * float64(req.Pax)

	// Convert crmdeal.JamaahInput to shared.JamaahInput
	sharedJamaahInput := shared.JamaahInput{
		ID:          req.Jamaah.ID,
		NamaLengkap: req.Jamaah.NamaLengkap,
		NoHP:        req.Jamaah.NoHP,
		Email:       req.Jamaah.Email,
		Alamat:      req.Jamaah.Alamat,
	}
	jamaahID, err := shared.ResolveJamaah(ctx, tx, brandID, sharedJamaahInput)
	if err != nil {
		if errors.Is(err, shared.ErrAmbiguousJamaah) {
			return nil, ErrAmbiguousJamaah
		}
		if errors.Is(err, shared.ErrNotFound) {
			return nil, ErrNotFound
		}
		if errors.Is(err, shared.ErrBrandCodeMissing) {
			return nil, ErrBrandCodeMissing
		}
		return nil, err
	}

	brandCode, err := r.brandCode(ctx, tx, brandID)
	if err != nil {
		return nil, err
	}
	bookingCode, err := shared.UniqueCode(ctx, tx, "bookings", "id_booking", brandCode, 4)
	if err != nil {
		return nil, err
	}

	bookingStatus := "baru"
	isSeatBlocked := false
	var holdExpiry any
	var holdKey any
	dealSubstatus := "dp_pending"

	switch req.CommitmentType {
	case "book_seat":
		if seatRemaining < req.Pax {
			return nil, ErrSeatUnavailable
		}
		if req.SeatHoldExpiresAt == nil {
			return nil, ErrInvalidPayment
		}
		expiresAt, err := time.Parse(time.RFC3339, *req.SeatHoldExpiresAt)
		if err != nil || !expiresAt.After(time.Now()) {
			return nil, ErrInvalidPayment
		}
		isSeatBlocked = true
		holdExpiry = expiresAt.UTC()
		holdKey = idempotencyKey
		dealSubstatus = "book_seat"
	case "lunas":
		if seatRemaining < req.Pax {
			return nil, ErrSeatUnavailable
		}
		if req.PaymentAmount == nil || math.Abs(*req.PaymentAmount-totalPrice) >= 0.5 {
			return nil, ErrInvalidPayment
		}
		dealSubstatus = "lunas_pending"
	case "dp":
		if req.PaymentAmount == nil || *req.PaymentAmount <= 0 || *req.PaymentAmount > totalPrice {
			return nil, ErrInvalidPayment
		}
	default:
		return nil, ErrInvalidPayment
	}

	result, err := tx.ExecContext(ctx,
		`INSERT INTO bookings
		 (id_booking,schedule_id,pic_jamaah_id,seat_count,status,is_seat_blocked,seat_hold_expires_at,seat_hold_key,total_harga,created_by)
		 VALUES (?,?,?,?,?,?,?,?,?,?)`,
		bookingCode, req.ScheduleID, jamaahID, req.Pax, bookingStatus,
		isSeatBlocked, holdExpiry, holdKey, totalPrice, createdBy,
	)
	if err != nil {
		return nil, fmt.Errorf("create booking: %w", err)
	}
	bookingID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("booking id: %w", err)
	}

	// CRM saat ini menerima jumlah pax dan satu PIC. Simpan satu detail per kursi
	// agar invariant booking_pax tetap sama dengan seat_count setelah migrasi multi-pax.
	for range req.Pax {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO booking_pax
			 (booking_id,jamaah_id,pax_type,room_type,harga_pax,counts_for_seat,pax_status)
			 VALUES (?,?,'reguler',?,?,TRUE,'aktif')`,
			bookingID, jamaahID, req.RoomType, unitPrice,
		); err != nil {
			return nil, fmt.Errorf("create booking pax: %w", err)
		}
	}

	if isSeatBlocked {
		if _, err := tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa=seat_sisa-? WHERE id=?`, req.Pax, req.ScheduleID); err != nil {
			return nil, fmt.Errorf("reserve seats: %w", err)
		}
	}

	var paymentID *int64
	if req.CommitmentType == "dp" || req.CommitmentType == "lunas" {
		paymentResult, err := tx.ExecContext(ctx,
			`INSERT INTO payments
			 (booking_id,jumlah,metode,tanggal,status,bukti_url,source,verified_by,verified_at)
			 VALUES (?,?,?,?,'pending',?,'crm',NULL,NULL)`,
			bookingID, *req.PaymentAmount, req.PaymentMethod, req.PaymentDate,
			req.PaymentProofURL,
		)
		if err != nil {
			return nil, fmt.Errorf("create payment: %w", err)
		}
		id, err := paymentResult.LastInsertId()
		if err != nil {
			return nil, fmt.Errorf("payment id: %w", err)
		}
		paymentID = &id
	}

	response := DealResponse{
		Status:            "completed",
		CommitmentType:    req.CommitmentType,
		DealSubstatus:     dealSubstatus,
		JamaahID:          jamaahID,
		BookingID:         bookingID,
		BookingCode:       bookingCode,
		BookingStatus:     bookingStatus,
		PaymentID:         paymentID,
		SeatHoldExpiresAt: req.SeatHoldExpiresAt,
	}
	responsePayload, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("encode response: %w", err)
	}
	_, err = tx.ExecContext(ctx,
		`UPDATE crm_deal_requests SET status='completed',jamaah_id=?,booking_id=?,payment_id=?,response_payload=? WHERE idempotency_key=?`,
		jamaahID, bookingID, paymentID, responsePayload, idempotencyKey,
	)
	if err != nil {
		return nil, fmt.Errorf("complete deal record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit deal: %w", err)
	}
	return &response, nil
}

func (r *Repository) brandCode(ctx context.Context, tx *sql.Tx, brandID int64) (string, error) {
	var code sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT kode_brand FROM brands WHERE id=? FOR UPDATE`, brandID).Scan(&code); err != nil {
		return "", fmt.Errorf("load brand code: %w", err)
	}
	if !code.Valid || strings.TrimSpace(code.String) == "" {
		return "", ErrBrandCodeMissing
	}
	return strings.ToUpper(strings.TrimSpace(code.String)), nil
}

// ReleaseExpiredSeatHolds melepas hold booking baru yang kedaluwarsa secara atomic per-booking.
func (r *Repository) ReleaseExpiredSeatHolds(ctx context.Context) (int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id FROM bookings
		WHERE status='baru' AND is_seat_blocked=TRUE
		  AND seat_hold_expires_at IS NOT NULL AND seat_hold_expires_at <= UTC_TIMESTAMP()
		ORDER BY id`)
	if err != nil {
		return 0, fmt.Errorf("find expired seat hold candidates: %w", err)
	}
	defer rows.Close()

	var candidateIDs []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		candidateIDs = append(candidateIDs, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(candidateIDs) == 0 {
		return 0, nil
	}

	var releasedCount int64
	for _, bookingID := range candidateIDs {
		released, err := r.releaseSingleSeatHold(ctx, bookingID)
		if err != nil {
			continue
		}
		if released {
			releasedCount++
		}
	}

	return releasedCount, nil
}

func (r *Repository) releaseSingleSeatHold(ctx context.Context, bookingID int64) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin tx for booking %d: %w", bookingID, err)
	}
	defer tx.Rollback()

	var isSeatBlocked bool
	var seatHoldExpiresAt sql.NullTime
	var scheduleID int64
	var seatCount int
	var status string

	err = tx.QueryRowContext(ctx, `
		SELECT is_seat_blocked, seat_hold_expires_at, schedule_id, seat_count, status
		FROM bookings
		WHERE id=?
		FOR UPDATE`, bookingID).Scan(&isSeatBlocked, &seatHoldExpiresAt, &scheduleID, &seatCount, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("lock booking %d: %w", bookingID, err)
	}

	now := time.Now().UTC()
	if !isSeatBlocked || !seatHoldExpiresAt.Valid || status != "baru" || seatHoldExpiresAt.Time.After(now) {
		return false, nil
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE schedules SET seat_sisa=LEAST(seat_total, seat_sisa+?) WHERE id=?`,
		seatCount, scheduleID,
	); err != nil {
		return false, fmt.Errorf("restore seat for schedule %d: %w", scheduleID, err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE bookings SET is_seat_blocked=FALSE, seat_hold_expires_at=NULL, seat_hold_key=NULL WHERE id=?`,
		bookingID,
	); err != nil {
		return false, fmt.Errorf("clear hold on booking %d: %w", bookingID, err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit release hold on booking %d: %w", bookingID, err)
	}

	return true, nil
}
