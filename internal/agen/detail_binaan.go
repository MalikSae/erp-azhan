package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
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

// RincianBonus: satu baris bonus pembinaan upline, catatan yang sama dengan yang dilihat agen binaan di Detail Jamaah
// miliknya (bonus_upline). Baris batal tetap ikut supaya yang hilang dari total bisa dilacak.
type RincianBonus struct {
	ID            int64   `json:"id"`
	Jamaah        string  `json:"jamaah"` // nama tersamar, mis. "Si** Am****"
	Paket         string  `json:"paket"`
	Tanggal       string  `json:"tanggal"` // YYYY-MM-DD, tanggal bonus tercatat
	Nominal       float64 `json:"nominal"`
	Status        string  `json:"status"`         // tertahan | tersedia | batal
	TersediaMulai string  `json:"tersedia_mulai"` // YYYY-MM-DD
	DibatalkanAt  *string `json:"dibatalkan_at"`  // YYYY-MM-DD, hanya untuk baris batal
}

// rincianRingkas: baris rincian terbaru di Detail Agen; selebihnya lewat Rincian Bonus berhalaman.
const rincianRingkas = 5

// MaksHalamanRincian: batas baris per halaman Rincian Bonus.
const MaksHalamanRincian = 50

// ErrStatusBonus: filter status Rincian Bonus bukan semua, tertahan, tersedia, atau batal.
var ErrStatusBonus = errors.New("status tidak valid")

// JumlahRincian: banyak baris rincian per status, untuk angka di chip filter.
type JumlahRincian struct {
	Semua    int `json:"semua"`
	Tertahan int `json:"tertahan"`
	Tersedia int `json:"tersedia"`
	Batal    int `json:"batal"`
}

// HalamanRincian: satu halaman Rincian Bonus menurut filter, ditambah jumlah per status (tanpa filter).
type HalamanRincian struct {
	Items  []RincianBonus `json:"items"`
	Jumlah JumlahRincian  `json:"jumlah"`
}

// namaTersamar menyisakan dua huruf pertama tiap kata: cukup untuk mencocokkan catatan, tidak cukup untuk mengenali
// atau menghubungi jamaah milik agen lain.
func namaTersamar(nama string) string {
	kata := strings.Fields(nama)
	for i, k := range kata {
		r := []rune(k)
		if len(r) > 2 {
			kata[i] = string(r[:2]) + strings.Repeat("*", len(r)-2)
		}
	}
	return strings.Join(kata, " ")
}

type DetailAgenBinaan struct {
	AkunID        int64          `json:"akun_id"`
	Nama          string         `json:"nama"`
	Kota          *string        `json:"kota"`
	Bergabung     *string        `json:"bergabung"` // YYYY-MM-DD, tanggal disetujui sebagai agen
	Status        string         `json:"status"`    // aktif | rintisan | nonaktif
	Jamaah        JamaahBinaan   `json:"jamaah"`
	PaketAktif    int            `json:"paket_aktif"`
	Bonus         BonusBinaan    `json:"bonus"`
	Rincian       []RincianBonus `json:"rincian"`        // paling banyak rincianRingkas baris terbaru
	JumlahRincian int            `json:"jumlah_rincian"` // semua baris rincian, termasuk yang batal
	WhatsAppURL   string         `json:"whatsapp_url"`   // kosong bila agen tidak punya nomor
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

	h, err := listRincianBonus(ctx, q, akunID, binaanID, "", 0, rincianRingkas)
	if err != nil {
		return nil, err
	}
	d.Rincian, d.JumlahRincian = h.Items, h.Jumlah.Semua
	return d, nil
}

// sqlRincianDari: bonus pembinaan akun (parameter 1) dari jamaah agen binaan (parameter 3); parameter 2 = jenis pembinaan.
const sqlRincianDari = `
	FROM transaksi_komisi tk
	JOIN jamaah j ON j.id = tk.jamaah_sumber_id
	JOIN bookings b ON b.id = tk.booking_id
	JOIN schedules s ON s.id = b.schedule_id
	WHERE tk.akun_penerima_id = ? AND tk.jenis = ? AND j.direkrut_oleh_akun_id = ?`

