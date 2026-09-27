package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"erp-azhan/api/internal/shared"
)

// Repository membungkus operasi agen. Setiap operasi tulis punya pasangan
// fungsi ...Tx(ctx, tx, ...) supaya bisa diuji di dalam transaksi yang
// di-rollback (internal/testdb).
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (r *Repository) withTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("agen: begin tx: %w", err)
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repository) GetStatusPortal(ctx context.Context, jamaahID int64) (*StatusPortal, error) {
	return getStatusPortal(ctx, r.db, jamaahID)
}

func (r *Repository) Ajukan(ctx context.Context, jamaahID int64, req AjukanRequest) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return ajukanTx(ctx, tx, jamaahID, req) })
}

func (r *Repository) UploadBuktiJamaah(ctx context.Context, jamaahID int64, url string) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return uploadBuktiJamaahTx(ctx, tx, jamaahID, url) })
}

func (r *Repository) ListPengajuan(ctx context.Context, brandID *int64) ([]Pengajuan, error) {
	return listPengajuan(ctx, r.db, brandID)
}

func (r *Repository) Setujui(ctx context.Context, jamaahID int64, brandID *int64, adminID int64) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return setujuiTx(ctx, tx, jamaahID, brandID, adminID) })
}

func (r *Repository) Tolak(ctx context.Context, jamaahID int64, brandID *int64, alasan string) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return tolakTx(ctx, tx, jamaahID, brandID, alasan) })
}

func (r *Repository) UbahStatus(ctx context.Context, jamaahID int64, brandID *int64, status string) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return ubahStatusTx(ctx, tx, jamaahID, brandID, status) })
}

func (r *Repository) VerifikasiPembayaran(ctx context.Context, ppaID int64, brandID *int64, adminID int64) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return verifikasiPembayaranTx(ctx, tx, ppaID, brandID, adminID) })
}

func (r *Repository) TolakPembayaran(ctx context.Context, ppaID int64, brandID *int64, adminID int64, catatan string) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return tolakPembayaranTx(ctx, tx, ppaID, brandID, adminID, catatan) })
}

func (r *Repository) UploadBuktiAdmin(ctx context.Context, ppaID int64, brandID *int64, adminID int64, url string) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return uploadBuktiAdminTx(ctx, tx, ppaID, brandID, adminID, url) })
}

// ─── Portal (jamaah yang login) ───────────────────────────────────────────────

// getStatusPortal mengembalikan state Syiar jamaah beserta siklus pembayaran terakhir.
func getStatusPortal(ctx context.Context, q querier, jamaahID int64) (*StatusPortal, error) {
	var s StatusPortal
	err := q.QueryRowContext(ctx, `
		SELECT j.status_agen, j.kode_referral, j.foto_agen_url, j.domisili,
		       b.name, b.biaya_pendaftaran_agen, b.no_wa_admin_travel
		FROM jamaah j JOIN brands b ON b.id = j.brand_id
		WHERE j.id = ?`, jamaahID).
		Scan(&s.StatusAgen, &s.KodeReferral, &s.FotoAgenURL, &s.Domisili,
			&s.BrandName, &s.BiayaPendaftaranAgen, &s.NoWAAdminTravel)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("agen.GetStatusPortal: %w", err)
	}
	if s.Pembayaran, err = latestPembayaran(ctx, q, jamaahID); err != nil {
		return nil, err
	}
	return &s, nil
}

// ajukanTx menyimpan kelengkapan agen, mengubah status menjadi 'pengajuan', dan
// membuat satu siklus pembayaran dengan nominal snapshot biaya brand (3.6.3).
func ajukanTx(ctx context.Context, tx *sql.Tx, jamaahID int64, req AjukanRequest) error {
	var status string
	var brandID int64
	var biaya float64
	err := tx.QueryRowContext(ctx, `
		SELECT j.status_agen, j.brand_id, b.biaya_pendaftaran_agen
		FROM jamaah j JOIN brands b ON b.id = j.brand_id
		WHERE j.id = ? FOR UPDATE`, jamaahID).Scan(&status, &brandID, &biaya)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("agen.Ajukan baca jamaah: %w", err)
	}
	if status != "tidak_aktif" {
		return ErrSudahMengajukan
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE jamaah SET foto_agen_url = ?, domisili = ?, menyetujui_syarat_ketentuan_agen_at = NOW(),
		       status_agen = 'pengajuan', diajukan_agen_at = NOW()
		WHERE id = ?`, req.FotoAgenURL, req.Domisili, jamaahID); err != nil {
		return fmt.Errorf("agen.Ajukan update jamaah: %w", err)
	}

	// Biaya Rp0: langkah pembayaran dilewati, status langsung terverifikasi
	// tanpa diverifikasi_oleh (dianggap otomatis oleh sistem).
	statusBayar := "menunggu_verifikasi"
	if biaya <= 0 {
		statusBayar = "terverifikasi"
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pembayaran_pendaftaran_agen (jamaah_id, brand_id, nominal_tagihan, status)
		VALUES (?, ?, ?, ?)`, jamaahID, brandID, biaya, statusBayar); err != nil {
		return fmt.Errorf("agen.Ajukan insert pembayaran: %w", err)
	}
	return nil
}

