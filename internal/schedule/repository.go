package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"erp-azhan/api/internal/benefit"
	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/itinerary"
	"erp-azhan/api/internal/shared"
)

// ErrNotFound dikembalikan saat schedule tidak ditemukan.
var ErrNotFound = errors.New("data tidak ditemukan")

// selectFull adalah query SELECT lengkap dengan LEFT JOIN ke airlines dan hotels (2x alias).
// Tambahkan WHERE / ORDER BY setelah konstanta ini sesuai kebutuhan.
const selectFull = `
	SELECT
		s.id,
		s.brand_id,
		s.category_id,
		COALESCE(cat.name, '') AS category_name,
		COALESCE(cat.slug, '') AS category_slug,
		cat.badge_color AS category_badge_color,
		s.jadwal_nama,
		s.status,
		s.views,
		s.is_direct_flight,
		s.seat_total,
		s.seat_sisa,
		s.seat_tampil,
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
		hm.photo_url AS hm_photo,
		hm.video_url AS hm_video,
		s.hotel_madinah_id,
		COALESCE(hmd.name, '') AS hmd_name,
		COALESCE(hmd.star_rating, 0) AS hmd_star,
		hmd.distance_m AS hmd_dist,
		hmd.photo_url AS hmd_photo,
		hmd.video_url AS hmd_video,
		s.harga_quad,
		s.harga_triple,
		s.harga_double,
		s.harga_infant,
		s.minimal_dp,
		COALESCE(s.minimal_dp, br.minimal_dp, 0),
		s.nominal_komisi_langsung,
		s.nominal_bonus_pembinaan,
		s.itinerary_id,
		COALESCE(s.include_items, '[]') AS include_items,
		COALESCE(s.exclude_items, '[]') AS exclude_items,
		COALESCE(s.brosur_url, '') AS brosur_url,
		COALESCE(s.brosur_thumb_url, '') AS brosur_thumb_url,
		s.created_at,
		s.updated_at
	FROM schedules s
	JOIN brands br ON br.id = s.brand_id
	LEFT JOIN package_categories cat ON cat.id = s.category_id
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
		SELECT s.id, s.brand_id, s.category_id, COALESCE(cat.name, ''), COALESCE(cat.slug, ''), cat.badge_color,
			s.jadwal_nama, s.status, s.is_direct_flight,
			s.maskapai_id, COALESCE(a.name, ''), a.logo_url,
			DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d') AS berangkat_tanggal,
			s.seat_total, s.seat_sisa, s.harga_quad, s.harga_triple, s.harga_double, s.harga_infant
		FROM schedules s
		LEFT JOIN package_categories cat ON cat.id = s.category_id
		LEFT JOIN airlines a ON a.id = s.maskapai_id
		WHERE 1=1`

	var args []interface{}
	if brandID != nil {
		q += " AND s.brand_id = ?"
		args = append(args, *brandID)
	}

	q += " ORDER BY s.created_at DESC LIMIT ?"
	args = append(args, shared.DefaultListCap)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("schedule.List: %w", err)
	}
	defer rows.Close()

	items := make([]ScheduleListItem, 0)
	for rows.Next() {
		var item ScheduleListItem
		var maskapaiID *int64
		var maskapaiName string
		var maskapaiLogo *string

		var categoryID sql.NullInt64
		var categoryName, categorySlug string
		var categoryBadge sql.NullString
		var hargaInfant sql.NullFloat64

		if err := rows.Scan(&item.ID, &item.BrandID, &categoryID, &categoryName, &categorySlug, &categoryBadge,
			&item.JadwalNama, &item.Status, &item.IsDirectFlight,
			&maskapaiID, &maskapaiName, &maskapaiLogo,
			&item.BerangkatTanggal, &item.SeatTotal, &item.SeatSisa,
			&item.HargaQuad, &item.HargaTriple, &item.HargaDouble, &hargaInfant); err != nil {
			return nil, fmt.Errorf("schedule.List scan: %w", err)
		}

		if categoryID.Valid {
			item.CategoryID = &categoryID.Int64
			item.Category = &CategoryRef{
				ID:         categoryID.Int64,
				Name:       categoryName,
				Slug:       categorySlug,
				BadgeColor: badgeColorPtr(categoryBadge),
			}
		}

		if hargaInfant.Valid {
			v := hargaInfant.Float64
			item.HargaInfant = &v
		}

		if maskapaiID != nil && *maskapaiID > 0 {
			item.Maskapai = &MaskapaiRef{
				ID:      *maskapaiID,
				Name:    maskapaiName,
				LogoURL: maskapaiLogo,
			}
		}

		th, errTransit := r.GetTransitHotelsByScheduleID(ctx, item.ID)
		if errTransit != nil {
			return nil, fmt.Errorf("schedule.List hotel transit: %w", errTransit)
		}
		item.TransitHotels = th

		items = append(items, item)
	}

	if len(items) > 0 {
		ids := make([]any, len(items))
		placeholders := make([]string, len(items))
		for i, it := range items {
			ids[i] = it.ID
			placeholders[i] = "?"
		}
		bQuery := fmt.Sprintf(`SELECT schedule_id, COUNT(*) FROM bookings WHERE schedule_id IN (%s) GROUP BY schedule_id`, strings.Join(placeholders, ","))
		bRows, err := r.db.QueryContext(ctx, bQuery, ids...)
		if err != nil {
			return nil, fmt.Errorf("schedule.List jumlah booking: %w", err)
		}
		defer bRows.Close()
		bCountMap := make(map[int64]int)
		for bRows.Next() {
			var schedID int64
			var count int
			if err := bRows.Scan(&schedID, &count); err != nil {
				return nil, fmt.Errorf("schedule.List jumlah booking: %w", err)
			}
			bCountMap[schedID] = count
		}
		if err := bRows.Err(); err != nil {
			return nil, fmt.Errorf("schedule.List jumlah booking: %w", err)
		}
		for i := range items {
			items[i].BookingCount = bCountMap[items[i].ID]
		}
	}

	return items, rows.Err()
}

type rowsQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// transitHotelsBySchedule mengambil hotel transit beberapa jadwal sekaligus, terurut per jadwal.
func transitHotelsBySchedule(ctx context.Context, q rowsQuerier, scheduleIDs []int64) (map[int64][]TransitHotel, error) {
	out := make(map[int64][]TransitHotel, len(scheduleIDs))
	if len(scheduleIDs) == 0 {
		return out, nil
	}
	args := make([]any, len(scheduleIDs))
	ph := make([]string, len(scheduleIDs))
	for i, id := range scheduleIDs {
		args[i] = id
		ph[i] = "?"
	}
	rows, err := q.QueryContext(ctx, `
		SELECT sth.schedule_id, sth.hotel_id, h.name, COALESCE(h.city, ''), COALESCE(h.star_rating, 0), h.photo_url, h.video_url
		FROM schedule_transit_hotels sth
		JOIN hotels h ON h.id = sth.hotel_id
		WHERE sth.schedule_id IN (`+strings.Join(ph, ",")+`)
		ORDER BY sth.schedule_id, sth.urutan ASC`, args...)
	if err != nil {
		return nil, fmt.Errorf("schedule.transitHotelsBySchedule: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sid int64
		var th TransitHotel
		var photoURL, videoURL sql.NullString
		if err := rows.Scan(&sid, &th.HotelID, &th.Nama, &th.Kota, &th.StarRating, &photoURL, &videoURL); err != nil {
			return nil, fmt.Errorf("schedule.transitHotelsBySchedule scan: %w", err)
		}
		if photoURL.Valid && photoURL.String != "" {
			th.PhotoURL = &photoURL.String
		}
		if videoURL.Valid && videoURL.String != "" {
			th.VideoURL = &videoURL.String
		}
		out[sid] = append(out[sid], th)
	}
	return out, rows.Err()
}

// ─── Get Detail ───────────────────────────────────────────────────────────────

// GetByID mengambil schedule lengkap beserta ref hotel/airline.
func (r *Repository) GetByID(ctx context.Context, id int64, brandID *int64) (*Schedule, error) {
	s, err := r.fetchFull(ctx, id, brandID)
	if err != nil || s == nil {
		return s, err
	}
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM transaksi_komisi tk JOIN bookings b ON b.id = tk.booking_id
		WHERE b.schedule_id = ? AND tk.dibatalkan_at IS NULL`, id).Scan(&s.KomisiAktif); err != nil {
		return nil, fmt.Errorf("schedule.GetByID komisi aktif: %w", err)
	}
	if s.BookingAktif, err = r.CountActiveBookings(ctx, id); err != nil {
		return nil, fmt.Errorf("schedule.GetByID booking aktif: %w", err)
	}
	return s, nil
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
			brand_id, category_id, jadwal_nama, status, views, is_direct_flight, seat_total, seat_sisa,
			maskapai_id, berangkat_tanggal, berangkat_jam, berangkat_kode_penerbangan, berangkat_bandara_asal, berangkat_bandara_tujuan,
			pulang_tanggal, pulang_jam, pulang_kode_penerbangan, pulang_bandara_asal, pulang_bandara_tujuan, transit_bandara,
			hotel_mekkah_id, hotel_madinah_id,
			harga_quad, harga_triple, harga_double, harga_infant, minimal_dp,
			nominal_komisi_langsung, nominal_bonus_pembinaan,
			itinerary_id, include_items, exclude_items,
			brosur_url, brosur_thumb_url
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := tx.ExecContext(ctx, q,
		inp.BrandID, nullInt64Ptr(inp.CategoryID), inp.JadwalNama, inp.Status, 0, inp.IsDirectFlight, inp.SeatTotal, inp.SeatSisa,
		nullInt64(inp.MaskapaiID),
		inp.BerangkatTanggal, nullString(inp.BerangkatJam), nullString(inp.BerangkatKodePenerbangan),
		nullString(inp.BerangkatBandaraAsal), nullString(inp.BerangkatBandaraTujuan),
		inp.PulangTanggal, nullString(inp.PulangJam), nullString(inp.PulangKodePenerbangan),
		nullString(inp.PulangBandaraAsal), nullString(inp.PulangBandaraTujuan), nullString(inp.TransitBandara),
		nullInt64(inp.HotelMekkahID), nullInt64(inp.HotelMadinahID),
		inp.HargaQuad, inp.HargaTriple, inp.HargaDouble, inp.HargaInfant, inp.MinimalDP,
		inp.NominalKomisiLangsung, inp.NominalBonusPembinaan,
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

	if err := r.UpdateTransitHotels(ctx, tx, id, inp.TransitHotelIDs); err != nil {
		return nil, fmt.Errorf("schedule.Create transit hotels: %w", err)
	}
	if inp.BenefitIDs != nil {
		if err := benefit.GantiTautan(ctx, tx, id, *inp.BenefitIDs); err != nil {
			return nil, err
		}
	}
	if inp.LokasiIDs != nil {
		if err := gantiLokasi(ctx, tx, id, *inp.LokasiIDs); err != nil {
			return nil, err
		}
	}

	if inp.ItineraryKhusus != nil {
		if err := simpanItineraryKhusus(ctx, tx, id, inp.ItineraryKhusus, inp.JadwalNama); err != nil {
			return nil, err
		}
	}

	if err := setKlasifikasiTx(ctx, tx, id, inp.Klasifikasi); err != nil {
		return nil, err
	}
	if err := tandaiPernahTayang(ctx, tx, id); err != nil {
		return nil, err
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

	if err := prepareScheduleUpdate(ctx, tx, id, &inp, brandID, finalBrandID); err != nil {
		return nil, err
	}

	// Tanpa itinerary_id dan tanpa itinerary_khusus: itinerary pribadi paket dipertahankan.
	khususLama, err := itinerary.KhususID(ctx, tx, id)
	if err != nil {
		return nil, err
	}
	if inp.ItineraryKhusus == nil && inp.ItineraryID == nil && khususLama != nil {
		inp.ItineraryID = khususLama
	}

	includeJSON, _ := json.Marshal(inp.IncludeItems)
	excludeJSON, _ := json.Marshal(inp.ExcludeItems)

	const q = `
		UPDATE schedules SET
			brand_id=?, category_id=?, jadwal_nama=?, is_direct_flight=?, seat_total=?, seat_sisa=?,
			maskapai_id=?, berangkat_tanggal=?, berangkat_jam=?, berangkat_kode_penerbangan=?, berangkat_bandara_asal=?, berangkat_bandara_tujuan=?,
			pulang_tanggal=?, pulang_jam=?, pulang_kode_penerbangan=?, pulang_bandara_asal=?, pulang_bandara_tujuan=?, transit_bandara=?,
			hotel_mekkah_id=?, hotel_madinah_id=?,
			harga_quad=?, harga_triple=?, harga_double=?, harga_infant=?, minimal_dp=?,
			nominal_komisi_langsung=?, nominal_bonus_pembinaan=?,
			itinerary_id=?, include_items=?, exclude_items=?,
			brosur_url=?, brosur_thumb_url=?
		WHERE id=?`

	_, err = tx.ExecContext(ctx, q,
		finalBrandID, nullInt64Ptr(inp.CategoryID), inp.JadwalNama, inp.IsDirectFlight, inp.SeatTotal, inp.SeatSisa,
		nullInt64(inp.MaskapaiID),
		inp.BerangkatTanggal, nullString(inp.BerangkatJam), nullString(inp.BerangkatKodePenerbangan),
		nullString(inp.BerangkatBandaraAsal), nullString(inp.BerangkatBandaraTujuan),
		inp.PulangTanggal, nullString(inp.PulangJam), nullString(inp.PulangKodePenerbangan),
		nullString(inp.PulangBandaraAsal), nullString(inp.PulangBandaraTujuan), nullString(inp.TransitBandara),
		nullInt64(inp.HotelMekkahID), nullInt64(inp.HotelMadinahID),
		inp.HargaQuad, inp.HargaTriple, inp.HargaDouble, inp.HargaInfant, inp.MinimalDP,
		inp.NominalKomisiLangsung, inp.NominalBonusPembinaan,
		inp.ItineraryID, includeJSON, excludeJSON,
		nullString(inp.BrosurURL), nullString(inp.BrosurThumbURL),
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("schedule.Update: %w", err)
	}

	if err := r.UpdateTransitHotels(ctx, tx, id, inp.TransitHotelIDs); err != nil {
		return nil, fmt.Errorf("schedule.Update transit hotels: %w", err)
	}
	// nil: klien lama yang tidak mengirim benefit_ids tidak menghapus benefit yang tersimpan.
	if inp.BenefitIDs != nil {
		if err := benefit.GantiTautan(ctx, tx, id, *inp.BenefitIDs); err != nil {
			return nil, err
		}
	}
	// nil: clients that do not send lokasi_ids keep the stored locations.
	if inp.LokasiIDs != nil {
		if err := ubahLokasi(ctx, tx, id, *inp.LokasiIDs); err != nil {
			return nil, err
		}
	}

	if inp.ItineraryKhusus != nil {
		if err := simpanItineraryKhusus(ctx, tx, id, inp.ItineraryKhusus, inp.JadwalNama); err != nil {
			return nil, err
		}
	} else if khususLama != nil {
		// Paket sekarang menunjuk master (atau tetap ke salinannya): salinan lain dibuang.
		if err := itinerary.HapusKhusus(ctx, tx, id, inp.ItineraryID); err != nil {
			return nil, err
		}
	}

	if err := setKlasifikasiTx(ctx, tx, id, inp.Klasifikasi); err != nil {
		return nil, err
	}
	if err := tandaiPernahTayang(ctx, tx, id); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("schedule.Update commit: %w", err)
	}

	return r.fetchFull(ctx, id, brandID)
}

// ─── UpdateStatus ─────────────────────────────────────────────────────────────

// ─── Seat ─────────────────────────────────────────────────────────────────────

// Delete menghapus schedule berdasarkan id.
func (r *Repository) Delete(ctx context.Context, id int64, brandID *int64) error {
	if err := r.exists(ctx, id, brandID); err != nil {
		return err
	}
	// Hanya paket draft yang dihapus permanen (keputusan 11 November 2026). Paket yang pernah tayang
	// diarsipkan, supaya tautan yang sudah tersebar tetap menjawab "sudah tidak tersedia", bukan 404.
	// Penanda pernah_tayang_at ikut diperiksa: paket yang pernah tayang lalu dikembalikan ke draft tetap
	// tidak bisa dihapus, karena tautannya sudah pernah beredar.
	// Status dibaca dengan baris terkunci supaya paket yang diterbitkan bersamaan tidak ikut terhapus.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("schedule.Delete tx: %w", err)
	}
	defer tx.Rollback()
	var status string
	var pernahTayang bool
	err = tx.QueryRowContext(ctx, `SELECT status, pernah_tayang_at IS NOT NULL FROM schedules WHERE id=? FOR UPDATE`, id).Scan(&status, &pernahTayang)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("schedule.Delete baca status: %w", err)
	}
	if pesan := tolakHapusPaket(status, pernahTayang); pesan != "" {
		return fmt.Errorf("%w: %s", ErrConflict, pesan)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM schedules WHERE id=?`, id); err != nil {
		return err
	}
	// Itinerary pribadi paket ikut dihapus (hari ikut lewat CASCADE); FK menunjuk baris ini sudah hilang.
	if err := itinerary.HapusKhusus(ctx, tx, id, nil); err != nil {
		return err
	}
	return tx.Commit()
}