var (
	sqlBonusTersedia = "tk.dibatalkan_at IS NULL AND " + komisi.SQLTersediaMulai + " <= CURRENT_DATE"
	sqlBonusTertahan = "tk.dibatalkan_at IS NULL AND " + komisi.SQLTersediaMulai + " > CURRENT_DATE"
)

// sqlStatusBonus: syarat tambahan untuk filter status; "" atau "semua" tanpa syarat.
func sqlStatusBonus(status string) (string, error) {
	switch status {
	case "", "semua":
		return "", nil
	case "batal":
		return " AND tk.dibatalkan_at IS NOT NULL", nil
	case "tersedia":
		return " AND " + sqlBonusTersedia, nil
	case "tertahan":
		return " AND " + sqlBonusTertahan, nil
	}
	return "", ErrStatusBonus
}

// listRincianBonus: satu halaman bonus pembinaan dari agen binaan (terbaru dulu), baris batal ikut. Akses sudah dicek pemanggil.
func listRincianBonus(ctx context.Context, q querier, akunID, binaanID int64, status string, offset, limit int) (*HalamanRincian, error) {
	syarat, err := sqlStatusBonus(status)
	if err != nil {
		return nil, err
	}
	h := &HalamanRincian{Items: []RincianBonus{}}
	if err := q.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(`+sqlBonusTertahan+`), 0), COALESCE(SUM(`+sqlBonusTersedia+`), 0), COALESCE(SUM(tk.dibatalkan_at IS NOT NULL), 0)`+
		sqlRincianDari, akunID, komisi.JenisPembinaan, binaanID).Scan(&h.Jumlah.Semua, &h.Jumlah.Tertahan, &h.Jumlah.Tersedia, &h.Jumlah.Batal); err != nil {
		return nil, fmt.Errorf("agen: jumlah rincian bonus: %w", err)
	}
	rows, err := q.QueryContext(ctx, `
		SELECT tk.id, j.nama_lengkap, s.jadwal_nama, DATE_FORMAT(tk.created_at, '%Y-%m-%d'), tk.nominal,
		       COALESCE(DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d'), ''), DATE_FORMAT(tk.dibatalkan_at, '%Y-%m-%d')`+
		sqlRincianDari+syarat+`
		ORDER BY tk.created_at DESC, tk.id DESC
		LIMIT ? OFFSET ?`, akunID, komisi.JenisPembinaan, binaanID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("agen: rincian bonus: %w", err)
	}
	defer rows.Close()
	hariIni := time.Now().In(zonaWIB).Format("2006-01-02") // sama dengan CURRENT_DATE pada filter dan jumlah di atas
	for rows.Next() {
		var it RincianBonus
		var nama, berangkat string
		if err := rows.Scan(&it.ID, &nama, &it.Paket, &it.Tanggal, &it.Nominal, &berangkat, &it.DibatalkanAt); err != nil {
			return nil, fmt.Errorf("agen: rincian bonus: %w", err)
		}
		it.Jamaah = namaTersamar(nama)
		it.TersediaMulai, it.Status = tersediaMulai(berangkat, hariIni)
		if it.DibatalkanAt != nil {
			it.Status = "batal"
		}
		h.Items = append(h.Items, it)
	}
	return h, rows.Err()
}

// RincianBonusBinaan: halaman Rincian Bonus untuk upline langsung agen binaan; agen lain mendapat ErrNotFound.
func (r *Repository) RincianBonusBinaan(ctx context.Context, akunID, binaanID int64, status string, offset, limit int) (*HalamanRincian, error) {
	if _, err := requireAgenAktif(ctx, r.db, akunID); err != nil {
		return nil, err
	}
	var ada int
	err := r.db.QueryRowContext(ctx, `SELECT 1 FROM customer_accounts WHERE id = ? AND upline_akun_id = ? AND status_agen IN ('aktif', 'nonaktif')`, binaanID, akunID).Scan(&ada)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("agen: rincian bonus akses: %w", err)
	}
	return listRincianBonus(ctx, r.db, akunID, binaanID, status, offset, limit)
}

func (r *Repository) DetailAgenBinaan(ctx context.Context, akunID, binaanID int64, now time.Time) (*DetailAgenBinaan, error) {
	if _, err := requireAgenAktif(ctx, r.db, akunID); err != nil {
		return nil, err
	}
	return detailAgenBinaan(ctx, r.db, akunID, binaanID, now)
}
