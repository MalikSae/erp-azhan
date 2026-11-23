// Package tahapan menyusun tahapan umroh yang dilihat jamaah dan agen: status pekerjaan travel, bukan tugas jamaah.
// Satu tempat untuk urutan, sumber status, dan tanggal selesai, supaya portal pelanggan dan portal agen membaca hal
// yang sama. Tahap tidak saling mengunci; urutan hanya untuk tampilan.
package tahapan

import (
	"context"
	"database/sql"
	"fmt"
)

// Kunci tahap, urut tampil.
const (
	Perlengkapan = "perlengkapan"
	Paspor       = "paspor"
	Siskopatuh   = "siskopatuh"
	Visa         = "visa"
	Tiket        = "tiket"
)

// Status tahap.
const (
	Diproses = "diproses"
	Selesai  = "selesai"
	// Perbaiki: hanya paspor, saat dokumen paspor ditolak admin (jamaah harus mengunggah ulang).
	Perbaiki = "perbaiki"
)

// Urutan tampil (keputusan 2026-11-23: perlengkapan biasanya diserahkan saat DP, tiket menjelang berangkat). Infant tidak
// mendapat jatah perlengkapan (booking.MarkPerlengkapanDiberikan menolaknya), jadi tahap Perlengkapan tidak ada untuk
// infant: tahap yang tidak berlaku disembunyikan, bukan dibiarkan diproses selamanya.
var Urutan = []string{Perlengkapan, Paspor, Siskopatuh, Visa, Tiket}

// Tahap: satu tahap satu jamaah. SelesaiAt YYYY-MM-DD (WIB), nil bila belum selesai atau tanggalnya tidak tercatat.
type Tahap struct {
	Kunci     string  `json:"kunci"`
	Status    string  `json:"status"`
	SelesaiAt *string `json:"selesai_at"`
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// selectPax: kolom sumber tahapan per pax aktif, alias bp (booking_pax), b (bookings), dk (dokumen paspor).
const selectPax = `
	SELECT bp.id, bp.jamaah_id, bp.pax_type = 'infant',
	       bp.perlengkapan_status = 'sudah_diberikan',
	       DATE_FORMAT(COALESCE(bp.perlengkapan_diserahkan_at, bp.perlengkapan_tanggal), '%Y-%m-%d'),
	       COALESCE(dk.status, ''), DATE_FORMAT(dk.updated_at, '%Y-%m-%d'),
	       bp.progress_siskopatuh, DATE_FORMAT(bp.progress_siskopatuh_at, '%Y-%m-%d'),
	       bp.progress_visa, DATE_FORMAT(bp.progress_visa_at, '%Y-%m-%d'),
	       b.progress_tiket, DATE_FORMAT(b.progress_tiket_at, '%Y-%m-%d')
	FROM booking_pax bp
	JOIN bookings b ON b.id = bp.booking_id
	LEFT JOIN dokumen_jamaah dk ON dk.jamaah_id = bp.jamaah_id AND dk.jenis = 'paspor'
	WHERE bp.pax_status = 'aktif' AND bp.booking_id = ?`

// Pax: tahapan satu pax.
type Pax struct {
	PaxID    int64
	JamaahID int64
	Tahapan  []Tahap
}

// MuatBooking: tahapan semua pax aktif satu booking, urut id pax. Akses booking dicek pemanggil.
func MuatBooking(ctx context.Context, q querier, bookingID int64) ([]Pax, error) {
	rows, err := q.QueryContext(ctx, selectPax+` ORDER BY bp.id`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("tahapan: %w", err)
	}
	defer rows.Close()
	out := []Pax{}
	for rows.Next() {
		var p Pax
		var infant, perl, sisko, visa, tiket bool
		var perlAt, paspor, pasporAt, siskoAt, visaAt, tiketAt sql.NullString
		if err := rows.Scan(&p.PaxID, &p.JamaahID, &infant, &perl, &perlAt, &paspor, &pasporAt, &sisko, &siskoAt, &visa, &visaAt, &tiket, &tiketAt); err != nil {
			return nil, fmt.Errorf("tahapan: %w", err)
		}
		p.Tahapan = []Tahap{}
		if !infant {
			p.Tahapan = append(p.Tahapan, centang(Perlengkapan, perl, perlAt))
		}
		p.Tahapan = append(p.Tahapan,
			tahapPaspor(paspor.String, pasporAt),
			centang(Siskopatuh, sisko, siskoAt),
			centang(Visa, visa, visaAt),
			centang(Tiket, tiket, tiketAt),
		)
		out = append(out, p)
	}
	return out, rows.Err()
}

// UntukJamaah: tahapan satu jamaah di satu booking; nil bila jamaah itu bukan pax aktif booking tersebut.
func UntukJamaah(ctx context.Context, q querier, bookingID, jamaahID int64) ([]Tahap, error) {
	semua, err := MuatBooking(ctx, q, bookingID)
	if err != nil {
		return nil, err
	}
	for _, p := range semua {
		if p.JamaahID == jamaahID {
			return p.Tahapan, nil
		}
	}
	return nil, nil
}

func tanggal(s sql.NullString) *string {
	if !s.Valid || s.String == "" {
		return nil
	}
	v := s.String
	return &v
}

// centang: tahap yang selesai dari penanda admin; tanggal hanya ikut bila tahapnya selesai.
func centang(kunci string, selesai bool, at sql.NullString) Tahap {
	if !selesai {
		return Tahap{Kunci: kunci, Status: Diproses}
	}
	return Tahap{Kunci: kunci, Status: Selesai, SelesaiAt: tanggal(at)}
}

// tahapPaspor: selesai saat dokumen paspor disetujui admin, perlu diperbaiki saat ditolak, selain itu diproses.
func tahapPaspor(status string, at sql.NullString) Tahap {
	switch status {
	case "approved":
		return Tahap{Kunci: Paspor, Status: Selesai, SelesaiAt: tanggal(at)}
	case "rejected":
		return Tahap{Kunci: Paspor, Status: Perbaiki}
	}
	return Tahap{Kunci: Paspor, Status: Diproses}
}
