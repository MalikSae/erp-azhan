package portal

import (
	"database/sql"
	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/jamaah"
	"net/http"
	"time"
)

// Never serialize the admin profile directly to the customer portal.
func profileView(j *jamaah.Jamaah) map[string]any {
	return map[string]any{"id": j.ID, "brand_id": j.BrandID, "id_jamaah": j.IDJamaah, "nama_lengkap": j.NamaLengkap, "jenis_kelamin": j.JenisKelamin, "nik": j.NIK, "tanggal_lahir": j.TanggalLahir, "tempat_lahir": j.TempatLahir, "no_paspor": j.NoPaspor, "no_hp": j.NoHP, "email": j.Email, "alamat": j.Alamat, "kota": j.Kota, "status_agen": j.StatusAgen}
}

type portalBooking struct {
	*booking.Booking
	DueAt              *string             `json:"due_at"`
	DPPerPax           float64             `json:"dp_per_pax"`
	FullPayment        bool                `json:"full_payment"`
	ReservationPayable bool                `json:"reservation_payable"`
	CanSubmitPayment   bool                `json:"can_submit_payment"`
	CanViewInvoice     bool                `json:"can_view_invoice"`
	PersonalPax        *booking.BookingPax `json:"personal_pax"`
}

func (h *Handler) bookingView(r *http.Request, b *booking.Booking, id int64) (*portalBooking, error) {
	v := &portalBooking{Booking: b}
	var due sql.NullTime
	var dp sql.NullFloat64
	var full sql.NullBool
	var payable bool
	err := h.db.QueryRowContext(r.Context(), `SELECT c.due_at,c.dp_per_pax,c.full_payment,(b.status IN ('baru','dp') AND b.is_seat_blocked=TRUE AND (b.seat_hold_expires_at IS NULL OR b.seat_hold_expires_at>NOW())) FROM bookings b LEFT JOIN booking_checkout c ON c.booking_id=b.id WHERE b.id=?`, b.ID).Scan(&due, &dp, &full, &payable)
	if err != nil {
		return nil, err
	}
	if due.Valid {
		s := due.Time.UTC().Format(time.RFC3339)
		v.DueAt = &s
	}
	v.DPPerPax = dp.Float64
	v.FullPayment = full.Bool
	v.ReservationPayable = payable
	pic := b.PicJamaahID != nil && *b.PicJamaahID == id
	v.CanSubmitPayment = pic && payable
	v.CanViewInvoice = pic
	for i := range b.Pax {
		if b.Pax[i].JamaahID == id {
			v.PersonalPax = &b.Pax[i]
			break
		}
	}
	return v, nil
}
