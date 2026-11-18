package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"erp-azhan/api/internal/identity"
)

// Aksi admin untuk aktivasi akun jamaah: Lepas Tautan Jamaah (satu-satunya jalan memindahkan tautan), Kosongkan NIK
// Agen (NIK jamaah tertaut yang dipegang akun lain), dan daftar agen dari tautan untuk audit.

var (
	// ErrAgenAktif: akun agen aktif atau nonaktif tidak boleh dilepas tautannya (nonaktifkan agen dulu).
	ErrAgenAktif = errors.New("Akun agen. Nonaktifkan agen dulu.")
	// ErrTidakTertaut: akun tidak tertaut ke jamaah.
	ErrTidakTertaut = errors.New("Akun belum tertaut ke jamaah.")
	// ErrNIKTidakBentrok: NIK agen akun tidak dipegang bersamaan dengan jamaah yang tertaut ke akun lain.
	ErrNIKTidakBentrok = errors.New("NIK agen tidak bentrok dengan jamaah tertaut.")
)

type AlasanRequest struct {
	Alasan string `json:"alasan"`
}

// potongRune memotong teks ke n karakter.
func potongRune(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:n])
}

// lepasTautanTx melepas tautan akun ke jamaah: jamaah_id dikosongkan, semua sesi akun dicabut, tercatat di
// akun_data_log dan jamaah_aktivasi_log (lepas_admin). Upline tidak diubah (admin memakai Ganti Upline). Ditolak untuk
// akun agen aktif atau nonaktif. brandID nil = Admin Master.
func lepasTautanTx(ctx context.Context, tx *sql.Tx, brandID *int64, akunID, adminID int64, req AlasanRequest) error {
	alasan, err := cekAlasan(req.Alasan)
	if err != nil {
		return err
	}
	var brand int64
	var status string
	var jamaah sql.NullInt64
	err = tx.QueryRowContext(ctx,
		`SELECT brand_id, status_agen, jamaah_id FROM customer_accounts WHERE id = ? AND dihapus_at IS NULL FOR UPDATE`, akunID).
		Scan(&brand, &status, &jamaah)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != brand) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("customer: kunci akun: %w", err)
	}
	if !jamaah.Valid {
		return ErrTidakTertaut
	}
	if status == "aktif" || status == "nonaktif" {
		return ErrAgenAktif
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_accounts SET jamaah_id = NULL WHERE id = ?`, akunID); err != nil {
		return fmt.Errorf("customer: lepas tautan: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_sessions SET revoked_at = NOW() WHERE account_id = ? AND revoked_at IS NULL`, akunID); err != nil {
		return fmt.Errorf("customer: cabut sesi: %w", err)
	}
	if err := catatLog(ctx, tx, akunID, adminID, "jamaah_id", strconv.FormatInt(jamaah.Int64, 10), nil, alasan); err != nil {
		return err
	}
	return catatLogAktivasi(ctx, tx, entriLogAktivasi{
		BrandID: brand, Aksi: "lepas_admin", JamaahID: jamaah.Int64, AkunID: akunID,
		OlehAdminID: adminID, Catatan: potongRune(alasan, 300),
	})
}