// simpanItineraryKhusus menyimpan itinerary pribadi paket dan mengarahkan schedules.itinerary_id ke sana.
// Judul kosong memakai nama paket.
func simpanItineraryKhusus(ctx context.Context, tx *sql.Tx, scheduleID int64, k *itinerary.KhususRequest, namaPaket string) error {
	title := strings.TrimSpace(k.Title)
	if title == "" {
		title = strings.TrimSpace(namaPaket)
	}
	id, err := itinerary.SimpanKhusus(ctx, tx, scheduleID, title, k.Days)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schedules SET itinerary_id=? WHERE id=?`, id, scheduleID); err != nil {
		return fmt.Errorf("schedule.simpanItineraryKhusus tunjuk: %w", err)
	}
	return nil
}

// ─── Existence checks (dipakai handler untuk validasi FK) ─────────────────────

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
	s, err := scanRow(rows)
	if err != nil {
		return nil, err
	}
	// Close before the follow-up queries so each request holds one pool connection, not two.
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("schedule.fetchFull close: %w", err)
	}

	th, err := r.GetTransitHotelsByScheduleID(ctx, s.ID)
	if err != nil {
		return nil, fmt.Errorf("schedule.GetByID hotel transit: %w", err)
	}
	s.TransitHotels = th

	bf, err := benefit.UntukJadwal(ctx, r.db, []int64{s.ID})
	if err != nil {
		return nil, fmt.Errorf("schedule.GetByID benefit: %w", err)
	}
	s.Benefits = bf[s.ID]
	if s.Benefits == nil {
		s.Benefits = []benefit.Ref{}
	}

	if s.Lokasi, err = lokasiJadwal(ctx, r.db, s.ID); err != nil {
		return nil, fmt.Errorf("schedule.GetByID lokasi: %w", err)
	}
	s.LokasiIDs = idLokasi(s.Lokasi)

	if s.ItineraryID != nil {
		if err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM itineraries WHERE id=? AND schedule_id=?)`, *s.ItineraryID, s.ID).Scan(&s.ItineraryKhusus); err != nil {
			return nil, fmt.Errorf("schedule.GetByID itinerary khusus: %w", err)
		}
	}

	dipegang, err := shared.SeatDipegangBooking(ctx, r.db, s.ID)
	if err != nil {
		return nil, fmt.Errorf("schedule.GetByID seat dipegang: %w", err)
	}
	s.SeatDipegang = &dipegang

	var bCount int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM bookings WHERE schedule_id = ?`, s.ID).Scan(&bCount); err != nil {
		return nil, fmt.Errorf("schedule.GetByID jumlah booking: %w", err)
	}
	s.BookingCount = bCount

	kl, err := r.loadKlasifikasi(ctx, []int64{s.ID})
	if err != nil {
		return nil, err
	}
	k, ok := kl[s.ID]
	s.applyKlasifikasi(k, ok)

	return s, nil
}

// GetTransitHotelsByScheduleID mengambil daftar hotel transit untuk schedule tertentu terurut ASC.
func (r *Repository) GetTransitHotelsByScheduleID(ctx context.Context, scheduleID int64) ([]TransitHotel, error) {
	q := `
		SELECT sth.hotel_id, h.name, COALESCE(h.city, ''), COALESCE(h.star_rating, 0), h.photo_url, h.video_url
		FROM schedule_transit_hotels sth
		JOIN hotels h ON h.id = sth.hotel_id
		WHERE sth.schedule_id = ?
		ORDER BY sth.urutan ASC`

	rows, err := r.db.QueryContext(ctx, q, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("schedule.GetTransitHotelsByScheduleID: %w", err)
	}
	defer rows.Close()

	items := make([]TransitHotel, 0)
	for rows.Next() {
		var th TransitHotel
		var photoURL sql.NullString
		var videoURL sql.NullString
		if err := rows.Scan(&th.HotelID, &th.Nama, &th.Kota, &th.StarRating, &photoURL, &videoURL); err != nil {
			return nil, fmt.Errorf("schedule.GetTransitHotelsByScheduleID scan: %w", err)
		}
		if photoURL.Valid && photoURL.String != "" {
			th.PhotoURL = &photoURL.String
		}
		if videoURL.Valid && videoURL.String != "" {
			th.VideoURL = &videoURL.String
		}
		items = append(items, th)
	}
	return items, rows.Err()
}

// UpdateTransitHotels memperbarui relasi hotel transit menggunakan pola replace-all (DELETE + INSERT).
func (r *Repository) UpdateTransitHotels(ctx context.Context, tx *sql.Tx, scheduleID int64, hotelIDs []int64) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM schedule_transit_hotels WHERE schedule_id = ?`, scheduleID); err != nil {
		return fmt.Errorf("schedule.UpdateTransitHotels delete: %w", err)
	}

	for i, hid := range hotelIDs {
		if hid <= 0 {
			continue
		}
		_, err := tx.ExecContext(ctx, `INSERT INTO schedule_transit_hotels (schedule_id, hotel_id, urutan) VALUES (?, ?, ?)`, scheduleID, hid, i+1)
		if err != nil {
			return fmt.Errorf("schedule.UpdateTransitHotels insert: %w", err)
		}
	}
	return nil
}

