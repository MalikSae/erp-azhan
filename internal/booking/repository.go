package booking

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"
)

const idBookingCharset = "23456789ABCDEFGHJKLMNPQRSTUVWXYZ"

// Sentinel errors
var (
	ErrNotFound                        = errors.New("data tidak ditemukan")
	ErrSeatHabis                       = errors.New("kursi sudah habis, tidak bisa konfirmasi DP")
	ErrInvalidStatus                   = errors.New("status tidak valid")
	ErrSeatBelumDiblokir               = errors.New("kursi booking belum diblokir")
	ErrSeatSudahDiblokir               = errors.New("kursi booking sudah diblokir oleh permintaan lain")
	ErrTemplatePerlengkapanBelumDiatur = errors.New("template set perlengkapan belum diatur untuk brand ini")
	ErrPerlengkapanSudahDiberikan      = errors.New("perlengkapan untuk booking ini sudah pernah diberikan")
	ErrPerlengkapanBelumDiberikan      = errors.New("perlengkapan belum pernah diberikan untuk booking ini")
	ErrKodeBrandNotSet                 = errors.New("kode_brand belum diatur untuk brand ini, hubungi Super Admin untuk mengatur di Kelola Brand")
	ErrGenerateIDBookingFailed         = errors.New("gagal generate ID Booking unik, coba lagi")
	ErrPaxAlreadyCancelled             = errors.New("pax sudah dalam status batal")
	ErrCannotChangeInfantRoom          = errors.New("tidak bisa mengubah tipe kamar untuk infant")
	ErrCannotChangeCancelledPaxRoom    = errors.New("tidak bisa mengubah tipe kamar untuk pax yang sudah batal")
)

type ErrStokKurang struct {
	Message string
}

func (e *ErrStokKurang) Error() string {
	return e.Message
}

type ErrSeatNotEnough struct {
	Message string
}

func (e *ErrSeatNotEnough) Error() string {
	return e.Message
}

// Status yang menandakan kursi sudah terkunci (pernah dp/lebih).
var lockedStatuses = map[string]bool{
	"dp":    true,
	"lunas": true,
}

var (
	ErrPaxBatal      = errors.New("tidak dapat mengubah progress untuk pax yang sudah dibatalkan")
	ErrManasikInfant = errors.New("Manasik tidak berlaku untuk infant")
)

// AllowedHeaderProgressFields adalah mapping dari key request header ke nama kolom bookings.
var AllowedHeaderProgressFields = map[string]string{
	"hotel":            "progress_hotel",
	"land_arrangement": "progress_land_arrangement",
}

// AllowedPaxProgressFields adalah mapping dari key request pax ke nama kolom booking_pax.
var AllowedPaxProgressFields = map[string]string{
	"visa":              "progress_visa",
	"siskopatuh":        "progress_siskopatuh",
	"manasik":           "progress_manasik",
	"vaksin_meningitis": "progress_vaksin_meningitis",
}

// selectBookingFull adalah query SELECT booking dengan JOIN ke jamaah + schedules + primary pax.
const selectBookingFull = `
	SELECT b.id, b.id_booking, b.schedule_id, s.brand_id, s.jadwal_nama, s.berangkat_tanggal,
		b.pic_jamaah_id, j.nama_lengkap,
		bp.room_type,
		bp.harga_pax,
		b.seat_count, b.status, b.is_seat_blocked, b.seat_hold_expires_at,
		b.total_harga, b.diskon, b.diskon_keterangan,
		COALESCE(bp.progress_visa, FALSE) AS progress_visa,
		b.progress_hotel,
		b.progress_land_arrangement,
		COALESCE(bp.progress_manasik, FALSE) AS progress_manasik,
		COALESCE(bp.progress_siskopatuh, FALSE) AS progress_siskopatuh,
		COALESCE(bp.progress_vaksin_meningitis, FALSE) AS progress_vaksin_meningitis,
		b.perlengkapan_status, DATE_FORMAT(b.perlengkapan_tanggal, '%Y-%m-%d') AS perlengkapan_tanggal, b.perlengkapan_diberikan_oleh,
		b.perlengkapan_jumlah_pax,
		b.created_by, b.created_at,
		s.is_ticket_confirmed,
		b.pic_jamaah_id AS primary_jamaah_id,
		(SELECT COUNT(*) FROM booking_pax WHERE booking_id = b.id AND pax_status = 'aktif') AS pax_count,
		(SELECT COUNT(*) FROM booking_pax WHERE booking_id = b.id AND pax_status = 'aktif' AND pax_type = 'reguler') AS regular_pax_count,
		(SELECT COUNT(*) FROM booking_pax WHERE booking_id = b.id AND pax_status = 'aktif' AND pax_type = 'infant') AS infant_pax_count
	FROM bookings b
	LEFT JOIN jamaah j ON j.id = b.pic_jamaah_id
	JOIN schedules s ON s.id = b.schedule_id
	LEFT JOIN booking_pax bp ON bp.id = (
		SELECT id FROM booking_pax 
		WHERE booking_id = b.id AND pax_status = 'aktif'
		ORDER BY CASE WHEN jamaah_id = b.pic_jamaah_id THEN 0 ELSE 1 END,
		         CASE WHEN room_type IS NOT NULL THEN 0 ELSE 1 END,
		         id ASC
		LIMIT 1
	)`

// Repository mengelola semua query ke tabel bookings dan booking_pax.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// List mengambil semua booking, filter brand via schedules.brand_id dan opsional jamaah_id serta status.
func (r *Repository) List(ctx context.Context, brandID *int64, jamaahID *int64, status string) ([]Booking, error) {
	q := selectBookingFull + " WHERE 1=1"
	var args []interface{}
	if brandID != nil {
		q += " AND s.brand_id = ?"
		args = append(args, *brandID)
	}
	if jamaahID != nil {
		q += " AND (b.pic_jamaah_id = ? OR EXISTS (SELECT 1 FROM booking_pax bp2 WHERE bp2.booking_id = b.id AND bp2.jamaah_id = ?))"
		args = append(args, *jamaahID, *jamaahID)
	}

	if status == "draft" {
		q += " AND b.status = 'draft'"
	} else if status == "non_draft" {
		q += " AND b.status != 'draft'"
	} else if status != "" && status != "all" {
		q += " AND b.status = ?"
		args = append(args, status)
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
		if b.JamaahID != nil && *b.JamaahID > 0 {
			hasPaspor, err := r.checkPasporUploaded(ctx, *b.JamaahID)
			if err == nil {
				b.ProgressPaspor = hasPaspor
			}
		}
		computeSiapBerangkat(b)
		items = append(items, *b)
	}
	return items, rows.Err()
}

// ─── GetByID ──────────────────────────────────────────────────────────────────

// GetByID mengambil booking lengkap, verify brand via schedule, beserta daftar pax dan addons.
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

	if b.JamaahID != nil && *b.JamaahID > 0 {
		hasPaspor, err := r.checkPasporUploaded(ctx, *b.JamaahID)
		if err == nil {
			b.ProgressPaspor = hasPaspor
		}
	}

	paxList, err := r.ListPaxByBookingID(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Pax = paxList
	computeSiapBerangkat(b)

	addons, err := r.ListAddons(ctx, id)
	if err != nil {
		return nil, err
	}
	b.Addons = addons
	return b, nil
}

