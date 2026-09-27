// Package komisi menampung logika komisi Agen Umroh ("Syiar").
// Lihat agen-azhan.md Bagian 4 dan 7.4.
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
// terpanggil lebih dari sekali untuk booking yang sama. Implementasinya WAJIB
// idempoten (Sprint 1: UNIQUE(booking_pax_id, jenis, jamaah_penerima_id) di
// transaksi_komisi).
//
// Berupa variabel agar test bisa memverifikasi pemanggilannya. Jangan diganti
// di kode produksi.
var OnBookingLunas = func(ctx context.Context, tx *sql.Tx, bookingID int64) error {
	return nil
}
