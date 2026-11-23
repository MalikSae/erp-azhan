package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"erp-azhan/api/internal/komisi"
)

// Detail satu agen binaan untuk upline langsungnya (membina). Hanya kinerja dan bonus pembinaan milik upline:
// nama jamaah, komisi milik agen binaan, NIK, rekening, dan nomor telepon sebagai teks tidak pernah ikut.

type JamaahBinaan struct {
	DP        int `json:"dp"`
	Lunas     int `json:"lunas"`
	Berangkat int `json:"berangkat"`
}

// BonusBinaan: bonus pembinaan yang diterima upline dari jamaah agen binaan ini (komisi aktif saja).
type BonusBinaan struct {
	Total    float64 `json:"total"`
	Tertahan float64 `json:"tertahan"`
	Tersedia float64 `json:"tersedia"`
}

type DetailAgenBinaan struct {
	AkunID      int64        `json:"akun_id"`
	Nama        string       `json:"nama"`
	Kota        *string      `json:"kota"`
	Bergabung   *string      `json:"bergabung"` // YYYY-MM-DD, tanggal disetujui sebagai agen
	Status      string       `json:"status"`    // aktif | rintisan | nonaktif
	Jamaah      JamaahBinaan `json:"jamaah"`
	PaketAktif  int          `json:"paket_aktif"`
	Bonus       BonusBinaan  `json:"bonus"`
	WhatsAppURL string       `json:"whatsapp_url"` // kosong bila agen tidak punya nomor
}

func detailAgenBinaan(ctx context.Context, q querier, akunID, binaanID int64, now time.Time) (*DetailAgenBinaan, error) {
	hariIni := now.In(zonaWIB).Format("2006-01-02")
	d := &DetailAgenBinaan{AkunID: binaanID}
	var statusAgen, noHP string
	// Hanya agen yang upline langsungnya akun ini (satu tingkat), sama dengan daftar Agen Binaan.
	err := q.QueryRowContext(ctx, `
		SELECT a.nama_lengkap, NULLIF(TRIM(a.domisili), ''), DATE_FORMAT(a.disetujui_agen_at, '%Y-%m-%d'), a.status_agen,
		       COALESCE(a.no_hp_normal, '')
		FROM customer_accounts a
		WHERE a.id = ? AND a.upline_akun_id = ? AND a.status_agen IN ('aktif', 'nonaktif')`, binaanID, akunID).
		Scan(&d.Nama, &d.Kota, &d.Bergabung, &statusAgen, &noHP)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("agen: detail binaan: %w", err)
	}
	if noHP != "" {
		d.WhatsAppURL = "https://wa.me/" + noHP
	}

	// Jamaah agen binaan per status (booking minimal DP); berangkat = tanggal berangkat sudah lewat.
	if err := q.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(s.berangkat_tanggal > ? AND b.status = 'dp'), 0),
		       COALESCE(SUM(s.berangkat_tanggal > ? AND b.status = 'lunas'), 0),
		       COALESCE(SUM(s.berangkat_tanggal <= ?), 0),
		       COUNT(DISTINCT CASE WHEN s.berangkat_tanggal > ? THEN b.schedule_id END)
		FROM jamaah j
		JOIN booking_pax bp ON bp.jamaah_id = j.id AND bp.pax_status = 'aktif'
		JOIN bookings b ON b.id = bp.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE j.direkrut_oleh_akun_id = ? AND `+bookingTercatat, hariIni, hariIni, hariIni, hariIni, binaanID).
		Scan(&d.Jamaah.DP, &d.Jamaah.Lunas, &d.Jamaah.Berangkat, &d.PaketAktif); err != nil {
		return nil, fmt.Errorf("agen: detail binaan jamaah: %w", err)
	}
	d.Status = statusBinaan(statusAgen, d.Jamaah.DP+d.Jamaah.Lunas+d.Jamaah.Berangkat)

	// Bonus pembinaan milik upline dari jamaah agen binaan ini.
	if err := q.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN `+komisi.SQLTersediaMulai+` > CURRENT_DATE THEN tk.nominal END), 0),
		       COALESCE(SUM(CASE WHEN `+komisi.SQLTersediaMulai+` <= CURRENT_DATE THEN tk.nominal END), 0)
		FROM transaksi_komisi tk
		JOIN jamaah j ON j.id = tk.jamaah_sumber_id
		JOIN bookings b ON b.id = tk.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE tk.akun_penerima_id = ? AND tk.jenis = ? AND tk.dibatalkan_at IS NULL AND j.direkrut_oleh_akun_id = ?`,
		akunID, komisi.JenisPembinaan, binaanID).Scan(&d.Bonus.Tertahan, &d.Bonus.Tersedia); err != nil {
		return nil, fmt.Errorf("agen: detail binaan bonus: %w", err)
	}
	d.Bonus.Total = d.Bonus.Tertahan + d.Bonus.Tersedia
	return d, nil
}

func (r *Repository) DetailAgenBinaan(ctx context.Context, akunID, binaanID int64, now time.Time) (*DetailAgenBinaan, error) {
	if _, err := requireAgenAktif(ctx, r.db, akunID); err != nil {
		return nil, err
	}
	return detailAgenBinaan(ctx, r.db, akunID, binaanID, now)
}
