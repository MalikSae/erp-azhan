package booking

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
)

const idBookingCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// Sentinel errors
var (
	ErrNotFound                        = errors.New("data tidak ditemukan")
	ErrSeatHabis                       = errors.New("kursi sudah habis, tidak bisa konfirmasi DP")
	ErrInvalidStatus                   = errors.New("status tidak valid")
	ErrSeatBelumDiblokir               = errors.New("kursi booking belum diblokir")
	ErrTemplatePerlengkapanBelumDiatur = errors.New("template set perlengkapan belum diatur untuk brand ini")
	ErrPerlengkapanSudahDiberikan      = errors.New("perlengkapan untuk booking ini sudah pernah diberikan")
	ErrPerlengkapanBelumDiberikan      = errors.New("perlengkapan belum pernah diberikan untuk booking ini")
	ErrKodeBrandNotSet                 = errors.New("kode_brand belum diatur untuk brand ini, hubungi Super Admin untuk mengatur di Kelola Brand")
	ErrGenerateIDBookingFailed         = errors.New("gagal generate ID Booking unik, coba lagi")
)

type ErrStokKurang struct {
	Message string
}

func (e *ErrStokKurang) Error() string {
	return e.Message
}

// Status yang menandakan kursi sudah terkunci (pernah dp/lebih).
var lockedStatuses = map[string]bool{
	"dp":              true,
	"lunas":           true,
	"dokumen_lengkap": true,
	"siap_berangkat":  true,
}

// AllowedProgressFields adalah mapping dari key request ke nama kolom database.
// Catatan: paspor tidak ada di sini karena status paspor computed dari dokumen_jamaah.
var AllowedProgressFields = map[string]string{
	"visa":              "progress_visa",
	"tiket":             "progress_tiket",
	"hotel":             "progress_hotel",
	"land_arrangement":  "progress_land_arrangement",
	"manasik":           "progress_manasik",
	"siskopatuh":        "progress_siskopatuh",
	"vaksin_meningitis": "progress_vaksin_meningitis",
}

// selectBookingFull adalah query SELECT booking dengan JOIN ke jamaah + schedules.
// Kolom progress_paspor di tabel bookings bersifat VESTIGIAL dan di-override secara dinamis
// berdasarkan keberadaan dokumen paspor di tabel dokumen_jamaah.
const selectBookingFull = `
	SELECT b.id, b.id_booking, b.schedule_id, s.jadwal_nama, s.berangkat_tanggal, b.jamaah_id, j.nama_lengkap,
		b.room_type, b.harga_dasar, b.status, b.is_seat_blocked, b.total_harga, b.diskon, b.diskon_keterangan,
		b.progress_paspor, b.progress_visa, b.progress_tiket, b.progress_hotel,
		b.progress_land_arrangement, b.progress_manasik, b.progress_siskopatuh, b.progress_vaksin_meningitis,
		b.perlengkapan_status, DATE_FORMAT(b.perlengkapan_tanggal, '%Y-%m-%d') AS perlengkapan_tanggal, b.perlengkapan_diberikan_oleh,
		b.created_by, b.created_at
	FROM bookings b
	JOIN jamaah j ON j.id = b.jamaah_id
	JOIN schedules s ON s.id = b.schedule_id`

// Repository mengelola semua query ke tabel bookings.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// List mengambil semua booking, filter brand via schedules.brand_id dan opsional jamaah_id.
func (r *Repository) List(ctx context.Context, brandID *int64, jamaahID *int64) ([]Booking, error) {
	q := selectBookingFull + " WHERE 1=1"
	var args []interface{}
	if brandID != nil {
		q += " AND s.brand_id = ?"
		args = append(args, *brandID)
	}
	if jamaahID != nil {
		q += " AND b.jamaah_id = ?"
		args = append(args, *jamaahID)
	}
	q += " ORDER BY b.created_at DESC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("booking.List: %w", err)
	}
	defer rows.Close()

	items := make([]Booking, 0)
	for rows.Next() {
		b, err := scanBookingRow(rows)
		if err != nil {
			return nil, err
		}
		hasPaspor, err := r.checkPasporUploaded(ctx, b.JamaahID)
		if err != nil {
			return nil, err
		}
		b.ProgressPaspor = hasPaspor
		computeSiapBerangkat(b)
		items = append(items, *b)
	}
	return items, rows.Err()
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

