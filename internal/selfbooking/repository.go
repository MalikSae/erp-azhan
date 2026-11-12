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
	ErrDuplicatePaxInBooking = errors.New("jamaah tidak boleh terdaftar lebih dari sekali dalam satu booking")
	ErrCutoffBooking         = errors.New("pendaftaran online untuk jadwal ini sudah ditutup (30 hari sebelum keberangkatan)")
	ErrAkunTidakValid        = errors.New("akun tidak valid untuk brand ini")
	// Booking agen (portal agen). Pesan sengaja umum: tidak membocorkan data jamaah atau pemilik kuncinya.
	ErrBukanAgenAktif   = errors.New("akun agen tidak aktif")
	ErrJamaahBukanMilik = errors.New("data jamaah belum dapat diproses. Periksa kembali NIK dan nama sesuai KTP")
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

// inisiator menentukan siapa yang membuat booking. Pemesan selalu sebuah akun (accountID). Booking publik
// (web, lewat link ref) dibuat akun biasa; booking portal agen diberi agenID = accountID: akun agen aktif
// itu pemesan sekaligus agen pembuat booking, tanpa captcha. Agen penentu komisi bukan inisiator: komisi
// mengikuti kunci jamaah (komisi.ProcessBookingLunas).
type inisiator struct {
	accountID int64
	agenID    int64
}

// ProcessBooking membuat booking atas nama akun pelanggan yang sedang login.
func (r *Repository) ProcessBooking(ctx context.Context, brandID, accountID int64, req BookingRequest) (*BookingResponse, error) {
	return r.processBooking(ctx, brandID, req, inisiator{accountID: accountID})
}