// uploadBuktiJamaahTx mencatat bukti transfer dari pemohon untuk siklus terakhirnya.
func uploadBuktiJamaahTx(ctx context.Context, tx *sql.Tx, jamaahID int64, url string) error {
	var ppaID int64
	err := tx.QueryRowContext(ctx, `
		SELECT id FROM pembayaran_pendaftaran_agen WHERE jamaah_id = ?
		ORDER BY id DESC LIMIT 1 FOR UPDATE`, jamaahID).Scan(&ppaID)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrTidakAdaPengajuan
	}
	if err != nil {
		return fmt.Errorf("agen.UploadBuktiJamaah cari siklus: %w", err)
	}
	return insertUpload(ctx, tx, ppaID, url, "jamaah", jamaahID)
}

// ─── Admin Travel ─────────────────────────────────────────────────────────────

// listPengajuan mengembalikan antrian jamaah berstatus 'pengajuan' (screen B1).
// brandID nil = Super Admin (semua brand).
func listPengajuan(ctx context.Context, db querier, brandID *int64) ([]Pengajuan, error) {
	q := `
		SELECT j.id, COALESCE(j.id_jamaah, ''), j.brand_id, j.nama_lengkap, j.no_hp,
		       j.foto_agen_url, j.domisili, j.menyetujui_syarat_ketentuan_agen_at, j.diajukan_agen_at,
		       rek.nama_lengkap
		FROM jamaah j
		LEFT JOIN jamaah rek ON rek.id = j.direkrut_oleh_jamaah_id
		WHERE j.status_agen = 'pengajuan'`
	args := []any{}
	if brandID != nil {
		q += ` AND j.brand_id = ?`
		args = append(args, *brandID)
	}
	q += ` ORDER BY j.diajukan_agen_at ASC, j.id ASC`

	rows, err := db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("agen.ListPengajuan: %w", err)
	}
	items := []Pengajuan{}
	for rows.Next() {
		var p Pengajuan
		if err := rows.Scan(&p.JamaahID, &p.IDJamaah, &p.BrandID, &p.NamaLengkap, &p.NoHP,
			&p.FotoAgenURL, &p.Domisili, &p.SetujuSKAt, &p.DiajukanAt, &p.DirekrutOleh); err != nil {
			rows.Close()
			return nil, fmt.Errorf("agen.ListPengajuan scan: %w", err)
		}
		items = append(items, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Pembayaran, err = latestPembayaran(ctx, db, items[i].JamaahID); err != nil {
			return nil, err
		}
	}
	return items, nil
}

// setujuiTx mengaktifkan agen. Tidak memeriksa status pembayaran sama sekali
// (§5 poin 20). Kode referral dibuat sekali; upline diambil dari agen yang
// merekrut jamaah ini (§5 poin 4).
func setujuiTx(ctx context.Context, tx *sql.Tx, jamaahID int64, brandID *int64, adminID int64) error {
	var kode sql.NullString
	if err := lockPengajuan(ctx, tx, jamaahID, brandID, &kode); err != nil {
		return err
	}
	if !kode.Valid || kode.String == "" {
		generated, err := shared.UniqueCode(ctx, tx, "jamaah", "kode_referral", "", 6)
		if err != nil {
			return fmt.Errorf("agen.Setujui kode referral: %w", err)
		}
		kode = sql.NullString{String: generated, Valid: true}
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE jamaah SET status_agen = 'aktif', kode_referral = ?, upline_jamaah_id = direkrut_oleh_jamaah_id,
		       disetujui_agen_at = NOW(), disetujui_agen_oleh = ?
		WHERE id = ?`, kode.String, adminID, jamaahID); err != nil {
		return fmt.Errorf("agen.Setujui update: %w", err)
	}
	return catatKeputusan(ctx, tx, jamaahID, "disetujui", nil)
}

// tolakTx mengembalikan status ke 'tidak_aktif' (identik dengan belum pernah
// mengajukan, 3.6.4-B) dan mencatat alasannya di siklus pembayaran.
func tolakTx(ctx context.Context, tx *sql.Tx, jamaahID int64, brandID *int64, alasan string) error {
	if err := lockPengajuan(ctx, tx, jamaahID, brandID, nil); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jamaah SET status_agen = 'tidak_aktif' WHERE id = ?`, jamaahID); err != nil {
		return fmt.Errorf("agen.Tolak update: %w", err)
	}
	return catatKeputusan(ctx, tx, jamaahID, "ditolak", &alasan)
}