// GetByID mengambil booking lengkap, verify brand via schedule, beserta daftar addons.
func (r *Repository) GetByID(ctx context.Context, id int64, brandID *int64) (*Booking, error) {
	q := selectBookingFull + " WHERE b.id=?"
	var args []interface{}
	args = append(args, id)
	if brandID != nil {
		q += " AND s.brand_id=?"
		args = append(args, *brandID)
	}

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("booking.GetByID: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("booking.GetByID rows: %w", err)
		}
		return nil, ErrNotFound
	}
	b, err := scanBookingRow(rows)
	if err != nil {
		return nil, err
	}

	hasPaspor, err := r.checkPasporUploaded(ctx, b.JamaahID)
	if err != nil {
		return nil, err
	}
	b.ProgressPaspor = hasPaspor
	computeSiapBerangkat(b)

	addons, err := r.ListAddons(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Addons = addons
	return b, nil
}

// GenerateIDBooking generates a 6-character unique booking ID: {kode_brand (2 chars)}{4 random chars from charset}.
func (r *Repository) GenerateIDBooking(ctx context.Context, tx *sql.Tx, brandID int64) (string, error) {
	var kodeBrand sql.NullString
	qBrand := "SELECT kode_brand FROM brands WHERE id = ?"
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, qBrand, brandID).Scan(&kodeBrand)
	} else {
		err = r.db.QueryRowContext(ctx, qBrand, brandID).Scan(&kodeBrand)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("booking.GenerateIDBooking get brand: %w", err)
	}

	if !kodeBrand.Valid || strings.TrimSpace(kodeBrand.String) == "" {
		return "", ErrKodeBrandNotSet
	}

	prefix := strings.ToUpper(strings.TrimSpace(kodeBrand.String))
	charsetLen := big.NewInt(int64(len(idBookingCharset)))

	for attempt := 0; attempt < 10; attempt++ {
		randomBytes := make([]byte, 4)
		for i := 0; i < 4; i++ {
			num, err := rand.Int(rand.Reader, charsetLen)
			if err != nil {
				return "", fmt.Errorf("booking.GenerateIDBooking rand: %w", err)
			}
			randomBytes[i] = idBookingCharset[num.Int64()]
		}

		code := prefix + string(randomBytes)

		var count int
		qCheck := "SELECT COUNT(*) FROM bookings WHERE id_booking = ?"
		if tx != nil {
			err = tx.QueryRowContext(ctx, qCheck, code).Scan(&count)
		} else {
			err = r.db.QueryRowContext(ctx, qCheck, code).Scan(&count)
		}
		if err != nil {
			return "", fmt.Errorf("booking.GenerateIDBooking check uniqueness: %w", err)
		}

		if count == 0 {
			return code, nil
		}
	}

	return "", ErrGenerateIDBookingFailed
}

// ─── Create ───────────────────────────────────────────────────────────────────

// CreateBooking membuat booking baru dengan status='baru' (dipaksa server).
// Tidak mengurangi seat_sisa — kursi baru berkurang saat status jadi 'dp'.
func (r *Repository) CreateBooking(ctx context.Context, req *CreateBookingRequest, createdBy int64, autoHarga *float64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.Create tx: %w", err)
	}
	defer tx.Rollback()

	var brandID int64
	err = tx.QueryRowContext(ctx, `SELECT brand_id FROM schedules WHERE id = ?`, req.ScheduleID).Scan(&brandID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("booking.Create find schedule brand: %w", err)
	}

	idBooking, err := r.GenerateIDBooking(ctx, tx, brandID)
	if err != nil {
		return nil, err
	}

	totalHarga := req.TotalHarga
	if totalHarga == nil {
		totalHarga = autoHarga
	}
	hargaDasar := totalHarga

	const q = `INSERT INTO bookings (id_booking, schedule_id, jamaah_id, room_type, harga_dasar, status, total_harga, created_by)
		VALUES (?, ?, ?, ?, ?, 'baru', ?, ?)`

	res, err := tx.ExecContext(ctx, q, idBooking, req.ScheduleID, req.JamaahID, req.RoomType, hargaDasar, totalHarga, createdBy)
	if err != nil {
		return nil, fmt.Errorf("booking.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("booking.Create LastInsertId: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.Create commit: %w", err)
	}

	return r.GetByID(ctx, id, nil)
}

