package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
)

// SamakanScheduleSeat godoc
// POST /api/admin/schedules/{id}/seat/samakan
// Mengisi seat_sisa = seat_total − seat dipegang booking. Tidak ada isian angka: semua seat
// terpakai wajib punya booking di ERP.
// Response 200: {"id": ..., "seat_sisa": ...}
func (h *Handler) SamakanScheduleSeat(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	sisa, err := h.repo.SamakanSeat(r.Context(), id, identity.GetBrandID(r.Context()), identity.GetAdminUserID(r.Context()))
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "seat_sisa": sisa})
}

// UpdateScheduleSeatTampil godoc
// PUT /api/admin/schedules/{id}/seat-tampil
// Body: {"seat_tampil": int|null}; null mematikan scarcity. Hanya label publik, seat_sisa tidak berubah.
// Response 200: {"id": ..., "seat_tampil": ...}
func (h *Handler) UpdateScheduleSeatTampil(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	req, err := parseSeatTampilBody(r.Body)
	if err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	seatTotal, err := h.repo.GetSeatTotal(r.Context(), id, brandID)
	if err != nil {
		handleRepoError(w, err)
		return
	}
	if req.SeatTampil != nil && (*req.SeatTampil < 1 || *req.SeatTampil > seatTotal) {
		shared.WriteError(w, http.StatusBadRequest, fmt.Sprintf("Seat tampil harus antara 1 dan %d.", seatTotal))
		return
	}

	if err := h.repo.UpdateSeatTampil(r.Context(), id, req.SeatTampil, brandID, identity.GetAdminUserID(r.Context())); err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]any{"id": id, "seat_tampil": req.SeatTampil})
}

// parseSeatTampilBody membaca {"seat_tampil": int|null}. Field wajib ada: hanya null eksplisit yang
// mematikan scarcity, supaya body kosong tidak mematikannya tanpa sengaja.
func parseSeatTampilBody(body io.Reader) (UpdateSeatTampilRequest, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(body).Decode(&raw); err != nil {
		return UpdateSeatTampilRequest{}, err
	}
	v, ok := raw["seat_tampil"]
	if !ok {
		return UpdateSeatTampilRequest{}, errors.New("seat_tampil wajib dikirim")
	}
	var req UpdateSeatTampilRequest
	if err := json.Unmarshal(v, &req.SeatTampil); err != nil {
		return UpdateSeatTampilRequest{}, err
	}
	return req, nil
}

// GetScheduleSeat godoc
// GET /api/admin/schedules/{id}/seat
// Response 200: SeatOverview (ringkasan seat + riwayat penyesuaian terakhir)
func (h *Handler) GetScheduleSeat(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	o, err := h.repo.SeatOverview(r.Context(), id, identity.GetBrandID(r.Context()))
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, o)
}

// GetSeatTotal mengambil seat_total dari schedule berdasarkan id (batas atas seat tampil).
func (r *Repository) GetSeatTotal(ctx context.Context, id int64, brandID *int64) (int, error) {
	var seatTotal int
	q := `SELECT seat_total FROM schedules WHERE id=?`
	var args []interface{}
	args = append(args, id)
	if brandID != nil {
		q += " AND brand_id=?"
		args = append(args, *brandID)
	}
	err := r.db.QueryRowContext(ctx, q, args...).Scan(&seatTotal)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("schedule.GetSeatTotal: %w", err)
	}
	return seatTotal, nil
}

// SamakanSeat mengisi seat_sisa = seat_total − seat dipegang booking (minimal 0). Hanya untuk
// memperbaiki sisa seat yang tidak cocok dengan booking; dicatat di riwayat seat (jenis 'samakan') dan log server.
func (r *Repository) SamakanSeat(ctx context.Context, id int64, brandID *int64, actorID int64) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	owner, before, after, err := samakanSeatTx(ctx, tx, id, brandID, actorID)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	if before != after {
		log.Printf("[AUDIT] schedule_seat_samakan id=%d brand=%d before=%d after=%d by=%d", id, owner, before, after, actorID)
	}
	return after, nil
}

