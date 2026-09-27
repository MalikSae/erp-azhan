package selfbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
)

// Booking oleh agen untuk jamaahnya (Jalur 1, agen-azhan.md 3.1, screen A5).
// Endpoint terautentikasi portal, jadi tanpa captcha; tetap dibatasi per agen
// karena setiap booking menahan kursi 24 jam.

const agenBookingLimit = 20 // booking agen per agen per jam

// resolvePaxAgen menetapkan jamaah untuk satu pax booking agen (keputusan D8):
//   - jamaahID diisi: harus jamaah milik agen ini (daftar "Jamaah Saya");
//   - nomor HP sudah terdaftar: hanya boleh bila milik agen ini, selain itu
//     ditolak dengan pesan umum tanpa data pemilik;
//   - selain itu: jamaah baru dibuat (tanpa PIN), baru=true.
func resolvePaxAgen(ctx context.Context, tx *sql.Tx, brandID, agenID int64, jamaahID *int64, nama string, noHP *string, jenisKelamin string, email *string, tanggalLahir *string) (int64, bool, error) {
	if jamaahID != nil && *jamaahID > 0 {
		var owner sql.NullInt64
		err := tx.QueryRowContext(ctx,
			`SELECT direkrut_oleh_jamaah_id FROM jamaah WHERE id = ? AND brand_id = ? FOR UPDATE`,
			*jamaahID, brandID).Scan(&owner)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && owner.Int64 != agenID) {
			return 0, false, ErrJamaahBukanMilik
		}
		if err != nil {
			return 0, false, fmt.Errorf("booking agen cek jamaah: %w", err)
		}
		return *jamaahID, false, nil
	}

	phone := ""
	if noHP != nil {
		phone = strings.TrimSpace(*noHP)
	}
	var jk *string
	if jenisKelamin != "" {
		jk = &jenisKelamin
	}
	if phone == "" {
		id, err := insertJamaahTanpaHP(ctx, tx, brandID, nama, jk, tanggalLahir)
		return id, err == nil, err
	}

	canonical, local := shared.PhoneVariants(phone)
	var existingID int64
	var owner sql.NullInt64
	err := tx.QueryRowContext(ctx, `
		SELECT id, direkrut_oleh_jamaah_id FROM jamaah
		WHERE brand_id = ? AND REGEXP_REPLACE(COALESCE(no_hp,''),'[^0-9]','') IN (?,?)
		ORDER BY id LIMIT 1 FOR UPDATE`, brandID, canonical, local).Scan(&existingID, &owner)
	if err == nil {
		if owner.Int64 == agenID {
			return existingID, false, nil
		}
		return 0, false, ErrNomorMilikLain
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, false, fmt.Errorf("booking agen cek nomor: %w", err)
	}
	id, err := shared.ResolveJamaah(ctx, tx, brandID, shared.JamaahInput{
		NamaLengkap:  nama,
		NoHP:         phone,
		Email:        email,
		JenisKelamin: jk,
	})
	if err != nil {
		return 0, false, fmt.Errorf("booking agen buat jamaah: %w", err)
	}
	if tl := tanggalLahirOrNil(tanggalLahir); tl != nil {
		if _, err := tx.ExecContext(ctx, `UPDATE jamaah SET tanggal_lahir = ? WHERE id = ?`, tl, id); err != nil {
			return 0, false, fmt.Errorf("booking agen tanggal lahir: %w", err)
		}
	}
	return id, true, nil
}

func insertJamaahTanpaHP(ctx context.Context, tx *sql.Tx, brandID int64, nama string, jk *string, tanggalLahir *string) (int64, error) {
	var brandCode sql.NullString
	var counter uint64
	if err := tx.QueryRowContext(ctx, `SELECT kode_brand, jamaah_counter FROM brands WHERE id=? FOR UPDATE`, brandID).Scan(&brandCode, &counter); err != nil {
		return 0, fmt.Errorf("lock brand: %w", err)
	}
	if !brandCode.Valid || strings.TrimSpace(brandCode.String) == "" {
		return 0, shared.ErrBrandCodeMissing
	}
	counter++
	if _, err := tx.ExecContext(ctx, `UPDATE brands SET jamaah_counter=? WHERE id=?`, counter, brandID); err != nil {
		return 0, fmt.Errorf("update counter: %w", err)
	}
	idJamaah := fmt.Sprintf("%s-%02d%02d%06d", strings.ToUpper(strings.TrimSpace(brandCode.String)), time.Now().Year()%100, int(time.Now().Month()), counter)
	kodeJamaah, err := shared.UniqueCode(ctx, tx, "jamaah", "kode_jamaah", "", 6)
	if err != nil {
		return 0, fmt.Errorf("kode jamaah: %w", err)
	}
	res, err := tx.ExecContext(ctx,
		`INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap, no_hp, jenis_kelamin, tanggal_lahir) VALUES (?, ?, ?, ?, NULL, ?, ?)`,
		brandID, idJamaah, kodeJamaah, nama, jk, tanggalLahirOrNil(tanggalLahir))
	if err != nil {
		return 0, fmt.Errorf("insert jamaah: %w", err)
	}
	return res.LastInsertId()
}

// brandAgen mengembalikan brand agen. Booking agen selalu di brand agen,
// brand_id dari body diabaikan.
func (r *Repository) brandAgen(ctx context.Context, agenID int64) (int64, error) {
	var brandID int64
	err := r.db.QueryRowContext(ctx, `SELECT brand_id FROM jamaah WHERE id = ? AND status_agen = 'aktif'`, agenID).Scan(&brandID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrBukanAgenAktif
	}
	return brandID, err
}

