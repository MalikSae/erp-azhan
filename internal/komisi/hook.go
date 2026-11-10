// Package komisi menampung logika komisi Agen Umroh ("Syiar").
// Lihat agen-azhan.md Bagian 4, 5, dan 7.4–7.7.
package komisi

import (
	"context"
	"database/sql"
	"fmt"
)

// OnBookingLunas adalah satu-satunya titik yang dipanggil setiap kali status
// booking berubah menjadi 'lunas', di dalam transaksi yang sama dengan
// perubahan status. Pemanggil saat ini:
//   - payment.SyncBookingStatusTx (pembayaran dikonfirmasi, ditolak, atau dibatalkan lewat void)
//   - booking.recalculateTotalTx (add-on/diskon berubah)
//
// Booking bisa mundur dari 'lunas' ke 'dp' lalu naik lagi, jadi fungsi ini bisa
// terpanggil lebih dari sekali untuk booking yang sama. Saat mundur, OnBookingTurunDariLunas
// (batal.go) menandai batal komisi booking itu, sehingga lunas berikutnya mencatat baris baru
// dengan penerima saat itu (K25). Pemanggilan ulang tanpa mundur lebih dulu tidak menambah
// baris: UNIQUE(booking_pax_id, jenis, aktif) hanya mengizinkan satu baris aktif per pax dan jenis.
//
// Berupa variabel agar test bisa memverifikasi pemanggilannya. Jangan diganti
// di kode produksi.
var OnBookingLunas = ProcessBookingLunas

// Nominal efektif komisi satu paket (tabel schedules): nilai paket bila diisi (termasuk 0 = sengaja tanpa
// komisi), selain itu komisi bawaan brand. Bawaan brand kosong (tersimpan 0) berarti brand belum ikut
// program, sehingga hasilnya NULL.
// Dipakai dengan alias tabel s (schedules) dan br (brands).
const (
	SQLKomisiLangsungEfektif = `COALESCE(s.nominal_komisi_langsung, NULLIF(br.ujroh_closing, 0))`
	SQLBonusPembinaanEfektif = `COALESCE(s.nominal_bonus_pembinaan, NULLIF(br.ujroh_pembinaan, 0))`
)

// SnapshotNominal menyalin nominal efektif komisi jadwal ke booking (keputusan D9).
// Dipanggil saat booking resmi dibuat (bukan draft) di semua jalur: booking
// admin, finalisasi draft, self-booking, dan CRM deal. Perubahan nominal jadwal atau
// bawaan brand setelahnya tidak mengubah booking yang sudah ada.
func SnapshotNominal(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE bookings b
		JOIN schedules s ON s.id = b.schedule_id
		JOIN brands br ON br.id = s.brand_id
		SET b.snapshot_komisi_langsung = `+SQLKomisiLangsungEfektif+`,
		    b.snapshot_bonus_pembinaan = `+SQLBonusPembinaanEfektif+`
		WHERE b.id = ?`, bookingID)
	if err != nil {
		return fmt.Errorf("komisi: snapshot nominal booking %d: %w", bookingID, err)
	}
	return nil
}
