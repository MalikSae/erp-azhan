package manifest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// Pax yang masuk manifest: pax aktif dari booking baru, DP, dan lunas (draft dan batal tidak).
const syaratPax = `b.status IN ('baru','dp','lunas') AND bp.pax_status = 'aktif'`

// ambilPaket memuat header manifest; ErrTidakAda bila paket tidak ada atau di luar brand admin.
func ambilPaket(ctx context.Context, q querier, scheduleID int64, scope *int64) (*Paket, error) {
	query := `
		SELECT s.id, COALESCE(s.brand_id, 0), s.jadwal_nama,
		       DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d'), DATE_FORMAT(s.pulang_tanggal, '%Y-%m-%d'),
		       COALESCE(a.name, ''), COALESCE(a.code, ''),
		       COALESCE(mp.pnr, ''), COALESCE(mp.detail_penerbangan, '')
		FROM schedules s
		LEFT JOIN airlines a ON a.id = s.maskapai_id
		LEFT JOIN manifest_paket mp ON mp.schedule_id = s.id
		WHERE s.id = ?`
	args := []any{scheduleID}
	if scope != nil {
		query += ` AND s.brand_id = ?`
		args = append(args, *scope)
	}
	var p Paket
	err := q.QueryRowContext(ctx, query, args...).Scan(&p.ID, &p.BrandID, &p.Nama, &p.BerangkatTanggal, &p.PulangTanggal,
		&p.Maskapai, &p.KodeMaskapai, &p.PNR, &p.DetailPenerbangan)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTidakAda
	}
	if err != nil {
		return nil, fmt.Errorf("manifest.ambilPaket: %w", err)
	}
	return &p, nil
}

// ambilPax memuat pax manifest, urut per booking (waktu dibuat) lalu per pax (urutan pengisian).
func ambilPax(ctx context.Context, q querier, scheduleID int64) ([]Pax, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT bp.id, b.id, COALESCE(b.id_booking, ''), b.status, j.id, j.nama_lengkap, COALESCE(j.nama_paspor, ''),
		       COALESCE(j.jenis_kelamin, ''), COALESCE(j.nik, ''), COALESCE(j.tempat_lahir, ''),
		       COALESCE(DATE_FORMAT(j.tanggal_lahir, '%Y-%m-%d'), ''), COALESCE(j.no_paspor, ''),
		       COALESCE(DATE_FORMAT(j.tanggal_paspor_keluar, '%Y-%m-%d'), ''),
		       COALESCE(DATE_FORMAT(j.paspor_berlaku_sampai, '%Y-%m-%d'), ''), COALESCE(j.tempat_paspor_keluar, ''),
		       bp.pax_type, COALESCE(bp.room_type, '')
		FROM booking_pax bp
		JOIN bookings b ON b.id = bp.booking_id
		JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE b.schedule_id = ? AND `+syaratPax+`
		ORDER BY b.created_at, b.id, bp.id`, scheduleID)
	if err != nil {
		return nil, fmt.Errorf("manifest.ambilPax: %w", err)
	}
	defer rows.Close()
	out := []Pax{}
	for rows.Next() {
		var p Pax
		if err := rows.Scan(&p.PaxID, &p.BookingID, &p.KodeBooking, &p.StatusBooking, &p.JamaahID, &p.Nama, &p.NamaPaspor,
			&p.JenisKelamin, &p.NIK, &p.TempatLahir, &p.TanggalLahir, &p.NoPaspor, &p.PasporTerbit, &p.PasporHabis,
			&p.KantorPaspor, &p.TipePax, &p.TipeKamar); err != nil {
			return nil, fmt.Errorf("manifest.ambilPax scan: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func terakhirUnduh(ctx context.Context, q querier, scheduleID int64) (*Unduhan, error) {
	var u Unduhan
	err := q.QueryRowContext(ctx, `
		SELECT COALESCE(NULLIF(au.display_name, ''), au.email, ''), DATE_FORMAT(mu.dibuat_at, '%Y-%m-%dT%H:%i:%s')
		FROM manifest_unduhan mu
		LEFT JOIN admin_users au ON au.id = mu.admin_user_id
		WHERE mu.schedule_id = ?
		ORDER BY mu.dibuat_at DESC, mu.id DESC LIMIT 1`, scheduleID).Scan(&u.Nama, &u.Waktu)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("manifest.terakhirUnduh: %w", err)
	}
	return &u, nil
}

// Ambil memuat manifest lengkap satu paket beserta hasil pemeriksaan data.
func Ambil(ctx context.Context, q querier, scheduleID int64, scope *int64) (*Manifest, error) {
	p, err := ambilPaket(ctx, q, scheduleID, scope)
	if err != nil {
		return nil, err
	}
	pax, err := ambilPax(ctx, q, scheduleID)
	if err != nil {
		return nil, err
	}
	m := &Manifest{Paket: *p, Pax: pax, LamaHari: lamaHari(p.BerangkatTanggal, p.PulangTanggal)}
	for i := range m.Pax {
		periksaPax(&m.Pax[i], p.BerangkatTanggal, p.PulangTanggal)
		if len(m.Pax[i].Masalah) > 0 {
			m.JumlahMasalah++
		}
	}
	if m.TerakhirUnduh, err = terakhirUnduh(ctx, q, scheduleID); err != nil {
		return nil, err
	}
	return m, nil
}

// SimpanInfo menyimpan PNR dan detail penerbangan paket.
func SimpanInfo(ctx context.Context, q querier, scheduleID int64, scope *int64, adminID int64, in InputInfo) error {
	if err := in.rapikan(); err != nil {
		return err
	}
	if _, err := ambilPaket(ctx, q, scheduleID, scope); err != nil {
		return err
	}
	_, err := q.ExecContext(ctx, `
		INSERT INTO manifest_paket (schedule_id, pnr, detail_penerbangan, diubah_oleh) VALUES (?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, 0))
		ON DUPLICATE KEY UPDATE pnr = VALUES(pnr), detail_penerbangan = VALUES(detail_penerbangan), diubah_oleh = VALUES(diubah_oleh)`,
		scheduleID, in.PNR, in.DetailPenerbangan, adminID)
	if err != nil {
		return fmt.Errorf("manifest.SimpanInfo: %w", err)
	}
	return nil
}

// CatatUnduhan menyimpan jejak siapa mengunduh manifest dan berapa pax di dalamnya.
func CatatUnduhan(ctx context.Context, q querier, scheduleID, adminID int64, jumlahPax int) error {
	_, err := q.ExecContext(ctx, `INSERT INTO manifest_unduhan (schedule_id, admin_user_id, jumlah_pax) VALUES (?, NULLIF(?, 0), ?)`,
		scheduleID, adminID, jumlahPax)
	if err != nil {
		return fmt.Errorf("manifest.CatatUnduhan: %w", err)
	}
	return nil
}
