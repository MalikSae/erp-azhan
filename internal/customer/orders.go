package customer

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/media"
	"erp-azhan/api/internal/payment"
	"github.com/go-chi/chi/v5"
)

// Orders melayani halaman "Pesanan Saya": daftar, detail, pembatalan, dan pembayaran
// milik akun yang sedang login. Semua query dibatasi customer_account_id akun itu.
type Orders struct {
	db          *sql.DB
	bookingRepo *booking.Repository
	paymentRepo *payment.Repository
}

func NewOrders(db *sql.DB, b *booking.Repository, p *payment.Repository) *Orders {
	return &Orders{db: db, bookingRepo: b, paymentRepo: p}
}

type orderSummary struct {
	ID                 int64      `json:"id"`
	BookingCode        string     `json:"booking_code"`
	Status             string     `json:"status"`
	Tab                string     `json:"tab"` // menunggu | sebagian | lunas | batal
	ScheduleID         int64      `json:"schedule_id"`
	JadwalNama         string     `json:"jadwal_nama"`
	BerangkatTanggal   string     `json:"berangkat_tanggal"`
	SeatCount          int        `json:"seat_count"`
	TotalHarga         float64    `json:"total_harga"`
	MinimalDP          float64    `json:"minimal_dp"`
	FullPayment        bool       `json:"full_payment"`
	Terbayar           float64    `json:"terbayar"`
	MenungguVerifikasi float64    `json:"menunggu_verifikasi"`
	Sisa               float64    `json:"sisa"`
	DueAt              *time.Time `json:"due_at"`
	SeatHoldExpiresAt  *time.Time `json:"seat_hold_expires_at"`
	Expired            bool       `json:"expired"`
	CanPay             bool       `json:"can_pay"`
	CanCancel          bool       `json:"can_cancel"`
	CreatedAt          time.Time  `json:"created_at"`
}

const orderSelect = `
	SELECT b.id, b.id_booking, b.status, b.schedule_id, s.jadwal_nama, DATE_FORMAT(s.berangkat_tanggal,'%Y-%m-%d'),
	       b.seat_count, COALESCE(b.total_harga,0), b.is_seat_blocked, b.seat_hold_expires_at,
	       c.due_at, COALESCE(c.dp_per_pax,0), COALESCE(c.full_payment,FALSE), b.created_at,
	       COALESCE(SUM(CASE WHEN p.status='confirmed' THEN p.jumlah END),0),
	       COALESCE(SUM(CASE WHEN p.status='pending' THEN p.jumlah END),0),
	       COUNT(p.id)
	FROM bookings b
	JOIN schedules s ON s.id = b.schedule_id
	LEFT JOIN booking_checkout c ON c.booking_id = b.id
	LEFT JOIN payments p ON p.booking_id = b.id
	WHERE b.customer_account_id = ? AND b.status <> 'draft'`

