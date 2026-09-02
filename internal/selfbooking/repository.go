package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrSeatHabis   = errors.New("kursi tidak cukup")
	ErrDuplicate   = errors.New("anda sudah memiliki booking aktif di jadwal ini")
	ErrNotFound    = errors.New("data tidak ditemukan")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) ProcessBooking(ctx context.Context, brandID int64, req BookingRequest) (*BookingResponse, error) {
	// Serializable ensures atomic check-then-act
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Lock schedule
	var scheduleStatus string
	var scheduleBrandID int64
	var seatSisa int
	var hargaQuad, hargaTriple, hargaDouble float64
	var hargaInfant, scheduleMinDP sql.NullFloat64

	err = tx.QueryRowContext(ctx, `
		SELECT brand_id, status, seat_sisa, harga_quad, harga_triple, harga_double, harga_infant, minimal_dp
		FROM schedules WHERE id=? FOR UPDATE`, req.ScheduleID).Scan(
		&scheduleBrandID, &scheduleStatus, &seatSisa,
		&hargaQuad, &hargaTriple, &hargaDouble, &hargaInfant, &scheduleMinDP,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock schedule: %w", err)
	}
	if scheduleStatus != "published" || scheduleBrandID != brandID {
		return nil, ErrNotFound
	}

	// 2. Resolve PIC (Jamaah Utama)
	picJamaahID, err := shared.ResolveJamaah(ctx, tx, brandID, shared.JamaahInput{
		NamaLengkap:  req.PIC.NamaLengkap,
		NoHP:         req.PIC.NoHP,
		Email:        req.PIC.Email,
		JenisKelamin: &req.PIC.JenisKelamin,
	})
	if err != nil {
		return nil, fmt.Errorf("resolve pic: %w", err)
	}

	// 3. Duplikasi check
	var dupCount int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM bookings b
		JOIN booking_pax bp ON bp.booking_id = b.id
		WHERE b.schedule_id = ? AND bp.jamaah_id = ? AND b.status NOT IN ('batal', 'draft')
	`, req.ScheduleID, picJamaahID).Scan(&dupCount)
	if err != nil {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}
	if dupCount > 0 {
		return nil, ErrDuplicate
	}

	// 4. Hitung kebutuhan seat (hanya pax reguler yang mengurangi seat)
	regulerPaxCount := 1 // PIC
	for _, a := range req.Anggota {
		if a.PaxType == "reguler" {
			regulerPaxCount++
		}
	}
	if seatSisa < regulerPaxCount {
		return nil, ErrSeatHabis
	}

	// 5. Update PIN PIC
	pinHash, err := bcrypt.GenerateFromPassword([]byte(req.PIC.PortalPIN), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash pin: %w", err)
	}
	_, err = tx.ExecContext(ctx, `UPDATE jamaah SET portal_pin_hash=? WHERE id=?`, string(pinHash), picJamaahID)
	if err != nil {
		return nil, fmt.Errorf("update pin: %w", err)
	}

	// get id_jamaah pic
	var idJamaahPIC string
	err = tx.QueryRowContext(ctx, `SELECT id_jamaah FROM jamaah WHERE id=?`, picJamaahID).Scan(&idJamaahPIC)
	if err != nil {
		return nil, fmt.Errorf("get id jamaah: %w", err)
	}

	// 6. Create jamaah records for anggota
	anggotaJamaahIDs := make([]int64, len(req.Anggota))
	for i, a := range req.Anggota {
		var brandCode string
		var counter uint64
		err = tx.QueryRowContext(ctx, `SELECT kode_brand, jamaah_counter FROM brands WHERE id=? FOR UPDATE`, brandID).Scan(&brandCode, &counter)
		if err != nil {
			return nil, fmt.Errorf("lock brand: %w", err)
		}
		counter++
		_, err = tx.ExecContext(ctx, `UPDATE brands SET jamaah_counter=? WHERE id=?`, counter, brandID)
		if err != nil {
			return nil, fmt.Errorf("update counter: %w", err)
		}
		idJamaah := fmt.Sprintf("%s-%02d%02d%06d", strings.ToUpper(strings.TrimSpace(brandCode)), time.Now().Year()%100, int(time.Now().Month()), counter)
		kodeJamaah, err := shared.UniqueCode(ctx, tx, "jamaah", "kode_jamaah", "", 6)
		if err != nil {
			return nil, fmt.Errorf("kode jamaah: %w", err)
		}

		res, err := tx.ExecContext(ctx, `
			INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap, no_hp, jenis_kelamin)
			VALUES (?, ?, ?, ?, ?, ?)
		`, brandID, idJamaah, kodeJamaah, a.NamaLengkap, a.NoHP, a.JenisKelamin)
		if err != nil {
			return nil, fmt.Errorf("insert anggota jamaah: %w", err)
		}
		anggotaID, _ := res.LastInsertId()
		anggotaJamaahIDs[i] = anggotaID
	}

	// 7. Generate booking code
	var brandCodeStr string
	err = tx.QueryRowContext(ctx, `SELECT kode_brand FROM brands WHERE id=?`, brandID).Scan(&brandCodeStr)
	if err != nil {
		return nil, fmt.Errorf("get brand code: %w", err)
	}
	bookingCode, err := shared.UniqueCode(ctx, tx, "bookings", "id_booking", strings.ToUpper(brandCodeStr), 4)
	if err != nil {
		return nil, fmt.Errorf("generate id booking: %w", err)
	}

	// 8. Create booking header
	seatHoldExpiresAt := time.Now().Add(24 * time.Hour).UTC()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, seat_count, status, is_seat_blocked, seat_hold_expires_at)
		VALUES (?, ?, ?, ?, 'baru', TRUE, ?)
	`, bookingCode, req.ScheduleID, picJamaahID, regulerPaxCount, seatHoldExpiresAt)
	if err != nil {
		return nil, fmt.Errorf("insert booking: %w", err)
	}
	bookingID, _ := res.LastInsertId()

	// 9. Create booking pax
	var paxSummary []PaxSummary
	totalHarga := 0.0

	// Helper harga
	getHarga := func(roomType string) float64 {
		switch roomType {
		case "Triple":
			return hargaTriple
		case "Double":
			return hargaDouble
		default:
			return hargaQuad
		}
	}

	// PIC
	picHarga := getHarga(req.PIC.RoomType)
	totalHarga += picHarga
	paxSummary = append(paxSummary, PaxSummary{
		Nama:     req.PIC.NamaLengkap,
		PaxType:  "reguler",
		RoomType: &req.PIC.RoomType,
		Harga:    picHarga,
	})
	_, err = tx.ExecContext(ctx, `
		INSERT INTO booking_pax (booking_id, jamaah_id, pax_type, room_type, harga_pax, counts_for_seat, pax_status)
		VALUES (?, ?, 'reguler', ?, ?, TRUE, 'aktif')
	`, bookingID, picJamaahID, req.PIC.RoomType, picHarga)
	if err != nil {
		return nil, fmt.Errorf("insert pic pax: %w", err)
	}

	// Anggota
	for i, a := range req.Anggota {
		countsForSeat := false
		var hrg float64
		var rType *string

		if a.PaxType == "reguler" {
			countsForSeat = true
			if a.RoomType == nil {
				return nil, errors.New("room_type anggota reguler wajib diisi")
			}
			rType = a.RoomType
			hrg = getHarga(*a.RoomType)
		} else { // infant
			if hargaInfant.Valid {
				hrg = hargaInfant.Float64
			}
		}

		totalHarga += hrg
		paxSummary = append(paxSummary, PaxSummary{
			Nama:     a.NamaLengkap,
			PaxType:  a.PaxType,
			RoomType: rType,
			Harga:    hrg,
		})
		_, err = tx.ExecContext(ctx, `
			INSERT INTO booking_pax (booking_id, jamaah_id, pax_type, room_type, harga_pax, counts_for_seat, pax_status)
			VALUES (?, ?, ?, ?, ?, ?, 'aktif')
		`, bookingID, anggotaJamaahIDs[i], a.PaxType, rType, hrg, countsForSeat)
		if err != nil {
			return nil, fmt.Errorf("insert anggota pax: %w", err)
		}
	}

	// Update total_harga
	_, err = tx.ExecContext(ctx, `UPDATE bookings SET total_harga=? WHERE id=?`, totalHarga, bookingID)
	if err != nil {
		return nil, fmt.Errorf("update total harga: %w", err)
	}

	// 10. Reserve seats
	_, err = tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa = seat_sisa - ? WHERE id=?`, regulerPaxCount, req.ScheduleID)
	if err != nil {
		return nil, fmt.Errorf("reserve seats: %w", err)
	}

	// 11. Minimal DP
	var globalMinDP float64
	err = tx.QueryRowContext(ctx, `SELECT minimal_dp FROM brands WHERE id=?`, brandID).Scan(&globalMinDP)
	if err != nil {
		return nil, fmt.Errorf("get brand minimal dp: %w", err)
	}

	dpPerReguler := globalMinDP
	if scheduleMinDP.Valid {
		dpPerReguler = scheduleMinDP.Float64
	}
	totalMinDP := dpPerReguler * float64(regulerPaxCount)

	// 12. Bank Accounts
	var bankAccounts []BankAccountInfo
	rows, err := tx.QueryContext(ctx, `SELECT id, bank_name, logo_url, account_number, account_holder, instructions FROM bank_accounts WHERE brand_id=? AND is_active=TRUE`, brandID)
	if err == nil {
		for rows.Next() {
			var b BankAccountInfo
			var logo, inst sql.NullString
			if err := rows.Scan(&b.ID, &b.BankName, &logo, &b.AccountNumber, &b.AccountHolder, &inst); err == nil {
				if logo.Valid && logo.String != "" {
					b.LogoURL = &logo.String
				}
				if inst.Valid && inst.String != "" {
					b.Instructions = &inst.String
				}
				bankAccounts = append(bankAccounts, b)
			}
		}
		rows.Close()
	}

	// 13. Create Portal Token
	portalToken, err := identity.GeneratePortalToken(picJamaahID)
	if err != nil {
		return nil, fmt.Errorf("generate portal token: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}

	return &BookingResponse{
		Status: "success",
		Booking: BookingSummary{
			BookingCode:       bookingCode,
			TotalHarga:        totalHarga,
			MinimalDP:         totalMinDP,
			SeatHoldExpiresAt: seatHoldExpiresAt.Format(time.RFC3339),
			PaxSummary:        paxSummary,
		},
		Jamaah: JamaahInfo{
			ID:       picJamaahID,
			IDJamaah: idJamaahPIC,
		},
		PortalToken:  portalToken,
		BankAccounts: bankAccounts,
	}, nil
}

// GetInvoiceByCode mengambil data lengkap invoice digital berdasarkan kode booking.
func (r *Repository) GetInvoiceByCode(ctx context.Context, bookingCode string) (*InvoiceResponse, error) {
	bookingCode = strings.TrimSpace(strings.ToUpper(bookingCode))
	if bookingCode == "" {
		return nil, ErrNotFound
	}

	// 1. Get booking
	var bookingID, brandID, scheduleID int64
	var status string
	var totalHarga float64
	var createdAt, seatHoldExpiresAt time.Time

	err := r.db.QueryRowContext(ctx, `
		SELECT b.id, s.brand_id, b.schedule_id, b.status, b.total_harga, b.created_at, COALESCE(b.seat_hold_expires_at, b.created_at)
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.id_booking = ?
	`, bookingCode).Scan(&bookingID, &brandID, &scheduleID, &status, &totalHarga, &createdAt, &seatHoldExpiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("get booking: %w", err)
	}

	// 2. Get Brand info
	var brand InvoiceBrandInfo
	var logo, phone, wa, address, city, prov sql.NullString
	err = r.db.QueryRowContext(ctx, `
		SELECT id, name, COALESCE(legalitas, name), logo_url, primary_color, phone, whatsapp_number, address, city, province
		FROM brands WHERE id = ?
	`, brandID).Scan(&brand.ID, &brand.Name, &brand.PTName, &logo, &brand.PrimaryColor, &phone, &wa, &address, &city, &prov)
	if err == nil {
		if logo.Valid && logo.String != "" {
			brand.LogoURL = &logo.String
		}
		if phone.Valid && phone.String != "" {
			brand.Phone = &phone.String
		}
		if wa.Valid && wa.String != "" {
			brand.WhatsappNumber = &wa.String
		}
		if address.Valid && address.String != "" {
			brand.Alamat = &address.String
		}
		if city.Valid && city.String != "" {
			brand.City = &city.String
		}
		if prov.Valid && prov.String != "" {
			brand.Province = &prov.String
		}
	}
	brand.PPIUNumber = stringPtr("401/2020")
	brand.Akreditasi = stringPtr("A")

	// 3. Get Schedule info
	var schedule InvoiceScheduleInfo
	schedule.ID = scheduleID
	var airlineName, airlineLogo, hmName, hmdName sql.NullString
	var hmStar, hmdStar sql.NullInt64
	var berangkatTanggal, pulangTanggal time.Time
	var scheduleMinDP sql.NullFloat64

	err = r.db.QueryRowContext(ctx, `
		SELECT s.jadwal_nama, s.berangkat_tanggal, s.pulang_tanggal, s.minimal_dp,
		       a.name, a.logo_url,
		       hm.name, hm.star_rating,
		       hmd.name, hmd.star_rating
		FROM schedules s
		LEFT JOIN airlines a ON a.id = s.maskapai_id
		LEFT JOIN hotels hm ON hm.id = s.hotel_mekkah_id
		LEFT JOIN hotels hmd ON hmd.id = s.hotel_madinah_id
		WHERE s.id = ?
	`, scheduleID).Scan(
		&schedule.JadwalNama, &berangkatTanggal, &pulangTanggal, &scheduleMinDP,
		&airlineName, &airlineLogo,
		&hmName, &hmStar,
		&hmdName, &hmdStar,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get schedule: %w", err)
	}
	schedule.BerangkatTanggal = berangkatTanggal.Format("2006-01-02")
	schedule.PulangTanggal = pulangTanggal.Format("2006-01-02")

	if airlineName.Valid && airlineName.String != "" {
		schedule.Maskapai = &InvoiceMaskapaiInfo{
			Name: airlineName.String,
		}
		if airlineLogo.Valid && airlineLogo.String != "" {
			schedule.Maskapai.LogoURL = &airlineLogo.String
		}
	}
	if hmName.Valid && hmName.String != "" {
		if hmStar.Valid && hmStar.Int64 > 0 {
			schedule.HotelMekkah = fmt.Sprintf("%s (★%d)", hmName.String, hmStar.Int64)
		} else {
			schedule.HotelMekkah = hmName.String
		}
	}
	if hmdName.Valid && hmdName.String != "" {
		if hmdStar.Valid && hmdStar.Int64 > 0 {
			schedule.HotelMadinah = fmt.Sprintf("%s (★%d)", hmdName.String, hmdStar.Int64)
		} else {
			schedule.HotelMadinah = hmdName.String
		}
	}

	// 4. Get Pax items + PIC
	var pic InvoicePICInfo
	var paxItems []InvoicePaxItem
	var regulerPaxCount int

	paxRows, err := r.db.QueryContext(ctx, `
		SELECT j.nama_lengkap, COALESCE(j.no_hp, ''), bp.pax_type, bp.room_type, bp.harga_pax
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE bp.booking_id = ?
		ORDER BY bp.id ASC
	`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("get pax items: %w", err)
	}
	defer paxRows.Close()

	idx := 0
	for paxRows.Next() {
		var pName, pPhone, pType string
		var rType sql.NullString
		var pHarga float64

		if err := paxRows.Scan(&pName, &pPhone, &pType, &rType, &pHarga); err == nil {
			if idx == 0 {
				pic.NamaLengkap = pName
				pic.NoHPMasked = maskPhone(pPhone)
			}
			idx++

			roomLabel := "INFANT"
			if pType == "reguler" {
				regulerPaxCount++
				if rType.Valid && rType.String != "" {
					switch strings.ToLower(rType.String) {
					case "quad":
						roomLabel = "Quad"
					case "triple":
						roomLabel = "Triple"
					case "double":
						roomLabel = "Double"
					default:
						roomLabel = rType.String
					}
				} else {
					roomLabel = "Reguler"
				}
			}

			paxItems = append(paxItems, InvoicePaxItem{
				NamaLengkap: pName,
				PaxType:     pType,
				RoomType:    roomLabel,
				Harga:       pHarga,
			})
		}
	}

	// 5. Financial details
	var dpPerPax float64 = 5000000
	if scheduleMinDP.Valid && scheduleMinDP.Float64 > 0 {
		dpPerPax = scheduleMinDP.Float64
	}
	totalMinDP := dpPerPax * float64(regulerPaxCount)

	var totalDibayar float64
	_ = r.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(jumlah), 0) FROM payments WHERE booking_id = ? AND status = 'verified'
	`, bookingID).Scan(&totalDibayar)

	sisaTagihan := totalHarga - totalDibayar
	if sisaTagihan < 0 {
		sisaTagihan = 0
	}

	// Status label
	statusLabel := "Menunggu Pembayaran DP"
	if status == "dp" {
		statusLabel = "DP Terverifikasi"
	} else if status == "lunas" {
		statusLabel = "Lunas"
	} else if status == "batal" {
		statusLabel = "Dibatalkan"
	} else if time.Now().After(seatHoldExpiresAt) && status == "draft" {
		statusLabel = "Batas Pembayaran Berakhir"
	}

	// 6. Bank accounts
	var bankAccounts []BankAccountInfo
	bRows, err := r.db.QueryContext(ctx, `
		SELECT id, bank_name, logo_url, account_number, account_holder, instructions
		FROM bank_accounts
		WHERE brand_id = ? AND is_active = TRUE
		ORDER BY sort_order, id
	`, brandID)
	if err == nil {
		for bRows.Next() {
			var b BankAccountInfo
			var logo, inst sql.NullString
			if err := bRows.Scan(&b.ID, &b.BankName, &logo, &b.AccountNumber, &b.AccountHolder, &inst); err == nil {
				if logo.Valid && logo.String != "" {
					b.LogoURL = &logo.String
				}
				if inst.Valid && inst.String != "" {
					b.Instructions = &inst.String
				}
				bankAccounts = append(bankAccounts, b)
			}
		}
		bRows.Close()
	}

	// Pelunasan H-45
	hMin45 := berangkatTanggal.AddDate(0, 0, -45)
	jatuhTempoStr := fmt.Sprintf("H-45 Keberangkatan (%s)", hMin45.Format("02 Jan 2006"))

	return &InvoiceResponse{
		BookingCode:       bookingCode,
		Status:            status,
		StatusLabel:       statusLabel,
		CreatedAt:         createdAt.Format(time.RFC3339),
		SeatHoldExpiresAt: seatHoldExpiresAt.Format(time.RFC3339),
		Brand:             brand,
		Schedule:          schedule,
		PIC:               pic,
		PaxItems:          paxItems,
		Financial: InvoiceFinancial{
			TotalHarga:          totalHarga,
			MinimalDP:           totalMinDP,
			TotalDibayar:        totalDibayar,
			SisaTagihan:         sisaTagihan,
			JatuhTempoPelunasan: jatuhTempoStr,
		},
		BankAccounts: bankAccounts,
	}, nil
}

func maskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) <= 6 {
		return phone
	}
	prefix := phone[:4]
	suffix := phone[len(phone)-4:]
	return prefix + "••••" + suffix
}

func stringPtr(s string) *string {
	return &s
}