// ─── UpdateBookingStatus ──────────────────────────────────────────────────────

// UpdateBookingStatus mengubah status booking + logika seat_sisa kritis.
// Semua dalam 1 transaction dengan SELECT ... FOR UPDATE pada schedule.
func (r *Repository) UpdateBookingStatus(ctx context.Context, bookingID int64, newStatus string) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateStatus tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Ambil booking existing (status lama + schedule_id)
	var oldStatus string
	var isSeatBlocked bool
	var scheduleID int64
	err = tx.QueryRowContext(ctx,
		`SELECT status, schedule_id, is_seat_blocked FROM bookings WHERE id=?`, bookingID,
	).Scan(&oldStatus, &scheduleID, &isSeatBlocked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateStatus get old: %w", err)
	}

	// 2. Lock schedule row untuk cek/ubah seat_sisa
	var seatSisa int
	err = tx.QueryRowContext(ctx,
		`SELECT seat_sisa FROM schedules WHERE id=? FOR UPDATE`, scheduleID,
	).Scan(&seatSisa)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateStatus lock schedule: %w", err)
	}

	// 3. Logic seat_sisa
	newIsLocked := lockedStatuses[newStatus]

	if newStatus == "batal" && isSeatBlocked {
		// Kembalikan kursi: status lama sudah terkunci, sekarang dibatalkan
		_, err = tx.ExecContext(ctx,
			`UPDATE schedules SET seat_sisa = seat_sisa + 1 WHERE id=?`, scheduleID)
		if err != nil {
			return nil, fmt.Errorf("booking.UpdateStatus restore seat: %w", err)
		}
		_, err = tx.ExecContext(ctx, `UPDATE bookings SET is_seat_blocked=FALSE WHERE id=?`, bookingID)
		if err != nil {
			return nil, fmt.Errorf("booking.UpdateStatus clear seat block: %w", err)
		}
	} else if newIsLocked && oldStatus == "baru" && !isSeatBlocked {
		// Transisi PERTAMA KALI ke status terkunci (baru→dp, baru→lunas, dst)
		if seatSisa <= 0 {
			return nil, ErrSeatHabis
		}
		_, err = tx.ExecContext(ctx,
			`UPDATE schedules SET seat_sisa = seat_sisa - 1 WHERE id=?`, scheduleID)
		if err != nil {
			return nil, fmt.Errorf("booking.UpdateStatus decrement seat: %w", err)
		}
		_, err = tx.ExecContext(ctx, `UPDATE bookings SET is_seat_blocked=TRUE WHERE id=?`, bookingID)
		if err != nil {
			return nil, fmt.Errorf("booking.UpdateStatus mark seat block: %w", err)
		}
	}
	// else: transisi lain (dp→lunas, dll) → seat_sisa tidak berubah

	// 4. Update status booking
	_, err = tx.ExecContext(ctx,
		`UPDATE bookings SET status=? WHERE id=?`, newStatus, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateStatus update: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.UpdateStatus commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, nil)
}

// CancelSeatBlock melepaskan kursi tanpa mengubah status maupun data booking.
func (r *Repository) CancelSeatBlock(ctx context.Context, bookingID int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.CancelSeatBlock tx: %w", err)
	}
	defer tx.Rollback()

	var scheduleID int64
	var isBlocked bool
	err = tx.QueryRowContext(ctx, `SELECT schedule_id, is_seat_blocked FROM bookings WHERE id=? FOR UPDATE`, bookingID).Scan(&scheduleID, &isBlocked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.CancelSeatBlock find: %w", err)
	}
	if !isBlocked {
		return nil, ErrSeatBelumDiblokir
	}
	if _, err = tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa=LEAST(seat_total, seat_sisa + 1) WHERE id=?`, scheduleID); err != nil {
		return nil, fmt.Errorf("booking.CancelSeatBlock restore seat: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE bookings SET is_seat_blocked=FALSE WHERE id=?`, bookingID); err != nil {
		return nil, fmt.Errorf("booking.CancelSeatBlock update: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.CancelSeatBlock commit: %w", err)
	}
	return r.GetByID(ctx, bookingID, nil)
}

