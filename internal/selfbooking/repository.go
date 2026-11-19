package selfbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

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

// inisiator: pemesan booking, selalu sebuah akun. Mode agen atau akun biasa ditentukan status akun itu di
// dalam transaksi (bacaPemesan), bukan oleh jalur yang dipanggil. Penerima komisi mengikuti kunci jamaah
// (komisi.ProcessBookingLunas).
type inisiator struct {
	accountID int64
}

// ProcessBooking membuat booking atas nama akun pelanggan yang sedang login, untuk semua jalur.
func (r *Repository) ProcessBooking(ctx context.Context, brandID, accountID int64, req BookingRequest) (*BookingResponse, error) {
	return r.processBooking(ctx, brandID, req, inisiator{accountID: accountID})
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
	asli := req // fingerprint idempotensi memakai request apa adanya, sebelum diisi server

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
	pem, err := bacaPemesan(ctx, tx, brandID, ini.accountID)
	if err != nil {
		return nil, err
	}
	ikut := !req.PemesanTidakIkut
	if pem.agen && ikut {
		if err := isiNIKPemesanAgen(ctx, tx, pem, &req); err != nil {
			return nil, err
		}
	}
	// Batas menurut status akun: agen aktif memakai batas hold seat agen, akun lain batas akun.
	if pem.agen {
		err = cekBatasSeatAgen(ctx, tx, pem.id, req.ScheduleID, seatDibutuhkan(req))
	} else {
		err = cekBatasAkun(ctx, tx, pem.id, req.ScheduleID, seatDibutuhkan(req), batasAkunDefault)
	}
	if err != nil {
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

	// 2. Penumpang pertama (PIC). Pemesan yang ikut berangkat dicocokkan lewat NIK dan nama (resolvePIC;
	// agen: NIK dari profil agen); pemesan yang tidak ikut memesankan jamaah pertama.
	if pem.id <= 0 {
		return nil, ErrAkunTidakValid
	}
	if req.PemesanTidakIkut {
		var hp *string
		if req.PIC.NoHP != "" {
			hp = &req.PIC.NoHP
		}
		picJamaahID, err = resolvePax(ctx, tx, brandID, pem, req.PIC.JamaahID, AnggotaInput{
			PaxType: "reguler", NamaLengkap: req.PIC.NamaLengkap, NoHP: hp, JenisKelamin: req.PIC.JenisKelamin, NIK: req.PIC.NIK, Nomor: 1,
		})
		if err != nil {
			return nil, err
		}
	} else {
		if picJamaahID, err = resolvePIC(ctx, tx, brandID, pem.id, req.PIC); err != nil {
			return nil, err
		}
		if pem.agen {
			diri, err := jamaahDiriAgen(ctx, tx, pem, picJamaahID)
			if err != nil {
				return nil, err
			}
			if !diri {
				return nil, ErrSayaSendiriBukanAnda
			}
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
		id, err := resolvePax(ctx, tx, brandID, pem, a.JamaahID, a)
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

	// Kunci jamaah lewat satu pintu (kunci.go); cookie referral tidak dibaca di sini.
	semuaJamaah := append([]int64{picJamaahID}, anggotaJamaahIDs...)
	if err := kunciJamaahPemesan(ctx, tx, pem, picJamaahID, ikut, semuaJamaah); err != nil {
		return nil, err
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
	`, bookingCode, req.ScheduleID, picJamaahID, regulerPaxCount, seatHoldExpiresAt, nullableID(ini.accountID), ikut, agenPembuat(pem))
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
	var sayaSendiri any // komisi langsung atas diri agen sendiri (K13), semua jalur
	if pem.agen && ikut {
		sayaSendiri = pem.id
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
	// Proses bayar: DP, lalu pembayaran bertahap dengan nominal bebas, lalu lunas paling lambat H-pelunasanHari (hari
	// itu masih termasuk). Booking yang dibuat dekat tanggal berangkat tetap boleh memakai DP dan pembayaran bertahap;
	// tenggatnya tidak pernah lebih awal dari hari ini, jadi booking mendesak punya sisa hari ini untuk melunasi.
	// Lewat tenggat tanpa lunas, booking dibatalkan otomatis (booking.BatalkanLewatBatasPelunasan).
	fullPayment := false
	hariIni := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dueAt := depDate.AddDate(0, 0, -pelunasanHari)
	if dueAt.Before(hariIni) {
		dueAt = hariIni
	}
	if req.ExpectedTotal != nil && (*req.ExpectedTotal != totalHarga || req.ExpectedDP == nil || *req.ExpectedDP != totalMinDP) {
		return nil, ErrQuoteChanged
	}

	// 12. Bank Accounts
	bankAccounts := []BankAccountInfo{}
	rows, err := tx.QueryContext(ctx, `SELECT id, bank_name, logo_url, account_number, account_holder, instructions FROM bank_accounts WHERE brand_id=? AND is_active=TRUE`, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var b BankAccountInfo
		var logo, inst sql.NullString
		if err := rows.Scan(&b.ID, &b.BankName, &logo, &b.AccountNumber, &b.AccountHolder, &inst); err != nil {
			return nil, err
		}
		if logo.Valid && logo.String != "" {
			b.LogoURL = &logo.String
		}
		if inst.Valid && inst.String != "" {
			b.Instructions = &inst.String
		}
		bankAccounts = append(bankAccounts, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close()

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
	// Jamaah yang dipakai ulang dari pihak lain: klaim pembuat yang belum pernah membayar gugur.
	if err := shared.GugurkanKlaimPembuatBooking(ctx, tx, bookingID); err != nil {
		return nil, err
	}
	var keyHash, fingerprint any
	if req.RequestKey != "" {
		keyHash = hashText(req.RequestKey)
		fingerprint = requestFingerprint(asli, ini)
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

// agenPembuat: isi bookings.dibuat_oleh_agen_akun_id, akun pemesan bila agen aktif (jalur mana pun), selain itu NULL.
func agenPembuat(p pemesan) any {
	if p.agen {
		return p.id
	}
	return nil
}