// badgeColorPtr mengubah kolom badge_color (nullable) menjadi pointer; kosong = nil (otomatis).
func badgeColorPtr(n sql.NullString) *string {
	if !n.Valid || n.String == "" {
		return nil
	}
	v := n.String
	return &v
}

// scanRow mem-scan satu baris dari *sql.Rows (hasil selectFull) ke *Schedule.
func scanRow(rows *sql.Rows) (*Schedule, error) {
	var (
		s              Schedule
		categoryID     sql.NullInt64
		categoryName   string
		categorySlug   string
		categoryBadge  sql.NullString
		maskapaiID     sql.NullInt64
		maskapaiName   string
		maskapaiLogo   sql.NullString
		hotelMekkahID  sql.NullInt64
		hmName         string
		hmStar         int
		hmDist         sql.NullInt64
		hmPhoto        sql.NullString
		hmVideo        sql.NullString
		hotelMadinahID sql.NullInt64
		hmdName        string
		hmdStar        int
		hmdDist        sql.NullInt64
		hmdPhoto       sql.NullString
		hmdVideo       sql.NullString
		hargaInfant    sql.NullFloat64
		minimalDP      sql.NullFloat64
		komisiLangsung sql.NullFloat64
		bonusPembinaan sql.NullFloat64
		itineraryID    sql.NullInt64
		includeJSON    []byte
		seatTampil     sql.NullInt64
		excludeJSON    []byte
	)

	err := rows.Scan(
		&s.ID, &s.BrandID, &categoryID, &categoryName, &categorySlug, &categoryBadge,
		&s.JadwalNama, &s.Status, &s.Views, &s.IsDirectFlight, &s.SeatTotal, &s.SeatSisa, &seatTampil,
		&maskapaiID, &maskapaiName, &maskapaiLogo,
		&s.BerangkatTanggal, &s.BerangkatJam, &s.BerangkatKodePenerbangan,
		&s.BerangkatBandaraAsal, &s.BerangkatBandaraTujuan,
		&s.PulangTanggal, &s.PulangJam, &s.PulangKodePenerbangan,
		&s.PulangBandaraAsal, &s.PulangBandaraTujuan, &s.TransitBandara,
		&hotelMekkahID, &hmName, &hmStar, &hmDist, &hmPhoto, &hmVideo,
		&hotelMadinahID, &hmdName, &hmdStar, &hmdDist, &hmdPhoto, &hmdVideo,
		&s.HargaQuad, &s.HargaTriple, &s.HargaDouble, &hargaInfant, &minimalDP, &s.EffectiveMinimalDP,
		&komisiLangsung, &bonusPembinaan,
		&itineraryID, &includeJSON, &excludeJSON,
		&s.BrosurURL, &s.BrosurThumbURL,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("schedule.scanRow: %w", err)
	}
	if seatTampil.Valid {
		v := int(seatTampil.Int64)
		s.SeatTampil = &v
	}

	if categoryID.Valid {
		s.CategoryID = &categoryID.Int64
		s.Category = &CategoryRef{
			ID:         categoryID.Int64,
			Name:       categoryName,
			Slug:       categorySlug,
			BadgeColor: badgeColorPtr(categoryBadge),
		}
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
		if hmPhoto.Valid && hmPhoto.String != "" {
			s.HotelMekkah.PhotoURL = &hmPhoto.String
		}
		if hmVideo.Valid && hmVideo.String != "" {
			s.HotelMekkah.VideoURL = &hmVideo.String
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
		if hmdPhoto.Valid && hmdPhoto.String != "" {
			s.HotelMadinah.PhotoURL = &hmdPhoto.String
		}
		if hmdVideo.Valid && hmdVideo.String != "" {
			s.HotelMadinah.VideoURL = &hmdVideo.String
		}
	}

	// Harga Infant nullable
	if hargaInfant.Valid {
		v := hargaInfant.Float64
		s.HargaInfant = &v
	}

	// Minimal DP nullable
	if komisiLangsung.Valid {
		v := komisiLangsung.Float64
		s.NominalKomisiLangsung = &v
	}
	if bonusPembinaan.Valid {
		v := bonusPembinaan.Float64
		s.NominalBonusPembinaan = &v
	}
	if minimalDP.Valid {
		v := minimalDP.Float64
		s.MinimalDP = &v
	}

	// ItineraryID nullable
	if itineraryID.Valid {
		s.ItineraryID = &itineraryID.Int64
	}

	// JSON arrays — default ke slice kosong bukan nil
	s.IncludeItems = []string{}
	s.ExcludeItems = []string{}
	if len(includeJSON) > 0 {
		_ = json.Unmarshal(includeJSON, &s.IncludeItems)
	}
	if len(excludeJSON) > 0 {
		_ = json.Unmarshal(excludeJSON, &s.ExcludeItems)
	}

	return &s, nil
}

// ─── SQL null helpers ─────────────────────────────────────────────────────────

func nullInt64Ptr(v *int64) sql.NullInt64 {
	if v == nil || *v == 0 {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *v, Valid: true}
}

func nullInt64(v int64) sql.NullInt64 {
	return sql.NullInt64{Int64: v, Valid: v != 0}
}

func nullString(v string) sql.NullString {
	return sql.NullString{String: v, Valid: v != ""}
}

func prepareScheduleUpdate(ctx context.Context, tx *sql.Tx, id int64, inp *ScheduleInput, brandID *int64, finalBrandID int64) error {
	var oldTotal, oldRemaining int
	var oldBrand int64
	var oldBerangkat string
	var oldTampil sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT seat_total, seat_sisa, seat_tampil, brand_id, DATE_FORMAT(berangkat_tanggal, '%Y-%m-%d') FROM schedules WHERE id=? FOR UPDATE`, id).
		Scan(&oldTotal, &oldRemaining, &oldTampil, &oldBrand, &oldBerangkat); err != nil {
		return err
	}
	if brandID != nil && oldBrand != *brandID {
		return ErrNotFound
	}
	if err := ubahTanggalBerangkatTx(ctx, tx, id, oldBerangkat, inp); err != nil {
		return err
	}
	if inp.ExpectedSeatTotal != nil && *inp.ExpectedSeatTotal != oldTotal {
		return fmt.Errorf("%w: kapasitas berubah, muat ulang paket", ErrConflict)
	}
	if finalBrandID != oldBrand {
		return fmt.Errorf("%w: brand paket tidak dapat diubah", ErrConflict)
	}
	// Kuota tetap: sisa seat tersimpan dipertahankan. Selisih dengan booking hanya diperbaiki lewat
	// POST /schedules/{id}/seat/samakan, bukan diam-diam saat simpan form.
	if inp.SeatTotal == oldTotal {
		inp.SeatSisa = oldRemaining
		return nil
	}
	dipegang, err := shared.SeatDipegangBooking(ctx, tx, id)
	if err != nil {
		return err
	}
	inp.SeatSisa, err = remainingAfterCapacityChange(dipegang, inp.SeatTotal)
	if err != nil {
		return err
	}
	// Ubah kuota seat lewat form edit ikut tercatat di riwayat penyesuaian seat, tanpa isian alasan.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schedule_seat_adjustments (schedule_id, jenis, admin_user_id, seat_before, seat_after, reason) VALUES (?,'kuota',?,?,?,?)`,
		id, nullInt64(inp.ActorID), oldRemaining, inp.SeatSisa, alasanUbahKuota(oldTotal, inp.SeatTotal)); err != nil {
		return fmt.Errorf("catat riwayat ubah kuota seat: %w", err)
	}
	// Seat tampil berlaku 1..seat_total: kuota yang diturunkan ikut menurunkan seat tampil (tercatat di riwayat).
	if oldTampil.Valid && int(oldTampil.Int64) > inp.SeatTotal {
		baru := inp.SeatTotal
		if err := updateSeatTampilTx(ctx, tx, id, &baru, brandID, inp.ActorID); err != nil {
			return err
		}
	}
	return nil
}

