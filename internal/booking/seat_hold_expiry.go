package booking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"erp-azhan/api/internal/shared"
)

// ReleaseExpiredSeatHolds melepas hold seat booking baru yang kedaluwarsa, atomic per booking.
func (r *Repository) ReleaseExpiredSeatHolds(ctx context.Context) (int64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id FROM bookings
		WHERE status='baru' AND is_seat_blocked=TRUE
		  AND seat_hold_expires_at IS NOT NULL AND seat_hold_expires_at <= NOW()
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

	var releasedCount int64
	for _, bookingID := range candidateIDs {
		released, err := r.releaseSingleSeatHold(ctx, bookingID)
		if err != nil {
			log.Printf("[ERROR] seat_hold_expiry booking=%d: %v", bookingID, err)
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
	released, err := releaseSingleSeatHoldTx(ctx, tx, bookingID)
	if err != nil {
		return false, err
	}
	// Commit juga saat tidak dilepas: batas waktu booking yang sudah dibayar ikut dihapus.
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit release hold on booking %d: %w", bookingID, err)
	}
	return released, nil
}

// releaseSingleSeatHoldTx melepas hold kedaluwarsa satu booking di dalam transaksi pemanggil.
func releaseSingleSeatHoldTx(ctx context.Context, tx *sql.Tx, bookingID int64) (bool, error) {
	var isSeatBlocked bool
	var seatHoldExpiresAt sql.NullTime
	var scheduleID int64
	var seatCount int
	var status string

	err := tx.QueryRowContext(ctx, `
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
	// Uang jamaah sudah masuk: seat tidak dilepas, batas waktunya dihapus.
	dibayar, err := adaPembayaranTerkonfirmasiTx(ctx, tx, bookingID)
	if err != nil {
		return false, err
	}
	if dibayar {
		if _, err := tx.ExecContext(ctx, `UPDATE bookings SET seat_hold_expires_at=NULL WHERE id=?`, bookingID); err != nil {
			return false, fmt.Errorf("clear expiry on paid booking %d: %w", bookingID, err)
		}
		return false, nil
	}

	if err := shared.RestoreSeats(ctx, tx, scheduleID, seatCount); err != nil {
		return false, fmt.Errorf("restore seat for schedule %d: %w", scheduleID, err)
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE bookings SET is_seat_blocked=FALSE, seat_hold_expires_at=NULL, seat_hold_key=NULL WHERE id=?`,
		bookingID,
	); err != nil {
		return false, fmt.Errorf("clear hold on booking %d: %w", bookingID, err)
	}
	return true, nil
}
