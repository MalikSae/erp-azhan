package perlengkapan

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"erp-azhan/api/internal/identity"
	"github.com/go-chi/chi/v5"
)

// Isi perlengkapan per kategori paket (bawaan) dan per paket (penimpa).
// Urutan yang berlaku untuk satu paket:
//   paket (perlengkapan_custom) -> kategori -> set global (perlengkapan_set_template)

var untukValid = map[string]bool{"semua": true, "pria": true, "wanita": true}

// SetItem adalah satu baris isi perlengkapan.
type SetItem struct {
	PerlengkapanItemID uint64  `json:"perlengkapan_item_id"`
	Nama               string  `json:"nama"`
	PhotoURL           *string `json:"photo_url"`
	Qty                int     `json:"qty"`
	Untuk              string  `json:"untuk"`
}

// SetItemInput adalah baris isi perlengkapan pada request simpan.
type SetItemInput struct {
	PerlengkapanItemID uint64 `json:"perlengkapan_item_id"`
	Qty                int    `json:"qty"`
	Untuk              string `json:"untuk"`
}

// ScheduleSet adalah isi perlengkapan efektif sebuah paket.
type ScheduleSet struct {
	Custom       bool      `json:"custom"`
	Source       string    `json:"source"` // paket | kategori | global
	KategoriNama *string   `json:"kategori_nama"`
	Items        []SetItem `json:"items"`
	// Inherited = isi yang berlaku bila paket tidak memakai set khusus (kategori/global).
	Inherited       []SetItem `json:"inherited_items"`
	InheritedSource string    `json:"inherited_source"`
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func scanSetItems(rows *sql.Rows) ([]SetItem, error) {
	defer rows.Close()
	items := make([]SetItem, 0)
	for rows.Next() {
		var it SetItem
		if err := rows.Scan(&it.PerlengkapanItemID, &it.Nama, &it.PhotoURL, &it.Qty, &it.Untuk); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func normalizeInputs(inputs []SetItemInput) ([]SetItemInput, error) {
	seen := map[uint64]bool{}
	out := make([]SetItemInput, 0, len(inputs))
	for _, in := range inputs {
		if in.PerlengkapanItemID == 0 {
			return nil, fmt.Errorf("%w: item tidak valid", ErrInvalidInput)
		}
		if in.Qty < 1 {
			return nil, fmt.Errorf("%w: jumlah minimal 1", ErrInvalidInput)
		}
		in.Untuk = strings.ToLower(strings.TrimSpace(in.Untuk))
		if in.Untuk == "" {
			in.Untuk = "semua"
		}
		if !untukValid[in.Untuk] {
			return nil, fmt.Errorf("%w: nilai 'untuk' harus semua, pria, atau wanita", ErrInvalidInput)
		}
		if seen[in.PerlengkapanItemID] {
			return nil, fmt.Errorf("%w: item yang sama tidak boleh dipilih dua kali", ErrInvalidInput)
		}
		seen[in.PerlengkapanItemID] = true
		out = append(out, in)
	}
	return out, nil
}

func (r *Repository) itemsExist(ctx context.Context, q querier, inputs []SetItemInput) error {
	if len(inputs) == 0 {
		return nil
	}
	ids := make([]string, len(inputs))
	args := make([]any, len(inputs))
	for i, in := range inputs {
		ids[i] = "?"
		args[i] = in.PerlengkapanItemID
	}
	var n int
	if err := q.QueryRowContext(ctx,
		fmt.Sprintf("SELECT COUNT(*) FROM perlengkapan_items WHERE id IN (%s)", strings.Join(ids, ",")), args...).Scan(&n); err != nil {
		return err
	}
	if n != len(inputs) {
		return fmt.Errorf("%w: ada item yang tidak ditemukan di katalog", ErrInvalidInput)
	}
	return nil
}

// ─── Kategori ─────────────────────────────────────────────────────────────────

func (r *Repository) GetCategorySet(ctx context.Context, categoryID int64) ([]SetItem, error) {
	var exists int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM package_categories WHERE id = ?`, categoryID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, ErrNotFound
	}
	return r.categorySet(ctx, r.db, categoryID)
}

func (r *Repository) categorySet(ctx context.Context, q querier, categoryID int64) ([]SetItem, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT c.perlengkapan_item_id, i.nama, i.photo_url, c.qty, c.untuk
		FROM package_category_perlengkapan c
		JOIN perlengkapan_items i ON i.id = c.perlengkapan_item_id
		WHERE c.category_id = ?
		ORDER BY i.nama ASC`, categoryID)
	if err != nil {
		return nil, err
	}
	return scanSetItems(rows)
}

func (r *Repository) PutCategorySet(ctx context.Context, categoryID int64, inputs []SetItemInput) ([]SetItem, error) {
	inputs, err := normalizeInputs(inputs)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM package_categories WHERE id = ?`, categoryID).Scan(&exists); err != nil {
		return nil, err
	}
	if exists == 0 {
		return nil, ErrNotFound
	}
	if err := r.itemsExist(ctx, tx, inputs); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM package_category_perlengkapan WHERE category_id = ?`, categoryID); err != nil {
		return nil, err
	}
	for _, in := range inputs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO package_category_perlengkapan (category_id, perlengkapan_item_id, qty, untuk) VALUES (?,?,?,?)`,
			categoryID, in.PerlengkapanItemID, in.Qty, in.Untuk); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.categorySet(ctx, r.db, categoryID)
}

// ─── Paket ────────────────────────────────────────────────────────────────────

func (r *Repository) scheduleSetItems(ctx context.Context, q querier, scheduleID int64) ([]SetItem, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT s.perlengkapan_item_id, i.nama, i.photo_url, s.qty, s.untuk
		FROM schedule_perlengkapan s
		JOIN perlengkapan_items i ON i.id = s.perlengkapan_item_id
		WHERE s.schedule_id = ?
		ORDER BY i.nama ASC`, scheduleID)
	if err != nil {
		return nil, err
	}
	return scanSetItems(rows)
}