func samakanSeatTx(ctx context.Context, tx *sql.Tx, id int64, brandID *int64, actorID int64) (owner int64, before, after int, err error) {
	var total int
	err = tx.QueryRowContext(ctx, `SELECT seat_total, seat_sisa, brand_id FROM schedules WHERE id=? FOR UPDATE`, id).Scan(&total, &before, &owner)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, 0, 0, ErrNotFound
	}
	if err != nil {
		return 0, 0, 0, err
	}
	if brandID != nil && owner != *brandID {
		return 0, 0, 0, ErrNotFound
	}
	dipegang, err := shared.SeatDipegangBooking(ctx, tx, id)
	if err != nil {
		return 0, 0, 0, err
	}
	after = total - dipegang
	if after < 0 {
		after = 0
	}
	if after != before {
		if _, err := tx.ExecContext(ctx, `UPDATE schedules SET seat_sisa=? WHERE id=?`, after, id); err != nil {
			return 0, 0, 0, err
		}
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO schedule_seat_adjustments (schedule_id, jenis, admin_user_id, seat_before, seat_after, reason) VALUES (?, 'samakan', ?, ?, ?, ?)`,
			id, nullInt64(actorID), before, after, "Samakan dengan booking"); err != nil {
			return 0, 0, 0, fmt.Errorf("catat riwayat samakan seat: %w", err)
		}
	}
	return owner, before, after, nil
}

// UpdateSeatTampil menyimpan angka scarcity (nil = nonaktif) dan mencatatnya di riwayat seat
// dengan jenis 'tampil' (0 = nonaktif). Tidak menyentuh seat_sisa.
func (r *Repository) UpdateSeatTampil(ctx context.Context, id int64, seatTampil *int, brandID *int64, actorID int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := updateSeatTampilTx(ctx, tx, id, seatTampil, brandID, actorID); err != nil {
		return err
	}
	return tx.Commit()
}

func updateSeatTampilTx(ctx context.Context, tx *sql.Tx, id int64, seatTampil *int, brandID *int64, actorID int64) error {
	var total int
	var current sql.NullInt64
	var owner int64
	err := tx.QueryRowContext(ctx, `SELECT seat_total, seat_tampil, brand_id FROM schedules WHERE id=? FOR UPDATE`, id).Scan(&total, &current, &owner)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if brandID != nil && owner != *brandID {
		return ErrNotFound
	}
	if seatTampil != nil && (*seatTampil < 1 || *seatTampil > total) {
		return fmt.Errorf("%w: kuota paket berubah, muat ulang paket", ErrConflict)
	}
	before := int(current.Int64) // NULL → 0
	after := 0
	var seatTampilDB any
	if seatTampil != nil {
		after = *seatTampil
		seatTampilDB = after
	}
	if before == after {
		return nil
	}
	if _, err := tx.ExecContext(ctx, `UPDATE schedules SET seat_tampil=? WHERE id=?`, seatTampilDB, id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO schedule_seat_adjustments (schedule_id, jenis, admin_user_id, seat_before, seat_after, reason) VALUES (?, 'tampil', ?, ?, ?, ?)`,
		id, nullInt64(actorID), before, after, "Seat tampil"); err != nil {
		return fmt.Errorf("catat riwayat angka tampil: %w", err)
	}
	return nil
}

// SeatOverview mengambil ringkasan seat dan 20 riwayat penyesuaian terakhir.
// brandID non-nil membatasi ke paket milik brand itu (ErrNotFound bila bukan).
func (r *Repository) SeatOverview(ctx context.Context, id int64, brandID *int64) (*SeatOverview, error) {
	var o SeatOverview
	var seatTampil sql.NullInt64
	err := r.db.QueryRowContext(ctx, `
		SELECT s.seat_total, s.seat_sisa, s.seat_tampil
		FROM schedules s WHERE s.id=? AND (? IS NULL OR s.brand_id=?)`, id, brandID, brandID).
		Scan(&o.SeatTotal, &o.SeatSisa, &seatTampil)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("schedule.SeatOverview: %w", err)
	}
	if o.SeatDialokasikan, err = shared.SeatDipegangBooking(ctx, r.db, id); err != nil {
		return nil, fmt.Errorf("schedule.SeatOverview seat dipegang: %w", err)
	}
	o.SeatSesuaiBooking = o.SeatTotal - o.SeatDialokasikan
	if o.SeatSesuaiBooking < 0 {
		o.SeatSesuaiBooking = 0
	}
	o.SeatCocok = o.SeatSisa == o.SeatSesuaiBooking
	if seatTampil.Valid {
		v := int(seatTampil.Int64)
		o.SeatTampil = &v
	}

	rows, err := r.db.QueryContext(ctx, `
		SELECT a.id, a.jenis, a.seat_before, a.seat_after, a.reason, u.email, a.created_at
		FROM schedule_seat_adjustments a LEFT JOIN admin_users u ON u.id=a.admin_user_id
		WHERE a.schedule_id=? ORDER BY a.id DESC LIMIT 20`, id)
	if err != nil {
		return nil, fmt.Errorf("schedule.SeatOverview riwayat: %w", err)
	}
	defer rows.Close()
	o.Adjustments = []SeatAdjustment{}
	for rows.Next() {
		var a SeatAdjustment
		var email sql.NullString
		if err := rows.Scan(&a.ID, &a.Jenis, &a.SeatBefore, &a.SeatAfter, &a.Reason, &email, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("schedule.SeatOverview scan: %w", err)
		}
		if email.Valid {
			a.AdminEmail = &email.String
		}
		o.Adjustments = append(o.Adjustments, a)
	}
	return &o, rows.Err()
}

// alasanUbahKuota: keterangan riwayat penyesuaian seat untuk perubahan kuota lewat form edit paket.
func alasanUbahKuota(kuotaLama, kuotaBaru int) string {
	return fmt.Sprintf("Ubah kuota seat lewat edit paket: %d menjadi %d", kuotaLama, kuotaBaru)
}
