package payment

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"erp-azhan/api/internal/identity"
)

// FinanceSummary adalah ringkasan keuangan untuk dashboard finance.
type FinanceSummary struct {
	Kas struct {
		HariIni        float64 `json:"hari_ini"`
		BulanIni       float64 `json:"bulan_ini"`
		BulanLalu      float64 `json:"bulan_lalu"`
		RefundBulanIni float64 `json:"refund_bulan_ini"`
		NetBulanIni    float64 `json:"net_bulan_ini"`
	} `json:"kas"`
	MenungguVerifikasi struct {
		Jumlah  int     `json:"jumlah"`
		Nominal float64 `json:"nominal"`
	} `json:"menunggu_verifikasi"`
	Piutang struct {
		JumlahBooking int     `json:"jumlah_booking"`
		Nominal       float64 `json:"nominal"`
	} `json:"piutang"`
	PerluRefund struct {
		JumlahBooking int     `json:"jumlah_booking"`
		Nominal       float64 `json:"nominal"`
	} `json:"perlu_refund"`
	PerBrand        []FinanceBrandRow `json:"per_brand"`
	PendingTerbaru  []FinancePending  `json:"pending_terbaru"`
}

// FinanceBrandRow merangkum angka keuangan satu brand (hanya untuk holding).
type FinanceBrandRow struct {
	BrandID            int64   `json:"brand_id"`
	BrandName          string  `json:"brand_name"`
	KasBulanIni        float64 `json:"kas_bulan_ini"`
	Piutang            float64 `json:"piutang"`
	MenungguVerifikasi int     `json:"menunggu_verifikasi"`
}

// FinancePending adalah pembayaran pending terbaru yang menunggu verifikasi.
type FinancePending struct {
	ID          int64     `json:"id"`
	BookingID   int64     `json:"booking_id"`
	IDBooking   string    `json:"id_booking"`
	Jumlah      float64   `json:"jumlah"`
	Metode      *string   `json:"metode"`
	JamaahName  string    `json:"jamaah_name"`
	BrandName   string    `json:"brand_name"`
	CreatedAt   time.Time `json:"created_at"`
}