// kosongkanNIKAgenTx mengosongkan nik_agen akun yang NIK-nya sama dengan jamaah yang tertaut ke akun lain. Izin menurut
// status akun seperti ubah NIK agen (pengajuan: agen.approve, agen aktif/nonaktif: agen.kaitan.edit).
func kosongkanNIKAgenTx(ctx context.Context, tx *sql.Tx, brandID *int64, akunID, adminID int64, izin IzinNIK, req AlasanRequest) error {
	alasan, err := cekAlasan(req.Alasan)
	if err != nil {
		return err
	}
	var brand int64
	var status string
	var nik sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT brand_id, status_agen, nik_agen FROM customer_accounts WHERE id = ? FOR UPDATE`, akunID).Scan(&brand, &status, &nik)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != brand) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("customer: kunci akun: %w", err)
	}
	if (status == "aktif" || status == "nonaktif") && !izin.Aktif || (status != "aktif" && status != "nonaktif") && !izin.Pengajuan {
		return ErrDilarang
	}
	if !nik.Valid || nik.String == "" {
		return ErrNIKTidakBentrok
	}
	var bentrok bool
	if err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM jamaah j JOIN customer_accounts ca ON ca.jamaah_id = j.id AND ca.id <> ?
		               WHERE j.brand_id = ? AND j.nik = ?)`, akunID, brand, nik.String).Scan(&bentrok); err != nil {
		return fmt.Errorf("customer: cek NIK agen bentrok: %w", err)
	}
	if !bentrok {
		return ErrNIKTidakBentrok
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_accounts SET nik_agen = NULL WHERE id = ?`, akunID); err != nil {
		return fmt.Errorf("customer: kosongkan NIK agen: %w", err)
	}
	return catatLog(ctx, tx, akunID, adminID, "nik_agen", samarkanNIK(nik.String), nil, alasan)
}

// AgenTautanBaris: akun agen (pengajuan, aktif, nonaktif) yang tertaut ke jamaah, untuk audit admin.
type AgenTautanBaris struct {
	AkunID       int64      `json:"akun_id"`
	NamaLengkap  string     `json:"nama_lengkap"`
	BrandID      int64      `json:"brand_id"`
	StatusAgen   string     `json:"status_agen"`
	DisetujuiAt  *time.Time `json:"disetujui_agen_at"`
	JamaahID     int64      `json:"jamaah_id"`
	JamaahNama   string     `json:"jamaah_nama"`
	KaitanStatus string     `json:"kaitan_status"`
	// PemilikKunciID: akun pemilik kunci jamaah tertaut (direkrut jamaah); nil bila tanpa agen.
	PemilikKunciID   *int64  `json:"pemilik_kunci_akun_id"`
	PemilikKunciNama *string `json:"pemilik_kunci_nama"`
	UplineID         *int64  `json:"upline_akun_id"`
	UplineNama       *string `json:"upline_nama"`
	// UplineSamaKunci: upline akun sama dengan pemilik kunci jamaah.
	UplineSamaKunci bool `json:"upline_sama_kunci"`
	// BookingPendukungAktif: ada booking dp/lunas dengan pax aktif jamaah tertaut. False berarti pendukung sudah batal.
	BookingPendukungAktif bool      `json:"booking_pendukung_aktif"`
	CreatedAt             time.Time `json:"created_at"`
}

type HasilAgenTautan struct {
	Items   []AgenTautanBaris `json:"items"`
	Total   int               `json:"total"`
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
}

// listAgenDariTautan: brandID nil = semua brand. Terbaru dulu.
func listAgenDariTautan(ctx context.Context, q adminQuerier, brandID *int64, page, perPage int) (*HasilAgenTautan, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 50
	}
	where := `a.jamaah_id IS NOT NULL AND a.status_agen IN ('pengajuan','aktif','nonaktif') AND a.dihapus_at IS NULL`
	args := []any{}
	if brandID != nil {
		where += ` AND a.brand_id = ?`
		args = append(args, *brandID)
	}
	h := &HasilAgenTautan{Items: []AgenTautanBaris{}, Page: page, PerPage: perPage}
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM customer_accounts a WHERE `+where, args...).Scan(&h.Total); err != nil {
		return nil, fmt.Errorf("customer: hitung agen dari tautan: %w", err)
	}
	rows, err := q.QueryContext(ctx, `
		SELECT a.id, a.nama_lengkap, a.brand_id, a.status_agen, a.disetujui_agen_at, a.created_at, j.id, j.nama_lengkap, j.kaitan_status,
		       j.direkrut_oleh_akun_id, ku.nama_lengkap, a.upline_akun_id, up.nama_lengkap,
		       EXISTS (SELECT 1 FROM booking_pax bp JOIN bookings b ON b.id = bp.booking_id AND b.status IN ('dp','lunas')
		               WHERE bp.jamaah_id = j.id AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant')
		FROM customer_accounts a
		JOIN jamaah j ON j.id = a.jamaah_id
		LEFT JOIN customer_accounts ku ON ku.id = j.direkrut_oleh_akun_id
		LEFT JOIN customer_accounts up ON up.id = a.upline_akun_id
		WHERE `+where+` ORDER BY a.id DESC LIMIT ? OFFSET ?`, append(args, perPage, (page-1)*perPage)...)
	if err != nil {
		return nil, fmt.Errorf("customer: agen dari tautan: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var b AgenTautanBaris
		var kunci, upline sql.NullInt64
		if err := rows.Scan(&b.AkunID, &b.NamaLengkap, &b.BrandID, &b.StatusAgen, &b.DisetujuiAt, &b.CreatedAt, &b.JamaahID, &b.JamaahNama, &b.KaitanStatus,
			&kunci, &b.PemilikKunciNama, &upline, &b.UplineNama, &b.BookingPendukungAktif); err != nil {
			return nil, fmt.Errorf("customer: scan agen dari tautan: %w", err)
		}
		if kunci.Valid {
			b.PemilikKunciID = &kunci.Int64
		}
		if upline.Valid {
			b.UplineID = &upline.Int64
		}
		b.UplineSamaKunci = kunci.Valid && upline.Valid && kunci.Int64 == upline.Int64
		h.Items = append(h.Items, b)
	}
	return h, rows.Err()
}

