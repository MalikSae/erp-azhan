package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNotFound dikembalikan saat schedule tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// selectFull adalah query SELECT lengkap dengan LEFT JOIN ke airlines dan hotels (2x alias).
// Tambahkan WHERE / ORDER BY setelah konstanta ini sesuai kebutuhan.
const selectFull = `
	SELECT
		s.id,
		s.brand_id,
		s.jadwal_nama,
		s.status,
		s.is_promo,
		s.is_ticket_confirmed,
		s.is_direct_flight,
		s.seat_total,
		s.seat_sisa,
		s.maskapai_id,
		COALESCE(a.name, '') AS maskapai_name,
		a.logo_url AS maskapai_logo,
		DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d') AS berangkat_tanggal,
		COALESCE(TIME_FORMAT(s.berangkat_jam, '%H:%i'), '') AS berangkat_jam,
		COALESCE(s.berangkat_kode_penerbangan, '') AS berangkat_kode_penerbangan,
		COALESCE(s.berangkat_bandara_asal, '') AS berangkat_bandara_asal,
		COALESCE(s.berangkat_bandara_tujuan, '') AS berangkat_bandara_tujuan,
		DATE_FORMAT(s.pulang_tanggal, '%Y-%m-%d') AS pulang_tanggal,
		COALESCE(s.pulang_jam, '') AS pulang_jam,
		COALESCE(s.pulang_kode_penerbangan, '') AS pulang_kode_penerbangan,
		COALESCE(s.pulang_bandara_asal, '') AS pulang_bandara_asal,
		COALESCE(s.pulang_bandara_tujuan, '') AS pulang_bandara_tujuan,
		COALESCE(s.transit_bandara, '') AS transit_bandara,
		s.hotel_mekkah_id,
		COALESCE(hm.name, '') AS hm_name,
		COALESCE(hm.star_rating, 0) AS hm_star,
		hm.distance_m AS hm_dist,
		s.hotel_madinah_id,
		COALESCE(hmd.name, '') AS hmd_name,
		COALESCE(hmd.star_rating, 0) AS hmd_star,
		hmd.distance_m AS hmd_dist,
		s.harga_quad,
		s.harga_triple,
		s.harga_double,
		s.harga_coret,
		s.itinerary_id,
		(
			SELECT COALESCE(JSON_ARRAYAGG(JSON_OBJECT('id', ao.id, 'name', ao.name)), '[]')
			FROM schedule_add_ons sao
			JOIN add_ons ao ON ao.id = sao.add_on_id
			WHERE sao.schedule_id = s.id
		) AS add_ons,
		COALESCE(s.include_items, '[]') AS include_items,
		COALESCE(s.exclude_items, '[]') AS exclude_items,
		COALESCE(s.brosur_url, '') AS brosur_url,
		COALESCE(s.brosur_thumb_url, '') AS brosur_thumb_url,
		s.created_at,
		s.updated_at
	FROM schedules s
	LEFT JOIN airlines a ON a.id = s.maskapai_id
	LEFT JOIN hotels hm ON hm.id = s.hotel_mekkah_id
	LEFT JOIN hotels hmd ON hmd.id = s.hotel_madinah_id`

// Repository mengelola semua query ke tabel schedules.
type Repository struct {
	db *sql.DB
}

// NewRepository membuat instance Repository dengan *sql.DB yang sudah ada.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── List ─────────────────────────────────────────────────────────────────────

