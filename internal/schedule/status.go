package schedule

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"erp-azhan/api/internal/booking"
	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
)

// UpdateScheduleStatus godoc
// PUT /api/admin/schedules/{id}/status
// Body: {"status": "draft"|"published"|"archived"}
// Response 200: {"id": ..., "status": "..."}
func (h *Handler) UpdateScheduleStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		shared.WriteError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}

	validStatuses := map[string]bool{StatusDraft: true, StatusPublished: true, StatusArchived: true}
	if !validStatuses[req.Status] {
		shared.WriteError(w, http.StatusBadRequest, "Status tidak valid, harus draft/published/archived.")
		return
	}

	brandID := identity.GetBrandID(r.Context())
	if req.Status == StatusPublished || req.Status == StatusArchived {
		s, err := h.repo.GetByID(r.Context(), id, brandID)
		if err != nil {
			handleRepoError(w, err)
			return
		}
		if req.Status == StatusPublished && s.BerangkatTanggal < todayWIB() {
			shared.WriteError(w, http.StatusBadRequest, errPublishLampau)
			return
		}
		if !benefitCukup(w, req.Status, len(s.Benefits)) {
			return
		}
	}
	// Syarat arsip (booking aktif) diperiksa repository dengan baris paket terkunci.
	resID, resStatus, err := h.repo.UpdateStatus(r.Context(), id, req.Status, brandID, todayWIB())
	if err != nil {
		handleRepoError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, map[string]any{"id": resID, "status": resStatus})
}

// UpdateStatus mengubah hanya kolom status. Baris paket dikunci dulu, sehingga syarat arsip (MP-09: paket
// belum berangkat tanpa booking aktif) tidak terlewati booking yang masuk bersamaan. hariIni dari pemanggil.
func (r *Repository) UpdateStatus(ctx context.Context, id int64, status string, brandID *int64, hariIni string) (int64, string, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, "", fmt.Errorf("schedule.UpdateStatus tx: %w", err)
	}
	defer tx.Rollback()
	if err := ubahStatusTx(ctx, tx, id, status, brandID, hariIni); err != nil {
		return 0, "", err
	}
	if err := tx.Commit(); err != nil {
		return 0, "", fmt.Errorf("schedule.UpdateStatus commit: %w", err)
	}
	return id, status, nil
}

func ubahStatusTx(ctx context.Context, tx *sql.Tx, id int64, status string, brandID *int64, hariIni string) error {
	var lama, berangkat string
	var owner int64
	err := tx.QueryRowContext(ctx,
		`SELECT status, brand_id, DATE_FORMAT(berangkat_tanggal, '%Y-%m-%d') FROM schedules WHERE id=? FOR UPDATE`, id).
		Scan(&lama, &owner, &berangkat)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("schedule.UpdateStatus baca: %w", err)
	}
	if brandID != nil && owner != *brandID {
		return ErrNotFound
	}
	if status == StatusArchived && lama != StatusArchived && berangkat >= hariIni {
		aktif, err := booking.HitungAktif(ctx, tx, id)
		if err != nil {
			return fmt.Errorf("schedule: hitung booking aktif: %w", err)
		}
		if aktif > 0 {
			return fmt.Errorf("%w: paket masih punya %d booking aktif", ErrConflict, aktif)
		}
	}
	// Satu pernyataan: status dan penanda pernah tayang berubah bersama.
	if _, err := tx.ExecContext(ctx, `
		UPDATE schedules SET status=?, pernah_tayang_at = IF(? = 'published', COALESCE(pernah_tayang_at, NOW()), pernah_tayang_at)
		WHERE id=?`, status, status, id); err != nil {
		return fmt.Errorf("schedule.UpdateStatus: %w", err)
	}
	return nil
}

// tandaiPernahTayang mengisi schedules.pernah_tayang_at saat paket berstatus published dan penandanya masih
// kosong. Dipanggil setelah paket dibuat atau diubah, di transaksi yang sama.
func tandaiPernahTayang(ctx context.Context, tx *sql.Tx, id int64) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE schedules SET pernah_tayang_at = NOW()
		WHERE id = ? AND status = 'published' AND pernah_tayang_at IS NULL`, id); err != nil {
		return fmt.Errorf("schedule: tandai pernah tayang: %w", err)
	}
	return nil
}