// CountActiveBookings menghitung booking aktif (baru/dp/lunas) pada paket (MP-09).
func (r *Repository) CountActiveBookings(ctx context.Context, scheduleID int64) (int, error) {
	return booking.HitungAktif(ctx, r.db, scheduleID)
}

// NameTaken: sudah ada paket dengan nama (tanpa beda huruf besar-kecil dan spasi tepi) dan tanggal berangkat
// yang sama di brand itu.
// excludeID mengecualikan paket yang sedang diedit (0 untuk paket baru).
func (r *Repository) NameTaken(ctx context.Context, brandID int64, nama, berangkat string, excludeID int64) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM schedules WHERE brand_id=? AND UPPER(TRIM(jadwal_nama))=UPPER(TRIM(?)) AND berangkat_tanggal=? AND id<>?`, brandID, nama, berangkat, excludeID).Scan(&n)
	return n > 0, err
}

// tolakHapusPaket: paket hanya boleh dihapus permanen bila berstatus draft dan belum pernah tayang.
// Mengembalikan pesan untuk pengguna, atau teks kosong bila boleh dihapus.
func tolakHapusPaket(status string, pernahTayang bool) string {
	switch {
	case status != StatusDraft:
		return "hanya paket draft yang dapat dihapus"
	case pernahTayang:
		return "paket pernah tayang tidak dapat dihapus"
	}
	return ""
}
