package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"erp-azhan/api/internal/komisi"
)

// Pencairan saldo agen: A7 (agen mengajukan) dan C1 (Admin Master memproses,
// lintas brand). agen-azhan.md §5 poin 9–11, 7.6, 7.7, keputusan D6/D7/L5.

const MinimalPencairan = 500_000

var (
	ErrPencairanMasihPending = errors.New("masih ada pengajuan pencairan yang sedang diproses")
	ErrPencairanMinimal      = errors.New("nominal pencairan minimal Rp500.000")
	ErrPencairanMelebihi     = errors.New("nominal melebihi saldo tersedia")
	ErrRekeningTidakLengkap  = errors.New("rekening tujuan wajib diisi lengkap (bank, nomor, atas nama)")
	ErrPencairanBukanPending = errors.New("pengajuan pencairan sudah diproses")
	ErrSaldoTidakCukup       = errors.New("saldo tersedia agen tidak lagi mencukupi, tolak pengajuan ini")
)

type Rekening struct {
	Bank     string `json:"rekening_bank"`
	Nomor    string `json:"rekening_nomor"`
	AtasNama string `json:"rekening_atas_nama"`
}

type AjukanPencairanRequest struct {
	Nominal float64 `json:"nominal"`
	Rekening
}

// PencairanPortal adalah state halaman A7.
type PencairanPortal struct {
	komisi.Saldo
	Minimal    float64            `json:"minimal"`
	AdaPending bool               `json:"ada_pending"`
	Rekening   Rekening           `json:"rekening"` // tersimpan dari pengajuan sebelumnya
	Riwayat    []PencairanRiwayat `json:"riwayat"`
}

type PencairanRiwayat struct {
	Pencairan
	Rekening
	PunyaBukti bool `json:"punya_bukti"`
}

// PencairanAdmin adalah satu baris antrian C1.
type PencairanAdmin struct {
	PencairanRiwayat
	JamaahAgenID   int64     `json:"jamaah_agen_id"`
	NamaAgen       string    `json:"nama_agen"`
	KodeReferral   *string   `json:"kode_referral"`
	BrandID        int64     `json:"brand_id"`
	BrandName      string    `json:"brand_name"`
	SaldoTersedia  float64   `json:"saldo_tersedia"` // sudah dikurangi pengajuan ini bila pending
	DiprosesOlehNm *string   `json:"diproses_oleh_nama"`
	BuktiURL       *string   `json:"bukti_transfer_keluar_url"`
	DiajukanAt     time.Time `json:"diajukan_at"`
}

func (req *AjukanPencairanRequest) normalize() error {
	req.Bank = strings.TrimSpace(req.Bank)
	req.Nomor = strings.TrimSpace(req.Nomor)
	req.AtasNama = strings.TrimSpace(req.AtasNama)
	if req.Bank == "" || req.Nomor == "" || req.AtasNama == "" ||
		utf8.RuneCountInString(req.Bank) > 100 || len(req.Nomor) > 50 || utf8.RuneCountInString(req.AtasNama) > 150 {
		return ErrRekeningTidakLengkap
	}
	if strings.Trim(req.Nomor, "0123456789 -") != "" {
		return ErrRekeningTidakLengkap
	}
	if req.Nominal < MinimalPencairan || req.Nominal != float64(int64(req.Nominal)) {
		return ErrPencairanMinimal
	}
	return nil
}

