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

	"erp-azhan/api/internal/identity"
)

// Booking oleh agen untuk jamaahnya (portal agen, spek bagian 8, screen A5).
// Endpoint terautentikasi akun agen, jadi tanpa captcha; tetap dibatasi per agen
// karena setiap booking menahan kursi 24 jam.

const agenBookingLimit = 20 // booking agen per agen per jam

// pesanJamaahTerkunci adalah peringatan booking agen untuk jamaah yang sudah terkunci ke agen lain (K16).
// Tidak menyebut pemilik kunci maupun data jamaah; hanya muncul bila NIK dan nama cocok.
const pesanJamaahTerkunci = "Jamaah %d sudah terikat agen lain. Booking ini tidak menghasilkan komisi untuk Anda."

// resolvePaxAgen menetapkan jamaah untuk satu pax booking agen. Jamaah dikenali dari NIK seperti di web
// publik (spek 4.3): NIK sama dan nama sama memakai ulang jamaah lama, NIK sama dengan nama berbeda ditolak
// generik, NIK baru membuat jamaah baru. Bayi tanpa NIK dicocokkan lewat nama dan tanggal lahir.
//
// jamaahID (pilihan "Jamaah Saya") hanya jalan pintas: jamaah itu tetap harus cocok dengan NIK dan nama
// yang dikirim, jadi id yang ditebak tidak membuka apa pun. Jamaah yang terkunci ke agen lain tidak lagi
// ditolak: terkunci=true dan pemanggil memberi peringatan, komisi tetap ke pemilik kunci.
func resolvePaxAgen(ctx context.Context, tx *sql.Tx, brandID, agenAkunID int64, jamaahID *int64, a AnggotaInput) (id int64, terkunci bool, err error) {
	if jamaahID != nil && *jamaahID > 0 {
		var nik, nama sql.NullString
		err = tx.QueryRowContext(ctx, `SELECT nik, nama_lengkap FROM jamaah WHERE id = ? AND brand_id = ? FOR UPDATE`,
			*jamaahID, brandID).Scan(&nik, &nama)
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, ErrJamaahBukanMilik
		}
		if err != nil {
			return 0, false, fmt.Errorf("booking agen cek jamaah: %w", err)
		}
		// Jamaah ber-NIK: NIK harus sama. Jamaah tanpa NIK (bayi): cukup nama yang sama.
		if (nik.String != "" && nikBersih(a.NIK) != nik.String) || normalizeName(a.NamaLengkap) != normalizeName(nama.String) {
			return 0, false, ErrJamaahBukanMilik
		}
		id = *jamaahID
	} else {
		id, err = resolveAnggotaPublik(ctx, tx, brandID, a)
		if err != nil {
			return 0, false, err
		}
	}
	var pemilik sql.NullInt64
	if err = tx.QueryRowContext(ctx, `SELECT direkrut_oleh_akun_id FROM jamaah WHERE id = ?`, id).Scan(&pemilik); err != nil {
		return 0, false, fmt.Errorf("booking agen cek kunci: %w", err)
	}
	return id, pemilik.Valid && pemilik.Int64 != agenAkunID, nil
}

// brandAgen mengembalikan brand agen. Booking agen selalu di brand agen,
// brand_id dari body diabaikan.
func (r *Repository) brandAgen(ctx context.Context, agenAkunID int64) (int64, error) {
	var brandID int64
	err := r.db.QueryRowContext(ctx, `SELECT brand_id FROM customer_accounts WHERE id = ? AND status_agen = 'aktif'`, agenAkunID).Scan(&brandID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrBukanAgenAktif
	}
	return brandID, err
}

// JamaahSayaItem adalah jamaah milik agen untuk dipilih saat booking ulang.
type JamaahSayaItem struct {
	ID           int64   `json:"id"`
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHPMasked   *string `json:"no_hp_masked"`
	JenisKelamin *string `json:"jenis_kelamin"`
}

