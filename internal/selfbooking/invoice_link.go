package selfbooking

import (
	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
	"net/http"
)

// Invoice links are capabilities; mint/retrieve only for the PIC or their active agent.
func (h *Handler) InvoiceLink(w http.ResponseWriter, r *http.Request) {
	userID := identity.GetPortalJamaahID(r.Context())
	var token string
	var bookingID int64
	accessErr := h.repo.db.QueryRowContext(r.Context(), `SELECT b.id FROM bookings b JOIN jamaah j ON j.id=b.pic_jamaah_id WHERE b.id=? AND (b.pic_jamaah_id=? OR (j.direkrut_oleh_jamaah_id=? AND EXISTS(SELECT 1 FROM jamaah a WHERE a.id=? AND a.status_agen='aktif')))`, chi.URLParam(r, "id"), userID, userID, userID).Scan(&bookingID)
	if accessErr != nil {
		writeError(w, 404, "invoice tidak ditemukan")
		return
	}
	fresh, tokenErr := newInvoiceToken()
	if tokenErr != nil {
		writeError(w, 500, "gagal memuat invoice")
		return
	}
	_, snapshotErr := h.repo.db.ExecContext(r.Context(), `INSERT IGNORE INTO booking_checkout(booking_id,dp_per_pax,full_payment,due_at,original_expires_at,invoice_token,terms_version) SELECT b.id,COALESCE(s.minimal_dp,br.minimal_dp,0),FALSE,GREATEST(DATE_SUB(s.berangkat_tanggal,INTERVAL 45 DAY),DATE_ADD(b.created_at,INTERVAL 1 DAY)),b.seat_hold_expires_at,?,'admin-legacy' FROM bookings b JOIN schedules s ON s.id=b.schedule_id JOIN brands br ON br.id=s.brand_id WHERE b.id=?`, fresh, bookingID)
	if snapshotErr != nil {
		writeError(w, 500, "gagal memuat invoice")
		return
	}
	err := h.repo.db.QueryRowContext(r.Context(), `SELECT c.invoice_token FROM booking_checkout c JOIN bookings b ON b.id=c.booking_id JOIN jamaah j ON j.id=b.pic_jamaah_id WHERE b.id=? AND (b.pic_jamaah_id=? OR (j.direkrut_oleh_jamaah_id=? AND EXISTS(SELECT 1 FROM jamaah a WHERE a.id=? AND a.status_agen='aktif')))`, chi.URLParam(r, "id"), userID, userID, userID).Scan(&token)
	if err != nil {
		writeError(w, 404, "invoice tidak ditemukan")
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	writeJSON(w, 200, map[string]string{"invoice_token": token})
}