func (r *Repository) scheduleHeader(ctx context.Context, q querier, scheduleID int64, brandID *int64) (custom bool, categoryID sql.NullInt64, kategoriNama sql.NullString, err error) {
	query := `SELECT s.perlengkapan_custom, s.category_id, c.name
		FROM schedules s LEFT JOIN package_categories c ON c.id = s.category_id
		WHERE s.id = ?`
	args := []any{scheduleID}
	if brandID != nil {
		query += " AND s.brand_id = ?"
		args = append(args, *brandID)
	}
	err = q.QueryRowContext(ctx, query, args...).Scan(&custom, &categoryID, &kategoriNama)
	if errors.Is(err, sql.ErrNoRows) {
		err = ErrNotFound
	}
	return
}

func (r *Repository) inheritedSet(ctx context.Context, q querier, categoryID sql.NullInt64) ([]SetItem, string, error) {
	if categoryID.Valid {
		items, err := r.categorySet(ctx, q, categoryID.Int64)
		if err != nil {
			return nil, "", err
		}
		if len(items) > 0 {
			return items, "kategori", nil
		}
	}
	rows, err := q.QueryContext(ctx, `
		SELECT t.perlengkapan_item_id, i.nama, i.photo_url, t.qty, 'semua'
		FROM perlengkapan_set_template t
		JOIN perlengkapan_items i ON i.id = t.perlengkapan_item_id
		ORDER BY i.nama ASC`)
	if err != nil {
		return nil, "", err
	}
	items, err := scanSetItems(rows)
	return items, "global", err
}

func (r *Repository) GetScheduleSet(ctx context.Context, scheduleID int64, brandID *int64) (*ScheduleSet, error) {
	custom, categoryID, kategoriNama, err := r.scheduleHeader(ctx, r.db, scheduleID, brandID)
	if err != nil {
		return nil, err
	}
	inherited, inheritedSource, err := r.inheritedSet(ctx, r.db, categoryID)
	if err != nil {
		return nil, err
	}
	res := &ScheduleSet{Custom: custom, Inherited: inherited, InheritedSource: inheritedSource}
	if kategoriNama.Valid {
		n := kategoriNama.String
		res.KategoriNama = &n
	}
	if custom {
		items, err := r.scheduleSetItems(ctx, r.db, scheduleID)
		if err != nil {
			return nil, err
		}
		res.Items = items
		res.Source = "paket"
	} else {
		res.Items = inherited
		res.Source = inheritedSource
	}
	return res, nil
}

