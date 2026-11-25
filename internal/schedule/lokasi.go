package schedule

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/go-sql-driver/mysql"
)

// Lokasi Dikunjungi (migration 148): Jelajah Haramain places picked per paket, ordered by admin.

// MaksLokasi is the max number of locations per paket.
const MaksLokasi = 12

// LokasiPaket is one location shown on a public paket (active places only).
type LokasiPaket struct {
	ID      int64  `json:"id"`
	Nama    string `json:"nama"`
	Kota    string `json:"kota"`
	FotoURL string `json:"foto_url"`
}

// LokasiAdmin is one linked location on the admin paket detail (inactive included).
type LokasiAdmin struct {
	LokasiPaket
	Aktif bool `json:"aktif"`
}

// LokasiError carries a message safe to show to the admin (400).
type LokasiError string

func (e LokasiError) Error() string { return string(e) }

type lokasiQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func placeholderID(ids []int64) (string, []any) {
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = "?"
		args[i] = id
	}
	return strings.Join(ph, ","), args
}

// cekDaftarLokasi checks the list shape: max 12, no duplicates, positive ids.
func cekDaftarLokasi(ids []int64) error {
	if len(ids) > MaksLokasi {
		return LokasiError(fmt.Sprintf("Lokasi maksimal %d", MaksLokasi))
	}
	seen := make(map[int64]bool, len(ids))
	for _, id := range ids {
		if id <= 0 {
			return LokasiError("Lokasi tidak ditemukan")
		}
		if seen[id] {
			return LokasiError("Lokasi tidak boleh dobel")
		}
		seen[id] = true
	}
	return nil
}

