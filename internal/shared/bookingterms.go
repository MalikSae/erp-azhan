package shared

import (
	"context"
	"database/sql"
	"math"
)

type rowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// RequiredBookingDP uses checkout snapshots, and current terms only for legacy
// administrative bookings which do not yet have a checkout record.
func RequiredBookingDP(ctx context.Context, q rowQuerier, bookingID int64, total float64) (float64, error) {
	var dp float64
	var full bool
	var count int
	// Urutan: snapshot checkout self-booking, snapshot DP booking (MP-02), lalu nilai
	// paket/brand saat ini hanya untuk booking lama tanpa snapshot.
	err := q.QueryRowContext(ctx, `SELECT COALESCE(c.dp_per_pax,b.dp_per_pax,s.minimal_dp,br.minimal_dp,0),COALESCE(c.full_payment,FALSE),
 (SELECT COUNT(*) FROM booking_pax p WHERE p.booking_id=b.id AND p.counts_for_seat=TRUE AND p.pax_status='aktif')
 FROM bookings b JOIN schedules s ON s.id=b.schedule_id JOIN brands br ON br.id=s.brand_id LEFT JOIN booking_checkout c ON c.booking_id=b.id WHERE b.id=?`, bookingID).Scan(&dp, &full, &count)
	if err != nil {
		return 0, err
	}
	if full {
		return total, nil
	}
	return math.Max(0, math.Min(total, dp*float64(count))), nil
}
