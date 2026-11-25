package agen

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"
)

// Prospek agen (docs/spek/SPEK-PROSPEK-AGEN-2026-11-25.md): akun yang daftar lewat tautan agen
// (customer_accounts.direkrut_oleh_akun_id = agen) dan belum sampai DP. Hanya kontak akun pemesan yang dibuka;
// NIK, nama tiap jamaah, dan dokumen tidak. Booking buatan agen sendiri tidak ikut (sudah di Booking Saya).

// Tahap prospek.
const (
	TahapBelumDP      = "belum_dp"      // booking baru, hold berjalan
	TahapHoldHabis    = "hold_habis"    // booking baru, hold sudah dilepas (masih bisa DP bila seat ada)
	TahapBelumBooking = "belum_booking" // akun tanpa booking
)

// HariProspek: prospek tanpa kabar lebih lama dari ini keluar dari daftar.
const HariProspek = 30

type Prospek struct {
	Tahap string `json:"tahap"`
	// Keterangan hanya untuk hold_habis yang tidak bisa lanjut DP: "jadwal_penuh" atau "pendaftaran_tutup".
	Keterangan *string `json:"keterangan"`
	AkunID     int64   `json:"akun_id"`
	Nama       string  `json:"nama"`
	// WhatsApp: angka berawalan 62, siap dipakai di wa.me; "" bila akun tanpa nomor.
	WhatsApp     string  `json:"whatsapp"`
	BookingID    *int64  `json:"booking_id"`
	KodeBooking  *string `json:"kode_booking"`
	Paket        *string `json:"paket"`
	Berangkat    *string `json:"berangkat"`   // YYYY-MM-DD
	Dibuat       string  `json:"dibuat_at"`   // waktu booking dibuat, atau akun daftar (YYYY-MM-DD HH:MM WIB)
	HoldSampai   *string `json:"hold_sampai"` // belum_dp saja (YYYY-MM-DD HH:MM WIB)
	JumlahJamaah int     `json:"jumlah_jamaah"`
	dibuat       time.Time
	hold         time.Time
}

func formatWIB(t time.Time) string { return t.In(zonaWIB).Format("2006-01-02 15:04") }

func prospekAgen(ctx context.Context, q querier, agenID int64, now time.Time) ([]Prospek, error) {
	if _, err := brandAkunAgen(ctx, q, agenID); err != nil {
		return nil, err
	}
	batas := now.AddDate(0, 0, -HariProspek)
	out := []Prospek{}

	rows, err := q.QueryContext(ctx, `
		SELECT b.id, COALESCE(b.id_booking, ''), b.created_at, b.is_seat_blocked, b.seat_hold_expires_at,
		       a.id, a.nama_lengkap, COALESCE(a.no_hp, ''),
		       s.jadwal_nama, DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d'),
		       (s.status <> 'published' OR s.berangkat_tanggal < DATE_ADD(CURRENT_DATE, INTERVAL 30 DAY)),
		       s.seat_sisa < b.seat_count,
		       (SELECT COUNT(*) FROM booking_pax bp WHERE bp.booking_id = b.id AND bp.pax_status = 'aktif')
		FROM bookings b
		JOIN customer_accounts a ON a.id = b.customer_account_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE a.direkrut_oleh_akun_id = ? AND a.dihapus_at IS NULL AND b.status = 'baru' AND b.created_at >= ?
		  AND (b.dibuat_oleh_agen_akun_id IS NULL OR b.dibuat_oleh_agen_akun_id <> ?)`, agenID, batas, agenID)
	if err != nil {
		return nil, fmt.Errorf("agen: prospek booking: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p Prospek
		var bookingID int64
		var kode, paket, berangkat string
		var holdAktif, tutup, penuh bool
		var hold sql.NullTime
		var hp string
		if err := rows.Scan(&bookingID, &kode, &p.dibuat, &holdAktif, &hold, &p.AkunID, &p.Nama, &hp,
			&paket, &berangkat, &tutup, &penuh, &p.JumlahJamaah); err != nil {
			return nil, fmt.Errorf("agen: scan prospek booking: %w", err)
		}
		p.BookingID, p.KodeBooking, p.Paket, p.Berangkat = &bookingID, &kode, &paket, &berangkat
		p.WhatsApp, _ = shared.PhoneVariants(hp)
		p.Dibuat = formatWIB(p.dibuat)
		if holdAktif && hold.Valid && hold.Time.After(now) {
			p.Tahap, p.hold = TahapBelumDP, hold.Time
			h := formatWIB(hold.Time)
			p.HoldSampai = &h
		} else {
			p.Tahap = TahapHoldHabis
			switch {
			case tutup:
				k := "pendaftaran_tutup"
				p.Keterangan = &k
			case penuh:
				k := "jadwal_penuh"
				p.Keterangan = &k
			}
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Akun tanpa booking sama sekali; akun yang sudah jadi agen (binaan) tidak dihitung prospek.
	rows2, err := q.QueryContext(ctx, `
		SELECT a.id, a.nama_lengkap, COALESCE(a.no_hp, ''), a.created_at
		FROM customer_accounts a
		WHERE a.direkrut_oleh_akun_id = ? AND a.dihapus_at IS NULL AND a.created_at >= ?
		  AND a.status_agen = 'tidak_aktif'
		  AND NOT EXISTS (SELECT 1 FROM bookings b WHERE b.customer_account_id = a.id)`, agenID, batas)
	if err != nil {
		return nil, fmt.Errorf("agen: prospek akun: %w", err)
	}
	defer rows2.Close()
	for rows2.Next() {
		p := Prospek{Tahap: TahapBelumBooking}
		var hp string
		if err := rows2.Scan(&p.AkunID, &p.Nama, &hp, &p.dibuat); err != nil {
			return nil, fmt.Errorf("agen: scan prospek akun: %w", err)
		}
		p.WhatsApp, _ = shared.PhoneVariants(hp)
		p.Dibuat = formatWIB(p.dibuat)
		out = append(out, p)
	}
	if err := rows2.Err(); err != nil {
		return nil, err
	}

	// Urutan: hold paling cepat habis, lalu hold habis terbaru, lalu belum booking terbaru.
	urutan := map[string]int{TahapBelumDP: 0, TahapHoldHabis: 1, TahapBelumBooking: 2}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if urutan[a.Tahap] != urutan[b.Tahap] {
			return urutan[a.Tahap] < urutan[b.Tahap]
		}
		if a.Tahap == TahapBelumDP {
			return a.hold.Before(b.hold)
		}
		return a.dibuat.After(b.dibuat)
	})
	return out, nil
}

func (r *Repository) ProspekAgen(ctx context.Context, agenID int64, now time.Time) ([]Prospek, error) {
	return prospekAgen(ctx, r.db, agenID, now)
}

// ProspekPortal GET /api/portal/agen/prospek: akun dari tautan agen yang belum sampai DP.
func (h *Handler) ProspekPortal(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ProspekAgen(r.Context(), identity.GetCustomerAccountID(r.Context()), time.Now())
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
}