// validasiLokasi checks the list and that every id exists in haramain_tempat. An inactive tempat is accepted
// only when it is already linked to that paket (scheduleID; 0 for a new paket).
func validasiLokasi(ctx context.Context, q rowsQuerier, scheduleID int64, ids []int64) error {
	if err := cekDaftarLokasi(ids); err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	ph, args := placeholderID(ids)
	args = append([]any{scheduleID}, args...)
	rows, err := q.QueryContext(ctx, `
		SELECT t.aktif OR EXISTS (SELECT 1 FROM schedule_lokasi sl WHERE sl.schedule_id = ? AND sl.tempat_id = t.id)
		FROM haramain_tempat t WHERE t.id IN (`+ph+`)`, args...)
	if err != nil {
		return fmt.Errorf("schedule.validasiLokasi: %w", err)
	}
	defer rows.Close()
	n, nonaktif := 0, false
	for rows.Next() {
		var boleh bool
		if err := rows.Scan(&boleh); err != nil {
			return fmt.Errorf("schedule.validasiLokasi scan: %w", err)
		}
		n++
		if !boleh {
			nonaktif = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("schedule.validasiLokasi: %w", err)
	}
	if n != len(ids) {
		return LokasiError("Lokasi tidak ditemukan")
	}
	if nonaktif {
		return LokasiError("Lokasi tidak aktif")
	}
	return nil
}

// gantiLokasi replaces all locations of a paket; called inside the paket save transaction.
func gantiLokasi(ctx context.Context, q lokasiQuerier, scheduleID int64, ids []int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM schedule_lokasi WHERE schedule_id = ?`, scheduleID); err != nil {
		return fmt.Errorf("schedule.gantiLokasi delete: %w", err)
	}
	if len(ids) == 0 {
		return nil
	}
	ph := make([]string, len(ids))
	args := make([]any, 0, len(ids)*3)
	for i, id := range ids {
		ph[i] = "(?, ?, ?)"
		args = append(args, scheduleID, id, i+1)
	}
	if _, err := q.ExecContext(ctx, `INSERT INTO schedule_lokasi (schedule_id, tempat_id, urutan) VALUES `+strings.Join(ph, ","), args...); err != nil {
		// FK (1452): a tempat was deleted after validasiLokasi.
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1452 {
			return LokasiError("Lokasi tidak ditemukan")
		}
		return fmt.Errorf("schedule.gantiLokasi insert: %w", err)
	}
	return nil
}

// ubahLokasi replaces a paket's locations on update. When the list changed it bumps
// schedules.updated_at so the public updated_since filter picks the paket up.
func ubahLokasi(ctx context.Context, q lokasiQuerier, scheduleID int64, ids []int64) error {
	lama, err := lokasiIDsJadwal(ctx, q, scheduleID)
	if err != nil {
		return err
	}
	if slices.Equal(lama, ids) {
		return nil
	}
	if err := gantiLokasi(ctx, q, scheduleID, ids); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `UPDATE schedules SET updated_at = NOW() WHERE id = ?`, scheduleID); err != nil {
		return fmt.Errorf("schedule.ubahLokasi touch: %w", err)
	}
	return nil
}

// TandaiDiubahLewatLokasi bumps schedules.updated_at of every paket linked to a Jelajah Haramain place,
// so the public updated_since filter picks them up after the place's shown fields change.
// Called by the haramain module; schedules stays written only by this module.
func TandaiDiubahLewatLokasi(ctx context.Context, q interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}, tempatID int64) error {
	_, err := q.ExecContext(ctx, `
		UPDATE schedules s JOIN schedule_lokasi sl ON sl.schedule_id = s.id
		SET s.updated_at = NOW() WHERE sl.tempat_id = ?`, tempatID)
	return err
}

// lokasiIDsJadwal returns the linked tempat ids of a paket in admin order (inactive included); [] when none.
func lokasiIDsJadwal(ctx context.Context, q rowsQuerier, scheduleID int64) ([]int64, error) {
	rows, err := q.QueryContext(ctx, `SELECT tempat_id FROM schedule_lokasi WHERE schedule_id = ? ORDER BY urutan, tempat_id`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("schedule.lokasiIDsJadwal: %w", err)
	}
	defer rows.Close()
	out := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("schedule.lokasiIDsJadwal scan: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// lokasiJadwal returns the linked tempat of a paket in admin order (inactive included); [] when none.
func lokasiJadwal(ctx context.Context, q rowsQuerier, scheduleID int64) ([]LokasiAdmin, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT t.id, t.nama, t.kota, t.foto_url, t.aktif
		FROM schedule_lokasi sl
		JOIN haramain_tempat t ON t.id = sl.tempat_id
		WHERE sl.schedule_id = ?
		ORDER BY sl.urutan, t.id`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("schedule.lokasiJadwal: %w", err)
	}
	defer rows.Close()
	out := []LokasiAdmin{}
	for rows.Next() {
		var l LokasiAdmin
		if err := rows.Scan(&l.ID, &l.Nama, &l.Kota, &l.FotoURL, &l.Aktif); err != nil {
			return nil, fmt.Errorf("schedule.lokasiJadwal scan: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// idLokasi returns the ids of locations, keeping order; [] when none.
func idLokasi(l []LokasiAdmin) []int64 {
	out := make([]int64, len(l))
	for i, x := range l {
		out[i] = x.ID
	}
	return out
}

// lokasiPublik loads active locations of several pakets in one query, ordered per paket.
// Pakets without locations have no key in the result.
func lokasiPublik(ctx context.Context, q rowsQuerier, scheduleIDs []int64) (map[int64][]LokasiPaket, error) {
	out := make(map[int64][]LokasiPaket, len(scheduleIDs))
	if len(scheduleIDs) == 0 {
		return out, nil
	}
	ph, args := placeholderID(scheduleIDs)
	rows, err := q.QueryContext(ctx, `
		SELECT sl.schedule_id, t.id, t.nama, t.kota, t.foto_url
		FROM schedule_lokasi sl
		JOIN haramain_tempat t ON t.id = sl.tempat_id AND t.aktif
		WHERE sl.schedule_id IN (`+ph+`)
		ORDER BY sl.schedule_id, sl.urutan, t.id`, args...)
	if err != nil {
		return nil, fmt.Errorf("schedule.lokasiPublik: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var sid int64
		var l LokasiPaket
		if err := rows.Scan(&sid, &l.ID, &l.Nama, &l.Kota, &l.FotoURL); err != nil {
			return nil, fmt.Errorf("schedule.lokasiPublik scan: %w", err)
		}
		out[sid] = append(out[sid], l)
	}
	return out, rows.Err()
}

// opsiLokasi lists active tempat in master order for the paket form.
func opsiLokasi(ctx context.Context, q rowsQuerier) ([]LokasiPaket, error) {
	rows, err := q.QueryContext(ctx, `SELECT id, nama, kota, foto_url FROM haramain_tempat WHERE aktif ORDER BY urutan, id`)
	if err != nil {
		return nil, fmt.Errorf("schedule.opsiLokasi: %w", err)
	}
	defer rows.Close()
	out := []LokasiPaket{}
	for rows.Next() {
		var l LokasiPaket
		if err := rows.Scan(&l.ID, &l.Nama, &l.Kota, &l.FotoURL); err != nil {
			return nil, fmt.Errorf("schedule.opsiLokasi scan: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// ValidasiLokasi checks a paket's lokasi_ids against the master.
func (r *Repository) ValidasiLokasi(ctx context.Context, scheduleID int64, ids []int64) error {
	return validasiLokasi(ctx, r.db, scheduleID, ids)
}

// LokasiPublik loads active locations of several pakets in one query.
func (r *Repository) LokasiPublik(ctx context.Context, scheduleIDs []int64) (map[int64][]LokasiPaket, error) {
	return lokasiPublik(ctx, r.db, scheduleIDs)
}

// lokasiSah validates lokasi_ids; user errors are written as 400.
func (h *Handler) lokasiSah(ctx context.Context, w http.ResponseWriter, scheduleID int64, ids []int64) bool {
	err := h.repo.ValidasiLokasi(ctx, scheduleID, ids)
	if err == nil {
		return true
	}
	var le LokasiError
	if errors.As(err, &le) {
		writeError(w, http.StatusBadRequest, le.Error())
		return false
	}
	log.Printf("[ERROR] schedule: validasi lokasi: %v", err)
	writeError(w, http.StatusInternalServerError, "Gagal memeriksa lokasi.")
	return false
}

// ListLokasiOptions godoc
// GET /api/admin/schedules/lokasi-options
// Response 200: [{id, nama, kota, foto_url}] active Jelajah Haramain places, master order.
func (h *Handler) ListLokasiOptions(w http.ResponseWriter, r *http.Request) {
	items, err := opsiLokasi(r.Context(), h.repo.db)
	if err != nil {
		log.Printf("[ERROR] schedule: opsi lokasi: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal memuat lokasi.")
		return
	}
	writeJSON(w, http.StatusOK, items)
}