// ListPaxByBookingID mengambil semua pax untuk sebuah booking.
func (r *Repository) ListPaxByBookingID(ctx context.Context, bookingID int64) ([]BookingPax, error) {
	const q = `
		SELECT bp.id, bp.booking_id, bp.jamaah_id, j.nama_lengkap, bp.pax_type, bp.room_type,
			bp.harga_pax, bp.counts_for_seat, bp.pax_status,
			bp.progress_visa, bp.progress_siskopatuh, bp.progress_manasik, bp.progress_vaksin_meningitis,
			bp.created_at, bp.updated_at
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE bp.booking_id = ?
		ORDER BY bp.id ASC`

	rows, err := r.db.QueryContext(ctx, q, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking.ListPax: %w", err)
	}
	defer rows.Close()

	items := make([]BookingPax, 0)
	for rows.Next() {
		var p BookingPax
		var roomType sql.NullString
		if err := rows.Scan(
			&p.ID, &p.BookingID, &p.JamaahID, &p.NamaJamaah, &p.PaxType, &roomType,
			&p.HargaPax, &p.CountsForSeat, &p.PaxStatus,
			&p.ProgressVisa, &p.ProgressSiskopatuh, &p.ProgressManasik, &p.ProgressVaksinMeningitis,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("booking.ListPax scan: %w", err)
		}
		if roomType.Valid {
			p.RoomType = &roomType.String
		}
		hasPaspor, err := r.checkPasporUploaded(ctx, p.JamaahID)
		if err == nil {
			p.ProgressPaspor = hasPaspor
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

// isInfantEligible mengecek apakah jamaah dengan tanggalLahir masih berusia < 2 tahun pada berangkatTanggal.
// Jika tanggalLahir nil/kosong, mengembalikan true (eligible karena data belum lengkap/fallback manual).
// Jika usia pada berangkatTanggal >= 2 tahun, mengembalikan false.
func isInfantEligible(tanggalLahir *string, berangkatTanggal *string) (bool, error) {
	if tanggalLahir == nil || strings.TrimSpace(*tanggalLahir) == "" {
		return true, nil
	}
	if berangkatTanggal == nil || strings.TrimSpace(*berangkatTanggal) == "" {
		return true, nil
	}

	dobStr := strings.TrimSpace(*tanggalLahir)
	if len(dobStr) >= 10 {
		dobStr = dobStr[:10]
	}
	dob, err := time.Parse("2006-01-02", dobStr)
	if err != nil {
		return true, nil
	}

	depStr := strings.TrimSpace(*berangkatTanggal)
	if len(depStr) >= 10 {
		depStr = depStr[:10]
	}
	dep, err := time.Parse("2006-01-02", depStr)
	if err != nil {
		return true, nil
	}

	twoYearsOld := dob.AddDate(2, 0, 0)
	if dep.After(twoYearsOld) || dep.Equal(twoYearsOld) {
		return false, nil // Berusia 2 tahun atau lebih pada hari keberangkatan
	}

	return true, nil // Berusia di bawah 2 tahun
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

// CreateBooking membuat booking baru dengan dukungan multi-pax.
func (r *Repository) CreateBooking(ctx context.Context, req *CreateBookingRequest, createdBy int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.Create tx: %w", err)
	}
	defer tx.Rollback()

	// 0. Validasi duplikasi jamaah dalam payload
	seenJamaah := make(map[int64]bool)
	for _, p := range req.Pax {
		if seenJamaah[p.JamaahID] {
			var namaLengkap string
			_ = tx.QueryRowContext(ctx, `SELECT nama_lengkap FROM jamaah WHERE id = ?`, p.JamaahID).Scan(&namaLengkap)
			if namaLengkap == "" {
				namaLengkap = fmt.Sprintf("ID %d", p.JamaahID)
			}
			return nil, fmt.Errorf("Jamaah %s didaftarkan lebih dari satu kali dalam booking ini", namaLengkap)
		}
		seenJamaah[p.JamaahID] = true
	}

	// 1. Lock schedule row
	var brandID int64
	var seatSisa int
	var hargaQuad, hargaTriple, hargaDouble float64
	var hargaInfant sql.NullFloat64
	var berangkatTanggal sql.NullString

	err = tx.QueryRowContext(ctx, `
		SELECT brand_id, seat_sisa, harga_quad, harga_triple, harga_double, harga_infant, DATE_FORMAT(berangkat_tanggal, '%Y-%m-%d') 
		FROM schedules 
		WHERE id = ? FOR UPDATE`, req.ScheduleID).
		Scan(&brandID, &seatSisa, &hargaQuad, &hargaTriple, &hargaDouble, &hargaInfant, &berangkatTanggal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("booking.Create find schedule: %w", err)
	}

	// Validasi PIC bukan infant
	picID := req.PicJamaahID
	if picID == 0 && len(req.Pax) > 0 {
		picID = req.Pax[0].JamaahID
	}
	if picID > 0 && berangkatTanggal.Valid {
		var picTanggalLahir sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT DATE_FORMAT(tanggal_lahir, '%Y-%m-%d') FROM jamaah WHERE id = ?`, picID).Scan(&picTanggalLahir)
		if err == nil && picTanggalLahir.Valid {
			eligible, _ := isInfantEligible(&picTanggalLahir.String, &berangkatTanggal.String)
			if eligible {
				return nil, fmt.Errorf("PIC (Kontak Utama) tidak boleh berstatus infant")
			}
		}
	}

	// 2. Hitung jumlah pax reguler
	regularCount := 0
	for _, p := range req.Pax {
		if p.PaxType == "reguler" {
			regularCount++
		}
	}

	// 3. Validasi kuota kursi
	if seatSisa < regularCount {
		return nil, &ErrSeatNotEnough{
			Message: fmt.Sprintf("Kuota kursi tidak mencukupi, sisa %d kursi", seatSisa),
		}
	}

	// 4. Update schedules seat_sisa jika ada pax reguler
	if regularCount > 0 {
		_, err = tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa = seat_sisa - ? WHERE id = ?`, regularCount, req.ScheduleID)
		if err != nil {
			return nil, fmt.Errorf("booking.Create decrement seat: %w", err)
		}
	}

	// 5. Generate id_booking
	idBooking, err := r.GenerateIDBooking(ctx, tx, brandID)
	if err != nil {
		return nil, err
	}

	// 6. Insert header ke bookings
	isSeatBlocked := regularCount > 0

	const qBooking = `INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, status, is_seat_blocked, created_by)
		VALUES (?, ?, ?, 'baru', ?, ?)`

	res, err := tx.ExecContext(ctx, qBooking, idBooking, req.ScheduleID, picID, isSeatBlocked, createdBy)
	if err != nil {
		return nil, fmt.Errorf("booking.Create insert header: %w", err)
	}

	bookingID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("booking.Create LastInsertId: %w", err)
	}

	// 7. Insert detail booking_pax
	const qPax = `INSERT INTO booking_pax (booking_id, jamaah_id, pax_type, room_type, harga_pax, counts_for_seat, pax_status)
		VALUES (?, ?, ?, ?, ?, ?, 'aktif')`

	for _, p := range req.Pax {
		var hargaPax float64
		var countsForSeat bool
		var roomTypeVal *string

		var namaLengkap string
		var tanggalLahir sql.NullString
		err := tx.QueryRowContext(ctx, `SELECT nama_lengkap, DATE_FORMAT(tanggal_lahir, '%Y-%m-%d') FROM jamaah WHERE id = ?`, p.JamaahID).Scan(&namaLengkap, &tanggalLahir)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil, fmt.Errorf("jamaah tidak ditemukan: ID %d", p.JamaahID)
			}
			return nil, fmt.Errorf("booking.Create find jamaah: %w", err)
		}

		if !tanggalLahir.Valid || strings.TrimSpace(tanggalLahir.String) == "" {
			return nil, fmt.Errorf("Jamaah %s belum memiliki tanggal lahir. Lengkapi data jamaah terlebih dahulu sebelum booking dapat diproses.", namaLengkap)
		}

		if p.PaxType == "infant" {
			if berangkatTanggal.Valid {
				eligible, _ := isInfantEligible(&tanggalLahir.String, &berangkatTanggal.String)
				if !eligible {
					return nil, fmt.Errorf("Jamaah %s berusia 2 tahun atau lebih pada tanggal keberangkatan, harus didaftarkan sebagai pax reguler", namaLengkap)
				}
			}

			if !hargaInfant.Valid {
				return nil, fmt.Errorf("paket ini tidak memiliki harga infant")
			}
			hargaPax = hargaInfant.Float64
			countsForSeat = false
			roomTypeVal = nil
		} else {
			countsForSeat = true
			if p.RoomType == nil {
				return nil, fmt.Errorf("room_type untuk pax reguler wajib diisi")
			}
			rt := *p.RoomType
			roomTypeVal = &rt
			switch rt {
			case "Quad":
				hargaPax = hargaQuad
			case "Triple":
				hargaPax = hargaTriple
			case "Double":
				hargaPax = hargaDouble
			default:
				return nil, fmt.Errorf("room_type tidak valid: %s", rt)
			}
		}

		_, err = tx.ExecContext(ctx, qPax, bookingID, p.JamaahID, p.PaxType, roomTypeVal, hargaPax, countsForSeat)
		if err != nil {
			return nil, fmt.Errorf("booking.Create insert pax: %w", err)
		}
	}

	// 8. Recalculate total harga
	if err := r.recalculateTotalTx(ctx, tx, bookingID); err != nil {
		return nil, fmt.Errorf("booking.Create recalc: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.Create commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, nil)
}

// CreateDraftBooking membuat booking berstatus 'draft' tanpa row-lock, tanpa validasi kuota kursi, tanpa pengurangan seat_sisa, dan tanpa generate id_booking.
func (r *Repository) CreateDraftBooking(ctx context.Context, req *CreateDraftBookingRequest, createdBy int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.CreateDraft tx: %w", err)
	}
	defer tx.Rollback()

	// Validasi duplikasi jamaah dalam draft payload
	seenJamaah := make(map[int64]bool)
	for _, p := range req.Pax {
		if p.JamaahID == 0 {
			continue
		}
		if seenJamaah[p.JamaahID] {
			var namaLengkap string
			_ = tx.QueryRowContext(ctx, `SELECT nama_lengkap FROM jamaah WHERE id = ?`, p.JamaahID).Scan(&namaLengkap)
			if namaLengkap == "" {
				namaLengkap = fmt.Sprintf("ID %d", p.JamaahID)
			}
			return nil, fmt.Errorf("Jamaah %s didaftarkan lebih dari satu kali dalam booking ini", namaLengkap)
		}
		seenJamaah[p.JamaahID] = true
	}

	// Ambil harga paket untuk kalkulasi live pax jika tipe kamar sudah dipilih
	var hargaQuad, hargaTriple, hargaDouble float64
	var hargaInfant sql.NullFloat64
	var berangkatTanggal sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT harga_quad, harga_triple, harga_double, harga_infant, DATE_FORMAT(berangkat_tanggal, '%Y-%m-%d') 
		FROM schedules WHERE id = ?`, req.ScheduleID).
		Scan(&hargaQuad, &hargaTriple, &hargaDouble, &hargaInfant, &berangkatTanggal)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("booking.CreateDraft find schedule: %w", err)
	}

	// Validasi PIC bukan infant jika tanggal lahir & tanggal berangkat ada
	if req.PicJamaahID != nil && *req.PicJamaahID > 0 && berangkatTanggal.Valid {
		var picTanggalLahir sql.NullString
		_ = tx.QueryRowContext(ctx, `SELECT DATE_FORMAT(tanggal_lahir, '%Y-%m-%d') FROM jamaah WHERE id = ?`, *req.PicJamaahID).Scan(&picTanggalLahir)
		if picTanggalLahir.Valid {
			eligible, _ := isInfantEligible(&picTanggalLahir.String, &berangkatTanggal.String)
			if eligible {
				return nil, fmt.Errorf("PIC (Kontak Utama) tidak boleh berstatus infant")
			}
		}
	}

	const qBooking = `INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, status, is_seat_blocked, created_by)
		VALUES (NULL, ?, ?, 'draft', FALSE, ?)`

	res, err := tx.ExecContext(ctx, qBooking, req.ScheduleID, req.PicJamaahID, createdBy)
	if err != nil {
		return nil, fmt.Errorf("booking.CreateDraft insert header: %w", err)
	}

	bookingID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("booking.CreateDraft LastInsertId: %w", err)
	}

	const qPax = `INSERT INTO booking_pax (booking_id, jamaah_id, pax_type, room_type, harga_pax, counts_for_seat, pax_status)
		VALUES (?, ?, ?, ?, ?, ?, 'aktif')`

	for _, p := range req.Pax {
		if p.JamaahID == 0 {
			continue
		}

		var hargaPax float64
		var countsForSeat bool
		var roomTypeVal *string

		if p.PaxType == "infant" {
			if hargaInfant.Valid {
				hargaPax = hargaInfant.Float64
			}
			countsForSeat = false
			roomTypeVal = nil
		} else {
			countsForSeat = true
			if p.RoomType != nil && *p.RoomType != "" {
				rt := *p.RoomType
				roomTypeVal = &rt
				switch rt {
				case "Quad":
					hargaPax = hargaQuad
				case "Triple":
					hargaPax = hargaTriple
				case "Double":
					hargaPax = hargaDouble
				}
			}
		}

		paxType := p.PaxType
		if paxType == "" {
			paxType = "reguler"
		}

		_, err = tx.ExecContext(ctx, qPax, bookingID, p.JamaahID, paxType, roomTypeVal, hargaPax, countsForSeat)
		if err != nil {
			return nil, fmt.Errorf("booking.CreateDraft insert pax: %w", err)
		}
	}

	if err := r.recalculateTotalTx(ctx, tx, bookingID); err != nil {
		return nil, fmt.Errorf("booking.CreateDraft recalc: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.CreateDraft commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, nil)
}

// UpdateDraftBooking memperbarui booking yang berstatus 'draft'.
func (r *Repository) UpdateDraftBooking(ctx context.Context, bookingID int64, req *CreateDraftBookingRequest) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateDraft tx: %w", err)
	}
	defer tx.Rollback()

	var status string
	err = tx.QueryRowContext(ctx, `SELECT status FROM bookings WHERE id = ? FOR UPDATE`, bookingID).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("booking.UpdateDraft get status: %w", err)
	}
	if status != "draft" {
		return nil, fmt.Errorf("hanya booking berstatus draft yang dapat diubah")
	}

	// Validasi duplikasi jamaah dalam draft payload
	seenJamaah := make(map[int64]bool)
	for _, p := range req.Pax {
		if p.JamaahID == 0 {
			continue
		}
		if seenJamaah[p.JamaahID] {
			var namaLengkap string
			_ = tx.QueryRowContext(ctx, `SELECT nama_lengkap FROM jamaah WHERE id = ?`, p.JamaahID).Scan(&namaLengkap)
			if namaLengkap == "" {
				namaLengkap = fmt.Sprintf("ID %d", p.JamaahID)
			}
			return nil, fmt.Errorf("Jamaah %s didaftarkan lebih dari satu kali dalam booking ini", namaLengkap)
		}
		seenJamaah[p.JamaahID] = true
	}

	var hargaQuad, hargaTriple, hargaDouble float64
	var hargaInfant sql.NullFloat64
	var berangkatTanggal sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT harga_quad, harga_triple, harga_double, harga_infant, DATE_FORMAT(berangkat_tanggal, '%Y-%m-%d') 
		FROM schedules WHERE id = ?`, req.ScheduleID).
		Scan(&hargaQuad, &hargaTriple, &hargaDouble, &hargaInfant, &berangkatTanggal)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateDraft find schedule: %w", err)
	}

	// Validasi PIC bukan infant jika tanggal lahir & tanggal berangkat ada
	if req.PicJamaahID != nil && *req.PicJamaahID > 0 && berangkatTanggal.Valid {
		var picTanggalLahir sql.NullString
		_ = tx.QueryRowContext(ctx, `SELECT DATE_FORMAT(tanggal_lahir, '%Y-%m-%d') FROM jamaah WHERE id = ?`, *req.PicJamaahID).Scan(&picTanggalLahir)
		if picTanggalLahir.Valid {
			eligible, _ := isInfantEligible(&picTanggalLahir.String, &berangkatTanggal.String)
			if eligible {
				return nil, fmt.Errorf("PIC (Kontak Utama) tidak boleh berstatus infant")
			}
		}
	}

	_, err = tx.ExecContext(ctx, `UPDATE bookings SET schedule_id = ?, pic_jamaah_id = ? WHERE id = ?`,
		req.ScheduleID, req.PicJamaahID, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateDraft update header: %w", err)
	}

	_, err = tx.ExecContext(ctx, `DELETE FROM booking_pax WHERE booking_id = ?`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateDraft delete pax: %w", err)
	}

	const qPax = `INSERT INTO booking_pax (booking_id, jamaah_id, pax_type, room_type, harga_pax, counts_for_seat, pax_status)
		VALUES (?, ?, ?, ?, ?, ?, 'aktif')`

	for _, p := range req.Pax {
		if p.JamaahID == 0 {
			continue
		}

		var hargaPax float64
		var countsForSeat bool
		var roomTypeVal *string

		if p.PaxType == "infant" {
			if hargaInfant.Valid {
				hargaPax = hargaInfant.Float64
			}
			countsForSeat = false
			roomTypeVal = nil
		} else {
			countsForSeat = true
			if p.RoomType != nil && *p.RoomType != "" {
				rt := *p.RoomType
				roomTypeVal = &rt
				switch rt {
				case "Quad":
					hargaPax = hargaQuad
				case "Triple":
					hargaPax = hargaTriple
				case "Double":
					hargaPax = hargaDouble
				}
			}
		}

		paxType := p.PaxType
		if paxType == "" {
			paxType = "reguler"
		}

		_, err = tx.ExecContext(ctx, qPax, bookingID, p.JamaahID, paxType, roomTypeVal, hargaPax, countsForSeat)
		if err != nil {
			return nil, fmt.Errorf("booking.UpdateDraft insert pax: %w", err)
		}
	}

	if err := r.recalculateTotalTx(ctx, tx, bookingID); err != nil {
		return nil, fmt.Errorf("booking.UpdateDraft recalc: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.UpdateDraft commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, nil)
}

// FinalizeBooking mengubah booking draft menjadi booking resmi ('baru') dengan row-lock, kuota validasi, pengurangan seat_sisa, dan generate id_booking.
func (r *Repository) FinalizeBooking(ctx context.Context, bookingID int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.Finalize tx: %w", err)
	}
	defer tx.Rollback()

	var status string
	var scheduleID int64
	var picJamaahID sql.NullInt64
	err = tx.QueryRowContext(ctx, `
		SELECT status, schedule_id, pic_jamaah_id FROM bookings WHERE id = ? FOR UPDATE`, bookingID).
		Scan(&status, &scheduleID, &picJamaahID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("booking.Finalize get booking: %w", err)
	}

	if status != "draft" {
		return nil, fmt.Errorf("booking bukan berstatus draft (status saat ini: %s)", status)
	}

	if !picJamaahID.Valid || picJamaahID.Int64 == 0 {
		return nil, fmt.Errorf("Kontak Utama (PIC) wajib dipilih sebelum finalisasi booking")
	}

	var activeRegularPax int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM booking_pax 
		WHERE booking_id = ? AND counts_for_seat = TRUE AND pax_status = 'aktif'`, bookingID).Scan(&activeRegularPax)
	if err != nil {
		return nil, fmt.Errorf("booking.Finalize count regular pax: %w", err)
	}

	if activeRegularPax == 0 {
		var totalPax int
		_ = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM booking_pax WHERE booking_id = ? AND pax_status = 'aktif'`, bookingID).Scan(&totalPax)
		if totalPax == 0 {
			return nil, fmt.Errorf("Minimal 1 jamaah reguler wajib didaftarkan dalam booking")
		}
		return nil, fmt.Errorf("Booking harus memiliki minimal 1 pax reguler (tidak boleh hanya infant)")
	}

	// Validasi PIC bukan infant
	if picJamaahID.Valid && picJamaahID.Int64 > 0 {
		var picTanggalLahir, picBerangkatTanggal sql.NullString
		err := tx.QueryRowContext(ctx, `
			SELECT DATE_FORMAT(j.tanggal_lahir, '%Y-%m-%d'), DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d')
			FROM jamaah j, schedules s
			WHERE j.id = ? AND s.id = ?`, picJamaahID.Int64, scheduleID).Scan(&picTanggalLahir, &picBerangkatTanggal)
		if err == nil && picTanggalLahir.Valid && picBerangkatTanggal.Valid {
			eligible, _ := isInfantEligible(&picTanggalLahir.String, &picBerangkatTanggal.String)
			if eligible {
				return nil, fmt.Errorf("PIC (Kontak Utama) tidak boleh berstatus infant")
			}
		}
	}

	// Validasi kelengkapan tanggal lahir, duplikasi, & umur infant untuk semua pax aktif
	paxRows, err := tx.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, DATE_FORMAT(j.tanggal_lahir, '%Y-%m-%d'), bp.pax_type, DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d')
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		JOIN bookings b ON b.id = bp.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking.Finalize check pax data: %w", err)
	}
	defer paxRows.Close()

	seenFinalizeJamaah := make(map[int64]bool)
	for paxRows.Next() {
		var jamaahID int64
		var namaLengkap, paxType string
		var tanggalLahir, berangkatTanggal sql.NullString
		if err := paxRows.Scan(&jamaahID, &namaLengkap, &tanggalLahir, &paxType, &berangkatTanggal); err != nil {
			return nil, fmt.Errorf("booking.Finalize scan pax data: %w", err)
		}
		if seenFinalizeJamaah[jamaahID] {
			return nil, fmt.Errorf("Jamaah %s didaftarkan lebih dari satu kali dalam booking ini", namaLengkap)
		}
		seenFinalizeJamaah[jamaahID] = true

		if !tanggalLahir.Valid || strings.TrimSpace(tanggalLahir.String) == "" {
			return nil, fmt.Errorf("Jamaah %s belum memiliki tanggal lahir. Lengkapi data jamaah terlebih dahulu sebelum booking dapat diproses.", namaLengkap)
		}
		if paxType == "infant" && berangkatTanggal.Valid {
			eligible, _ := isInfantEligible(&tanggalLahir.String, &berangkatTanggal.String)
			if !eligible {
				return nil, fmt.Errorf("Jamaah %s berusia 2 tahun atau lebih pada tanggal keberangkatan, harus didaftarkan sebagai pax reguler", namaLengkap)
			}
		}
	}
	if err := paxRows.Err(); err != nil {
		return nil, fmt.Errorf("booking.Finalize paxRows err: %w", err)
	}

	// 1. Lock schedule row
	var brandID int64
	var seatSisa int
	err = tx.QueryRowContext(ctx, `
		SELECT brand_id, seat_sisa FROM schedules WHERE id = ? FOR UPDATE`, scheduleID).
		Scan(&brandID, &seatSisa)
	if err != nil {
		return nil, fmt.Errorf("booking.Finalize lock schedule: %w", err)
	}

	// 2. Cek kuota kursi
	if seatSisa < activeRegularPax {
		return nil, &ErrSeatNotEnough{
			Message: fmt.Sprintf("Kuota kursi tidak mencukupi, sisa %d kursi", seatSisa),
		}
	}

	// 3. Decrement seat_sisa
	_, err = tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa = seat_sisa - ? WHERE id = ?`, activeRegularPax, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("booking.Finalize decrement seat: %w", err)
	}

	// 4. Generate id_booking
	idBooking, err := r.GenerateIDBooking(ctx, tx, brandID)
	if err != nil {
		return nil, err
	}

	// 5. Update status ke 'baru' dan is_seat_blocked = true
	_, err = tx.ExecContext(ctx, `
		UPDATE bookings SET id_booking = ?, status = 'baru', is_seat_blocked = TRUE WHERE id = ?`,
		idBooking, bookingID)
	if err != nil {
		return nil, fmt.Errorf("booking.Finalize update booking: %w", err)
	}

	if err := r.recalculateTotalTx(ctx, tx, bookingID); err != nil {
		return nil, fmt.Errorf("booking.Finalize recalc: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.Finalize commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, nil)
}

// ─── Cancel Pax ───────────────────────────────────────────────────────────────

// CancelPax membatalkan status satu pax pada booking dan mengembalikan kuota kursi jika eligible.
func (r *Repository) CancelPax(ctx context.Context, bookingID int64, paxID int64, brandID *int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.CancelPax tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Verify booking & brand
	var scheduleID int64
	var scheduleBrandID int64
	var isSeatBlocked bool
	err = tx.QueryRowContext(ctx, `
		SELECT b.schedule_id, s.brand_id, b.is_seat_blocked
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ? FOR UPDATE`, bookingID).Scan(&scheduleID, &scheduleBrandID, &isSeatBlocked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.CancelPax verify booking: %w", err)
	}
	if brandID != nil && *brandID != scheduleBrandID {
		return nil, ErrNotFound
	}

	// 2. Lock and get pax status
	var paxStatus string
	var countsForSeat bool
	err = tx.QueryRowContext(ctx, `
		SELECT pax_status, counts_for_seat
		FROM booking_pax
		WHERE id = ? AND booking_id = ? FOR UPDATE`, paxID, bookingID).Scan(&paxStatus, &countsForSeat)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.CancelPax get pax: %w", err)
	}
	if paxStatus == "batal" {
		return nil, ErrPaxAlreadyCancelled
	}

	// 3. Update status pax
	_, err = tx.ExecContext(ctx, `UPDATE booking_pax SET pax_status = 'batal' WHERE id = ?`, paxID)
	if err != nil {
		return nil, fmt.Errorf("booking.CancelPax update status: %w", err)
	}

	// 4. Restore seat if counts_for_seat is true and seat was blocked
	if countsForSeat && isSeatBlocked {
		_, err = tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa = LEAST(seat_total, seat_sisa + 1) WHERE id = ?`, scheduleID)
		if err != nil {
			return nil, fmt.Errorf("booking.CancelPax restore seat: %w", err)
		}
	}

	// 5. Cek active pax
	var activePaxCount int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM booking_pax WHERE booking_id = ? AND pax_status = 'aktif'`, bookingID).Scan(&activePaxCount)
	if err != nil {
		return nil, fmt.Errorf("booking.CancelPax check active count: %w", err)
	}
	if activePaxCount == 0 {
		_, err = tx.ExecContext(ctx, `UPDATE bookings SET status = 'batal', is_seat_blocked = FALSE WHERE id = ?`, bookingID)
		if err != nil {
			return nil, fmt.Errorf("booking.CancelPax cancel booking: %w", err)
		}
	}

	// 6. Commit (tanpa recalculate total harga)
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.CancelPax commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, brandID)
}

// ─── Update Pax Room Type ─────────────────────────────────────────────────────

// UpdatePaxRoomType mengubah tipe kamar satu pax dan memperbarui total_harga booking secara otomatis.
func (r *Repository) UpdatePaxRoomType(ctx context.Context, bookingID int64, paxID int64, newRoomType string, brandID *int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdatePaxRoomType tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Verify booking & brand
	var scheduleID int64
	var scheduleBrandID int64
	err = tx.QueryRowContext(ctx, `
		SELECT b.schedule_id, s.brand_id
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ? FOR UPDATE`, bookingID).Scan(&scheduleID, &scheduleBrandID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.UpdatePaxRoomType verify booking: %w", err)
	}
	if brandID != nil && *brandID != scheduleBrandID {
		return nil, ErrNotFound
	}

	// 2. Lock and verify pax
	var paxType, paxStatus string
	err = tx.QueryRowContext(ctx, `
		SELECT pax_type, pax_status
		FROM booking_pax
		WHERE id = ? AND booking_id = ? FOR UPDATE`, paxID, bookingID).Scan(&paxType, &paxStatus)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.UpdatePaxRoomType get pax: %w", err)
	}
	if paxType == "infant" {
		return nil, ErrCannotChangeInfantRoom
	}
	if paxStatus == "batal" {
		return nil, ErrCannotChangeCancelledPaxRoom
	}

	// 3. Lookup new price from schedule
	var col string
	switch newRoomType {
	case "Quad":
		col = "harga_quad"
	case "Triple":
		col = "harga_triple"
	case "Double":
		col = "harga_double"
	default:
		return nil, fmt.Errorf("room_type tidak valid: %s", newRoomType)
	}

	var newHarga float64
	qPrice := fmt.Sprintf("SELECT %s FROM schedules WHERE id = ?", col)
	if err := tx.QueryRowContext(ctx, qPrice, scheduleID).Scan(&newHarga); err != nil {
		return nil, fmt.Errorf("booking.UpdatePaxRoomType lookup price: %w", err)
	}

	// 4. Update booking_pax
	_, err = tx.ExecContext(ctx, `UPDATE booking_pax SET room_type = ?, harga_pax = ? WHERE id = ?`, newRoomType, newHarga, paxID)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdatePaxRoomType update pax: %w", err)
	}

	// 5. Recalculate total_harga
	if err := r.recalculateTotalTx(ctx, tx, bookingID); err != nil {
		return nil, fmt.Errorf("booking.UpdatePaxRoomType recalc: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.UpdatePaxRoomType commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, brandID)
}

// ─── UpdateBookingStatus ──────────────────────────────────────────────────────

// UpdateBookingStatus mengubah status booking + logika seat_sisa.
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
		`SELECT status, schedule_id, is_seat_blocked FROM bookings WHERE id=? FOR UPDATE`, bookingID,
	).Scan(&oldStatus, &scheduleID, &isSeatBlocked)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateStatus get old: %w", err)
	}

	// 2. Hitung jumlah pax reguler aktif
	var activeRegularPax int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM booking_pax 
		WHERE booking_id = ? AND counts_for_seat = TRUE AND pax_status = 'aktif'`, bookingID).Scan(&activeRegularPax)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateStatus count regular pax: %w", err)
	}

	// 3. Lock schedule row untuk cek/ubah seat_sisa
	var seatSisa int
	err = tx.QueryRowContext(ctx,
		`SELECT seat_sisa FROM schedules WHERE id=? FOR UPDATE`, scheduleID,
	).Scan(&seatSisa)
	if err != nil {
		return nil, fmt.Errorf("booking.UpdateStatus lock schedule: %w", err)
	}

	// 4. Logic seat_sisa
	newIsLocked := lockedStatuses[newStatus]

	if newStatus == "batal" && isSeatBlocked {
		if activeRegularPax > 0 {
			_, err = tx.ExecContext(ctx,
				`UPDATE schedules SET seat_sisa = LEAST(seat_total, seat_sisa + ?) WHERE id=?`, activeRegularPax, scheduleID)
			if err != nil {
				return nil, fmt.Errorf("booking.UpdateStatus restore seat: %w", err)
			}
		}
		_, err = tx.ExecContext(ctx, `UPDATE bookings SET is_seat_blocked=FALSE,seat_hold_expires_at=NULL,seat_hold_key=NULL WHERE id=?`, bookingID)
		if err != nil {
			return nil, fmt.Errorf("booking.UpdateStatus clear seat block: %w", err)
		}
		_, err = tx.ExecContext(ctx, `UPDATE booking_pax SET pax_status='batal' WHERE booking_id=?`, bookingID)
		if err != nil {
			return nil, fmt.Errorf("booking.UpdateStatus cancel pax: %w", err)
		}
	} else if newIsLocked && oldStatus == "baru" && !isSeatBlocked {
		if seatSisa < activeRegularPax {
			return nil, ErrSeatHabis
		}
		if activeRegularPax > 0 {
			_, err = tx.ExecContext(ctx,
				`UPDATE schedules SET seat_sisa = seat_sisa - ? WHERE id=?`, activeRegularPax, scheduleID)
			if err != nil {
				return nil, fmt.Errorf("booking.UpdateStatus decrement seat: %w", err)
			}
			_, err = tx.ExecContext(ctx, `UPDATE bookings SET is_seat_blocked=TRUE WHERE id=?`, bookingID)
			if err != nil {
				return nil, fmt.Errorf("booking.UpdateStatus mark seat block: %w", err)
			}
		}
	}
	// DP/Lunas mengubah hold sementara menjadi reservasi tanpa kedaluwarsa.
	if newIsLocked {
		if _, err = tx.ExecContext(ctx, `UPDATE bookings SET seat_hold_expires_at=NULL,seat_hold_key=NULL WHERE id=?`, bookingID); err != nil {
			return nil, fmt.Errorf("booking.UpdateStatus clear seat expiry: %w", err)
		}
	}

	// 5. Update status booking
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
	var activeRegularPax int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM booking_pax 
		WHERE booking_id = ? AND counts_for_seat = TRUE AND pax_status = 'aktif'`, bookingID).Scan(&activeRegularPax)
	if err != nil {
		return nil, fmt.Errorf("booking.CancelSeatBlock count regular pax: %w", err)
	}

	if activeRegularPax > 0 {
		if _, err = tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa=LEAST(seat_total, seat_sisa + ?) WHERE id=?`, activeRegularPax, scheduleID); err != nil {
			return nil, fmt.Errorf("booking.CancelSeatBlock restore seat: %w", err)
		}
	}
	if _, err = tx.ExecContext(ctx, `UPDATE bookings SET is_seat_blocked=FALSE,seat_hold_expires_at=NULL,seat_hold_key=NULL WHERE id=?`, bookingID); err != nil {
		return nil, fmt.Errorf("booking.CancelSeatBlock update: %w", err)
	}
	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.CancelSeatBlock commit: %w", err)
	}
	return r.GetByID(ctx, bookingID, nil)
}