// ─── Addons & Diskon ──────────────────────────────────────────────────────────

func (r *Repository) recalculateTotalTx(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	const q = `
		UPDATE bookings
		SET total_harga = GREATEST(0, COALESCE(harga_dasar, 0) + (
			SELECT COALESCE(SUM(nominal), 0) FROM booking_addons WHERE booking_id = ?
		) - diskon)
		WHERE id = ?`
	if _, err := tx.ExecContext(ctx, q, bookingID, bookingID); err != nil {
		return err
	}

	// Cek status saat ini dan total_paid untuk auto-update status lunas/dp
	var totalHarga sql.NullFloat64
	var currentStatus string
	if err := tx.QueryRowContext(ctx, `SELECT total_harga, status FROM bookings WHERE id=?`, bookingID).Scan(&totalHarga, &currentStatus); err != nil {
		return err
	}
	if currentStatus == "batal" || currentStatus == "baru" {
		return nil
	}

	var totalPaid float64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah), 0) FROM payments WHERE booking_id=? AND status='confirmed'`, bookingID).Scan(&totalPaid); err != nil {
		return err
	}

	targetHarga := 0.0
	if totalHarga.Valid {
		targetHarga = totalHarga.Float64
	}

	if targetHarga > 0 && totalPaid >= targetHarga && currentStatus != "lunas" {
		_, err := tx.ExecContext(ctx, `UPDATE bookings SET status='lunas' WHERE id=?`, bookingID)
		return err
	} else if totalPaid < targetHarga && currentStatus == "lunas" {
		_, err := tx.ExecContext(ctx, `UPDATE bookings SET status='dp' WHERE id=?`, bookingID)
		return err
	}

	return nil
}

// ListAddons mengambil daftar add-on untuk sebuah booking.
func (r *Repository) ListAddons(ctx context.Context, bookingID int64) ([]BookingAddon, error) {
	const q = `SELECT id, booking_id, nama, nominal, created_at FROM booking_addons WHERE booking_id=? ORDER BY id ASC`
	rows, err := r.db.QueryContext(ctx, q, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking.ListAddons: %w", err)
	}
	defer rows.Close()

	items := make([]BookingAddon, 0)
	for rows.Next() {
		var a BookingAddon
		if err := rows.Scan(&a.ID, &a.BookingID, &a.Nama, &a.Nominal, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("booking.ListAddons scan: %w", err)
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// AddAddon menambahkan addon baru dan menghitung ulang total_harga.
func (r *Repository) AddAddon(ctx context.Context, bookingID int64, nama string, nominal float64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `INSERT INTO booking_addons (booking_id, nama, nominal) VALUES (?, ?, ?)`, bookingID, nama, nominal)
	if err != nil {
		return fmt.Errorf("booking.AddAddon insert: %w", err)
	}

	if err := r.recalculateTotalTx(ctx, tx, bookingID); err != nil {
		return fmt.Errorf("booking.AddAddon recalc: %w", err)
	}

	return tx.Commit()
}

// DeleteAddon menghapus addon dan menghitung ulang total_harga.
func (r *Repository) DeleteAddon(ctx context.Context, bookingID int64, addonID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `DELETE FROM booking_addons WHERE id=? AND booking_id=?`, addonID, bookingID)
	if err != nil {
		return fmt.Errorf("booking.DeleteAddon delete: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}

	if err := r.recalculateTotalTx(ctx, tx, bookingID); err != nil {
		return fmt.Errorf("booking.DeleteAddon recalc: %w", err)
	}

	return tx.Commit()
}

// UpdateDiskon mengubah nilai diskon & keterangan lalu menghitung ulang total_harga.
func (r *Repository) UpdateDiskon(ctx context.Context, bookingID int64, diskon float64, keterangan *string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `UPDATE bookings SET diskon=?, diskon_keterangan=? WHERE id=?`, diskon, keterangan, bookingID)
	if err != nil {
		return fmt.Errorf("booking.UpdateDiskon update: %w", err)
	}

	if err := r.recalculateTotalTx(ctx, tx, bookingID); err != nil {
		return fmt.Errorf("booking.UpdateDiskon recalc: %w", err)
	}

	return tx.Commit()
}

// ─── Existence checks (dipakai handler) ───────────────────────────────────────

// ScheduleExistsForBrand memeriksa apakah schedule_id ada (dan brand cocok kalau scoped).
func (r *Repository) ScheduleExistsForBrand(ctx context.Context, scheduleID int64, brandID *int64) (bool, error) {
	q := `SELECT COUNT(*) FROM schedules WHERE id=?`
	var args []interface{}
	args = append(args, scheduleID)
	if brandID != nil {
		q += " AND brand_id=?"
		args = append(args, *brandID)
	}
	var count int
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&count)
	return count > 0, err
}

// JamaahExistsForBrand memeriksa apakah jamaah_id ada (dan brand cocok kalau scoped).
func (r *Repository) JamaahExistsForBrand(ctx context.Context, jamaahID int64, brandID *int64) (bool, error) {
	q := `SELECT COUNT(*) FROM jamaah WHERE id=?`
	var args []interface{}
	args = append(args, jamaahID)
	if brandID != nil {
		q += " AND brand_id=?"
		args = append(args, *brandID)
	}
	var count int
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&count)
	return count > 0, err
}

// GetScheduleHarga mengambil harga sesuai room_type dari schedule.
func (r *Repository) GetScheduleHarga(ctx context.Context, scheduleID int64, roomType string) (*float64, error) {
	var col string
	switch roomType {
	case "Quad":
		col = "harga_quad"
	case "Triple":
		col = "harga_triple"
	case "Double":
		col = "harga_double"
	default:
		return nil, fmt.Errorf("room_type tidak valid: %s", roomType)
	}
	q := fmt.Sprintf("SELECT %s FROM schedules WHERE id=?", col)
	var harga float64
	err := r.db.QueryRowContext(ctx, q, scheduleID).Scan(&harga)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.GetScheduleHarga: %w", err)
	}
	return &harga, nil
}

// UpdateProgress melakukan partial update terhadap kolom progress_* pada sebuah booking.
func (r *Repository) UpdateProgress(ctx context.Context, bookingID int64, brandID *int64, updates map[string]bool) (*Booking, error) {
	if len(updates) == 0 {
		return r.GetByID(ctx, bookingID, brandID)
	}

	// Verify existence and brand
	if _, err := r.GetByID(ctx, bookingID, brandID); err != nil {
		return nil, err
	}

	var setClauses []string
	var args []interface{}

	for key, val := range updates {
		colName, ok := AllowedProgressFields[key]
		if !ok {
			return nil, fmt.Errorf("item progress tidak valid: %s", key)
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", colName))
		args = append(args, val)
	}

	args = append(args, bookingID)
	q := fmt.Sprintf("UPDATE bookings SET %s WHERE id = ?", strings.Join(setClauses, ", "))

	if _, err := r.db.ExecContext(ctx, q, args...); err != nil {
		return nil, fmt.Errorf("booking.UpdateProgress: %w", err)
	}

	return r.GetByID(ctx, bookingID, brandID)
}

// checkPasporUploaded memeriksa apakah jamaah memiliki dokumen paspor yang sudah diupload (file_url tidak null/kosong).
func (r *Repository) checkPasporUploaded(ctx context.Context, jamaahID int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) 
		FROM dokumen_jamaah 
		WHERE jamaah_id = ? AND jenis = 'paspor' AND file_url IS NOT NULL AND file_url != ''
	`, jamaahID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("booking.checkPasporUploaded: %w", err)
	}
	return count > 0, nil
}

