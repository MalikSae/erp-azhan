package selfbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/shared"
)

var (
	ErrSeatHabis = errors.New("seat tidak cukup")
	ErrDuplicate = errors.New("anda sudah memiliki booking aktif di jadwal ini")
	ErrNotFound  = errors.New("data tidak ditemukan")
	// Nomor WhatsApp anggota dipakai jamaah lain dengan nama berbeda. Pesan generik (sama gayanya
	// dengan pesan NIK) agar tidak membuka keberadaan data pihak lain; percobaan dibatasi per akun.
	ErrAnggotaNameMismatch   = errors.New("data jamaah belum dapat diproses. Periksa kembali nama dan nomor WhatsApp anggota, kosongkan nomornya, atau hubungi kami via WhatsApp")
	ErrDuplicatePaxInBooking = errors.New("jamaah tidak boleh terdaftar lebih dari sekali dalam satu booking")
	ErrCutoffBooking         = errors.New("pendaftaran online untuk jadwal ini sudah ditutup (30 hari sebelum keberangkatan)")
	ErrAkunTidakValid        = errors.New("akun tidak valid untuk brand ini")
	// Booking agen (Jalur 1). Pesan sengaja umum: tidak membocorkan data
	// pemilik nomor yang sudah terdaftar (D8).
	ErrBukanAgenAktif   = errors.New("akun agen tidak aktif")
	ErrNomorMilikLain   = errors.New("nomor sudah terdaftar, minta jamaah booking sendiri atau hubungi admin")
	ErrJamaahBukanMilik = errors.New("jamaah tidak ada di daftar Jamaah Saya")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func normalizeName(s string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(s)), " "))
}

// inisiator menentukan siapa yang membuat booking. Booking publik (Jalur 2)
// dibuat pelanggan yang login (accountID): akun itu pemilik booking dan
// jamaah PIC-nya ditentukan dari akun. Booking agen (Jalur 1, agenID) dibuat
// agen untuk jamaahnya: agen bukan PIC, dan hanya jamaah yang dibuat di booking
// ini yang diikat ke agen (keputusan D8).
type inisiator struct {
	accountID int64
	agenID    int64
}

// ProcessBooking membuat booking atas nama akun pelanggan yang sedang login.
func (r *Repository) ProcessBooking(ctx context.Context, brandID, accountID int64, req BookingRequest) (*BookingResponse, error) {
	return r.processBooking(ctx, brandID, req, inisiator{accountID: accountID})
}

// ProcessBookingAgen POST /api/portal/agen/bookings (Jalur 1).
func (r *Repository) ProcessBookingAgen(ctx context.Context, brandID int64, req BookingRequest, agenID int64) (*BookingResponse, error) {
	return r.processBooking(ctx, brandID, req, inisiator{agenID: agenID})
}

func (r *Repository) processBooking(ctx context.Context, brandID int64, req BookingRequest, ini inisiator) (*BookingResponse, error) {
	// Serializable ensures atomic check-then-act
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()
	resp, err := processBookingTx(ctx, tx, brandID, req, ini)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	return resp, nil
}

