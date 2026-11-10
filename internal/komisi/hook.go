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
// terpanggil lebih dari sekali untuk booking yang sama. Saat mundur, OnBookingTurunDariLunas
// (batal.go) menandai batal komisi booking itu, sehingga lunas berikutnya mencatat baris baru
// dengan penerima saat itu (K25). Pemanggilan ulang tanpa mundur lebih dulu tidak menambah
// baris: UNIQUE(booking_pax_id, jenis, aktif) hanya mengizinkan satu baris aktif per pax dan jenis.
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
