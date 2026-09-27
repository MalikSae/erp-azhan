// Package komisi menampung logika komisi Agen Umroh ("Syiar").
// Lihat agen-azhan.md Bagian 4, 5, dan 7.4–7.7.
package komisi

import (
	"context"
	"database/sql"
)

// OnBookingLunas adalah satu-satunya titik yang dipanggil setiap kali status
// booking berubah menjadi 'lunas', di dalam transaksi yang sama dengan
// perubahan status. Pemanggil saat ini:
//   - payment.syncBookingStatusTx (pembayaran dikonfirmasi/ditolak/dihapus)
//   - booking.recalculateTotalTx (add-on/diskon berubah)
//
// Booking bisa mundur dari 'lunas' ke 'dp' lalu naik lagi, jadi fungsi ini bisa
// terpanggil lebih dari sekali untuk booking yang sama. ProcessBookingLunas
// idempoten: UNIQUE(booking_pax_id, jenis, jamaah_penerima_id) di
// transaksi_komisi dan pertama_lunas_at yang hanya diisi sekali.
//
// Berupa variabel agar test bisa memverifikasi pemanggilannya. Jangan diganti
// di kode produksi.
var OnBookingLunas = ProcessBookingLunas

// SnapshotNominal menyalin nominal komisi jadwal ke booking (keputusan D9).
// Dipanggil saat booking resmi dibuat (bukan draft) di semua jalur: booking
// admin, finalisasi draft, self-booking, dan CRM deal.
func SnapshotNominal(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	_, err := tx.ExecContext(ctx, `
		UPDATE bookings b JOIN schedules s ON s.id = b.schedule_id
		SET b.snapshot_komisi_langsung = s.nominal_komisi_langsung,
		    b.snapshot_bonus_pembinaan = s.nominal_bonus_pembinaan
		WHERE b.id = ?`, bookingID)
	return err
}
