package agen

// Data bagian agen di tab Akun app Hana (keputusan 15 November 2026): mutasi dompet gabungan, agen binaan, data jamaah,
// dan klik referral. Semua untuk agen aktif; agen nonaktif ditolak oleh requireAgenAktif (layar A4).

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// zonaWIB: tanggal klik dan "bulan berjalan" dibaca sebagai waktu Indonesia Barat.
var zonaWIB = time.FixedZone("WIB", 7*3600)

// maksMutasi: jumlah baris mutasi terbaru yang dikirim ke app.
const maksMutasi = 50

// ─── Mutasi dompet ───────────────────────────────────────────────────────────

// Mutasi: satu baris riwayat dompet. Nominal bertanda: positif menambah saldo, negatif mengurangi.
type Mutasi struct {
	ID      string    `json:"id"`
	Jenis   string    `json:"jenis"` // komisi | koreksi | pencairan
	Judul   string    `json:"judul"`
	Tanggal time.Time `json:"tanggal"`
	Nominal float64   `json:"nominal"`
}

// listMutasiAgen menggabungkan komisi masuk, koreksi (komisi batal pada tanggal batalnya), dan pencairan yang
// diproses atau disetujui, urut terbaru. Cashback tidak masuk dompet (hanya potongan booking berikutnya).
func listMutasiAgen(ctx context.Context, q querier, akunID int64) ([]Mutasi, error) {
	out := []Mutasi{}
	rows, err := q.QueryContext(ctx, `
		SELECT tk.id, tk.nominal, tk.created_at, tk.dibatalkan_at, sb.nama_lengkap
		FROM transaksi_komisi tk
		JOIN jamaah sb ON sb.id = tk.jamaah_sumber_id
		WHERE tk.akun_penerima_id = ? AND tk.jenis <> 'cashback'
		ORDER BY GREATEST(tk.created_at, COALESCE(tk.dibatalkan_at, tk.created_at)) DESC, tk.id DESC
		LIMIT ?`, akunID, maksMutasi)
	if err != nil {
		return nil, fmt.Errorf("agen: mutasi komisi: %w", err)
	}
	for rows.Next() {
		var id int64
		var nominal float64
		var dibuat time.Time
		var batal sql.NullTime
		var sumber string
		if err := rows.Scan(&id, &nominal, &dibuat, &batal, &sumber); err != nil {
			rows.Close()
			return nil, fmt.Errorf("agen: mutasi komisi: %w", err)
		}
		out = append(out, Mutasi{ID: fmt.Sprintf("k-%d", id), Jenis: "komisi", Judul: "Komisi · " + sumber, Tanggal: dibuat, Nominal: nominal})
		if batal.Valid {
			out = append(out, Mutasi{ID: fmt.Sprintf("b-%d", id), Jenis: "koreksi", Judul: "Koreksi · " + sumber, Tanggal: batal.Time, Nominal: -nominal})
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	rows, err = q.QueryContext(ctx, `
		SELECT id, nominal_diajukan, diajukan_at, rekening_bank
		FROM pengajuan_pencairan
		WHERE akun_agen_id = ? AND status IN ('pending', 'disetujui')
		ORDER BY diajukan_at DESC, id DESC
		LIMIT ?`, akunID, maksMutasi)
	if err != nil {
		return nil, fmt.Errorf("agen: mutasi pencairan: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var nominal float64
		var diajukan time.Time
		var bank sql.NullString
		if err := rows.Scan(&id, &nominal, &diajukan, &bank); err != nil {
			return nil, fmt.Errorf("agen: mutasi pencairan: %w", err)
		}
		judul := "Pencairan"
		if b := strings.TrimSpace(bank.String); b != "" {
			judul = "Pencairan ke " + b
		}
		out = append(out, Mutasi{ID: fmt.Sprintf("p-%d", id), Jenis: "pencairan", Judul: judul, Tanggal: diajukan, Nominal: -nominal})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Tanggal.After(out[j].Tanggal) })
	if len(out) > maksMutasi {
		out = out[:maksMutasi]
	}
	return out, nil
}

func (r *Repository) ListMutasiAgen(ctx context.Context, akunID int64) ([]Mutasi, error) {
	if _, err := requireAgenAktif(ctx, r.db, akunID); err != nil {
		return nil, err
	}
	return listMutasiAgen(ctx, r.db, akunID)
}

// ─── Data jamaah ─────────────────────────────────────────────────────────────

// bookingTercatat: booking minimal DP (bukan draft, baru, atau batal).
const bookingTercatat = `b.status IN ('dp', 'lunas')`

// JamaahAgen: satu jamaah dalam satu booking minimal DP, dari jamaah yang terikat ke agen.
type JamaahAgen struct {
	BookingID     int64  `json:"booking_id"`
	JamaahID      int64  `json:"jamaah_id"`
	Nama          string `json:"nama"`
	Paket         string `json:"paket"`
	Jenis         string `json:"jenis"`          // umroh | haji | wisata
	TanggalDaftar string `json:"tanggal_daftar"` // YYYY-MM-DD, tanggal booking dibuat
	Status        string `json:"status"`         // dp | lunas | berangkat
}

func listDataJamaah(ctx context.Context, q querier, akunID int64, now time.Time) ([]JamaahAgen, error) {
	hariIni := now.In(zonaWIB).Format("2006-01-02")
	rows, err := q.QueryContext(ctx, `
		SELECT b.id, j.id, j.nama_lengkap, s.jadwal_nama, s.journey_type, DATE_FORMAT(b.created_at, '%Y-%m-%d'),
			b.status, DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d')
		FROM jamaah j
		JOIN booking_pax bp ON bp.jamaah_id = j.id AND bp.pax_status = 'aktif'
		JOIN bookings b ON b.id = bp.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE j.direkrut_oleh_akun_id = ? AND `+bookingTercatat+`
		ORDER BY b.created_at DESC, b.id DESC, bp.id`, akunID)
	if err != nil {
		return nil, fmt.Errorf("agen: data jamaah: %w", err)
	}
	defer rows.Close()
	out := []JamaahAgen{}
	for rows.Next() {
		var it JamaahAgen
		var jenis, status, berangkat string
		if err := rows.Scan(&it.BookingID, &it.JamaahID, &it.Nama, &it.Paket, &jenis, &it.TanggalDaftar, &status, &berangkat); err != nil {
			return nil, fmt.Errorf("agen: data jamaah: %w", err)
		}
		it.Jenis = jenis
		if jenis == "wisata_halal" {
			it.Jenis = "wisata"
		}
		it.Status = status
		if berangkat != "" && berangkat <= hariIni {
			it.Status = "berangkat"
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

func (r *Repository) ListDataJamaahAgen(ctx context.Context, akunID int64, now time.Time) ([]JamaahAgen, error) {
	if _, err := requireAgenAktif(ctx, r.db, akunID); err != nil {
		return nil, err
	}
	return listDataJamaah(ctx, r.db, akunID, now)
}

// ─── Agen binaan ─────────────────────────────────────────────────────────────

// AgenBinaan: agen yang upline-nya agen ini.
type AgenBinaan struct {
	AkunID       int64   `json:"akun_id"`
	Nama         string  `json:"nama"`
	Kota         *string `json:"kota"`
	Bergabung    *string `json:"bergabung"` // YYYY-MM-DD, tanggal disetujui sebagai agen
	JumlahJamaah int     `json:"jumlah_jamaah"`
	PaketAktif   int     `json:"paket_aktif"`
	Status       string  `json:"status"` // aktif | rintisan | nonaktif
}

func listAgenBinaan(ctx context.Context, q querier, akunID int64, now time.Time) ([]AgenBinaan, error) {
	hariIni := now.In(zonaWIB).Format("2006-01-02")
	rows, err := q.QueryContext(ctx, `
		SELECT a.id, a.nama_lengkap, NULLIF(TRIM(a.domisili), ''), DATE_FORMAT(a.disetujui_agen_at, '%Y-%m-%d'), a.status_agen,
			(SELECT COUNT(*) FROM jamaah j
				JOIN booking_pax bp ON bp.jamaah_id = j.id AND bp.pax_status = 'aktif'
				JOIN bookings b ON b.id = bp.booking_id
				WHERE j.direkrut_oleh_akun_id = a.id AND `+bookingTercatat+`),
			(SELECT COUNT(DISTINCT b.schedule_id) FROM jamaah j
				JOIN booking_pax bp ON bp.jamaah_id = j.id AND bp.pax_status = 'aktif'
				JOIN bookings b ON b.id = bp.booking_id
				JOIN schedules s ON s.id = b.schedule_id
				WHERE j.direkrut_oleh_akun_id = a.id AND `+bookingTercatat+` AND s.berangkat_tanggal > ?)
		FROM customer_accounts a
		WHERE a.upline_akun_id = ? AND a.status_agen IN ('aktif', 'nonaktif')
		ORDER BY a.nama_lengkap, a.id`, hariIni, akunID)
	if err != nil {
		return nil, fmt.Errorf("agen: agen binaan: %w", err)
	}
	defer rows.Close()
	out := []AgenBinaan{}
	for rows.Next() {
		var it AgenBinaan
		var statusAgen string
		if err := rows.Scan(&it.AkunID, &it.Nama, &it.Kota, &it.Bergabung, &statusAgen, &it.JumlahJamaah, &it.PaketAktif); err != nil {
			return nil, fmt.Errorf("agen: agen binaan: %w", err)
		}
		it.Status = statusBinaan(statusAgen, it.JumlahJamaah)
		out = append(out, it)
	}
	return out, rows.Err()
}

// statusBinaan: Nonaktif bila keagenan nonaktif; Rintisan bila aktif tetapi belum punya jamaah dengan booking minimal
// DP; selain itu Aktif.
func statusBinaan(statusAgen string, jumlahJamaah int) string {
	switch {
	case statusAgen != StatusAgenAktif:
		return "nonaktif"
	case jumlahJamaah == 0:
		return "rintisan"
	default:
		return "aktif"
	}
}

func (r *Repository) ListAgenBinaan(ctx context.Context, akunID int64, now time.Time) ([]AgenBinaan, error) {
	if _, err := requireAgenAktif(ctx, r.db, akunID); err != nil {
		return nil, err
	}
	return listAgenBinaan(ctx, r.db, akunID, now)
}

// ─── Klik referral ───────────────────────────────────────────────────────────

// klikBulanIni: jumlah klik referral agen pada bulan berjalan (WIB).
func klikBulanIni(ctx context.Context, q querier, akunID int64, now time.Time) (int, error) {
	t := now.In(zonaWIB)
	awal := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, zonaWIB).Format("2006-01-02")
	var n int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM referral_klik WHERE akun_id = ? AND tanggal >= ?`, akunID, awal).Scan(&n); err != nil {
		return 0, fmt.Errorf("agen: klik referral: %w", err)
	}
	return n, nil
}

// HashPengunjung: penanda pengunjung yang disimpan (sha256). Tanpa id dari situs, dipakai IP dan user agent.
func HashPengunjung(id, ip, userAgent string) string {
	sumber := strings.TrimSpace(id)
	if sumber == "" {
		sumber = "ipua|" + ip + "|" + userAgent
	}
	h := sha256.Sum256([]byte(sumber))
	return hex.EncodeToString(h[:])
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// catatKlikReferral mencatat satu klik link referral agen aktif di brand itu. Klik kedua dari pengunjung yang sama di
// hari yang sama diabaikan. Kode yang tidak dikenal juga diabaikan tanpa galat, supaya endpoint publik tidak bisa
// dipakai menebak kode referral. baru = true hanya bila satu baris klik ditambahkan.
func catatKlikReferral(ctx context.Context, db interface {
	querier
	execer
}, brandID int64, kode, pengunjungHash string, now time.Time) (baru bool, err error) {
	kode = strings.ToUpper(strings.TrimSpace(kode))
	if brandID <= 0 || kode == "" || len(kode) > 20 {
		return false, nil
	}
	var akunID int64
	err = db.QueryRowContext(ctx, `SELECT id FROM customer_accounts WHERE brand_id = ? AND kode_referral = ? AND status_agen = 'aktif'`,
		brandID, kode).Scan(&akunID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("agen: cari kode referral: %w", err)
	}
	res, err := db.ExecContext(ctx, `INSERT IGNORE INTO referral_klik (akun_id, tanggal, pengunjung) VALUES (?, ?, ?)`,
		akunID, now.In(zonaWIB).Format("2006-01-02"), pengunjungHash)
	if err != nil {
		return false, fmt.Errorf("agen: catat klik referral: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("agen: catat klik referral, jumlah baris: %w", err)
	}
	return n == 1, nil
}

func (r *Repository) CatatKlikReferral(ctx context.Context, brandID int64, kode, pengunjungHash string, now time.Time) (bool, error) {
	return catatKlikReferral(ctx, r.db, brandID, kode, pengunjungHash, now)
}