// List mengambil semua schedule (semua status), ringkas, diurutkan created_at DESC.
func (r *Repository) List(ctx context.Context, brandID *int64) ([]ScheduleListItem, error) {
	q := `
		SELECT s.id, s.brand_id, s.jadwal_nama, s.status, s.is_promo, s.is_ticket_confirmed, s.is_direct_flight,
			s.maskapai_id, COALESCE(a.name, ''), a.logo_url,
			DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d') AS berangkat_tanggal,
			s.seat_total, s.seat_sisa, s.harga_quad, s.harga_triple, s.harga_double,
			(
				SELECT COALESCE(JSON_ARRAYAGG(JSON_OBJECT('id', ao.id, 'name', ao.name)), '[]')
				FROM schedule_add_ons sao
				JOIN add_ons ao ON ao.id = sao.add_on_id
				WHERE sao.schedule_id = s.id
			) AS add_ons
		FROM schedules s
		LEFT JOIN airlines a ON a.id = s.maskapai_id
		WHERE 1=1`

	var args []interface{}
	if brandID != nil {
		q += " AND s.brand_id = ?"
		args = append(args, *brandID)
	}

	q += " ORDER BY s.created_at DESC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("schedule.List: %w", err)
	}
	defer rows.Close()

	items := make([]ScheduleListItem, 0)
	for rows.Next() {
		var item ScheduleListItem
		var addOnsJSON []byte
		var maskapaiID *int64
		var maskapaiName string
		var maskapaiLogo *string

		if err := rows.Scan(&item.ID, &item.BrandID, &item.JadwalNama, &item.Status, &item.IsPromo, &item.IsTicketConfirmed, &item.IsDirectFlight,
			&maskapaiID, &maskapaiName, &maskapaiLogo,
			&item.BerangkatTanggal, &item.SeatTotal, &item.SeatSisa,
			&item.HargaQuad, &item.HargaTriple, &item.HargaDouble, &addOnsJSON); err != nil {
			return nil, fmt.Errorf("schedule.List scan: %w", err)
		}

		if maskapaiID != nil && *maskapaiID > 0 {
			item.Maskapai = &MaskapaiRef{
				ID:      *maskapaiID,
				Name:    maskapaiName,
				LogoURL: maskapaiLogo,
			}
		}

		item.AddOns = []AddOnRef{}
		if len(addOnsJSON) > 0 {
			_ = json.Unmarshal(addOnsJSON, &item.AddOns)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ListPublic mengambil schedule yang published saja untuk brand tertentu, diurutkan berangkat_tanggal ASC.
func (r *Repository) ListPublic(ctx context.Context, brandID int64) ([]*PublicSchedule, error) {
	q := selectFull + " WHERE s.status = 'published' AND s.brand_id = ? ORDER BY s.berangkat_tanggal ASC"
	rows, err := r.db.QueryContext(ctx, q, brandID)
	if err != nil {
		return nil, fmt.Errorf("schedule.ListPublic: %w", err)
	}
	defer rows.Close()

	items := make([]*PublicSchedule, 0)
	for rows.Next() {
		s, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, s.ToPublic())
	}
	return items, rows.Err()
}

// ─── Get Detail ───────────────────────────────────────────────────────────────

// GetByID mengambil schedule lengkap beserta ref hotel/airline.
func (r *Repository) GetByID(ctx context.Context, id int64, brandID *int64) (*Schedule, error) {
	return r.fetchFull(ctx, id, brandID)
}

// ─── Create ───────────────────────────────────────────────────────────────────

// Create menyisipkan schedule baru dengan status="draft".
func (r *Repository) Create(ctx context.Context, inp ScheduleInput) (*Schedule, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("schedule.Create tx: %w", err)
	}
	defer tx.Rollback()

	includeJSON, _ := json.Marshal(inp.IncludeItems)
	excludeJSON, _ := json.Marshal(inp.ExcludeItems)

	const q = `
		INSERT INTO schedules (
			brand_id, jadwal_nama, status, is_promo, is_ticket_confirmed, is_direct_flight, seat_total, seat_sisa,
			maskapai_id, berangkat_tanggal, berangkat_jam, berangkat_kode_penerbangan, berangkat_bandara_asal, berangkat_bandara_tujuan,
			pulang_tanggal, pulang_jam, pulang_kode_penerbangan, pulang_bandara_asal, pulang_bandara_tujuan, transit_bandara,
			hotel_mekkah_id, hotel_madinah_id,
			harga_quad, harga_triple, harga_double, harga_coret,
			itinerary_id, include_items, exclude_items,
			brosur_url, brosur_thumb_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := tx.ExecContext(ctx, q,
		inp.BrandID, inp.JadwalNama, inp.Status, inp.IsPromo, inp.IsTicketConfirmed, inp.IsDirectFlight, inp.SeatTotal, inp.SeatSisa,
		nullInt64(inp.MaskapaiID),
		inp.BerangkatTanggal, nullString(inp.BerangkatJam), nullString(inp.BerangkatKodePenerbangan),
		nullString(inp.BerangkatBandaraAsal), nullString(inp.BerangkatBandaraTujuan),
		inp.PulangTanggal, nullString(inp.PulangJam), nullString(inp.PulangKodePenerbangan),
		nullString(inp.PulangBandaraAsal), nullString(inp.PulangBandaraTujuan), nullString(inp.TransitBandara),
		nullInt64(inp.HotelMekkahID), nullInt64(inp.HotelMadinahID),
		inp.HargaQuad, inp.HargaTriple, inp.HargaDouble, inp.HargaCoret,
		inp.ItineraryID, includeJSON, excludeJSON,
		nullString(inp.BrosurURL), nullString(inp.BrosurThumbURL),
	)
	if err != nil {
		return nil, fmt.Errorf("schedule.Create: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("schedule.Create LastInsertId: %w", err)
	}

	if len(inp.AddOnIDs) > 0 {
		for _, addonID := range inp.AddOnIDs {
			_, err := tx.ExecContext(ctx, `INSERT INTO schedule_add_ons (schedule_id, add_on_id) VALUES (?, ?)`, id, addonID)
			if err != nil {
				return nil, fmt.Errorf("schedule.Create addon: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("schedule.Create commit: %w", err)
	}

	return r.fetchFull(ctx, id, nil) // return created without brand check since we just created it
}

// ─── Update ───────────────────────────────────────────────────────────────────

// Update memperbarui semua field schedule kecuali status.
func (r *Repository) Update(ctx context.Context, id int64, inp ScheduleInput, brandID *int64, finalBrandID int64) (*Schedule, error) {
	if err := r.exists(ctx, id, brandID); err != nil {
		return nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("schedule.Update tx: %w", err)
	}
	defer tx.Rollback()

	includeJSON, _ := json.Marshal(inp.IncludeItems)
	excludeJSON, _ := json.Marshal(inp.ExcludeItems)

	const q = `
		UPDATE schedules SET
			brand_id=?, jadwal_nama=?, status=COALESCE(NULLIF(?, ''), status), is_promo=?, is_ticket_confirmed=?, is_direct_flight=?, seat_total=?, seat_sisa=?,
			maskapai_id=?, berangkat_tanggal=?, berangkat_jam=?, berangkat_kode_penerbangan=?, berangkat_bandara_asal=?, berangkat_bandara_tujuan=?,
			pulang_tanggal=?, pulang_jam=?, pulang_kode_penerbangan=?, pulang_bandara_asal=?, pulang_bandara_tujuan=?, transit_bandara=?,
			hotel_mekkah_id=?, hotel_madinah_id=?,
			harga_quad=?, harga_triple=?, harga_double=?, harga_coret=?,
			itinerary_id=?, include_items=?, exclude_items=?,
			brosur_url=?, brosur_thumb_url=?
		WHERE id=?`

	_, err = tx.ExecContext(ctx, q,
		finalBrandID, inp.JadwalNama, inp.Status, inp.IsPromo, inp.IsTicketConfirmed, inp.IsDirectFlight, inp.SeatTotal, inp.SeatSisa,
		nullInt64(inp.MaskapaiID),
		inp.BerangkatTanggal, nullString(inp.BerangkatJam), nullString(inp.BerangkatKodePenerbangan),
		nullString(inp.BerangkatBandaraAsal), nullString(inp.BerangkatBandaraTujuan),
		inp.PulangTanggal, nullString(inp.PulangJam), nullString(inp.PulangKodePenerbangan),
		nullString(inp.PulangBandaraAsal), nullString(inp.PulangBandaraTujuan), nullString(inp.TransitBandara),
		nullInt64(inp.HotelMekkahID), nullInt64(inp.HotelMadinahID),
		inp.HargaQuad, inp.HargaTriple, inp.HargaDouble, inp.HargaCoret,
		inp.ItineraryID, includeJSON, excludeJSON,
		nullString(inp.BrosurURL), nullString(inp.BrosurThumbURL),
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("schedule.Update: %w", err)
	}

	// Update schedule_add_ons (replace-all)
	if _, err := tx.ExecContext(ctx, `DELETE FROM schedule_add_ons WHERE schedule_id=?`, id); err != nil {
		return nil, fmt.Errorf("schedule.Update delete addons: %w", err)
	}
	if len(inp.AddOnIDs) > 0 {
		for _, addonID := range inp.AddOnIDs {
			if _, err := tx.ExecContext(ctx, `INSERT INTO schedule_add_ons (schedule_id, add_on_id) VALUES (?, ?)`, id, addonID); err != nil {
				return nil, fmt.Errorf("schedule.Update insert addon: %w", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("schedule.Update commit: %w", err)
	}

	return r.fetchFull(ctx, id, brandID)
}

// ─── UpdateStatus ─────────────────────────────────────────────────────────────

// UpdateStatus mengubah hanya kolom status.
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status string, brandID *int64) (int64, string, error) {
	if err := r.exists(ctx, id, brandID); err != nil {
		return 0, "", err
	}
	_, err := r.db.ExecContext(ctx, `UPDATE schedules SET status=? WHERE id=?`, status, id)
	if err != nil {
		return 0, "", fmt.Errorf("schedule.UpdateStatus: %w", err)
	}
	return id, status, nil
}

// ─── UpdateSeat ───────────────────────────────────────────────────────────────

// GetSeatTotal mengambil seat_total dari schedule berdasarkan id.
// Digunakan handler untuk validasi range seat_sisa sebelum update.
func (r *Repository) GetSeatTotal(ctx context.Context, id int64, brandID *int64) (int, error) {
	var seatTotal int
	q := `SELECT seat_total FROM schedules WHERE id=?`
	var args []interface{}
	args = append(args, id)
	if brandID != nil {
		q += " AND brand_id=?"
		args = append(args, *brandID)
	}
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&seatTotal)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("schedule.GetSeatTotal: %w", err)
	}
	return seatTotal, nil
}

// UpdateSeat mengupdate kolom seat_sisa saja (tanpa validasi, sudah dilakukan handler).
func (r *Repository) UpdateSeat(ctx context.Context, id int64, seatSisa int) error {
	_, err := r.db.ExecContext(ctx, `UPDATE schedules SET seat_sisa=? WHERE id=?`, seatSisa, id)
	if err != nil {
		return fmt.Errorf("schedule.UpdateSeat: %w", err)
	}
	return nil
}

// ─── Delete ───────────────────────────────────────────────────────────────────

// Delete menghapus schedule berdasarkan id.
func (r *Repository) Delete(ctx context.Context, id int64, brandID *int64) error {
	if err := r.exists(ctx, id, brandID); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `DELETE FROM schedules WHERE id=?`, id)
	return err
}

// ─── Existence checks (dipakai handler untuk validasi FK) ─────────────────────

// AirlineExists memeriksa apakah airline dengan id tersebut ada.
func (r *Repository) AirlineExists(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM airlines WHERE id=?`, id).Scan(&count)
	return count > 0, err
}

