package booking

import (
	"context"
	"database/sql"
	"errors"
	"net/http"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/komisi"
)

// AgenKomisi membaca ringkasan agen dan komisi booking untuk Detail Booking (keputusan 2026-11-24).
// Brand dibatasi seperti GetByID; nominal hanya untuk pemegang izin komisi.view.
func (r *Repository) AgenKomisi(ctx context.Context, id int64, brandID *int64, lihatNominal bool) (*komisi.BookingAgenKomisi, error) {
	if brandID != nil {
		var ada bool
		if err := r.db.QueryRowContext(ctx, `
			SELECT EXISTS (SELECT 1 FROM bookings b JOIN schedules s ON s.id = b.schedule_id WHERE b.id = ? AND s.brand_id = ?)`,
			id, *brandID).Scan(&ada); err != nil {
			return nil, err
		}
		if !ada {
			return nil, ErrNotFound
		}
	}
	out, err := komisi.RingkasanBooking(ctx, r.db, id, lihatNominal)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return out, err
}

// GetAgenKomisi GET /api/admin/bookings/{id}/agen-komisi
func (h *Handler) GetAgenKomisi(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()
	out, err := h.repo.AgenKomisi(ctx, id, identity.GetBrandID(ctx), identity.HasPermission(ctx, "komisi.view"))
	if err != nil {
		handleRepoError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