// ubahStatusTx men-toggle agen antara 'aktif' dan 'nonaktif' (screen B2a).
func ubahStatusTx(ctx context.Context, tx *sql.Tx, jamaahID int64, brandID *int64, status string) error {
	if status != "aktif" && status != "nonaktif" {
		return ErrStatusTidakValid
	}
	var cur string
	var jBrand int64
	err := tx.QueryRowContext(ctx, `SELECT status_agen, brand_id FROM jamaah WHERE id = ? FOR UPDATE`, jamaahID).Scan(&cur, &jBrand)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != jBrand) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("agen.UbahStatus baca: %w", err)
	}
	if cur != "aktif" && cur != "nonaktif" {
		return ErrStatusTidakValid
	}
	if _, err := tx.ExecContext(ctx, `UPDATE jamaah SET status_agen = ? WHERE id = ?`, status, jamaahID); err != nil {
		return fmt.Errorf("agen.UbahStatus update: %w", err)
	}
	return nil
}

// verifikasiPembayaranTx menandai pembayaran terverifikasi (dari menunggu atau ditolak).
func verifikasiPembayaranTx(ctx context.Context, tx *sql.Tx, ppaID int64, brandID *int64, adminID int64) error {
	status, err := lockPembayaran(ctx, tx, ppaID, brandID)
	if err != nil {
		return err
	}
	if status == "terverifikasi" {
		return ErrPembayaranFinal
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE pembayaran_pendaftaran_agen SET status = 'terverifikasi', diverifikasi_oleh = ?, diverifikasi_at = NOW()
		WHERE id = ?`, adminID, ppaID); err != nil {
		return fmt.Errorf("agen.VerifikasiPembayaran update: %w", err)
	}
	return nil
}

// tolakPembayaranTx menolak pembayaran yang sedang menunggu verifikasi.
func tolakPembayaranTx(ctx context.Context, tx *sql.Tx, ppaID int64, brandID *int64, adminID int64, catatan string) error {
	status, err := lockPembayaran(ctx, tx, ppaID, brandID)
	if err != nil {
		return err
	}
	if status != "menunggu_verifikasi" {
		return ErrPembayaranBukanMenunggu
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE pembayaran_pendaftaran_agen SET status = 'ditolak', catatan_penolakan = ?,
		       diverifikasi_oleh = ?, diverifikasi_at = NOW()
		WHERE id = ?`, catatan, adminID, ppaID); err != nil {
		return fmt.Errorf("agen.TolakPembayaran update: %w", err)
	}
	return nil
}

// uploadBuktiAdminTx mencatat bukti transfer yang diunggah Admin Travel atas nama pemohon.
func uploadBuktiAdminTx(ctx context.Context, tx *sql.Tx, ppaID int64, brandID *int64, adminID int64, url string) error {
	if _, err := lockPembayaran(ctx, tx, ppaID, brandID); err != nil {
		return err
	}
	return insertUpload(ctx, tx, ppaID, url, "admin", adminID)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// lockPengajuan mengunci jamaah yang berstatus 'pengajuan' di brand yang benar.
func lockPengajuan(ctx context.Context, tx *sql.Tx, jamaahID int64, brandID *int64, kode *sql.NullString) error {
	var status string
	var jBrand int64
	var k sql.NullString
	err := tx.QueryRowContext(ctx, `SELECT status_agen, brand_id, kode_referral FROM jamaah WHERE id = ? FOR UPDATE`, jamaahID).
		Scan(&status, &jBrand, &k)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != jBrand) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("agen: kunci jamaah: %w", err)
	}
	if status != "pengajuan" {
		return ErrBukanPengajuan
	}
	if kode != nil {
		*kode = k
	}
	return nil
}