// HotelExists memeriksa apakah hotel dengan id tersebut ada.
func (r *Repository) HotelExists(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM hotels WHERE id=?`, id).Scan(&count)
	return count > 0, err
}

// ItineraryExists memeriksa apakah itinerary dengan id tersebut ada.
func (r *Repository) ItineraryExists(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM itineraries WHERE id=?`, id).Scan(&count)
	return count > 0, err
}

// AddOnExists memeriksa apakah add-on dengan id tersebut ada.
func (r *Repository) AddOnExists(ctx context.Context, id int64) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM add_ons WHERE id=?`, id).Scan(&count)
	return count > 0, err
}

// ─── Internal helpers ─────────────────────────────────────────────────────────

func (r *Repository) exists(ctx context.Context, id int64, brandID *int64) error {
	var count int
	q := `SELECT COUNT(*) FROM schedules WHERE id=?`
	var args []interface{}
	args = append(args, id)
	if brandID != nil {
		q += " AND brand_id=?"
		args = append(args, *brandID)
	}
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&count)
	if err != nil {
		return fmt.Errorf("schedule.exists: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) fetchFull(ctx context.Context, id int64, brandID *int64) (*Schedule, error) {
	q := selectFull + " WHERE s.id=?"
	var args []interface{}
	args = append(args, id)
	if brandID != nil {
		q += " AND s.brand_id=?"
		args = append(args, *brandID)
	}
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("schedule.fetchFull: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("schedule.fetchFull rows: %w", err)
		}
		return nil, ErrNotFound
	}
	return scanRow(rows)
}

// scanRow mem-scan satu baris dari *sql.Rows (hasil selectFull) ke *Schedule.
func scanRow(rows *sql.Rows) (*Schedule, error) {
	var (
		s              Schedule
		maskapaiID     sql.NullInt64
		maskapaiName   string
		maskapaiLogo   sql.NullString
		hotelMekkahID  sql.NullInt64
		hmName         string
		hmStar         int
		hmDist         sql.NullInt64
		hotelMadinahID sql.NullInt64
		hmdName        string
		hmdStar        int
		hmdDist        sql.NullInt64
		hargaCoret     sql.NullFloat64
		itineraryID    sql.NullInt64
		addOnsJSON     []byte
		includeJSON    []byte
		excludeJSON    []byte
	)

	err := rows.Scan(
		&s.ID, &s.BrandID, &s.JadwalNama, &s.Status, &s.IsPromo, &s.IsTicketConfirmed, &s.IsDirectFlight, &s.SeatTotal, &s.SeatSisa,
		&maskapaiID, &maskapaiName, &maskapaiLogo,
		&s.BerangkatTanggal, &s.BerangkatJam, &s.BerangkatKodePenerbangan,
		&s.BerangkatBandaraAsal, &s.BerangkatBandaraTujuan,
		&s.PulangTanggal, &s.PulangJam, &s.PulangKodePenerbangan,
		&s.PulangBandaraAsal, &s.PulangBandaraTujuan, &s.TransitBandara,
		&hotelMekkahID, &hmName, &hmStar, &hmDist,
		&hotelMadinahID, &hmdName, &hmdStar, &hmdDist,
		&s.HargaQuad, &s.HargaTriple, &s.HargaDouble, &hargaCoret,
		&itineraryID, &addOnsJSON, &includeJSON, &excludeJSON,
		&s.BrosurURL, &s.BrosurThumbURL,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("schedule.scanRow: %w", err)
	}

	// Maskapai ref (nullable FK)
	if maskapaiID.Valid {
		s.Maskapai = &MaskapaiRef{ID: maskapaiID.Int64, Name: maskapaiName}
		if maskapaiLogo.Valid && maskapaiLogo.String != "" {
			s.Maskapai.LogoURL = &maskapaiLogo.String
		}
	}

	// Hotel Mekkah ref
	if hotelMekkahID.Valid {
		s.HotelMekkah.ID = hotelMekkahID.Int64
		s.HotelMekkah.Name = hmName
		s.HotelMekkah.StarRating = hmStar
		if hmDist.Valid {
			v := int(hmDist.Int64)
			s.HotelMekkah.DistanceM = &v
		}
	}

	// Hotel Madinah ref
	if hotelMadinahID.Valid {
		s.HotelMadinah.ID = hotelMadinahID.Int64
		s.HotelMadinah.Name = hmdName
		s.HotelMadinah.StarRating = hmdStar
		if hmdDist.Valid {
			v := int(hmdDist.Int64)
			s.HotelMadinah.DistanceM = &v
		}
	}

	// Harga Coret nullable
	if hargaCoret.Valid {
		v := hargaCoret.Float64
		s.HargaCoret = &v
	}

	// ItineraryID nullable
	if itineraryID.Valid {
		s.ItineraryID = &itineraryID.Int64
	}

	// JSON arrays — default ke slice kosong bukan nil
	s.IncludeItems = []string{}
	s.ExcludeItems = []string{}
	s.AddOns = []AddOnRef{}
	if len(includeJSON) > 0 {
		_ = json.Unmarshal(includeJSON, &s.IncludeItems)
	}
	if len(excludeJSON) > 0 {
		_ = json.Unmarshal(excludeJSON, &s.ExcludeItems)
	}
	if len(addOnsJSON) > 0 {
		_ = json.Unmarshal(addOnsJSON, &s.AddOns)
	}

	return &s, nil
}

// ─── SQL null helpers ─────────────────────────────────────────────────────────

func nullInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v != 0}
}

func nullString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: v != ""}
}
