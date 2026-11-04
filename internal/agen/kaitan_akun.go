package agen

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"erp-azhan/api/internal/identity"
)

// Override agen akun (screen C4, spek 4.4): Admin Master memasangkan akun tanpa agen, atau mengganti/
// melepas agen yang mengikat akun. Setiap perubahan dicatat di akun_kaitan_log.

var ErrKaitanAkunSiklus = errors.New("agen tujuan berada di bawah akun ini (rantai pengikat berputar)")

type AkunKaitan struct {
	AkunID      int64   `json:"akun_id"`
	NamaLengkap string  `json:"nama_lengkap"`
	Email       string  `json:"email"`
	NoHP        string  `json:"no_hp"`
	BrandID     int64   `json:"brand_id"`
	BrandName   string  `json:"brand_name"`
	StatusAgen  string  `json:"status_agen"`
	AgenID      *int64  `json:"agen_id"`
	AgenNama    *string `json:"agen_nama"`
	UplineID    *int64  `json:"upline_akun_id"`
	UplineNama  *string `json:"upline_nama"`
}

type GantiAgenAkunRequest struct {
	AgenAkunID *int64 `json:"agen_akun_id"` // null = lepas upline; akun yang sudah menjadi agen ikut mengganti upline_akun_id
	Alasan     string `json:"alasan"`
}

type AkunKaitanLog struct {
	ID           int64     `json:"id"`
	AgenLamaNama *string   `json:"agen_lama_nama"`
	AgenBaruNama *string   `json:"agen_baru_nama"`
	UplineDiubah bool      `json:"upline_diubah"`
	DigantiOleh  string    `json:"diganti_oleh"`
	DigantiAt    time.Time `json:"diganti_at"`
	Alasan       string    `json:"alasan"`
}

const qAkunKaitan = `
	SELECT a.id, a.nama_lengkap, a.email, a.no_hp, a.brand_id, b.name, a.status_agen,
	       a.direkrut_oleh_akun_id, ag.nama_lengkap, a.upline_akun_id, up.nama_lengkap
	FROM customer_accounts a
	JOIN brands b ON b.id = a.brand_id
	LEFT JOIN customer_accounts ag ON ag.id = a.direkrut_oleh_akun_id
	LEFT JOIN customer_accounts up ON up.id = a.upline_akun_id`

func scanAkunKaitan(sc interface{ Scan(...any) error }) (AkunKaitan, error) {
	var a AkunKaitan
	var agen, up sql.NullInt64
	if err := sc.Scan(&a.AkunID, &a.NamaLengkap, &a.Email, &a.NoHP, &a.BrandID, &a.BrandName, &a.StatusAgen,
		&agen, &a.AgenNama, &up, &a.UplineNama); err != nil {
		return a, err
	}
	if agen.Valid {
		a.AgenID = &agen.Int64
	}
	if up.Valid {
		a.UplineID = &up.Int64
	}
	return a, nil
}