// ListJamaahSaya: jamaah yang terkunci ke agen ini (direkrut_oleh_akun_id = agen).
func (r *Repository) ListJamaahSaya(ctx context.Context, agenAkunID int64, cari string) ([]JamaahSayaItem, error) {
	like := "%" + strings.TrimSpace(cari) + "%"
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, nama_lengkap, no_hp, jenis_kelamin FROM jamaah
		WHERE direkrut_oleh_akun_id = ? AND (? = '%%' OR nama_lengkap LIKE ?)
		ORDER BY nama_lengkap LIMIT 50`, agenAkunID, like, like)
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
	agenID := identity.GetCustomerAccountID(r.Context())
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
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	agenID := identity.GetCustomerAccountID(r.Context())
	limitKey := "agen:" + strconv.FormatInt(agenID, 10)
	if !h.bookingByAgen.allow(limitKey) {
		writeError(w, http.StatusTooManyRequests, "terlalu banyak booking, coba lagi dalam 1 jam")
		return
	}
	h.bookingByAgen.record(limitKey)
	// Percobaan NIK yang gagal dibatasi per akun, sama seperti web publik (jangan jadi oracle data jamaah).
	if h.identitasGagal != nil && !h.identitasGagal.allow(limitKey) {
		writeErrorCode(w, http.StatusTooManyRequests, "Terlalu banyak percobaan dengan data jamaah yang belum dapat diproses. Coba lagi dalam 10 menit atau hubungi admin.", KodeIdentitasTerlaluSering)
		return
	}

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
	if msg := validateCheckout(&req, true); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	resp, err := h.repo.ProcessBookingAgen(r.Context(), brandID, req, agenID)
	if err != nil {
		if errors.Is(err, ErrNIKAtasNamaLain) || errors.Is(err, ErrJamaahBukanMilik) {
			if h.identitasGagal != nil {
				h.identitasGagal.record(limitKey)
			}
		}
		h.writeAgenBookingError(w, err)
		return
	}
	log.Printf("[AUDIT] booking_agen agen_id=%d booking=%s", agenID, resp.Booking.BookingCode)
	writeJSON(w, http.StatusOK, resp)
}

// validateAgenBooking sama dengan gate booking publik, minus captcha. Nomor WhatsApp jamaah hanya kontak
// (spek 4.3), jadi tidak wajib; identitas jamaah ada di NIK (validateCheckout).
func validateAgenBooking(req *BookingRequest) string {
	if req.ScheduleID <= 0 {
		return "schedule_id wajib diisi"
	}
	req.PIC.NamaLengkap = strings.TrimSpace(req.PIC.NamaLengkap)
	req.PIC.NoHP = strings.TrimSpace(req.PIC.NoHP)
	req.PIC.JenisKelamin = strings.TrimSpace(req.PIC.JenisKelamin)
	req.PIC.RoomType = strings.TrimSpace(req.PIC.RoomType)
	if req.PIC.JamaahID == nil && (req.PIC.NamaLengkap == "" || req.PIC.JenisKelamin == "") {
		return "data jamaah utama tidak lengkap"
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
	case errors.Is(err, ErrQuoteChanged), errors.Is(err, ErrRequestConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrBukanAgenAktif):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrJamaahBukanMilik), errors.Is(err, ErrNIKAtasNamaLain),
		errors.Is(err, ErrSeatHabis), errors.Is(err, ErrDuplicate),
		errors.Is(err, ErrDuplicatePaxInBooking), errors.Is(err, ErrCutoffBooking),
		errors.Is(err, ErrTanggalLahirTidakValid), errors.Is(err, ErrUsiaInfant), errors.Is(err, ErrInfantUnavailable), errors.Is(err, ErrInvalidDP):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "jadwal tidak ditemukan")
	default:
		log.Printf("[ERROR] booking agen: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
	}
}
