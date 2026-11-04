package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
)

// Admin mengubah data akun (layar Akun Pelanggan). Hanya nama, nomor WhatsApp, domisili, dan NIK agen; email,
// kata sandi, dan foto tidak. Tiap kolom yang berubah dicatat di akun_data_log beserta alasannya.

var (
	ErrTidakBerubah = errors.New("tidak ada data yang berubah")
	ErrDilarang     = errors.New("anda tidak memiliki izin untuk mengubah data ini")
	nikPola         = regexp.MustCompile(`^\d{16}$`)
)

type ErrValidasi struct{ Pesan string }

func (e *ErrValidasi) Error() string { return e.Pesan }

// ErrBentrokData: nilai baru sudah dipakai akun lain di brand yang sama. AkunID dan Nama terisi bila pemakainya diketahui.
type ErrBentrokData struct {
	Kolom  string
	AkunID int64
	Nama   string
}

func (e *ErrBentrokData) Error() string {
	label := "Nomor WhatsApp"
	if e.Kolom == "nik_agen" {
		label = "NIK"
	}
	if e.Nama != "" {
		return fmt.Sprintf("%s sudah dipakai akun %s", label, e.Nama)
	}
	return label + " sudah dipakai akun lain"
}

// IzinNIK: izin yang dimiliki admin untuk mengubah NIK agen. Pengajuan (agen.approve) berlaku saat akun belum
// menjadi agen; Aktif (agen.kaitan.edit, Admin Master) berlaku sesudah agen aktif atau nonaktif.
type IzinNIK struct{ Pengajuan, Aktif bool }

type UbahProfilRequest struct {
	NamaLengkap string `json:"nama_lengkap"`
	NoHP        string `json:"no_hp"`
	Domisili    string `json:"domisili"`
	Alasan      string `json:"alasan"`
}

type UbahNIKRequest struct {
	NIK    string `json:"nik"`
	Alasan string `json:"alasan"`
}

// normalisasiHP mengikuti kolom no_hp_normal: angka saja, awalan 0 menjadi 62.
func normalisasiHP(s string) string {
	d := nonDigit.ReplaceAllString(s, "")
	if strings.HasPrefix(d, "0") {
		return "62" + d[1:]
	}
	return d
}

func cekAlasan(a string) (string, error) {
	a = strings.TrimSpace(a)
	if n := utf8.RuneCountInString(a); n < 5 || n > 1000 {
		return "", &ErrValidasi{"Alasan wajib diisi (5 sampai 1000 karakter)."}
	}
	return a, nil
}

func catatLog(ctx context.Context, tx *sql.Tx, akunID, adminID int64, kolom string, lama, baru any, alasan string) error {
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO akun_data_log (akun_id, kolom, nilai_lama, nilai_baru, diubah_oleh, alasan) VALUES (?, ?, ?, ?, ?, ?)`,
		akunID, kolom, lama, baru, adminID, alasan); err != nil {
		return fmt.Errorf("customer: log data akun: %w", err)
	}
	return nil
}

func pemakai(ctx context.Context, tx *sql.Tx, kolom string, brandID, selfID int64, nilai string) (*ErrBentrokData, error) {
	var q string
	switch kolom {
	case "no_hp":
		q = `SELECT id, nama_lengkap FROM customer_accounts WHERE brand_id = ? AND id <> ? AND no_hp_normal = ? LIMIT 1`
	default:
		q = `SELECT id, nama_lengkap FROM customer_accounts WHERE brand_id = ? AND id <> ? AND nik_agen = ? LIMIT 1`
	}
	var b ErrBentrokData
	err := tx.QueryRowContext(ctx, q, brandID, selfID, nilai).Scan(&b.AkunID, &b.Nama)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("customer: cari pemakai %s: %w", kolom, err)
	}
	b.Kolom = kolom
	return &b, nil
}

func duplikat(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062
}

// ubahProfilTx: brandID nil = Admin Master (semua brand); admin brand hanya boleh mengubah akun brand-nya.
func ubahProfilTx(ctx context.Context, tx *sql.Tx, brandID *int64, akunID, adminID int64, req UbahProfilRequest) error {
	alasan, err := cekAlasan(req.Alasan)
	if err != nil {
		return err
	}
	nama, hp, dom := strings.TrimSpace(req.NamaLengkap), strings.TrimSpace(req.NoHP), strings.TrimSpace(req.Domisili)
	if !ValidName(nama) {
		return &ErrValidasi{"Nama hanya boleh berisi huruf, spasi, titik, koma, apostrof, dan tanda hubung."}
	}
	if !ValidPhone(hp) {
		return &ErrValidasi{"Nomor WhatsApp tidak valid."}
	}
	if utf8.RuneCountInString(dom) > 100 {
		return &ErrValidasi{"Domisili maksimal 100 karakter."}
	}

	var brand int64
	var namaLama, hpLama string
	var domLama sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT brand_id, nama_lengkap, no_hp, domisili FROM customer_accounts WHERE id = ? FOR UPDATE`, akunID).
		Scan(&brand, &namaLama, &hpLama, &domLama)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != brand) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("customer: kunci akun: %w", err)
	}

	type ubah struct{ kolom, lama, baru string }
	var daftar []ubah
	if nama != namaLama {
		daftar = append(daftar, ubah{"nama_lengkap", namaLama, nama})
	}
	if hp != hpLama {
		daftar = append(daftar, ubah{"no_hp", hpLama, hp})
	}
	if dom != domLama.String {
		daftar = append(daftar, ubah{"domisili", domLama.String, dom})
	}
	if len(daftar) == 0 {
		return ErrTidakBerubah
	}
	if normalisasiHP(hp) != normalisasiHP(hpLama) {
		b, err := pemakai(ctx, tx, "no_hp", brand, akunID, normalisasiHP(hp))
		if err != nil {
			return err
		}
		if b != nil {
			return b
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_accounts SET nama_lengkap = ?, no_hp = ?, domisili = NULLIF(?, '') WHERE id = ?`, nama, hp, dom, akunID); err != nil {
		if duplikat(err) {
			return &ErrBentrokData{Kolom: "no_hp"}
		}
		return fmt.Errorf("customer: ubah profil akun: %w", err)
	}
	for _, u := range daftar {
		var lama, baru any = u.lama, u.baru
		if u.lama == "" {
			lama = nil
		}
		if u.baru == "" {
			baru = nil
		}
		if err := catatLog(ctx, tx, akunID, adminID, u.kolom, lama, baru, alasan); err != nil {
			return err
		}
	}
	return nil
}

// ubahNIKAgenTx: pengajuan ditangani izin Pengajuan, agen aktif atau nonaktif hanya izin Aktif (Admin Master), karena NIK
// agen menentukan siapa yang boleh memakai "Saya sendiri" untuk komisi atas dirinya. NIK di log disamarkan.
func ubahNIKAgenTx(ctx context.Context, tx *sql.Tx, brandID *int64, akunID, adminID int64, izin IzinNIK, req UbahNIKRequest) error {
	alasan, err := cekAlasan(req.Alasan)
	if err != nil {
		return err
	}
	nik := strings.TrimSpace(req.NIK)
	if !nikPola.MatchString(nik) {
		return &ErrValidasi{"NIK harus 16 digit angka."}
	}
	var brand int64
	var status string
	var lama sql.NullString
	err = tx.QueryRowContext(ctx, `SELECT brand_id, status_agen, nik_agen FROM customer_accounts WHERE id = ? FOR UPDATE`, akunID).Scan(&brand, &status, &lama)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != brand) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("customer: kunci akun: %w", err)
	}
	if (status == "aktif" || status == "nonaktif") && !izin.Aktif || (status != "aktif" && status != "nonaktif") && !izin.Pengajuan {
		return ErrDilarang
	}
	if lama.Valid && lama.String == nik {
		return ErrTidakBerubah
	}
	b, err := pemakai(ctx, tx, "nik_agen", brand, akunID, nik)
	if err != nil {
		return err
	}
	if b != nil {
		return b
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_accounts SET nik_agen = ? WHERE id = ?`, nik, akunID); err != nil {
		if duplikat(err) {
			return &ErrBentrokData{Kolom: "nik_agen"}
		}
		return fmt.Errorf("customer: ubah NIK agen: %w", err)
	}
	var lamaLog any
	if lama.Valid {
		lamaLog = samarkanNIK(lama.String)
	}
	return catatLog(ctx, tx, akunID, adminID, "nik_agen", lamaLog, samarkanNIK(nik), alasan)
}