func processBookingTx(ctx context.Context, tx *sql.Tx, brandID int64, req BookingRequest, ini inisiator) (*BookingResponse, error) {
	var err error
	modeAgen := ini.agenID > 0
	if modeAgen {
		if err := agen.ValidasiAgenAktif(ctx, tx, brandID, ini.agenID); err != nil {
			return nil, ErrBukanAgenAktif
		}
	}
	// Jamaah yang dibuat oleh booking agen ini; hanya mereka yang diikat (D8).
	var jamaahBaruAgen []int64

	// 1. Lock schedule
	var scheduleStatus string
	var scheduleBrandID int64
	var seatSisa int
	var hargaQuad, hargaTriple, hargaDouble float64
	var hargaInfant, scheduleMinDP sql.NullFloat64
	var berangkatTanggal time.Time

	err = tx.QueryRowContext(ctx, `
		SELECT brand_id, status, seat_sisa, harga_quad, harga_triple, harga_double, harga_infant, minimal_dp, berangkat_tanggal
		FROM schedules WHERE id=? FOR UPDATE`, req.ScheduleID).Scan(
		&scheduleBrandID, &scheduleStatus, &seatSisa,
		&hargaQuad, &hargaTriple, &hargaDouble, &hargaInfant, &scheduleMinDP, &berangkatTanggal,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("lock schedule: %w", err)
	}
	if scheduleBrandID != brandID {
		return nil, ErrNotFound
	}
	if replay, replayErr := replayCheckout(ctx, tx, req, ini); replayErr != nil || replay != nil {
		return replay, replayErr
	}
	if scheduleStatus != "published" {
		return nil, ErrNotFound
	}
	if ini.accountID > 0 {
		butuh := 1 // PIC
		for _, a := range req.Anggota {
			if a.PaxType == "reguler" {
				butuh++
			}
		}
		if err := cekBatasAkun(ctx, tx, ini.accountID, req.ScheduleID, butuh, batasAkunDefault); err != nil {
			return nil, err
		}
	}

	for _, pax := range req.Anggota {
		if pax.PaxType == "infant" && !hargaInfant.Valid {
			return nil, ErrInfantUnavailable
		}
	}
	// Cutoff H-30: online booking ditutup 30 hari sebelum tanggal keberangkatan
	now := time.Now().In(shared.LoadConfig().Location())
	cutoffDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, 30)
	depDate := time.Date(berangkatTanggal.Year(), berangkatTanggal.Month(), berangkatTanggal.Day(), 0, 0, 0, 0, now.Location())
	if depDate.Before(cutoffDate) {
		return nil, ErrCutoffBooking
	}
	if err := validasiTanggalLahirAnggota(req.Anggota, depDate, time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())); err != nil {
		return nil, err
	}

	var picJamaahID int64

	// 2a. Booking agen: PIC baru dibuat tanpa PIN (jamaah mengaktifkan akun
	// portalnya sendiri), atau dipilih dari "Jamaah Saya".
	if modeAgen {
		noHP := req.PIC.NoHP
		var baru bool
		picJamaahID, baru, err = resolvePaxAgen(ctx, tx, brandID, ini.agenID, req.PIC.JamaahID, req.PIC.NamaLengkap, &noHP, req.PIC.JenisKelamin, req.PIC.Email, nil)
		if err != nil {
			return nil, err
		}
		if baru {
			jamaahBaruAgen = append(jamaahBaruAgen, picJamaahID)
		}
	}

	// 2. Booking publik: PIC ditentukan dari akun pelanggan yang login.
	if ini.accountID > 0 {
		if req.PemesanTidakIkut {
			// Pemesan tidak berangkat: PIC booking = jamaah pertama, diperlakukan seperti anggota.
			var hp *string
			if req.PIC.NoHP != "" {
				hp = &req.PIC.NoHP
			}
			picJamaahID, err = resolveAnggotaPublik(ctx, tx, brandID, AnggotaInput{
				PaxType: "reguler", NamaLengkap: req.PIC.NamaLengkap, NoHP: hp, JenisKelamin: req.PIC.JenisKelamin, NIK: req.PIC.NIK, Nomor: 1,
			})
		} else {
			picJamaahID, err = resolveAkunPIC(ctx, tx, brandID, ini.accountID, req.PIC)
			if err == nil {
				err = pasangNIKPIC(ctx, tx, brandID, picJamaahID, nikBersih(req.PIC.NIK))
			}
		}
		if err != nil {
			return nil, err
		}
	} else if !modeAgen {
		return nil, ErrAkunTidakValid
	}

	// 3. Duplikasi check
	var dupCount int
	err = tx.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM bookings b
		JOIN booking_pax bp ON bp.booking_id = b.id
		WHERE b.schedule_id = ? AND bp.jamaah_id = ? AND bp.pax_status='aktif' AND b.status NOT IN ('batal', 'draft')
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

	// get id_jamaah pic
	var idJamaahPIC string
	err = tx.QueryRowContext(ctx, `SELECT id_jamaah FROM jamaah WHERE id=?`, picJamaahID).Scan(&idJamaahPIC)
	if err != nil {
		return nil, fmt.Errorf("get id jamaah: %w", err)
	}

	// 6. Create or reuse jamaah records for anggota
	anggotaJamaahIDs := make([]int64, len(req.Anggota))
	for i, a := range req.Anggota {
		if modeAgen {
			id, baru, err := resolvePaxAgen(ctx, tx, brandID, ini.agenID, a.JamaahID, a.NamaLengkap, a.NoHP, a.JenisKelamin, nil, a.TanggalLahir)
			if err != nil {
				return nil, err
			}
			if baru {
				jamaahBaruAgen = append(jamaahBaruAgen, id)
			}
			anggotaJamaahIDs[i] = id
			continue
		}
		a.Nomor = i + 2
		id, err := resolveAnggotaPublik(ctx, tx, brandID, a)
		if err != nil {
			return nil, err
		}
		anggotaJamaahIDs[i] = id
	}

	// Verifikasi tambahan: pastikan tidak ada duplikasi jamaah_id dalam satu booking
	allJamaahMap := make(map[int64]bool)
	allJamaahMap[picJamaahID] = true
	for _, aid := range anggotaJamaahIDs {
		if allJamaahMap[aid] {
			return nil, ErrDuplicatePaxInBooking
		}
		allJamaahMap[aid] = true
	}
	// The schedule lock serializes booking admission; check every resolved member.
	for i, aid := range anggotaJamaahIDs {
		var duplicate int
		if err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM booking_pax bp JOIN bookings b ON b.id=bp.booking_id WHERE b.schedule_id=? AND bp.jamaah_id=? AND bp.pax_status='aktif' AND b.status NOT IN ('batal','draft')`, req.ScheduleID, aid).Scan(&duplicate); err != nil {
			return nil, err
		}
		if duplicate > 0 {
			return nil, ErrDuplicate
		}
		if req.Anggota[i].PaxType == "infant" {
			var dob sql.NullTime
			if err = tx.QueryRowContext(ctx, `SELECT tanggal_lahir FROM jamaah WHERE id=?`, aid).Scan(&dob); err != nil {
				return nil, err
			}
			if !dob.Valid {
				return nil, ErrTanggalLahirTidakValid
			}
			stored := dob.Time.Format("2006-01-02")
			if req.Anggota[i].TanggalLahir == nil || stored != strings.TrimSpace(*req.Anggota[i].TanggalLahir) {
				return nil, ErrUsiaInfant
			}
			if err = validasiTanggalLahirAnggota([]AnggotaInput{{PaxType: "infant", TanggalLahir: &stored}}, depDate, now); err != nil {
				return nil, err
			}
		}
	}

	if modeAgen {
		// Jalur 1 (D8): hanya jamaah yang dibuat di booking ini yang diikat.
		for _, id := range jamaahBaruAgen {
			if err := agen.IkatJamaah(ctx, tx, id, ini.agenID, agen.SumberJalur1); err != nil {
				return nil, err
			}
		}
	} else {
		// Kaitan agen Jalur 2 (3.2–3.4): kode referral dari cookie, lalu auto-bind
		// seluruh pax yang belum punya kaitan ke agen booking ini.
		agenReferral, err := agen.ResolveKodeReferral(ctx, tx, brandID, req.KodeReferral)
		if err != nil {
			return nil, err
		}
		if err := agen.IkatRombongan(ctx, tx, picJamaahID, anggotaJamaahIDs, agenReferral); err != nil {
			return nil, err
		}
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
		INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, seat_count, status, is_seat_blocked, seat_hold_expires_at, customer_account_id)
		VALUES (?, ?, ?, ?, 'baru', TRUE, ?, ?)
	`, bookingCode, req.ScheduleID, picJamaahID, regulerPaxCount, seatHoldExpiresAt, nullableID(ini.accountID))
	if err != nil {
		return nil, fmt.Errorf("insert booking: %w", err)
	}
	bookingID, _ := res.LastInsertId()
	if err := komisi.SnapshotNominal(ctx, tx, bookingID); err != nil {
		return nil, fmt.Errorf("snapshot komisi: %w", err)
	}

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
	var pelunasanHari int
	err = tx.QueryRowContext(ctx, `SELECT minimal_dp, batas_pelunasan_hari FROM brands WHERE id=?`, brandID).Scan(&globalMinDP, &pelunasanHari)
	if err != nil {
		return nil, fmt.Errorf("get brand minimal dp: %w", err)
	}

	dpPerReguler := globalMinDP
	if scheduleMinDP.Valid {
		dpPerReguler = scheduleMinDP.Float64
	}
	if dpPerReguler < 0 || dpPerReguler > hargaQuad || dpPerReguler > hargaTriple || dpPerReguler > hargaDouble {
		return nil, ErrInvalidDP
	}
	totalMinDP := dpPerReguler * float64(regulerPaxCount)
	fullPayment := !depDate.After(now.AddDate(0, 0, pelunasanHari))
	dueAt := depDate.AddDate(0, 0, -pelunasanHari)
	if fullPayment {
		totalMinDP = totalHarga
		dueAt = seatHoldExpiresAt
	}
	if req.ExpectedTotal != nil && (*req.ExpectedTotal != totalHarga || req.ExpectedDP == nil || *req.ExpectedDP != totalMinDP) {
		return nil, ErrQuoteChanged
	}

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

	// 13. Simpan jamaah rombongan ke akun pemesan sebagai bahan prefill checkout berikutnya.
	if ini.accountID > 0 {
		if err := simpanTravelers(ctx, tx, ini.accountID, travelersUntukSimpan(req)); err != nil {
			return nil, err
		}
	}

	response := &BookingResponse{
		Status: "success",
		Booking: BookingSummary{
			BookingID:         bookingID,
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
		BankAccounts: bankAccounts,
	}
	invoiceToken, err := newInvoiceToken()
	if err != nil {
		return nil, err
	}
	response.Booking.InvoiceToken = invoiceToken
	raw, err := json.Marshal(struct {
		Response BookingResponse
	}{*response})
	if err != nil {
		return nil, err
	}
	var keyHash, fingerprint any
	if req.RequestKey != "" {
		keyHash = hashText(req.RequestKey)
		fingerprint = requestFingerprint(req, ini)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO booking_checkout(booking_id,dp_per_pax,full_payment,due_at,original_expires_at,invoice_token,terms_version,accepted_at,request_key_hash,request_hash,response_json) VALUES(?,?,?,?,?,?,?,NOW(),?,?,?)`, bookingID, dpPerReguler, fullPayment, dueAt, seatHoldExpiresAt, invoiceToken, req.TermsVersion, keyHash, fingerprint, raw)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func maskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) <= 8 {
		return "••••"
	}
	prefix := phone[:4]
	suffix := phone[len(phone)-4:]
	return prefix + "••••" + suffix
}

func stringPtr(s string) *string {
	return &s
}

// resolveAnggotaPublik mencari atau membuat jamaah untuk anggota booking publik.
// Nomor WhatsApp yang sudah terdaftar dipakai ulang hanya bila namanya cocok.
func resolveAnggotaPublik(ctx context.Context, tx *sql.Tx, brandID int64, a AnggotaInput) (int64, error) {
	nik := nikBersih(a.NIK)
	if nik != "" {
		// NIK sudah terdaftar: pakai ulang bila nama cocok, tolak bila beda.
		id, nama, ada, err := cariJamaahByNIK(ctx, tx, brandID, nik)
		if err != nil {
			return 0, err
		}
		if ada {
			if normalizeName(a.NamaLengkap) != normalizeName(nama) {
				return 0, errNIKBeda(a.Nomor)
			}
			return id, nil
		}
	}
	id, err := resolveAnggotaTanpaNIK(ctx, tx, brandID, a)
	if err != nil {
		return 0, err
	}
	if nik != "" {
		if err := isiNIKBilaKosong(ctx, tx, id, nik); err != nil {
			return 0, err
		}
	}
	return id, nil
}

func resolveAnggotaTanpaNIK(ctx context.Context, tx *sql.Tx, brandID int64, a AnggotaInput) (int64, error) {
	var err error
	var rawPhone string
	if a.NoHP != nil {
		rawPhone = strings.TrimSpace(*a.NoHP)
	}

	if rawPhone == "" {
		// No HP kosong (NULL) -> Selalu INSERT baru dengan no_hp NULL
		var brandCode string
		var counter uint64
		err = tx.QueryRowContext(ctx, `SELECT kode_brand, jamaah_counter FROM brands WHERE id=? FOR UPDATE`, brandID).Scan(&brandCode, &counter)
		if err != nil {
			return 0, fmt.Errorf("lock brand: %w", err)
		}
		counter++
		_, err = tx.ExecContext(ctx, `UPDATE brands SET jamaah_counter=? WHERE id=?`, counter, brandID)
		if err != nil {
			return 0, fmt.Errorf("update counter: %w", err)
		}
		idJamaah := fmt.Sprintf("%s-%02d%02d%06d", strings.ToUpper(strings.TrimSpace(brandCode)), time.Now().Year()%100, int(time.Now().Month()), counter)
		kodeJamaah, err := shared.UniqueCode(ctx, tx, "jamaah", "kode_jamaah", "", 6)
		if err != nil {
			return 0, fmt.Errorf("kode jamaah: %w", err)
		}

		res, err := tx.ExecContext(ctx, `
		INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap, no_hp, jenis_kelamin, tanggal_lahir)
		VALUES (?, ?, ?, ?, NULL, ?, ?)
	`, brandID, idJamaah, kodeJamaah, a.NamaLengkap, a.JenisKelamin, tanggalLahirOrNil(a.TanggalLahir))
		if err != nil {
			return 0, fmt.Errorf("insert anggota jamaah: %w", err)
		}
		anggotaID, _ := res.LastInsertId()
		return anggotaID, nil
	} else {
		// No HP terisi -> Cari apakah sudah terdaftar
		canonicalAnggota, localAnggota := shared.PhoneVariants(rawPhone)
		var existingAnggotaID int64
		var existingAnggotaNama string

		err = tx.QueryRowContext(ctx, `
		SELECT id, nama_lengkap 
		FROM jamaah 
		WHERE brand_id = ? AND REGEXP_REPLACE(COALESCE(no_hp,''),'[^0-9]','') IN (?,?) 
		ORDER BY id LIMIT 1 FOR UPDATE
	`, brandID, canonicalAnggota, localAnggota).Scan(&existingAnggotaID, &existingAnggotaNama)

		if errors.Is(err, sql.ErrNoRows) {
			// Belum terdaftar -> INSERT baru dengan no_hp
			var brandCode string
			var counter uint64
			err = tx.QueryRowContext(ctx, `SELECT kode_brand, jamaah_counter FROM brands WHERE id=? FOR UPDATE`, brandID).Scan(&brandCode, &counter)
			if err != nil {
				return 0, fmt.Errorf("lock brand: %w", err)
			}
			counter++
			_, err = tx.ExecContext(ctx, `UPDATE brands SET jamaah_counter=? WHERE id=?`, counter, brandID)
			if err != nil {
				return 0, fmt.Errorf("update counter: %w", err)
			}
			idJamaah := fmt.Sprintf("%s-%02d%02d%06d", strings.ToUpper(strings.TrimSpace(brandCode)), time.Now().Year()%100, int(time.Now().Month()), counter)
			kodeJamaah, err := shared.UniqueCode(ctx, tx, "jamaah", "kode_jamaah", "", 6)
			if err != nil {
				return 0, fmt.Errorf("kode jamaah: %w", err)
			}

			res, err := tx.ExecContext(ctx, `
			INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap, no_hp, jenis_kelamin, tanggal_lahir)
			VALUES (?, ?, ?, ?, ?, ?, ?)
		`, brandID, idJamaah, kodeJamaah, a.NamaLengkap, rawPhone, a.JenisKelamin, tanggalLahirOrNil(a.TanggalLahir))
			if err != nil {
				return 0, fmt.Errorf("insert anggota jamaah: %w", err)
			}
			anggotaID, _ := res.LastInsertId()
			return anggotaID, nil
		} else if err != nil {
			return 0, fmt.Errorf("find anggota: %w", err)
		} else {
			// Sudah terdaftar -> Verifikasi Nama
			if normalizeName(a.NamaLengkap) != normalizeName(existingAnggotaNama) {
				return 0, ErrAnggotaNameMismatch
			}
			// Cocok -> pakai ulang baris jamaah itu, JANGAN UPDATE field apa pun
			return existingAnggotaID, nil
		}
	}
}

// travelersUntukSimpan: bila pemesan tidak ikut, jamaah pertama (di data "pic") juga disimpan.
func travelersUntukSimpan(req BookingRequest) []AnggotaInput {
	if !req.PemesanTidakIkut {
		return req.Anggota
	}
	var hp *string
	if req.PIC.NoHP != "" {
		hp = &req.PIC.NoHP
	}
	pertama := AnggotaInput{PaxType: "reguler", NamaLengkap: req.PIC.NamaLengkap, NoHP: hp, JenisKelamin: req.PIC.JenisKelamin, NIK: req.PIC.NIK}
	return append([]AnggotaInput{pertama}, req.Anggota...)
}