// FinanceSummary menghitung ringkasan keuangan; brandID != nil membatasi ke satu brand.
func (r *Repository) FinanceSummary(ctx context.Context, brandID *int64) (*FinanceSummary, error) {
	out := &FinanceSummary{PerBrand: []FinanceBrandRow{}, PendingTerbaru: []FinancePending{}}
	bf, args := "", []any{}
	if brandID != nil {
		bf = " AND s.brand_id = ?"
		args = []any{*brandID}
	}
	rep := func(n int) []any { // argumen brand diulang untuk tiap kemunculan filter
		var a []any
		for i := 0; i < n; i++ {
			a = append(a, args...)
		}
		return a
	}

	// Kas masuk (pembayaran terkonfirmasi, tanggal efektif = tanggal bayar).
	qKas := `
		SELECT
			COALESCE(SUM(CASE WHEN d = CURDATE() THEN jumlah END), 0),
			COALESCE(SUM(CASE WHEN DATE_FORMAT(d,'%Y%m') = DATE_FORMAT(CURDATE(),'%Y%m') THEN jumlah END), 0),
			COALESCE(SUM(CASE WHEN DATE_FORMAT(d,'%Y%m') = DATE_FORMAT(CURDATE() - INTERVAL 1 MONTH,'%Y%m') THEN jumlah END), 0)
		FROM (
			SELECT COALESCE(p.tanggal, DATE(p.created_at)) AS d, p.jumlah
			FROM payments p JOIN bookings b ON b.id = p.booking_id JOIN schedules s ON s.id = b.schedule_id
			WHERE p.status = 'confirmed'` + bf + `
		) x`
	if err := r.db.QueryRowContext(ctx, qKas, rep(1)...).Scan(&out.Kas.HariIni, &out.Kas.BulanIni, &out.Kas.BulanLalu); err != nil {
		return nil, fmt.Errorf("payment.FinanceSummary kas: %w", err)
	}
	qRefund := `
		SELECT COALESCE(SUM(rf.jumlah), 0)
		FROM booking_refunds rf JOIN bookings b ON b.id = rf.booking_id JOIN schedules s ON s.id = b.schedule_id
		WHERE DATE_FORMAT(rf.tanggal,'%Y%m') = DATE_FORMAT(CURDATE(),'%Y%m')` + bf
	if err := r.db.QueryRowContext(ctx, qRefund, rep(1)...).Scan(&out.Kas.RefundBulanIni); err != nil {
		return nil, fmt.Errorf("payment.FinanceSummary refund: %w", err)
	}
	out.Kas.NetBulanIni = out.Kas.BulanIni - out.Kas.RefundBulanIni

	// Menunggu verifikasi.
	qPend := `
		SELECT COUNT(*), COALESCE(SUM(p.jumlah), 0)
		FROM payments p JOIN bookings b ON b.id = p.booking_id JOIN schedules s ON s.id = b.schedule_id
		WHERE p.status = 'pending' AND b.status <> 'draft'` + bf
	if err := r.db.QueryRowContext(ctx, qPend, rep(1)...).Scan(&out.MenungguVerifikasi.Jumlah, &out.MenungguVerifikasi.Nominal); err != nil {
		return nil, fmt.Errorf("payment.FinanceSummary pending: %w", err)
	}

	// Piutang: sisa tagihan booking aktif (baru/dp).
	qPiutang := `
		SELECT COUNT(*), COALESCE(SUM(sisa), 0) FROM (
			SELECT b.total_harga - COALESCE((SELECT SUM(jumlah) FROM payments WHERE booking_id = b.id AND status = 'confirmed'), 0) AS sisa
			FROM bookings b JOIN schedules s ON s.id = b.schedule_id
			WHERE b.status IN ('baru','dp') AND b.total_harga > 0` + bf + `
		) x WHERE sisa > 0`
	if err := r.db.QueryRowContext(ctx, qPiutang, rep(1)...).Scan(&out.Piutang.JumlahBooking, &out.Piutang.Nominal); err != nil {
		return nil, fmt.Errorf("payment.FinanceSummary piutang: %w", err)
	}

	// Perlu refund: booking batal dengan dana masuk yang belum dikembalikan penuh.
	qRefundDue := `
		SELECT COUNT(*), COALESCE(SUM(sisa), 0) FROM (
			SELECT COALESCE((SELECT SUM(jumlah) FROM payments WHERE booking_id = b.id AND status = 'confirmed'), 0)
				 - COALESCE((SELECT SUM(jumlah) FROM booking_refunds WHERE booking_id = b.id), 0) AS sisa
			FROM bookings b JOIN schedules s ON s.id = b.schedule_id
			WHERE b.status = 'batal'` + bf + `
		) x WHERE sisa > 0`
	if err := r.db.QueryRowContext(ctx, qRefundDue, rep(1)...).Scan(&out.PerluRefund.JumlahBooking, &out.PerluRefund.Nominal); err != nil {
		return nil, fmt.Errorf("payment.FinanceSummary perlu refund: %w", err)
	}

	// Per brand (holding saja).
	if brandID == nil {
		const qBrand = `
			SELECT br.id, br.name,
				COALESCE((SELECT SUM(p.jumlah) FROM payments p JOIN bookings b ON b.id = p.booking_id JOIN schedules s ON s.id = b.schedule_id
					WHERE s.brand_id = br.id AND p.status = 'confirmed'
					AND DATE_FORMAT(COALESCE(p.tanggal, DATE(p.created_at)),'%Y%m') = DATE_FORMAT(CURDATE(),'%Y%m')), 0),
				COALESCE((SELECT SUM(x.sisa) FROM (
					SELECT s.brand_id,
						b.total_harga - COALESCE((SELECT SUM(jumlah) FROM payments WHERE booking_id = b.id AND status = 'confirmed'), 0) AS sisa
					FROM bookings b JOIN schedules s ON s.id = b.schedule_id
					WHERE b.status IN ('baru','dp') AND b.total_harga > 0
				) x WHERE x.brand_id = br.id AND x.sisa > 0), 0),
				(SELECT COUNT(*) FROM payments p JOIN bookings b ON b.id = p.booking_id JOIN schedules s ON s.id = b.schedule_id
					WHERE s.brand_id = br.id AND p.status = 'pending' AND b.status <> 'draft')
			FROM brands br ORDER BY br.name`
		rows, err := r.db.QueryContext(ctx, qBrand)
		if err != nil {
			return nil, fmt.Errorf("payment.FinanceSummary per brand: %w", err)
		}
		for rows.Next() {
			var row FinanceBrandRow
			if err := rows.Scan(&row.BrandID, &row.BrandName, &row.KasBulanIni, &row.Piutang, &row.MenungguVerifikasi); err != nil {
				rows.Close()
				return nil, err
			}
			out.PerBrand = append(out.PerBrand, row)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}

	// Pembayaran pending terbaru.
	qList := `
		SELECT p.id, p.booking_id, COALESCE(b.id_booking, ''), p.jumlah, p.metode, j.nama_lengkap, br.name, p.created_at
		FROM payments p
		JOIN bookings b ON b.id = p.booking_id
		JOIN jamaah j ON j.id = b.pic_jamaah_id
		JOIN schedules s ON s.id = b.schedule_id
		JOIN brands br ON br.id = s.brand_id
		WHERE p.status = 'pending' AND b.status <> 'draft'` + bf + `
		ORDER BY p.created_at DESC LIMIT 8`
	rows, err := r.db.QueryContext(ctx, qList, rep(1)...)
	if err != nil {
		return nil, fmt.Errorf("payment.FinanceSummary pending list: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p FinancePending
		if err := rows.Scan(&p.ID, &p.BookingID, &p.IDBooking, &p.Jumlah, &p.Metode, &p.JamaahName, &p.BrandName, &p.CreatedAt); err != nil {
			return nil, err
		}
		out.PendingTerbaru = append(out.PendingTerbaru, p)
	}
	return out, rows.Err()
}

// FinanceSummary godoc
// GET /api/admin/finance/ringkasan  (report.finance.view; admin brand otomatis hanya brand-nya)
func (h *Handler) FinanceSummary(w http.ResponseWriter, r *http.Request) {
	s, err := h.repo.FinanceSummary(r.Context(), identity.GetBrandID(r.Context()))
	if err != nil {
		log.Printf("[ERROR] payment: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil ringkasan keuangan")
		return
	}
	writeJSON(w, http.StatusOK, s)
}
