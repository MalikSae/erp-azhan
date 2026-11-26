package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
)

// Daftar "Jamaah Saya" untuk agen. Booking agen sendiri lewat jalur inti (handler.go, CreateBookingAgen).

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

// maksJamaahSaya: batas daftar Jamaah Saya. Pemilih di web memuat semuanya sekali lalu menyaring di klien,
// jadi batasnya harus jauh di atas jumlah jamaah satu agen (dulu 50: jamaah ke-51 tidak bisa dipilih).
const maksJamaahSaya = 1000

// ListJamaahSaya: jamaah yang terkunci ke agen ini (direkrut_oleh_akun_id = agen).
func (r *Repository) ListJamaahSaya(ctx context.Context, agenAkunID int64, cari string) ([]JamaahSayaItem, error) {
	return listJamaahSaya(ctx, r.db, agenAkunID, cari)
}

// listJamaahSaya hanya memuat jamaah yang datanya boleh dilihat agen (aturan akses data jamaah minimal DP, audit
// keamanan 2026-11-26): data yang dibuat agen sendiri, jamaah dari booking yang dibuat agen, atau jamaah dengan booking
// minimal DP. Anggota rombongan prospek yang belum DP baru muncul setelah DP, sama dengan menu Jamaah agen.
func listJamaahSaya(ctx context.Context, q interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}, agenAkunID int64, cari string) ([]JamaahSayaItem, error) {
	like := "%" + strings.TrimSpace(cari) + "%"
	rows, err := q.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, j.no_hp, j.jenis_kelamin FROM jamaah j
		WHERE j.direkrut_oleh_akun_id = ? AND (? = '%%' OR j.nama_lengkap LIKE ?)
		  AND (j.dibuat_oleh_akun_id = ? OR EXISTS (
		        SELECT 1 FROM booking_pax bp JOIN bookings b ON b.id = bp.booking_id
		        WHERE bp.jamaah_id = j.id AND bp.pax_status = 'aktif'
		          AND (b.status IN ('dp', 'lunas') OR (b.dibuat_oleh_agen_akun_id = ? AND b.status IN ('baru', 'dp', 'lunas')))))
		ORDER BY j.nama_lengkap LIMIT ?`, agenAkunID, like, like, agenAkunID, agenAkunID, maksJamaahSaya)
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
		tulisGalatBooking(w, err, true)
		return
	}
	items, err := h.repo.ListJamaahSaya(r.Context(), agenID, r.URL.Query().Get("q"))
	if err != nil {
		tulisGalatBooking(w, err, true)
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
}