func cariAkunKaitan(ctx context.Context, q querier, brandID *int64, cari string) ([]AkunKaitan, error) {
	cari = strings.TrimSpace(cari)
	if len(cari) < 3 {
		return []AkunKaitan{}, nil
	}
	like := "%" + cari + "%"
	query := qAkunKaitan + ` WHERE (a.nama_lengkap LIKE ? OR a.email LIKE ? OR a.no_hp LIKE ?)`
	args := []any{like, like, like}
	if brandID != nil {
		query += ` AND a.brand_id = ?`
		args = append(args, *brandID)
	}
	query += ` ORDER BY a.nama_lengkap LIMIT 30`
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("agen: cari akun kaitan: %w", err)
	}
	defer rows.Close()
	items := []AkunKaitan{}
	for rows.Next() {
		a, err := scanAkunKaitan(rows)
		if err != nil {
			return nil, fmt.Errorf("agen: scan akun kaitan: %w", err)
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

// gantiAgenAkunTx: brandID nil = Super Admin (semua brand); admin brand hanya boleh mengubah akun brand-nya sendiri.
func gantiAgenAkunTx(ctx context.Context, tx *sql.Tx, brandID *int64, akunID, adminID int64, req GantiAgenAkunRequest) error {
	a, err := scanAkunKaitan(tx.QueryRowContext(ctx, qAkunKaitan+` WHERE a.id = ? FOR UPDATE OF a`, akunID))
	if errors.Is(err, sql.ErrNoRows) || (err == nil && brandID != nil && *brandID != a.BrandID) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("agen: kunci akun kaitan: %w", err)
	}
	var baru any
	if req.AgenAkunID != nil {
		if *req.AgenAkunID == akunID {
			return ErrAgenTidakValid
		}
		if err := ValidasiAgenAktif(ctx, tx, a.BrandID, *req.AgenAkunID); err != nil {
			return err
		}
		// Rantai pengikat agen tujuan tidak boleh melewati akun ini. Tiap baris di rantai dikunci supaya dua override
		// bersamaan (A ke B dan B ke A) tidak sama-sama lolos dan membentuk siklus.
		cur := *req.AgenAkunID
		for i := 0; i < 50; i++ {
			var next sql.NullInt64
			if err := tx.QueryRowContext(ctx, `SELECT direkrut_oleh_akun_id FROM customer_accounts WHERE id = ? FOR UPDATE`, cur).Scan(&next); err != nil {
				return fmt.Errorf("agen: telusuri rantai pengikat: %w", err)
			}
			if !next.Valid {
				break
			}
			if next.Int64 == akunID {
				return ErrKaitanAkunSiklus
			}
			cur = next.Int64
		}
		baru = *req.AgenAkunID
	}
	samaID := func(x, y *int64) bool { return (x == nil && y == nil) || (x != nil && y != nil && *x == *y) }
	// Upline satu konsep dengan pengajak: akun yang sudah menjadi agen selalu ikut dipindah (bonus pembinaan menuju upline).
	ubahUpline := a.StatusAgen == "aktif" || a.StatusAgen == "nonaktif"
	if samaID(a.AgenID, req.AgenAkunID) && (!ubahUpline || samaID(a.UplineID, req.AgenAkunID)) {
		return ErrKaitanTidakBerubah
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_accounts SET direkrut_oleh_akun_id = ? WHERE id = ?`, baru, akunID); err != nil {
		return fmt.Errorf("agen: ganti agen akun: %w", err)
	}
	if ubahUpline {
		if _, err := tx.ExecContext(ctx, `UPDATE customer_accounts SET upline_akun_id = ? WHERE id = ?`, baru, akunID); err != nil {
			return fmt.Errorf("agen: ganti upline akun: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO akun_kaitan_log (akun_id, direkrut_oleh_akun_id_lama, direkrut_oleh_akun_id_baru, upline_diubah, diganti_oleh, alasan)
		VALUES (?, ?, ?, ?, ?, ?)`, akunID, a.AgenID, baru, ubahUpline, adminID, req.Alasan); err != nil {
		return fmt.Errorf("agen: log kaitan akun: %w", err)
	}
	return nil
}

func listAkunKaitanLog(ctx context.Context, q querier, brandID *int64, akunID int64) ([]AkunKaitanLog, error) {
	var ada bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM customer_accounts WHERE id = ? AND (? IS NULL OR brand_id = ?))`,
		akunID, brandID, brandID).Scan(&ada); err != nil {
		return nil, fmt.Errorf("agen: cek akun log kaitan: %w", err)
	}
	if !ada {
		return nil, ErrNotFound
	}
	rows, err := q.QueryContext(ctx, `
		SELECT l.id, lama.nama_lengkap, baru.nama_lengkap, l.upline_diubah, COALESCE(au.display_name, au.email), l.diganti_at, l.alasan
		FROM akun_kaitan_log l
		LEFT JOIN customer_accounts lama ON lama.id = l.direkrut_oleh_akun_id_lama
		LEFT JOIN customer_accounts baru ON baru.id = l.direkrut_oleh_akun_id_baru
		JOIN admin_users au ON au.id = l.diganti_oleh
		WHERE l.akun_id = ? ORDER BY l.id DESC`, akunID)
	if err != nil {
		return nil, fmt.Errorf("agen: log kaitan akun: %w", err)
	}
	defer rows.Close()
	items := []AkunKaitanLog{}
	for rows.Next() {
		var l AkunKaitanLog
		if err := rows.Scan(&l.ID, &l.AgenLamaNama, &l.AgenBaruNama, &l.UplineDiubah, &l.DigantiOleh, &l.DigantiAt, &l.Alasan); err != nil {
			return nil, fmt.Errorf("agen: scan log kaitan akun: %w", err)
		}
		items = append(items, l)
	}
	return items, rows.Err()
}

func (r *Repository) CariAkunKaitan(ctx context.Context, brandID *int64, cari string) ([]AkunKaitan, error) {
	return cariAkunKaitan(ctx, r.db, brandID, cari)
}

func (r *Repository) GantiAgenAkun(ctx context.Context, brandID *int64, akunID, adminID int64, req GantiAgenAkunRequest) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return gantiAgenAkunTx(ctx, tx, brandID, akunID, adminID, req) })
}

func (r *Repository) ListAkunKaitanLog(ctx context.Context, brandID *int64, akunID int64) ([]AkunKaitanLog, error) {
	return listAkunKaitanLog(ctx, r.db, brandID, akunID)
}

// CariAkunKaitan GET /api/admin/kaitan-agen/akun?q=&brand_id=
func (h *Handler) CariAkunKaitan(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.CariAkunKaitan(r.Context(), adminBrandScope(r), r.URL.Query().Get("q"))
	if err != nil {
		handleSprint5Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// LogKaitanAkun GET /api/admin/kaitan-agen/akun/{akunID}/log
func (h *Handler) LogKaitanAkun(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	items, err := h.repo.ListAkunKaitanLog(r.Context(), adminBrandScope(r), id)
	if err != nil {
		handleSprint5Error(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// GantiAgenAkun PUT /api/admin/kaitan-agen/akun/{akunID} {agen_akun_id|null, alasan}
func (h *Handler) GantiAgenAkun(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	var req GantiAgenAkunRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "request body tidak valid")
		return
	}
	req.Alasan = strings.TrimSpace(req.Alasan)
	if n := utf8.RuneCountInString(req.Alasan); n < 5 || n > 1000 {
		writeError(w, http.StatusBadRequest, "alasan wajib diisi (5–1000 karakter)")
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	if err := h.repo.GantiAgenAkun(r.Context(), adminBrandScope(r), id, adminID, req); err != nil {
		handleSprint5Error(w, err)
		return
	}
	log.Printf("[AUDIT] kaitan_akun_diganti akun_id=%d admin_id=%d", id, adminID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "agen akun diganti"})
}
