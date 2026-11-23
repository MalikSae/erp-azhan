package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/shared"
)

// Detail satu jamaah untuk agen perekrutnya (follow-up pembayaran dan kelengkapan). Hanya status dan hitungan:
// nilai isian, dokumen, dan nomor telepon sebagai teks tidak pernah ikut (keputusan akses data jamaah 11 November 2026).

type PembayaranJamaah struct {
	Total        float64 `json:"total"`
	Dibayar      float64 `json:"dibayar"`
	Sisa         float64 `json:"sisa"`
	TenggatLunas string  `json:"tenggat_lunas"` // YYYY-MM-DD
}

type HitunganJamaah struct {
	Terisi int `json:"terisi"`
	Total  int `json:"total"`
}

type KelengkapanJamaah struct {
	Data    HitunganJamaah `json:"data"`
	Dokumen HitunganJamaah `json:"dokumen"`
}

type TahapanJamaah struct {
	Visa         bool `json:"visa"`
	Manasik      bool `json:"manasik"`
	Perlengkapan bool `json:"perlengkapan"`
}

type KomisiJamaah struct {
	Nominal       float64 `json:"nominal"`
	Status        string  `json:"status"`         // tertahan | tersedia | batal
	TersediaMulai string  `json:"tersedia_mulai"` // YYYY-MM-DD
}

// DetailJamaahAgen: Pembayaran dan Kelengkapan nil untuk jamaah yang sudah berangkat.
type DetailJamaahAgen struct {
	BookingID    int64              `json:"booking_id"`
	JamaahID     int64              `json:"jamaah_id"`
	Nama         string             `json:"nama"`
	NIKTersamar  string             `json:"nik_tersamar"`
	Paket        string             `json:"paket"`
	Jenis        string             `json:"jenis"`
	KodeBooking  string             `json:"kode_booking"`
	Berangkat    string             `json:"berangkat"` // YYYY-MM-DD
	Status       string             `json:"status"`    // dp | lunas | berangkat
	Pembayaran   *PembayaranJamaah  `json:"pembayaran"`
	Kelengkapan  *KelengkapanJamaah `json:"kelengkapan"`
	Tahapan      TahapanJamaah      `json:"tahapan"`
	Komisi       *KomisiJamaah      `json:"komisi"`
	BonusUpline  *KomisiJamaah      `json:"bonus_upline"`  // bonus pembinaan upline dari jamaah ini; nil bila tidak ada
	BisaLengkapi bool               `json:"bisa_lengkapi"` // agen pembuat data jamaah ini: boleh membuka halaman lengkapi data
	WhatsAppURL  string             `json:"whatsapp_url"`  // kosong bila jamaah tidak punya nomor
}

func nikTersamar(nik string) string {
	if len(nik) <= 4 {
		return strings.Repeat("•", len(nik))
	}
	return strings.Repeat("•", len(nik)-4) + nik[len(nik)-4:]
}

// sqlHitungData: dua ekspresi SQL (terisi, total) atas data wajib jamaah; alias j dan bp. Infant tidak menghitung data yang tidak berlaku.
func sqlHitungData() string {
	var terisi, total []string
	for _, k := range shared.KunciDataWajibJamaah {
		ada := fmt.Sprintf("(NULLIF(TRIM(CAST(j.%s AS CHAR)), '') IS NOT NULL)", k)
		berlaku := "1"
		if !shared.DataWajibBerlaku("infant", k) {
			berlaku = "(bp.pax_type <> 'infant')"
		}
		terisi = append(terisi, fmt.Sprintf("(%s AND %s)", berlaku, ada))
		total = append(total, berlaku)
	}
	return strings.Join(terisi, " + ") + ", " + strings.Join(total, " + ")
}