// BlockSeat menahan seluruh seat_count booking baru dengan idempotency key.
func (r *Repository) BlockSeat(ctx context.Context, bookingID int64, expiresAt time.Time, idempotencyKey string) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.BlockSeat tx: %w", err)
	}
	defer tx.Rollback()

	var scheduleID int64
	var seatCount int
	var status string
	var isBlocked bool
	var existingKey sql.NullString
	err = tx.QueryRowContext(ctx,
		`SELECT schedule_id,seat_count,status,is_seat_blocked,seat_hold_key FROM bookings WHERE id=? FOR UPDATE`, bookingID,
	).Scan(&scheduleID, &seatCount, &status, &isBlocked, &existingKey)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("booking.BlockSeat find: %w", err)
	}
	if status != "baru" {
		return nil, ErrInvalidStatus
	}
	if isBlocked {
		if !existingKey.Valid || existingKey.String != idempotencyKey {
			return nil, ErrSeatSudahDiblokir
		}
		if err := tx.Commit(); err != nil {
			return nil, err
		}
		return r.GetByID(ctx, bookingID, nil)
	}

	var seatRemaining int
	if err := tx.QueryRowContext(ctx, `SELECT seat_sisa FROM schedules WHERE id=? FOR UPDATE`, scheduleID).Scan(&seatRemaining); err != nil {
		return nil, fmt.Errorf("booking.BlockSeat lock schedule: %w", err)
	}
	if seatRemaining < seatCount {
		return nil, ErrSeatHabis
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa=seat_sisa-? WHERE id=?`, seatCount, scheduleID); err != nil {
		return nil, fmt.Errorf("booking.BlockSeat reserve: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE bookings SET is_seat_blocked=TRUE,seat_hold_expires_at=?,seat_hold_key=? WHERE id=?`,
		expiresAt.UTC(), idempotencyKey, bookingID,
	); err != nil {
		return nil, fmt.Errorf("booking.BlockSeat update: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.BlockSeat commit: %w", err)
	}
	return r.GetByID(ctx, bookingID, nil)
}