func computeSiapBerangkat(b *Booking) {
	b.SiapBerangkat = b.ProgressPaspor && b.ProgressVisa && b.ProgressTiket && b.ProgressHotel &&
		b.ProgressLandArrangement && b.ProgressManasik && b.ProgressSiskopatuh && b.ProgressVaksinMeningitis
}

// ─── Internal scan helper ─────────────────────────────────────────────────────

func scanBookingRow(rows *sql.Rows) (*Booking, error) {
	var b Booking
	var idBooking sql.NullString
	var totalHarga sql.NullFloat64
	var hargaDasar sql.NullFloat64
	var diskonKeterangan sql.NullString
	var createdBy sql.NullInt64
	var berangkatTanggal sql.NullString
	var vestigialPaspor bool
	var perlengkapanStatus string
	var perlengkapanTanggal sql.NullString
	var perlengkapanDiberikanOleh sql.NullInt64

	err := rows.Scan(
		&b.ID, &idBooking, &b.ScheduleID, &b.JadwalNama, &berangkatTanggal, &b.JamaahID, &b.NamaJamaah,
		&b.RoomType, &hargaDasar, &b.Status, &b.IsSeatBlocked, &totalHarga, &b.Diskon, &diskonKeterangan,
		&vestigialPaspor, &b.ProgressVisa, &b.ProgressTiket, &b.ProgressHotel,
		&b.ProgressLandArrangement, &b.ProgressManasik, &b.ProgressSiskopatuh, &b.ProgressVaksinMeningitis,
		&perlengkapanStatus, &perlengkapanTanggal, &perlengkapanDiberikanOleh,
		&createdBy, &b.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("booking.scanRow: %w", err)
	}
	if idBooking.Valid {
		b.IDBooking = idBooking.String
	}
	if berangkatTanggal.Valid {
		b.BerangkatTanggal = &berangkatTanggal.String
	}
	if hargaDasar.Valid {
		b.HargaDasar = &hargaDasar.Float64
	}
	if totalHarga.Valid {
		b.TotalHarga = &totalHarga.Float64
	}
	if diskonKeterangan.Valid {
		b.DiskonKeterangan = &diskonKeterangan.String
	}
	if createdBy.Valid {
		b.CreatedBy = &createdBy.Int64
	}
	b.PerlengkapanStatus = perlengkapanStatus
	if perlengkapanTanggal.Valid {
		b.PerlengkapanTanggal = &perlengkapanTanggal.String
	}
	if perlengkapanDiberikanOleh.Valid {
		b.PerlengkapanDiberikanOleh = &perlengkapanDiberikanOleh.Int64
	}
	computeSiapBerangkat(&b)
	b.Addons = make([]BookingAddon, 0)
	return &b, nil
}