// lockPembayaran mengunci siklus pembayaran di brand yang benar dan mengembalikan statusnya.
func lockPembayaran(ctx context.Context, tx *sql.Tx, ppaID int64, brandID *int64) (string, error) {
	var status string
	var pBrand int64
	err := tx.QueryRowContext(ctx, `SELECT status, brand_id FROM pembayaran_pendaftaran_agen WHERE id = ? FOR UPDATE`, ppaID).
		Scan(&status, &pBrand)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != pBrand) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("agen: kunci pembayaran: %w", err)
	}
	return status, nil
}

// catatKeputusan menulis audit trail keputusan keagenan ke siklus terakhir (7.10).
func catatKeputusan(ctx context.Context, tx *sql.Tx, jamaahID int64, keputusan string, alasan *string) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE pembayaran_pendaftaran_agen SET keputusan_agen = ?, keputusan_agen_at = NOW(), alasan_ditolak_agen = ?
		WHERE id = (SELECT id FROM (SELECT MAX(id) AS id FROM pembayaran_pendaftaran_agen WHERE jamaah_id = ?) t)`,
		keputusan, alasan, jamaahID)
	if err != nil {
		return fmt.Errorf("agen: catat keputusan: %w", err)
	}
	return nil
}

// insertUpload menambah riwayat upload (tidak pernah menimpa), menyinkronkan
// bukti terakhir, dan mengembalikan status 'ditolak' ke 'menunggu_verifikasi'
// (3.6.4-A). Siklus yang keagenannya sudah ditolak tidak menerima upload lagi.
func insertUpload(ctx context.Context, tx *sql.Tx, ppaID int64, url, tipe string, olehID int64) error {
	var status, keputusan string
	if err := tx.QueryRowContext(ctx, `SELECT status, keputusan_agen FROM pembayaran_pendaftaran_agen WHERE id = ?`, ppaID).
		Scan(&status, &keputusan); err != nil {
		return fmt.Errorf("agen: baca siklus: %w", err)
	}
	if keputusan == "ditolak" {
		return ErrSiklusDitutup
	}
	if status == "terverifikasi" {
		return ErrPembayaranFinal
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO pembayaran_pendaftaran_agen_upload (pembayaran_pendaftaran_agen_id, bukti_transfer_url, diupload_oleh_tipe, diupload_oleh_id)
		VALUES (?, ?, ?, ?)`, ppaID, url, tipe, olehID); err != nil {
		return fmt.Errorf("agen: insert upload: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE pembayaran_pendaftaran_agen SET bukti_transfer_url = ?,
		       status = IF(status = 'ditolak', 'menunggu_verifikasi', status)
		WHERE id = ?`, url, ppaID); err != nil {
		return fmt.Errorf("agen: sinkron bukti: %w", err)
	}
	return nil
}

// latestPembayaran mengambil siklus pembayaran terakhir jamaah + riwayat upload.
func latestPembayaran(ctx context.Context, q querier, jamaahID int64) (*Pembayaran, error) {
	var p Pembayaran
	err := q.QueryRowContext(ctx, `
		SELECT id, nominal_tagihan, status, bukti_transfer_url, catatan_penolakan, diverifikasi_at,
		       keputusan_agen, keputusan_agen_at, alasan_ditolak_agen, created_at
		FROM pembayaran_pendaftaran_agen WHERE jamaah_id = ? ORDER BY id DESC LIMIT 1`, jamaahID).
		Scan(&p.ID, &p.NominalTagihan, &p.Status, &p.BuktiTransferURL, &p.CatatanPenolakan, &p.DiverifikasiAt,
			&p.KeputusanAgen, &p.KeputusanAgenAt, &p.AlasanDitolakAgen, &p.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("agen: baca pembayaran: %w", err)
	}
	rows, err := q.QueryContext(ctx, `
		SELECT id, bukti_transfer_url, diupload_oleh_tipe, created_at
		FROM pembayaran_pendaftaran_agen_upload WHERE pembayaran_pendaftaran_agen_id = ? ORDER BY id ASC`, p.ID)
	if err != nil {
		return nil, fmt.Errorf("agen: baca upload: %w", err)
	}
	defer rows.Close()
	p.Uploads = []Upload{}
	for rows.Next() {
		var u Upload
		if err := rows.Scan(&u.ID, &u.BuktiTransferURL, &u.DiuploadOlehTipe, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("agen: scan upload: %w", err)
		}
		p.Uploads = append(p.Uploads, u)
	}
	return &p, rows.Err()
}