func listPencairanRiwayat(ctx context.Context, q querier, where string, args ...any) ([]PencairanRiwayat, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT id, nominal_diajukan, status, diajukan_at, diproses_at, catatan_penolakan,
		       rekening_bank, rekening_nomor, rekening_atas_nama, bukti_transfer_keluar_url IS NOT NULL
		FROM pengajuan_pencairan WHERE `+where+` ORDER BY diajukan_at DESC, id DESC`, args...)
	if err != nil {
		return nil, fmt.Errorf("agen: riwayat pencairan: %w", err)
	}
	defer rows.Close()
	items := []PencairanRiwayat{}
	for rows.Next() {
		var p PencairanRiwayat
		if err := rows.Scan(&p.ID, &p.NominalDiajukan, &p.Status, &p.DiajukanAt, &p.DiprosesAt, &p.CatatanPenolakan,
			&p.Bank, &p.Nomor, &p.AtasNama, &p.PunyaBukti); err != nil {
			return nil, fmt.Errorf("agen: scan riwayat pencairan: %w", err)
		}
		items = append(items, p)
	}
	return items, rows.Err()
}

func getPencairanPortal(ctx context.Context, q querier, agenID int64) (*PencairanPortal, error) {
	if _, err := requireAgenAktif(ctx, q, agenID); err != nil {
		return nil, err
	}
	var p PencairanPortal
	p.Minimal = MinimalPencairan
	var bank, nomor, atas sql.NullString
	if err := q.QueryRowContext(ctx,
		`SELECT rekening_bank_agen, rekening_nomor_agen, rekening_atas_nama_agen FROM jamaah WHERE id = ?`, agenID).
		Scan(&bank, &nomor, &atas); err != nil {
		return nil, fmt.Errorf("agen: rekening agen: %w", err)
	}
	p.Rekening = Rekening{Bank: bank.String, Nomor: nomor.String, AtasNama: atas.String}
	var err error
	if p.Saldo, err = komisi.HitungSaldo(ctx, q, agenID); err != nil {
		return nil, err
	}
	if p.Riwayat, err = listPencairanRiwayat(ctx, q, "jamaah_agen_id = ?", agenID); err != nil {
		return nil, err
	}
	for _, r := range p.Riwayat {
		if r.Status == "pending" {
			p.AdaPending = true
		}
	}
	return &p, nil
}

// ajukanPencairanTx: baris agen dikunci supaya dua pengajuan bersamaan tidak
// lolos cek "satu pending" dan cek saldo (7.6).
func ajukanPencairanTx(ctx context.Context, tx *sql.Tx, agenID int64, req AjukanPencairanRequest) (int64, error) {
	if err := req.normalize(); err != nil {
		return 0, err
	}
	var status string
	var brandID int64
	err := tx.QueryRowContext(ctx, `SELECT status_agen, brand_id FROM jamaah WHERE id = ? FOR UPDATE`, agenID).Scan(&status, &brandID)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("agen: kunci agen: %w", err)
	}
	if status != "aktif" {
		return 0, ErrBukanAgenAktif
	}
	var pending int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM pengajuan_pencairan WHERE jamaah_agen_id = ? AND status = 'pending'`, agenID).Scan(&pending); err != nil {
		return 0, fmt.Errorf("agen: cek pending: %w", err)
	}
	if pending > 0 {
		return 0, ErrPencairanMasihPending
	}
	saldo, err := komisi.HitungSaldo(ctx, tx, agenID)
	if err != nil {
		return 0, err
	}
	if req.Nominal > saldo.Tersedia {
		return 0, ErrPencairanMelebihi
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE jamaah SET rekening_bank_agen = ?, rekening_nomor_agen = ?, rekening_atas_nama_agen = ? WHERE id = ?`,
		req.Bank, req.Nomor, req.AtasNama, agenID); err != nil {
		return 0, fmt.Errorf("agen: simpan rekening: %w", err)
	}
	res, err := tx.ExecContext(ctx, `
		INSERT INTO pengajuan_pencairan (brand_id, jamaah_agen_id, nominal_diajukan, rekening_bank, rekening_nomor, rekening_atas_nama)
		VALUES (?, ?, ?, ?, ?, ?)`, brandID, agenID, req.Nominal, req.Bank, req.Nomor, req.AtasNama)
	if err != nil {
		return 0, fmt.Errorf("agen: insert pencairan: %w", err)
	}
	return res.LastInsertId()
}

// lockPencairan mengunci pengajuan pending beserta baris agennya.
func lockPencairan(ctx context.Context, tx *sql.Tx, id int64) (agenID int64, err error) {
	var status string
	err = tx.QueryRowContext(ctx, `SELECT jamaah_agen_id, status FROM pengajuan_pencairan WHERE id = ? FOR UPDATE`, id).Scan(&agenID, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("agen: kunci pencairan: %w", err)
	}
	if status != "pending" {
		return 0, ErrPencairanBukanPending
	}
	if _, err := tx.ExecContext(ctx, `SELECT id FROM jamaah WHERE id = ? FOR UPDATE`, agenID); err != nil {
		return 0, fmt.Errorf("agen: kunci agen: %w", err)
	}
	return agenID, nil
}

// setujuiPencairanTx: saldo dicek ulang saat disetujui (7.6). Pengajuan ini
// sudah ikut mengurangi saldo tersedia, jadi saldo tidak boleh negatif.
func setujuiPencairanTx(ctx context.Context, tx *sql.Tx, id, adminID int64, buktiURL string) error {
	agenID, err := lockPencairan(ctx, tx, id)
	if err != nil {
		return err
	}
	saldo, err := komisi.HitungSaldo(ctx, tx, agenID)
	if err != nil {
		return err
	}
	if saldo.Tersedia < 0 {
		return ErrSaldoTidakCukup
	}
	_, err = tx.ExecContext(ctx, `
		UPDATE pengajuan_pencairan SET status = 'disetujui', bukti_transfer_keluar_url = ?, diproses_oleh = ?, diproses_at = NOW()
		WHERE id = ?`, buktiURL, adminID, id)
	if err != nil {
		return fmt.Errorf("agen: setujui pencairan: %w", err)
	}
	return nil
}

func tolakPencairanTx(ctx context.Context, tx *sql.Tx, id, adminID int64, catatan string) error {
	if _, err := lockPencairan(ctx, tx, id); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, `
		UPDATE pengajuan_pencairan SET status = 'ditolak', catatan_penolakan = ?, diproses_oleh = ?, diproses_at = NOW()
		WHERE id = ?`, catatan, adminID, id)
	if err != nil {
		return fmt.Errorf("agen: tolak pencairan: %w", err)
	}
	return nil
}

func listPencairanAdmin(ctx context.Context, q querier, status string, brandID *int64) ([]PencairanAdmin, error) {
	query := `
		SELECT p.id, p.nominal_diajukan, p.status, p.diajukan_at, p.diproses_at, p.catatan_penolakan,
		       p.rekening_bank, p.rekening_nomor, p.rekening_atas_nama, p.bukti_transfer_keluar_url IS NOT NULL,
		       p.jamaah_agen_id, j.nama_lengkap, j.kode_referral, p.brand_id, b.name, COALESCE(au.display_name, au.email), p.bukti_transfer_keluar_url
		FROM pengajuan_pencairan p
		JOIN jamaah j ON j.id = p.jamaah_agen_id
		JOIN brands b ON b.id = p.brand_id
		LEFT JOIN admin_users au ON au.id = p.diproses_oleh
		WHERE 1=1`
	args := []any{}
	if status == "pending" || status == "disetujui" || status == "ditolak" {
		query += ` AND p.status = ?`
		args = append(args, status)
	}
	if brandID != nil {
		query += ` AND p.brand_id = ?`
		args = append(args, *brandID)
	}
	query += ` ORDER BY p.diajukan_at ASC, p.id ASC LIMIT 500`
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("agen: list pencairan admin: %w", err)
	}
	items := []PencairanAdmin{}
	for rows.Next() {
		var p PencairanAdmin
		if err := rows.Scan(&p.ID, &p.NominalDiajukan, &p.Status, &p.DiajukanAt, &p.DiprosesAt, &p.CatatanPenolakan,
			&p.Bank, &p.Nomor, &p.AtasNama, &p.PunyaBukti,
			&p.JamaahAgenID, &p.NamaAgen, &p.KodeReferral, &p.BrandID, &p.BrandName, &p.DiprosesOlehNm, &p.BuktiURL); err != nil {
			rows.Close()
			return nil, fmt.Errorf("agen: scan pencairan admin: %w", err)
		}
		p.Pencairan.DiajukanAt = p.DiajukanAt
		items = append(items, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Status != "pending" {
			continue
		}
		s, err := komisi.HitungSaldo(ctx, q, items[i].JamaahAgenID)
		if err != nil {
			return nil, err
		}
		items[i].SaldoTersedia = s.Tersedia
	}
	return items, nil
}

// buktiPencairan mengembalikan URL bukti transfer keluar. agenID > 0 membatasi
// ke pengajuan milik agen itu (portal).
func buktiPencairan(ctx context.Context, q querier, id, agenID int64) (string, error) {
	query := `SELECT bukti_transfer_keluar_url FROM pengajuan_pencairan WHERE id = ? AND status = 'disetujui'`
	args := []any{id}
	if agenID > 0 {
		query += ` AND jamaah_agen_id = ?`
		args = append(args, agenID)
	}
	var url sql.NullString
	err := q.QueryRowContext(ctx, query, args...).Scan(&url)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && !url.Valid) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("agen: bukti pencairan: %w", err)
	}
	return url.String, nil
}

func (r *Repository) GetPencairanPortal(ctx context.Context, agenID int64) (*PencairanPortal, error) {
	return getPencairanPortal(ctx, r.db, agenID)
}

func (r *Repository) AjukanPencairan(ctx context.Context, agenID int64, req AjukanPencairanRequest) (int64, error) {
	var id int64
	err := r.withTx(ctx, func(tx *sql.Tx) error {
		var e error
		id, e = ajukanPencairanTx(ctx, tx, agenID, req)
		return e
	})
	return id, err
}

func (r *Repository) SetujuiPencairan(ctx context.Context, id, adminID int64, buktiURL string) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return setujuiPencairanTx(ctx, tx, id, adminID, buktiURL) })
}

func (r *Repository) TolakPencairan(ctx context.Context, id, adminID int64, catatan string) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return tolakPencairanTx(ctx, tx, id, adminID, catatan) })
}

func (r *Repository) ListPencairanAdmin(ctx context.Context, status string, brandID *int64) ([]PencairanAdmin, error) {
	return listPencairanAdmin(ctx, r.db, status, brandID)
}

func (r *Repository) BuktiPencairan(ctx context.Context, id, agenID int64) (string, error) {
	return buktiPencairan(ctx, r.db, id, agenID)
}