func detailJamaahAgen(ctx context.Context, q querier, akunID, bookingID, jamaahID int64, now time.Time) (*DetailJamaahAgen, error) {
	hariIni := now.In(zonaWIB).Format("2006-01-02")
	d := &DetailJamaahAgen{BookingID: bookingID, JamaahID: jamaahID}
	var jenis, nik, noHP, tenggat, paxType, perlStatus string
	var totalHarga float64
	var dataTerisi, dataTotal int
	var progVisa, progManasik bool
	var dibuatOleh sql.NullInt64
	err := q.QueryRowContext(ctx, `
		SELECT j.nama_lengkap, COALESCE(j.nik, ''), COALESCE(j.no_hp_normal, ''), j.dibuat_oleh_akun_id,
		       s.jadwal_nama, s.journey_type, b.id_booking, COALESCE(DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d'), ''), b.status, b.total_harga,
		       COALESCE(DATE_FORMAT(c.due_at, '%Y-%m-%d'), DATE_FORMAT(DATE_SUB(s.berangkat_tanggal, INTERVAL br.batas_pelunasan_hari DAY), '%Y-%m-%d'), ''),
		       bp.pax_type, bp.progress_visa, bp.progress_manasik, COALESCE(bp.perlengkapan_status, ''),
		       `+sqlHitungData()+`
		FROM jamaah j
		JOIN booking_pax bp ON bp.jamaah_id = j.id AND bp.pax_status = 'aktif' AND bp.booking_id = ?
		JOIN bookings b ON b.id = bp.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		JOIN brands br ON br.id = s.brand_id
		LEFT JOIN booking_checkout c ON c.booking_id = b.id
		WHERE j.id = ? AND j.direkrut_oleh_akun_id = ? AND `+bookingTercatat, bookingID, jamaahID, akunID).
		Scan(&d.Nama, &nik, &noHP, &dibuatOleh, &d.Paket, &jenis, &d.KodeBooking, &d.Berangkat, &d.Status, &totalHarga,
			&tenggat, &paxType, &progVisa, &progManasik, &perlStatus, &dataTerisi, &dataTotal)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("agen: detail jamaah: %w", err)
	}
	d.Jenis = jenis
	if jenis == "wisata_halal" {
		d.Jenis = "wisata"
	}
	d.NIKTersamar = nikTersamar(nik)
	d.Tahapan = TahapanJamaah{Visa: progVisa, Manasik: progManasik, Perlengkapan: perlStatus == "sudah_diberikan"}
	d.BisaLengkapi = dibuatOleh.Valid && dibuatOleh.Int64 == akunID
	if noHP != "" {
		d.WhatsAppURL = "https://wa.me/" + noHP
	}

	if d.Berangkat != "" && d.Berangkat <= hariIni {
		d.Status = "berangkat"
	} else {
		var dibayar float64
		if err := q.QueryRowContext(ctx, `SELECT COALESCE(SUM(jumlah), 0) FROM payments WHERE booking_id = ? AND status = 'confirmed'`, bookingID).Scan(&dibayar); err != nil {
			return nil, fmt.Errorf("agen: detail jamaah pembayaran: %w", err)
		}
		sisa := totalHarga - dibayar
		if sisa < 0 {
			sisa = 0
		}
		wajib := shared.DokumenWajibJamaah(paxType)
		args := []any{jamaahID}
		marks := make([]string, len(wajib))
		for i, w := range wajib {
			marks[i] = "?"
			args = append(args, w)
		}
		var setuju int
		if err := q.QueryRowContext(ctx, `SELECT COUNT(DISTINCT jenis) FROM dokumen_jamaah WHERE jamaah_id = ? AND status = 'approved' AND jenis IN (`+strings.Join(marks, ",")+`)`, args...).Scan(&setuju); err != nil {
			return nil, fmt.Errorf("agen: detail jamaah dokumen: %w", err)
		}
		d.Pembayaran = &PembayaranJamaah{Total: totalHarga, Dibayar: dibayar, Sisa: sisa, TenggatLunas: tenggat}
		d.Kelengkapan = &KelengkapanJamaah{
			Data:    HitunganJamaah{Terisi: dataTerisi, Total: dataTotal},
			Dokumen: HitunganJamaah{Terisi: setuju, Total: len(wajib)},
		}
	}

	var err2 error
	if d.Komisi, err2 = komisiJamaah(ctx, q, "tk.akun_penerima_id = ?", akunID, bookingID, jamaahID, d.Berangkat, hariIni); err2 != nil {
		return nil, err2
	}
	// Bonus pembinaan yang diterima upline dari jamaah ini: catatan yang sama dengan Rincian Bonus di sisi upline.
	if d.BonusUpline, err2 = komisiJamaah(ctx, q, "tk.jenis = '"+komisi.JenisPembinaan+"' AND tk.akun_penerima_id <> ?", akunID, bookingID, jamaahID, d.Berangkat, hariIni); err2 != nil {
		return nil, err2
	}
	return d, nil
}

// komisiJamaah menjumlah komisi (tanpa cashback) satu jamaah di satu booking menurut syarat penerima. nil bila tidak
// ada baris. Status: batal bila semua barisnya dibatalkan, tersedia bila tanggal berangkat + hari tahan sudah lewat.
func komisiJamaah(ctx context.Context, q querier, syaratPenerima string, akunID, bookingID, jamaahID int64, berangkat, hariIni string) (*KomisiJamaah, error) {
	var aktif float64
	var batal, baris int
	if err := q.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN tk.dibatalkan_at IS NULL THEN tk.nominal END), 0),
		       COALESCE(SUM(tk.dibatalkan_at IS NOT NULL), 0), COUNT(*)
		FROM transaksi_komisi tk
		WHERE tk.jenis <> 'cashback' AND `+syaratPenerima+` AND tk.booking_id = ? AND tk.jamaah_sumber_id = ?`, akunID, bookingID, jamaahID).
		Scan(&aktif, &batal, &baris); err != nil {
		return nil, fmt.Errorf("agen: detail jamaah komisi: %w", err)
	}
	if baris == 0 {
		return nil, nil
	}
	k := &KomisiJamaah{Nominal: aktif, Status: "tertahan"}
	k.TersediaMulai, k.Status = tersediaMulai(berangkat, hariIni)
	if aktif == 0 && batal > 0 {
		k.Status = "batal"
	}
	return k, nil
}

// tersediaMulai: tanggal komisi menjadi tersedia (berangkat + hari tahan) dan statusnya hari ini.
func tersediaMulai(berangkat, hariIni string) (mulai, status string) {
	tgl, err := time.ParseInLocation("2006-01-02", berangkat, zonaWIB)
	if err != nil {
		return "", "tertahan"
	}
	mulai = tgl.AddDate(0, 0, komisi.HariTahanSetelahBerangkat).Format("2006-01-02")
	if mulai <= hariIni {
		return mulai, "tersedia"
	}
	return mulai, "tertahan"
}

func (r *Repository) DetailJamaahAgen(ctx context.Context, akunID, bookingID, jamaahID int64, now time.Time) (*DetailJamaahAgen, error) {
	if _, err := requireAgenAktif(ctx, r.db, akunID); err != nil {
		return nil, err
	}
	return detailJamaahAgen(ctx, r.db, akunID, bookingID, jamaahID, now)
}