// ProcessBookingAgen POST /api/portal/agen/bookings (portal agen).
func (r *Repository) ProcessBookingAgen(ctx context.Context, brandID int64, req BookingRequest, agenID int64) (*BookingResponse, error) {
	return r.processBooking(ctx, brandID, req, inisiator{accountID: agenID, agenID: agenID})
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
	// Peringatan booking agen: jamaah yang sudah terkunci ke agen lain (komisi ke pemilik kunci).
	var peringatan []string

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
	if !modeAgen {
		butuh := 1 // PIC
		for _, a := range req.Anggota {
			if a.PaxType == "reguler" {
				butuh++
			}
		}
		if err := cekBatasAkun(ctx, tx, ini.accountID, req.ScheduleID, butuh, batasAkunDefault); err != nil {
			return nil, err
		}
	} else if err := cekBatasSeatAgen(ctx, tx, ini.agenID, req.ScheduleID, seatDibutuhkan(req)); err != nil {
		return nil, err
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

	// 2. Penumpang pertama (PIC). Booking agen: dikenali dari NIK seperti penumpang lain; "Saya sendiri"
	// menandai penumpang itu sebagai agen sendiri. Booking publik: pemesan yang ikut berangkat dicocokkan
	// lewat NIK dan nama (resolvePIC); pemesan yang tidak ikut memesankan jamaah pertama.
	if ini.accountID <= 0 {
		return nil, ErrAkunTidakValid
	}
	if modeAgen {
		var hp *string
		if req.PIC.NoHP != "" {
			hp = &req.PIC.NoHP
		}
		var terkunci bool
		picJamaahID, terkunci, err = resolvePaxAgen(ctx, tx, brandID, ini.agenID, req.PIC.JamaahID, AnggotaInput{
			PaxType: "reguler", NamaLengkap: req.PIC.NamaLengkap, NoHP: hp, JenisKelamin: req.PIC.JenisKelamin, NIK: req.PIC.NIK, Nomor: 1,
		})
		if err != nil {
			return nil, err
		}
		if req.PIC.SayaSendiri {
			if err := pastikanSayaSendiri(ctx, tx, ini.agenID, picJamaahID); err != nil {
				return nil, err
			}
		} else if terkunci {
			peringatan = append(peringatan, fmt.Sprintf(pesanJamaahTerkunci, 1))
		}
	} else if req.PemesanTidakIkut {
		// Pemesan tidak berangkat: PIC booking = jamaah pertama, diperlakukan seperti anggota.
		var hp *string
		if req.PIC.NoHP != "" {
			hp = &req.PIC.NoHP
		}
		picJamaahID, err = resolveAnggotaPublik(ctx, tx, brandID, ini.accountID, AnggotaInput{
			PaxType: "reguler", NamaLengkap: req.PIC.NamaLengkap, NoHP: hp, JenisKelamin: req.PIC.JenisKelamin, NIK: req.PIC.NIK, Nomor: 1,
		})
		if err != nil {
			return nil, err
		}
	} else {
		if picJamaahID, err = resolvePIC(ctx, tx, brandID, ini.accountID, req.PIC); err != nil {
			return nil, err
		}
		// Pemesan ikut berangkat: bila jamaahnya (NIK cocok) sudah dikunci agen X dan akunnya belum punya pengikat, akun menjadi downline X.
		if err := agen.IkatAkunDariJamaah(ctx, tx, ini.accountID, picJamaahID); err != nil {
			return nil, err
		}
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
		a.Nomor = i + 2
		if modeAgen {
			id, terkunci, err := resolvePaxAgen(ctx, tx, brandID, ini.agenID, a.JamaahID, a)
			if err != nil {
				return nil, err
			}
			if terkunci {
				peringatan = append(peringatan, fmt.Sprintf(pesanJamaahTerkunci, a.Nomor))
			}
			anggotaJamaahIDs[i] = id
			continue
		}
		id, err := resolveAnggotaPublik(ctx, tx, brandID, ini.accountID, a)
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

	// Kunci jamaah ke agen (spek 4.2, first-set-wins: hanya jamaah belum_ditentukan yang diikat). Portal agen:
	// agen pembuat booking. Web publik: agen yang mengikat akun pemesan; cookie referral tidak dibaca di sini.
	// Tanpa keduanya, jamaah tetap belum_ditentukan.
	semuaJamaah := append([]int64{picJamaahID}, anggotaJamaahIDs...)
	if modeAgen {
		if err := ikatJamaahAgen(ctx, tx, semuaJamaah, ini.agenID); err != nil {
			return nil, err
		}
	} else {
		agenAkun, err := agen.UplineAkun(ctx, tx, ini.accountID)
		if err != nil {
			return nil, err
		}
		if err := agen.IkatRombongan(ctx, tx, semuaJamaah, agenAkun, agen.SumberReferral); err != nil {
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
		INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, seat_count, status, is_seat_blocked, seat_hold_expires_at, customer_account_id, pemesan_ikut, dibuat_oleh_agen_akun_id)
		VALUES (?, ?, ?, ?, 'baru', TRUE, ?, ?, ?, ?)
	`, bookingCode, req.ScheduleID, picJamaahID, regulerPaxCount, seatHoldExpiresAt, nullableID(ini.accountID), pemesanIkut(modeAgen, req), nullableID(ini.agenID))
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
	var sayaSendiri any // komisi langsung atas diri agen sendiri (K13), hanya portal agen
	if modeAgen && req.PIC.SayaSendiri {
		sayaSendiri = ini.agenID
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO booking_pax (booking_id, jamaah_id, pax_type, room_type, harga_pax, counts_for_seat, pax_status, saya_sendiri_akun_id)
		VALUES (?, ?, 'reguler', ?, ?, TRUE, 'aktif', ?)
	`, bookingID, picJamaahID, req.PIC.RoomType, picHarga, sayaSendiri)
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
		Peringatan:   peringatan,
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
	// Jamaah yang dipakai ulang dari pihak lain: klaim pembuat yang belum pernah membayar gugur.
	if err := shared.GugurkanKlaimPembuatBooking(ctx, tx, bookingID); err != nil {
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
// Jamaah dikenali dari NIK (nama harus cocok); tanpa NIK (bayi) dari nama dan tanggal lahir.
// pembuatAkunID: akun pemesan; dicatat sebagai pembuat hanya bila jamaah baru dibuat di sini (0 = tidak dicatat).
//
// NIK yang belum terdaftar selalu membuat jamaah baru. Jamaah lama TIDAK dicocokkan lewat nama + tanggal lahir
// bila NIK diisi: nama dan tanggal lahir mudah diketahui orang lain, sehingga pencocokan itu membuat siapa pun bisa
// mengklaim jamaah orang lain (mengikatnya ke agen, mengisi NIK palsu, dan mendapat akses terbatas ke datanya).
func resolveAnggotaPublik(ctx context.Context, tx *sql.Tx, brandID, pembuatAkunID int64, a AnggotaInput) (int64, error) {
	nik := nikBersih(a.NIK)
	if nik == "" {
		return resolveAnggotaTanpaNIK(ctx, tx, brandID, pembuatAkunID, a)
	}
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
	id, err = buatAnggotaBaru(ctx, tx, brandID, pembuatAkunID, a, tanggalLahirOrNil(a.TanggalLahir))
	if err != nil {
		return 0, err
	}
	if err := isiNIKBilaKosong(ctx, tx, id, nik); err != nil {
		return 0, err
	}
	return id, nil
}

// resolveAnggotaTanpaNIK mencari atau membuat jamaah untuk pax tanpa NIK (bayi). Nomor WhatsApp bukan kunci
// identitas (spek 4.3): bayi dicocokkan lewat nama dan tanggal lahir; bila tidak ada yang cocok, jamaah baru
// dibuat dengan nomor sebagai kontak biasa. Jamaah lama yang cocok dipakai ulang tanpa menimpa data apa pun.
// Hanya jamaah yang belum punya NIK yang boleh cocok: jamaah ber-NIK hanya dikenali lewat NIK-nya.
func resolveAnggotaTanpaNIK(ctx context.Context, tx *sql.Tx, brandID, pembuatAkunID int64, a AnggotaInput) (int64, error) {
	dob := tanggalLahirOrNil(a.TanggalLahir)
	if dob != nil {
		rows, err := tx.QueryContext(ctx, `SELECT id, nama_lengkap FROM jamaah WHERE brand_id=? AND tanggal_lahir=? AND (nik IS NULL OR nik='') ORDER BY id FOR UPDATE`, brandID, dob)
		if err != nil {
			return 0, fmt.Errorf("cari jamaah by nama dan tanggal lahir: %w", err)
		}
		var cocok int64
		for rows.Next() {
			var id int64
			var nama string
			if err := rows.Scan(&id, &nama); err != nil {
				rows.Close()
				return 0, err
			}
			if cocok == 0 && normalizeName(nama) == normalizeName(a.NamaLengkap) {
				cocok = id
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return 0, err
		}
		if cocok > 0 {
			return cocok, nil
		}
	}
	return buatAnggotaBaru(ctx, tx, brandID, pembuatAkunID, a, dob)
}

// buatAnggotaBaru membuat jamaah baru untuk anggota booking publik, mencatat akun pembuat dan tanggal lahir.
func buatAnggotaBaru(ctx context.Context, tx *sql.Tx, brandID, pembuatAkunID int64, a AnggotaInput, dob any) (int64, error) {
	var hp string
	if a.NoHP != nil {
		hp = strings.TrimSpace(*a.NoHP)
	}
	jk := a.JenisKelamin
	id, err := shared.CreateJamaah(ctx, tx, brandID, shared.JamaahInput{NamaLengkap: a.NamaLengkap, NoHP: hp, JenisKelamin: &jk})
	if err != nil {
		return 0, fmt.Errorf("insert anggota jamaah: %w", err)
	}
	// Akun pemesan menjadi pembuat jamaah baru: dasar akses penuh atas data jamaah ini di portal.
	if pembuatAkunID > 0 {
		if _, err := tx.ExecContext(ctx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id=?`, pembuatAkunID, id); err != nil {
			return 0, fmt.Errorf("catat pembuat jamaah: %w", err)
		}
	}
	if dob != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE jamaah SET tanggal_lahir=? WHERE id=?`, dob, id); err != nil {
			return 0, fmt.Errorf("tanggal lahir anggota: %w", err)
		}
	}
	return id, nil
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

// pemesanIkut: apakah pemilik akun yang memesan ikut berangkat. Disimpan di bookings.pemesan_ikut supaya
// detail booking tidak perlu menebaknya. Booking agen: ikut hanya bila jamaah pertama "Saya sendiri".
// Booking publik: ikut kecuali pemesan memilih memesankan untuk orang lain.
func pemesanIkut(modeAgen bool, req BookingRequest) bool {
	if modeAgen {
		return req.PIC.SayaSendiri
	}
	return !req.PemesanTidakIkut
}