func scanOrder(row interface{ Scan(...any) error }) (*orderSummary, int, error) {
	var o orderSummary
	var blocked bool
	var hold, due sql.NullTime
	var dpPerPax float64
	var paymentCount int
	if err := row.Scan(&o.ID, &o.BookingCode, &o.Status, &o.ScheduleID, &o.JadwalNama, &o.BerangkatTanggal,
		&o.SeatCount, &o.TotalHarga, &blocked, &hold, &due, &dpPerPax, &o.FullPayment, &o.CreatedAt,
		&o.Terbayar, &o.MenungguVerifikasi, &paymentCount); err != nil {
		return nil, 0, err
	}
	if hold.Valid {
		t := hold.Time
		o.SeatHoldExpiresAt = &t
	}
	if due.Valid {
		t := due.Time
		o.DueAt = &t
	}
	now := time.Now()
	// Reservasi aktif = kursi masih ditahan dan belum lewat batas.
	holdActive := blocked && (hold.Valid == false || hold.Time.After(now))
	o.Expired = o.Status == "baru" && !holdActive
	o.MinimalDP = o.TotalHarga
	if !o.FullPayment {
		o.MinimalDP = minFloat(o.TotalHarga, dpPerPax*float64(o.SeatCount))
	}
	o.Sisa = maxFloat(0, o.TotalHarga-o.Terbayar)

	switch {
	case o.Status == "batal" || o.Expired:
		o.Tab = "batal"
	case o.Status == "lunas" || o.Status == "dokumen_lengkap" || o.Status == "siap_berangkat" || (o.TotalHarga > 0 && o.Terbayar >= o.TotalHarga):
		o.Tab = "lunas"
	case o.Terbayar > 0 || o.Status == "dp":
		o.Tab = "sebagian"
	default:
		o.Tab = "menunggu"
	}
	o.CanPay = (o.Tab == "menunggu" || o.Tab == "sebagian") && o.Sisa > 0 && (holdActive || o.Status == "dp")
	o.CanCancel = o.Status == "baru" && !o.Expired && paymentCount == 0
	return &o, paymentCount, nil
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// List GET /api/account/bookings?tab=
func (o *Orders) List(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	rows, err := o.db.QueryContext(r.Context(), orderSelect+` GROUP BY b.id ORDER BY b.id DESC LIMIT 200`, acc.ID)
	if err != nil {
		log.Printf("[ERROR] account list bookings: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar pesanan")
		return
	}
	defer rows.Close()
	tab := strings.TrimSpace(r.URL.Query().Get("tab"))
	items := []*orderSummary{}
	for rows.Next() {
		it, _, err := scanOrder(rows)
		if err != nil {
			log.Printf("[ERROR] account scan booking: %v", err)
			writeError(w, http.StatusInternalServerError, "gagal membaca pesanan")
			return
		}
		if tab == "" || tab == it.Tab {
			items = append(items, it)
		}
	}
	writeJSON(w, http.StatusOK, items)
}

type paxView struct {
	Nama     string  `json:"nama"`
	PaxType  string  `json:"pax_type"`
	RoomType *string `json:"room_type"`
	Harga    float64 `json:"harga"`
	Status   string  `json:"status"`
}

type bankView struct {
	ID            int64   `json:"id"`
	BankName      string  `json:"bank_name"`
	LogoURL       *string `json:"logo_url"`
	AccountNumber string  `json:"account_number"`
	AccountHolder string  `json:"account_holder"`
	Instructions  *string `json:"instructions"`
}

func (o *Orders) banks(ctx context.Context, brandID int64) ([]bankView, error) {
	rows, err := o.db.QueryContext(ctx, `SELECT id, bank_name, logo_url, account_number, account_holder, instructions FROM bank_accounts WHERE brand_id=? AND is_active=TRUE ORDER BY sort_order, id`, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []bankView{}
	for rows.Next() {
		var b bankView
		var logo sql.NullString
		if err := rows.Scan(&b.ID, &b.BankName, &logo, &b.AccountNumber, &b.AccountHolder, &b.Instructions); err != nil {
			return nil, err
		}
		if logo.Valid && strings.TrimSpace(logo.String) != "" {
			b.LogoURL = &logo.String
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// owned memuat ringkasan pesanan milik akun; ErrNotFound bila bukan miliknya.
func (o *Orders) owned(ctx context.Context, accountID, bookingID int64) (*orderSummary, int, error) {
	row := o.db.QueryRowContext(ctx, orderSelect+` AND b.id = ? GROUP BY b.id`, accountID, bookingID)
	it, n, err := scanOrder(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, 0, ErrNotFound
	}
	return it, n, err
}

func bookingIDParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id pesanan tidak valid")
		return 0, false
	}
	return id, true
}

// Detail GET /api/account/bookings/{id}
func (o *Orders) Detail(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	order, _, err := o.owned(r.Context(), acc.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "pesanan tidak ditemukan")
		return
	}
	if err != nil {
		log.Printf("[ERROR] account booking detail: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil pesanan")
		return
	}
	pax := []paxView{}
	rows, err := o.db.QueryContext(r.Context(), `SELECT j.nama_lengkap, bp.pax_type, bp.room_type, bp.harga_pax, bp.pax_status FROM booking_pax bp JOIN jamaah j ON j.id=bp.jamaah_id WHERE bp.booking_id=? ORDER BY bp.id`, id)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var p paxView
			if rows.Scan(&p.Nama, &p.PaxType, &p.RoomType, &p.Harga, &p.Status) == nil {
				pax = append(pax, p)
			}
		}
	}
	payments, err := o.paymentRepo.ListByBookingID(r.Context(), id, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil riwayat pembayaran")
		return
	}
	if payments == nil {
		payments = []payment.Payment{}
	}
	banks, err := o.banks(r.Context(), acc.BrandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil rekening")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"order": order, "pax": pax, "payments": payments, "bank_accounts": banks})
}

// Cancel POST /api/account/bookings/{id}/cancel: hanya sebelum ada pembayaran tercatat.
func (o *Orders) Cancel(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	order, _, err := o.owned(r.Context(), acc.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "pesanan tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil pesanan")
		return
	}
	if !order.CanCancel {
		writeError(w, http.StatusConflict, "pesanan ini tidak bisa dibatalkan sendiri, silakan hubungi admin")
		return
	}
	// Periksa ulang di dalam UpdateBookingStatus lewat status: hanya 'baru' yang lolos di atas.
	if _, err = o.bookingRepo.UpdateBookingStatus(r.Context(), id, "batal"); err != nil {
		log.Printf("[ERROR] account cancel booking %d: %v", id, err)
		writeError(w, http.StatusInternalServerError, "gagal membatalkan pesanan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "batal"})
}

// ListPayments GET /api/account/bookings/{id}/payments
func (o *Orders) ListPayments(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	if _, _, err := o.owned(r.Context(), acc.ID, id); err != nil {
		writeError(w, http.StatusNotFound, "pesanan tidak ditemukan")
		return
	}
	payments, err := o.paymentRepo.ListByBookingID(r.Context(), id, nil)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil riwayat pembayaran")
		return
	}
	if payments == nil {
		payments = []payment.Payment{}
	}
	writeJSON(w, http.StatusOK, payments)
}

// CreatePayment POST /api/account/bookings/{id}/payments: konfirmasi transfer manual.
func (o *Orders) CreatePayment(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	order, _, err := o.owned(r.Context(), acc.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "pesanan tidak ditemukan")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil pesanan")
		return
	}
	if !order.CanPay {
		writeError(w, http.StatusConflict, "pesanan ini tidak bisa menerima pembayaran, silakan hubungi admin")
		return
	}
	var req payment.CreatePaymentRequest
	if !decode(w, r, &req) {
		return
	}
	req.Source = "portal"
	if req.Tanggal == nil {
		writeError(w, http.StatusBadRequest, "tanggal transfer wajib diisi")
		return
	}
	transferDate, dateErr := time.Parse("2006-01-02", *req.Tanggal)
	today := time.Now().In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02")
	if dateErr != nil || transferDate.Format("2006-01-02") > today {
		writeError(w, http.StatusBadRequest, "tanggal transfer tidak valid atau berada di masa depan")
		return
	}
	if req.Jumlah <= 0 || req.BankAccountID == nil || req.SenderName == nil || strings.TrimSpace(*req.SenderName) == "" || req.BuktiURL == nil || strings.TrimSpace(*req.BuktiURL) == "" {
		writeError(w, http.StatusBadRequest, "rekening tujuan, nominal, nama pengirim, dan bukti transfer wajib diisi")
		return
	}
	if err := media.ValidateCustomerUpload(r.Context(), o.db, *req.BuktiURL, acc.ID); err != nil {
		writeError(w, http.StatusBadRequest, "bukti transfer harus berupa unggahan akun Anda yang masih tersedia")
		return
	}
	var accountBrand int64
	if err := o.db.QueryRowContext(r.Context(), `SELECT brand_id FROM bank_accounts WHERE id=? AND is_active=TRUE`, *req.BankAccountID).Scan(&accountBrand); err != nil || accountBrand != acc.BrandID {
		writeError(w, http.StatusBadRequest, "rekening tujuan tidak tersedia untuk brand ini")
		return
	}
	if req.Jumlah > order.Sisa {
		writeError(w, http.StatusBadRequest, "nominal melebihi sisa tagihan")
		return
	}
	item, err := o.paymentRepo.Create(r.Context(), id, &req, nil)
	if err != nil {
		if errors.Is(err, payment.ErrBookingClosed) || errors.Is(err, payment.ErrOverpayment) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		log.Printf("[ERROR] account create payment: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan konfirmasi pembayaran")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// BankAccounts GET /api/account/bank-accounts
func (o *Orders) BankAccounts(w http.ResponseWriter, r *http.Request) {
	banks, err := o.banks(r.Context(), FromContext(r.Context()).BrandID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil rekening")
		return
	}
	writeJSON(w, http.StatusOK, banks)
}

type travelerView struct {
	ID           int64   `json:"id"`
	NamaLengkap  string  `json:"nama_lengkap"`
	JenisKelamin string  `json:"jenis_kelamin"`
	NoHP         *string `json:"no_hp"`
	TanggalLahir *string `json:"tanggal_lahir"`
	NIK          *string `json:"nik"`
}

// Travelers GET /api/account/travelers
func (o *Orders) Travelers(w http.ResponseWriter, r *http.Request) {
	rows, err := o.db.QueryContext(r.Context(), `SELECT id, nama_lengkap, jenis_kelamin, no_hp, DATE_FORMAT(tanggal_lahir,'%Y-%m-%d'), nik FROM customer_travelers WHERE account_id=? ORDER BY id DESC LIMIT 300`, FromContext(r.Context()).ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "gagal mengambil jamaah tersimpan")
		return
	}
	defer rows.Close()
	out := []travelerView{}
	for rows.Next() {
		var t travelerView
		if rows.Scan(&t.ID, &t.NamaLengkap, &t.JenisKelamin, &t.NoHP, &t.TanggalLahir, &t.NIK) == nil {
			out = append(out, t)
		}
	}
	writeJSON(w, http.StatusOK, out)
}