// JamaahSayaItem adalah jamaah milik agen untuk dipilih saat repeat order.
type JamaahSayaItem struct {
	ID           int64   `json:"id"`
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHPMasked   *string `json:"no_hp_masked"`
	JenisKelamin *string `json:"jenis_kelamin"`
}

// ListJamaahSaya: jamaah yang terikat ke agen ini (direkrut_oleh = agen).
func (r *Repository) ListJamaahSaya(ctx context.Context, agenID int64, cari string) ([]JamaahSayaItem, error) {
	like := "%" + strings.TrimSpace(cari) + "%"
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, nama_lengkap, no_hp, jenis_kelamin FROM jamaah
		WHERE direkrut_oleh_jamaah_id = ? AND (? = '%%' OR nama_lengkap LIKE ?)
		ORDER BY nama_lengkap LIMIT 50`, agenID, like, like)
	if err != nil {
		return nil, fmt.Errorf("jamaah saya: %w", err)
	}
	defer rows.Close()
	items := []JamaahSayaItem{}
	for rows.Next() {
		var it JamaahSayaItem
		var hp, jk sql.NullString
		if err := rows.Scan(&it.ID, &it.NamaLengkap, &hp, &jk); err != nil {
			return nil, err
		}
		if hp.Valid && hp.String != "" {
			m := maskPhone(hp.String)
			it.NoHPMasked = &m
		}
		if jk.Valid {
			it.JenisKelamin = &jk.String
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

// ListJamaahSaya GET /api/portal/agen/jamaah
func (h *Handler) ListJamaahSaya(w http.ResponseWriter, r *http.Request) {
	agenID := identity.GetPortalJamaahID(r.Context())
	if _, err := h.repo.brandAgen(r.Context(), agenID); err != nil {
		h.writeAgenBookingError(w, err)
		return
	}
	items, err := h.repo.ListJamaahSaya(r.Context(), agenID, r.URL.Query().Get("q"))
	if err != nil {
		h.writeAgenBookingError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// CreateBookingAgen POST /api/portal/agen/bookings
func (h *Handler) CreateBookingAgen(w http.ResponseWriter, r *http.Request) {
	agenID := identity.GetPortalJamaahID(r.Context())
	limitKey := "agen:" + strconv.FormatInt(agenID, 10)
	if !h.bookingByAgen.allow(limitKey) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak booking, coba lagi dalam 1 jam")
		return
	}
	h.bookingByAgen.record(limitKey)

	brandID, err := h.repo.brandAgen(r.Context(), agenID)
	if err != nil {
		h.writeAgenBookingError(w, err)
		return
	}

	var req BookingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	if msg := validateAgenBooking(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	resp, err := h.repo.ProcessBookingAgen(r.Context(), brandID, req, agenID)
	if err != nil {
		h.writeAgenBookingError(w, err)
		return
	}
	log.Printf("[AUDIT] booking_agen agen_id=%d booking=%s", agenID, resp.Booking.BookingCode)
	writeJSON(w, http.StatusOK, resp)
}

// validateAgenBooking sama dengan gate booking publik, minus captcha dan PIN
// (jamaah baru mengaktifkan akun portalnya sendiri).
func validateAgenBooking(req *BookingRequest) string {
	if req.ScheduleID <= 0 {
		return "schedule_id wajib diisi"
	}
	req.PIC.NamaLengkap = strings.TrimSpace(req.PIC.NamaLengkap)
	req.PIC.NoHP = strings.TrimSpace(req.PIC.NoHP)
	req.PIC.JenisKelamin = strings.TrimSpace(req.PIC.JenisKelamin)
	req.PIC.RoomType = strings.TrimSpace(req.PIC.RoomType)
	req.PIC.PortalPIN = ""
	req.KodeReferral = ""
	if req.PIC.JamaahID == nil && (req.PIC.NamaLengkap == "" || req.PIC.NoHP == "" || req.PIC.JenisKelamin == "") {
		return "data pendaftar utama tidak lengkap"
	}
	if req.PIC.RoomType != "Quad" && req.PIC.RoomType != "Triple" && req.PIC.RoomType != "Double" {
		return "tipe kamar PIC tidak valid"
	}
	for i, a := range req.Anggota {
		req.Anggota[i].NamaLengkap = strings.TrimSpace(a.NamaLengkap)
		req.Anggota[i].JenisKelamin = strings.TrimSpace(a.JenisKelamin)
		req.Anggota[i].PaxType = strings.TrimSpace(a.PaxType)
		if a.JamaahID == nil && (req.Anggota[i].NamaLengkap == "" || req.Anggota[i].JenisKelamin == "") {
			return "data anggota tidak lengkap"
		}
		switch req.Anggota[i].PaxType {
		case "reguler":
			if a.RoomType == nil || (*a.RoomType != "Quad" && *a.RoomType != "Triple" && *a.RoomType != "Double") {
				return "tipe kamar reguler tidak valid"
			}
		case "infant":
			if a.TanggalLahir == nil || *a.TanggalLahir == "" {
				return "tanggal lahir wajib untuk infant"
			}
		default:
			return "pax_type anggota tidak valid"
		}
	}
	if 1+len(req.Anggota) > 9 {
		return "maksimal jamaah dalam satu booking adalah 9"
	}
	return ""
}

func (h *Handler) writeAgenBookingError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrBukanAgenAktif):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNomorMilikLain), errors.Is(err, ErrJamaahBukanMilik),
		errors.Is(err, ErrSeatHabis), errors.Is(err, ErrDuplicate),
		errors.Is(err, ErrDuplicatePaxInBooking), errors.Is(err, ErrCutoffBooking),
		errors.Is(err, ErrTanggalLahirTidakValid), errors.Is(err, ErrUsiaInfant):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "jadwal tidak ditemukan")
	default:
		log.Printf("[ERROR] booking agen: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
	}
}