// LepasTautan POST /api/admin/akun/{id}/lepas-tautan {alasan}. Izin akun.edit (dipasang di rute).
func (h *AdminAkun) LepasTautan(w http.ResponseWriter, r *http.Request) {
	id, ok := idAkun(w, r)
	if !ok {
		return
	}
	var req AlasanRequest
	if !decode(w, r, &req) {
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.dalamTx(r.Context(), func(tx *sql.Tx) error { return lepasTautanTx(r.Context(), tx, adminBrandScope(r), id, adminID, req) }); err != nil {
		tulisGalatUbah(w, err)
		return
	}
	log.Printf("[AUDIT] akun_tautan_dilepas akun_id=%d admin_id=%d", id, adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "tautan jamaah dilepas"})
}

// KosongkanNIKAgen POST /api/admin/akun/{id}/kosongkan-nik-agen {alasan}. Izin menurut status akun (lihat kosongkanNIKAgenTx).
func (h *AdminAkun) KosongkanNIKAgen(w http.ResponseWriter, r *http.Request) {
	id, ok := idAkun(w, r)
	if !ok {
		return
	}
	var req AlasanRequest
	if !decode(w, r, &req) {
		return
	}
	izin := IzinNIK{Pengajuan: identity.HasPermission(r.Context(), "agen.approve"), Aktif: identity.HasPermission(r.Context(), "agen.kaitan.edit")}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.dalamTx(r.Context(), func(tx *sql.Tx) error {
		return kosongkanNIKAgenTx(r.Context(), tx, adminBrandScope(r), id, adminID, izin, req)
	}); err != nil {
		tulisGalatUbah(w, err)
		return
	}
	log.Printf("[AUDIT] akun_nik_agen_dikosongkan akun_id=%d admin_id=%d", id, adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "NIK agen dikosongkan"})
}

// AgenDariTautan GET /api/admin/akun/agen-dari-tautan?page=&per_page=&brand_id=. Izin akun.view (grup rute).
func (h *AdminAkun) AgenDariTautan(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	page, _ := strconv.Atoi(strings.TrimSpace(qs.Get("page")))
	perPage, _ := strconv.Atoi(strings.TrimSpace(qs.Get("per_page")))
	hasil, err := listAgenDariTautan(r.Context(), h.db, adminBrandScope(r), page, perPage)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil daftar agen dari tautan.")
		return
	}
	writeJSON(w, http.StatusOK, hasil)
}

type resetHitungRequest struct {
	KodeBooking string `json:"kode_booking"`
	NIK         string `json:"nik"`
}

// ResetHitung membuka kunci penghitung harian satu kode booking dan/atau satu NIK (jamaah sah yang terkunci
// oleh tebakan pihak lain). Tercatat di jamaah_aktivasi_log (reset_admin).
func (h *AktivasiHandler) ResetHitung(w http.ResponseWriter, r *http.Request) {
	if !h.cfg.Aktif {
		http.NotFound(w, r)
		return
	}
	brand := adminBrandScope(r)
	if brand == nil {
		writeError(w, http.StatusBadRequest, "Pilih brand.")
		return
	}
	var req resetHitungRequest
	if !decode(w, r, &req) {
		return
	}
	kode := strings.ToUpper(strings.TrimSpace(req.KodeBooking))
	nik := strings.TrimSpace(req.NIK)
	if (kode == "") == (nik == "") {
		writeError(w, http.StatusBadRequest, "Isi kode booking atau NIK.")
		return
	}
	b := h.core.batas
	var kodeH, nikH string
	var err error
	switch {
	case kode != "":
		if !polaKodeBooking.MatchString(kode) {
			writeError(w, http.StatusBadRequest, "Format kode booking tidak valid.")
			return
		}
		kodeH = b.hash("kode", *brand, kode)
		_, err = h.core.db.ExecContext(r.Context(), `DELETE FROM jamaah_aktivasi_hitung WHERE kunci_hash = ? AND jenis = ?`, kodeH, batasKode24.jenis)
	default:
		if !ValidNIK(nik) {
			writeError(w, http.StatusBadRequest, "NIK harus 16 digit.")
			return
		}
		nikH = b.hash("nik", *brand, nik)
		_, err = h.core.db.ExecContext(r.Context(), `DELETE FROM jamaah_aktivasi_hitung WHERE kunci_hash = ? AND jenis = ?`, nikH, batasNIK24.jenis)
	}
	if err == nil {
		err = catatLogAktivasi(r.Context(), h.core.db, entriLogAktivasi{
			BrandID: *brand, Aksi: "reset_admin", KodeHash: kodeH, NIKHash: nikH, OlehAdminID: identity.GetAdminUserID(r.Context()),
		})
	}
	if err != nil {
		log.Printf("[ERROR] reset penghitung aktivasi: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	log.Printf("[AUDIT] aktivasi_hitung_direset brand_id=%d admin_id=%d", *brand, identity.GetAdminUserID(r.Context()))
	writeJSON(w, http.StatusOK, map[string]string{"message": "penghitung direset"})
}