// ─── Perlengkapan Distribusi ──────────────────────────────────────────────────

// MarkPerlengkapanDiberikan mendistribusikan perlengkapan sesuai template set brand dalam 1 transaction.
func (r *Repository) MarkPerlengkapanDiberikan(ctx context.Context, bookingID int64, adminID int64, brandID *int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Ambil booking existing & brand_id dari schedule terkait
	var (
		status             string
		perlengkapanStatus string
		scheduleBrandID    int64
	)
	qBooking := `
		SELECT b.status, b.perlengkapan_status, s.brand_id
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ?
	`
	err = tx.QueryRowContext(ctx, qBooking, bookingID).Scan(&status, &perlengkapanStatus, &scheduleBrandID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan find booking: %w", err)
	}

	if brandID != nil && *brandID != scheduleBrandID {
		return nil, ErrNotFound
	}

	if perlengkapanStatus == "sudah_diberikan" {
		return nil, ErrPerlengkapanSudahDiberikan
	}

	// 2. Ambil template set global & stok untuk brand tersebut dengan row lock FOR UPDATE
	qSet := `
		SELECT t.perlengkapan_item_id, t.qty, i.nama, s.stok_tersedia
		FROM perlengkapan_set_template t
		JOIN perlengkapan_stok s ON t.perlengkapan_item_id = s.perlengkapan_item_id AND s.brand_id = ?
		JOIN perlengkapan_items i ON t.perlengkapan_item_id = i.id
		FOR UPDATE
	`
	rows, err := tx.QueryContext(ctx, qSet, scheduleBrandID)
	if err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan lock set: %w", err)
	}
	defer rows.Close()

	type setItemRow struct {
		ItemID       uint64
		Qty          int
		Nama         string
		StokTersedia int
	}

	var setItems []setItemRow
	for rows.Next() {
		var item setItemRow
		if err := rows.Scan(&item.ItemID, &item.Qty, &item.Nama, &item.StokTersedia); err != nil {
			return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan scan set: %w", err)
		}
		setItems = append(setItems, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan rows: %w", err)
	}

	if len(setItems) == 0 {
		return nil, ErrTemplatePerlengkapanBelumDiatur
	}

	// 3. Cek apakah ada stok yang kurang
	var kurangItems []string
	for _, item := range setItems {
		if item.StokTersedia < item.Qty {
			kurangItems = append(kurangItems, fmt.Sprintf("%s (tersedia %d, butuh %d)", item.Nama, item.StokTersedia, item.Qty))
		}
	}
	if len(kurangItems) > 0 {
		return nil, &ErrStokKurang{
			Message: "stok tidak cukup untuk: " + strings.Join(kurangItems, ", "),
		}
	}

	// 4. Potong stok untuk setiap item di set pada tabel perlengkapan_stok
	for _, item := range setItems {
		_, err := tx.ExecContext(ctx, "UPDATE perlengkapan_stok SET stok_tersedia = stok_tersedia - ? WHERE brand_id = ? AND perlengkapan_item_id = ?", item.Qty, scheduleBrandID, item.ItemID)
		if err != nil {
			return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan potong stok: %w", err)
		}
	}

	// 5. Update status perlengkapan pada booking
	qUpdateBooking := `
		UPDATE bookings
		SET perlengkapan_status = 'sudah_diberikan',
		    perlengkapan_tanggal = CURDATE(),
		    perlengkapan_diberikan_oleh = ?
		WHERE id = ?
	`
	if _, err := tx.ExecContext(ctx, qUpdateBooking, adminID, bookingID); err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan update booking: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, brandID)
}

