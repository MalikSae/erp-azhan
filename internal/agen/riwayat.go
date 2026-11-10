package agen

import (
	"context"
	"fmt"
)

// Riwayat untuk layar admin: keputusan atas satu siklus pengajuan (keputusan_agen_log) dan perubahan
// status agen antara aktif dan nonaktif (agen_status_log). Keduanya hanya dibaca dashboard, tidak portal.

func listRiwayatKeputusan(ctx context.Context, q querier, ppaID int64) ([]KeputusanLog, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT l.aksi, l.catatan, COALESCE(au.display_name, au.email), l.created_at
		FROM keputusan_agen_log l
		LEFT JOIN admin_users au ON au.id = l.admin_id
		WHERE l.pembayaran_pendaftaran_agen_id = ? ORDER BY l.id DESC`, ppaID)
	if err != nil {
		return nil, fmt.Errorf("agen: riwayat keputusan: %w", err)
	}
	defer rows.Close()
	items := []KeputusanLog{}
	for rows.Next() {
		var k KeputusanLog
		if err := rows.Scan(&k.Aksi, &k.Catatan, &k.OlehNama, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("agen: scan riwayat keputusan: %w", err)
		}
		items = append(items, k)
	}
	return items, rows.Err()
}

func listRiwayatStatus(ctx context.Context, q querier, akunID int64) ([]StatusLog, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT l.status_lama, l.status_baru, l.alasan, COALESCE(au.display_name, au.email), l.diubah_at
		FROM agen_status_log l
		LEFT JOIN admin_users au ON au.id = l.diubah_oleh
		WHERE l.akun_id = ? ORDER BY l.id DESC`, akunID)
	if err != nil {
		return nil, fmt.Errorf("agen: riwayat status: %w", err)
	}
	defer rows.Close()
	items := []StatusLog{}
	for rows.Next() {
		var s StatusLog
		if err := rows.Scan(&s.StatusLama, &s.StatusBaru, &s.Alasan, &s.OlehNama, &s.DiubahAt); err != nil {
			return nil, fmt.Errorf("agen: scan riwayat status: %w", err)
		}
		items = append(items, s)
	}
	return items, rows.Err()
}