func (r *Repository) PutScheduleSet(ctx context.Context, scheduleID int64, brandID *int64, custom bool, inputs []SetItemInput) (*ScheduleSet, error) {
	inputs, err := normalizeInputs(inputs)
	if err != nil {
		return nil, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, _, _, err := r.scheduleHeader(ctx, tx, scheduleID, brandID); err != nil {
		return nil, err
	}
	if err := r.itemsExist(ctx, tx, inputs); err != nil {
		return nil, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM schedule_perlengkapan WHERE schedule_id = ?`, scheduleID); err != nil {
		return nil, err
	}
	if custom {
		for _, in := range inputs {
			if _, err := tx.ExecContext(ctx,
				`INSERT INTO schedule_perlengkapan (schedule_id, perlengkapan_item_id, qty, untuk) VALUES (?,?,?,?)`,
				scheduleID, in.PerlengkapanItemID, in.Qty, in.Untuk); err != nil {
				return nil, err
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schedules SET perlengkapan_custom = ? WHERE id = ?`, custom, scheduleID); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return r.GetScheduleSet(ctx, scheduleID, brandID)
}

// ─── Handler ──────────────────────────────────────────────────────────────────

type putSetRequest struct {
	Custom *bool          `json:"custom"`
	Items  []SetItemInput `json:"items"`
}

func parseIDParam(w http.ResponseWriter, r *http.Request, h *Handler) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		h.sendError(w, http.StatusBadRequest, "ID tidak valid")
		return 0, false
	}
	return id, true
}

func (h *Handler) setError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		h.sendError(w, http.StatusNotFound, "Data tidak ditemukan")
	case errors.Is(err, ErrInvalidInput):
		h.sendError(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), ErrInvalidInput.Error()+": "))
	default:
		h.sendError(w, http.StatusInternalServerError, "Gagal memproses isi perlengkapan")
	}
}

// GetCategorySet: GET /api/admin/categories/{id}/perlengkapan
func (h *Handler) GetCategorySet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, h)
	if !ok {
		return
	}
	items, err := h.repo.GetCategorySet(r.Context(), id)
	if err != nil {
		h.setError(w, err)
		return
	}
	h.sendJSON(w, http.StatusOK, map[string]any{"items": items})
}

// PutCategorySet: PUT /api/admin/categories/{id}/perlengkapan
func (h *Handler) PutCategorySet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, h)
	if !ok {
		return
	}
	var req putSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}
	items, err := h.repo.PutCategorySet(r.Context(), id, req.Items)
	if err != nil {
		h.setError(w, err)
		return
	}
	h.sendJSON(w, http.StatusOK, map[string]any{"items": items})
}

// GetScheduleSet: GET /api/admin/schedules/{id}/perlengkapan
func (h *Handler) GetScheduleSet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, h)
	if !ok {
		return
	}
	res, err := h.repo.GetScheduleSet(r.Context(), id, identity.GetBrandID(r.Context()))
	if err != nil {
		h.setError(w, err)
		return
	}
	h.sendJSON(w, http.StatusOK, res)
}

// PutScheduleSet: PUT /api/admin/schedules/{id}/perlengkapan
// custom=false menghapus set khusus sehingga paket kembali mengikuti kategori.
func (h *Handler) PutScheduleSet(w http.ResponseWriter, r *http.Request) {
	id, ok := parseIDParam(w, r, h)
	if !ok {
		return
	}
	var req putSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Custom == nil {
		h.sendError(w, http.StatusBadRequest, "Payload request tidak valid")
		return
	}
	res, err := h.repo.PutScheduleSet(r.Context(), id, identity.GetBrandID(r.Context()), *req.Custom, req.Items)
	if err != nil {
		h.setError(w, err)
		return
	}
	h.sendJSON(w, http.StatusOK, res)
}