// BatalkanPerlengkapan membatalkan status perlengkapan dan mengembalikan stok item ke database.
func (r *Repository) BatalkanPerlengkapan(ctx context.Context, bookingID int64, brandID *int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.BatalkanPerlengkapan tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Ambil booking existing & brand_id
	var (
		perlengkapanStatus string
		scheduleBrandID    int64
	)
	qBooking := `
		SELECT b.perlengkapan_status, s.brand_id
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ?
	`
	err = tx.QueryRowContext(ctx, qBooking, bookingID).Scan(&perlengkapanStatus, &scheduleBrandID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.BatalkanPerlengkapan find booking: %w", err)
	}

	if brandID != nil && *brandID != scheduleBrandID {
		return nil, ErrNotFound
	}

	if perlengkapanStatus != "sudah_diberikan" {
		return nil, ErrPerlengkapanBelumDiberikan
	}

	// 2. Ambil template set dengan row lock FOR UPDATE
	qSet := `
		SELECT perlengkapan_item_id, qty
		FROM perlengkapan_set_template
		FOR UPDATE
	`
	rows, err := tx.QueryContext(ctx, qSet)
	if err != nil {
		return nil, fmt.Errorf("booking.BatalkanPerlengkapan lock set: %w", err)
	}
	defer rows.Close()

	type setItemQty struct {
		ItemID uint64
		Qty    int
	}
	var setItems []setItemQty
	for rows.Next() {
		var item setItemQty
		if err := rows.Scan(&item.ItemID, &item.Qty); err != nil {
			return nil, fmt.Errorf("booking.BatalkanPerlengkapan scan: %w", err)
		}
		setItems = append(setItems, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("booking.BatalkanPerlengkapan rows: %w", err)
	}

	// 3. Kembalikan stok item ke tabel perlengkapan_stok
	for _, item := range setItems {
		_, err := tx.ExecContext(ctx, "UPDATE perlengkapan_stok SET stok_tersedia = stok_tersedia + ? WHERE brand_id = ? AND perlengkapan_item_id = ?", item.Qty, scheduleBrandID, item.ItemID)
		if err != nil {
			return nil, fmt.Errorf("booking.BatalkanPerlengkapan kembalikan stok: %w", err)
		}
	}

	// 4. Reset booking status perlengkapan
	qUpdateBooking := `
		UPDATE bookings
		SET perlengkapan_status = 'belum_diberikan',
		    perlengkapan_tanggal = NULL,
		    perlengkapan_diberikan_oleh = NULL
		WHERE id = ?
	`
	if _, err := tx.ExecContext(ctx, qUpdateBooking, bookingID); err != nil {
		return nil, fmt.Errorf("booking.BatalkanPerlengkapan update booking: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.BatalkanPerlengkapan commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, brandID)
}
