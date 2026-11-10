package komisi

import (
	"context"
	"database/sql"
	"fmt"
)

// Komisi mengikuti lunas terakhir (spek keagenan K25, merevisi K20). Booking yang turun dari 'lunas' ke 'dp'
// atau 'baru' (pembayaran dibatalkan, diskon dihapus, tipe kamar diganti ke yang lebih mahal) belum lunas,
// jadi komisinya batal. Saat booking lunas lagi, ProcessBookingLunas mencatat baris baru dengan penerima
// saat itu. Pembatalan booking atau pax setelah lunas tidak membatalkan komisi. Void pembayaran pada booking
// yang sudah batal membatalkannya bila sisa pembayaran tidak lagi menutup total harga (payment.voidTx):
// void menyatakan pembayaran itu tidak pernah ada.
//
// Baris yang batal tidak dihapus: dibatalkan_at dan sebab_batal diisi, dan baris itu tidak lagi dihitung di
// saldo, total, dan peringkat. Kunci unik uq_komisi_pax_jenis_aktif menjamin hanya ada satu baris aktif per
// pax dan jenis komisi.

// HariTahanSetelahBerangkat: komisi pindah dari tertahan ke tersedia sekian hari kalender setelah tanggal
// berangkat jadwal (K26). Berangkat 20 Desember berarti tersedia mulai 23 Desember.
const HariTahanSetelahBerangkat = 3

// Sebab pembatalan komisi yang dicatat di transaksi_komisi.sebab_batal.
const (
	SebabStatusTurun   = "status_turun"
	SebabVoid          = "void_pembayaran"
	SebabDiskonDihapus = "diskon_dihapus"
	SebabGantiKamar    = "ganti_kamar"
	SebabAturanBaru    = "penyesuaian_aturan" // migrasi 126
)

type kunciSebabBatal struct{}

// DenganSebabBatal menitipkan sebab pembatalan ke context, supaya pemanggil yang tahu penyebab turunnya
// status (void, hapus diskon, ganti kamar) tidak perlu mengubah tanda tangan sinkronisasi status booking.
func DenganSebabBatal(ctx context.Context, sebab string) context.Context {
	return context.WithValue(ctx, kunciSebabBatal{}, sebab)
}

func sebabBatal(ctx context.Context) string {
	if s, ok := ctx.Value(kunciSebabBatal{}).(string); ok && s != "" {
		return s
	}
	return SebabStatusTurun
}

// BatalkanKomisiBookingTx menandai batal semua baris komisi aktif milik satu booking dan mengembalikan
// jumlah baris yang berubah. Dipanggil di transaksi yang sama dengan perubahan status booking.
func BatalkanKomisiBookingTx(ctx context.Context, tx *sql.Tx, bookingID int64) (int64, error) {
	res, err := tx.ExecContext(ctx, `
		UPDATE transaksi_komisi SET dibatalkan_at = NOW(), sebab_batal = ?
		WHERE booking_id = ? AND dibatalkan_at IS NULL`, sebabBatal(ctx), bookingID)
	if err != nil {
		return 0, fmt.Errorf("komisi: batalkan komisi booking %d: %w", bookingID, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("komisi: jumlah komisi batal booking %d: %w", bookingID, err)
	}
	return n, nil
}

// OnBookingTurunDariLunas dipanggil setiap kali status booking berubah dari 'lunas' ke 'dp' atau 'baru'.
// Berupa variabel agar test bisa memverifikasi pemanggilannya. Jangan diganti di kode produksi.
var OnBookingTurunDariLunas = BatalkanKomisiBookingTx