func (h *AdminAkun) dalamTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := h.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("customer: mulai transaksi: %w", err)
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func tulisGalatUbah(w http.ResponseWriter, err error) {
	var val *ErrValidasi
	var bentrok *ErrBentrokData
	switch {
	case errors.As(err, &val):
		writeError(w, http.StatusBadRequest, val.Pesan)
	case errors.Is(err, ErrTidakBerubah):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, ErrDilarang):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "akun tidak ditemukan")
	case errors.As(err, &bentrok):
		writeJSON(w, http.StatusConflict, map[string]any{"error": bentrok.Error(), "akun_id": bentrok.AkunID, "nama": bentrok.Nama})
	default:
		log.Printf("[ERROR] ubah data akun: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan perubahan")
	}
}

func idAkun(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id akun tidak valid")
		return 0, false
	}
	return id, true
}

// UbahProfil PUT /api/admin/akun/{id} {nama_lengkap, no_hp, domisili, alasan}. Izin akun.edit.
func (h *AdminAkun) UbahProfil(w http.ResponseWriter, r *http.Request) {
	id, ok := idAkun(w, r)
	if !ok {
		return
	}
	var req UbahProfilRequest
	if !decode(w, r, &req) {
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.dalamTx(r.Context(), func(tx *sql.Tx) error { return ubahProfilTx(r.Context(), tx, adminBrandScope(r), id, adminID, req) }); err != nil {
		tulisGalatUbah(w, err)
		return
	}
	log.Printf("[AUDIT] akun_profil_diubah akun_id=%d admin_id=%d", id, adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "data akun diperbarui"})
}

// UbahNIK PUT /api/admin/akun/{id}/nik-agen {nik, alasan}. Izin menurut status akun (lihat ubahNIKAgenTx).
func (h *AdminAkun) UbahNIK(w http.ResponseWriter, r *http.Request) {
	id, ok := idAkun(w, r)
	if !ok {
		return
	}
	var req UbahNIKRequest
	if !decode(w, r, &req) {
		return
	}
	izin := IzinNIK{Pengajuan: identity.HasPermission(r.Context(), "agen.approve"), Aktif: identity.HasPermission(r.Context(), "agen.kaitan.edit")}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.dalamTx(r.Context(), func(tx *sql.Tx) error {
		return ubahNIKAgenTx(r.Context(), tx, adminBrandScope(r), id, adminID, izin, req)
	}); err != nil {
		tulisGalatUbah(w, err)
		return
	}
	log.Printf("[AUDIT] akun_nik_agen_diubah akun_id=%d admin_id=%d", id, adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "NIK agen diperbarui"})
}