// ─── Addons & Diskon ──────────────────────────────────────────────────────────

func (r *Repository) recalculateTotalTx(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	const q = `
		UPDATE bookings
		SET total_harga = GREATEST(0, (
			SELECT COALESCE(SUM(harga_pax), 0) FROM booking_pax WHERE booking_id = ?
		) + (
			SELECT COALESCE(SUM(nominal), 0) FROM booking_addons WHERE booking_id = ?
		) - diskon)
		WHERE id = ?`
	if _, err := tx.ExecContext(ctx, q, bookingID, bookingID, bookingID); err != nil {
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

// ─── Existence checks ─────────────────────────────────────────────────────────

// ScheduleExistsForBrand memeriksa apakah schedule_id ada.
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

// JamaahExistsForBrand memeriksa apakah jamaah_id ada.
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

// UpdateProgress melakukan update terhadap kolom progress header (hotel, land_arrangement).
func (r *Repository) UpdateProgress(ctx context.Context, bookingID int64, brandID *int64, updates map[string]bool) (*Booking, error) {
	if len(updates) == 0 {
		return r.GetByID(ctx, bookingID, brandID)
	}

	// Verify existence and brand
	if _, err := r.GetByID(ctx, bookingID, brandID); err != nil {
		return nil, err
	}

	var headerSetClauses []string
	var headerArgs []interface{}

	for key, val := range updates {
		colName, ok := AllowedHeaderProgressFields[key]
		if !ok {
			return nil, fmt.Errorf("item progress '%s' tidak valid atau gunakan endpoint progress per-pax untuk item ini", key)
		}
		headerSetClauses = append(headerSetClauses, fmt.Sprintf("%s = ?", colName))
		headerArgs = append(headerArgs, val)
	}

	if len(headerSetClauses) > 0 {
		headerArgs = append(headerArgs, bookingID)
		qHeader := fmt.Sprintf("UPDATE bookings SET %s WHERE id = ?", strings.Join(headerSetClauses, ", "))
		if _, err := r.db.ExecContext(ctx, qHeader, headerArgs...); err != nil {
			return nil, fmt.Errorf("booking.UpdateProgress: %w", err)
		}
	}

	return r.GetByID(ctx, bookingID, brandID)
}

// UpdatePaxProgress melakukan update terhadap kolom progress individual pax (visa, siskopatuh, manasik, vaksin_meningitis).
func (r *Repository) UpdatePaxProgress(ctx context.Context, bookingID int64, paxID int64, brandID *int64, updates map[string]bool) (*Booking, error) {
	if len(updates) == 0 {
		return r.GetByID(ctx, bookingID, brandID)
	}

	// Verify booking existence & brand
	if _, err := r.GetByID(ctx, bookingID, brandID); err != nil {
		return nil, err
	}

	// Verify pax existence & status
	var paxType, paxStatus string
	err := r.db.QueryRowContext(ctx, `SELECT pax_type, pax_status FROM booking_pax WHERE id = ? AND booking_id = ?`, paxID, bookingID).Scan(&paxType, &paxStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("booking.UpdatePaxProgress verify pax: %w", err)
	}

	if paxStatus != "aktif" {
		return nil, ErrPaxBatal
	}

	var setClauses []string
	var args []interface{}

	for key, val := range updates {
		colName, ok := AllowedPaxProgressFields[key]
		if !ok {
			return nil, fmt.Errorf("item progress pax '%s' tidak valid", key)
		}
		if key == "manasik" && paxType == "infant" {
			return nil, ErrManasikInfant
		}
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", colName))
		args = append(args, val)
	}

	if len(setClauses) > 0 {
		args = append(args, paxID, bookingID)
		q := fmt.Sprintf("UPDATE booking_pax SET %s WHERE id = ? AND booking_id = ?", strings.Join(setClauses, ", "))
		if _, err := r.db.ExecContext(ctx, q, args...); err != nil {
			return nil, fmt.Errorf("booking.UpdatePaxProgress update: %w", err)
		}
	}

	return r.GetByID(ctx, bookingID, brandID)
}

// checkPasporUploaded memeriksa apakah jamaah memiliki dokumen paspor yang sudah diupload.
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
	headerLengkap := b.ProgressTiket && b.ProgressHotel && b.ProgressLandArrangement
	activePaxCount := 0
	allPaxLengkap := true

	for _, p := range b.Pax {
		if p.PaxStatus != "aktif" {
			continue
		}
		activePaxCount++
		manasikReq := true
		if p.PaxType != "infant" {
			manasikReq = p.ProgressManasik
		}
		paxLengkap := p.ProgressPaspor && p.ProgressVisa && p.ProgressSiskopatuh && p.ProgressVaksinMeningitis && manasikReq
		if !paxLengkap {
			allPaxLengkap = false
		}
	}

	b.SiapBerangkat = (activePaxCount > 0) && headerLengkap && allPaxLengkap
}

// ─── Internal scan helper ─────────────────────────────────────────────────────

func scanBookingRow(rows *sql.Rows) (*Booking, error) {
	var b Booking
	var idBooking sql.NullString
	var picJamaahID sql.NullInt64
	var namaJamaah sql.NullString
	var primaryJamaahID sql.NullInt64
	var totalHarga sql.NullFloat64
	var hargaDasar sql.NullFloat64
	var diskonKeterangan sql.NullString
	var createdBy sql.NullInt64
	var berangkatTanggal sql.NullString
	var perlengkapanStatus string
	var perlengkapanTanggal sql.NullString
	var perlengkapanDiberikanOleh sql.NullInt64
	var isTicketConfirmed bool
	var seatHoldExpiresAt sql.NullTime
	var roomType sql.NullString
	var perlengkapanJumlahPax sql.NullInt64

	err := rows.Scan(
		&b.ID, &idBooking, &b.ScheduleID, &b.BrandID, &b.JadwalNama, &berangkatTanggal,
		&picJamaahID, &namaJamaah,
		&roomType, &hargaDasar,
		&b.SeatCount, &b.Status, &b.IsSeatBlocked, &seatHoldExpiresAt,
		&totalHarga, &b.Diskon, &diskonKeterangan,
		&b.ProgressVisa, &b.ProgressHotel, &b.ProgressLandArrangement,
		&b.ProgressManasik, &b.ProgressSiskopatuh, &b.ProgressVaksinMeningitis,
		&perlengkapanStatus, &perlengkapanTanggal, &perlengkapanDiberikanOleh,
		&perlengkapanJumlahPax,
		&createdBy, &b.CreatedAt,
		&isTicketConfirmed,
		&primaryJamaahID,
		&b.PaxCount,
		&b.RegularPaxCount,
		&b.InfantPaxCount,
	)
	if err != nil {
		return nil, fmt.Errorf("booking.scanRow: %w", err)
	}
	b.ProgressTiket = isTicketConfirmed
	if roomType.Valid && strings.TrimSpace(roomType.String) != "" {
		b.RoomType = &roomType.String
	}
	if idBooking.Valid && strings.TrimSpace(idBooking.String) != "" {
		b.IDBooking = &idBooking.String
	}
	if picJamaahID.Valid && picJamaahID.Int64 > 0 {
		b.PicJamaahID = &picJamaahID.Int64
	}
	if primaryJamaahID.Valid && primaryJamaahID.Int64 > 0 {
		b.JamaahID = &primaryJamaahID.Int64
	}
	if namaJamaah.Valid && strings.TrimSpace(namaJamaah.String) != "" {
		b.NamaJamaah = &namaJamaah.String
	}
	if berangkatTanggal.Valid {
		b.BerangkatTanggal = &berangkatTanggal.String
	}
	if hargaDasar.Valid && hargaDasar.Float64 > 0 {
		b.HargaDasar = &hargaDasar.Float64
	}
	if totalHarga.Valid {
		b.TotalHarga = &totalHarga.Float64
	}
	if seatHoldExpiresAt.Valid {
		value := seatHoldExpiresAt.Time.UTC().Format(time.RFC3339)
		b.SeatHoldExpiresAt = &value
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
	if perlengkapanJumlahPax.Valid {
		v := int(perlengkapanJumlahPax.Int64)
		b.PerlengkapanJumlahPax = &v
	}
	computeSiapBerangkat(&b)
	b.Addons = make([]BookingAddon, 0)
	b.Pax = make([]BookingPax, 0)
	return &b, nil
}

// ─── Perlengkapan Distribusi ──────────────────────────────────────────────────

// MarkPerlengkapanDiberikan mendistribusikan perlengkapan sesuai template set brand dikalikan pax non-infant aktif.
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

	// 2. Hitung jumlah pax reguler aktif (infant tidak dapat jatah perlengkapan)
	var jumlahPaxEligible int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM booking_pax 
		WHERE booking_id = ? AND pax_status = 'aktif' AND pax_type = 'reguler'`, bookingID).Scan(&jumlahPaxEligible)
	if err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan count eligible pax: %w", err)
	}
	if jumlahPaxEligible == 0 {
		return nil, errors.New("tidak ada pax reguler aktif untuk didistribusikan perlengkapan")
	}

	// 3. Ambil template set global & stok untuk brand tersebut dengan row lock FOR UPDATE
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

	// 4. Cek apakah ada stok yang kurang untuk kebutuhan (item.Qty * jumlahPaxEligible)
	var kurangItems []string
	for _, item := range setItems {
		butuh := item.Qty * jumlahPaxEligible
		if item.StokTersedia < butuh {
			kurangItems = append(kurangItems, fmt.Sprintf("%s (tersedia %d, butuh %d)", item.Nama, item.StokTersedia, butuh))
		}
	}
	if len(kurangItems) > 0 {
		return nil, &ErrStokKurang{
			Message: "stok tidak cukup untuk: " + strings.Join(kurangItems, ", "),
		}
	}

	// 5. Potong stok untuk setiap item di set pada tabel perlengkapan_stok
	for _, item := range setItems {
		butuh := item.Qty * jumlahPaxEligible
		_, err := tx.ExecContext(ctx, "UPDATE perlengkapan_stok SET stok_tersedia = stok_tersedia - ? WHERE brand_id = ? AND perlengkapan_item_id = ?", butuh, scheduleBrandID, item.ItemID)
		if err != nil {
			return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan potong stok: %w", err)
		}
	}

	// 6. Update status perlengkapan pada booking beserta perlengkapan_jumlah_pax
	qUpdateBooking := `
		UPDATE bookings
		SET perlengkapan_status = 'sudah_diberikan',
		    perlengkapan_tanggal = CURDATE(),
		    perlengkapan_diberikan_oleh = ?,
		    perlengkapan_jumlah_pax = ?
		WHERE id = ?
	`
	if _, err := tx.ExecContext(ctx, qUpdateBooking, adminID, jumlahPaxEligible, bookingID); err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan update booking: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("booking.MarkPerlengkapanDiberikan commit: %w", err)
	}

	return r.GetByID(ctx, bookingID, brandID)
}

// BatalkanPerlengkapan membatalkan status perlengkapan dan mengembalikan stok item ke database sesuai jumlah pax tersimpan.
func (r *Repository) BatalkanPerlengkapan(ctx context.Context, bookingID int64, brandID *int64) (*Booking, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("booking.BatalkanPerlengkapan tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Ambil booking existing, brand_id, dan perlengkapan_jumlah_pax
	var (
		perlengkapanStatus    string
		scheduleBrandID       int64
		perlengkapanJumlahPax sql.NullInt64
	)
	qBooking := `
		SELECT b.perlengkapan_status, s.brand_id, b.perlengkapan_jumlah_pax
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id = ?
	`
	err = tx.QueryRowContext(ctx, qBooking, bookingID).Scan(&perlengkapanStatus, &scheduleBrandID, &perlengkapanJumlahPax)
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

	// Ambil jumlah pax tersimpan (fallback 1 untuk data legacy)
	jumlahPaxTersimpan := 1
	if perlengkapanJumlahPax.Valid && perlengkapanJumlahPax.Int64 > 0 {
		jumlahPaxTersimpan = int(perlengkapanJumlahPax.Int64)
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

	// 3. Kembalikan stok item ke tabel perlengkapan_stok (item.Qty * jumlahPaxTersimpan)
	for _, item := range setItems {
		kembalikan := item.Qty * jumlahPaxTersimpan
		_, err := tx.ExecContext(ctx, "UPDATE perlengkapan_stok SET stok_tersedia = stok_tersedia + ? WHERE brand_id = ? AND perlengkapan_item_id = ?", kembalikan, scheduleBrandID, item.ItemID)
		if err != nil {
			return nil, fmt.Errorf("booking.BatalkanPerlengkapan kembalikan stok: %w", err)
		}
	}

	// 4. Reset booking status perlengkapan dan perlengkapan_jumlah_pax
	qUpdateBooking := `
		UPDATE bookings
		SET perlengkapan_status = 'belum_diberikan',
		    perlengkapan_tanggal = NULL,
		    perlengkapan_diberikan_oleh = NULL,
		    perlengkapan_jumlah_pax = NULL
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
