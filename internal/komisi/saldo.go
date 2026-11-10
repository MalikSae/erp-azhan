package komisi

import (
	"context"
	"database/sql"
	"fmt"
)

// RowQuerier dipenuhi *sql.DB dan *sql.Tx.
type RowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Saldo seorang akun agen (agen-azhan.md 7.7 dan 7.13, keputusan D4 & D6).
type Saldo struct {
	// Tersedia: komisi aktif (tanpa cashback) yang jadwal sumbernya sudah lewat HariTahanSetelahBerangkat
	// hari dari tanggal berangkat, dikurangi pencairan yang disetujui maupun masih pending. Bisa negatif:
	// komisi yang sudah dicairkan lalu batal, atau jadwal yang diundur, menjadi utang yang tertutup komisi
	// berikutnya (K27). Selama negatif, pencairan baru tertolak dengan sendirinya.
	Tersedia float64 `json:"saldo_tersedia"`
	// Tertahan: komisi aktif (tanpa cashback) yang belum melewati batas hari itu.
	Tertahan float64 `json:"saldo_tertahan"`
	// KreditCashback: cashback yang belum dipakai. Tidak bisa dicairkan,
	// hanya untuk potongan booking berikutnya.
	KreditCashback float64 `json:"kredit_cashback"`
}

// SQLTersediaMulai: tanggal sebuah komisi menjadi tersedia, untuk jadwal beralias s.
var SQLTersediaMulai = fmt.Sprintf("DATE_ADD(s.berangkat_tanggal, INTERVAL %d DAY)", HariTahanSetelahBerangkat)

var qSaldo = `
SELECT
  COALESCE((SELECT SUM(tk.nominal) FROM transaksi_komisi tk
            JOIN bookings b ON b.id = tk.booking_id
            JOIN schedules s ON s.id = b.schedule_id
            WHERE tk.akun_penerima_id = ? AND tk.jenis <> 'cashback' AND tk.dibatalkan_at IS NULL
              AND ` + SQLTersediaMulai + ` <= CURRENT_DATE), 0),
  COALESCE((SELECT SUM(tk.nominal) FROM transaksi_komisi tk
            JOIN bookings b ON b.id = tk.booking_id
            JOIN schedules s ON s.id = b.schedule_id
            WHERE tk.akun_penerima_id = ? AND tk.jenis <> 'cashback' AND tk.dibatalkan_at IS NULL
              AND ` + SQLTersediaMulai + ` > CURRENT_DATE), 0),
  COALESCE((SELECT SUM(nominal_diajukan) FROM pengajuan_pencairan
            WHERE akun_agen_id = ? AND status IN ('disetujui','pending')), 0),
  COALESCE((SELECT SUM(nominal) FROM transaksi_komisi
            WHERE akun_penerima_id = ? AND jenis = 'cashback' AND dibatalkan_at IS NULL), 0),
  COALESCE((SELECT SUM(nominal) FROM pemakaian_cashback WHERE akun_id = ?), 0)`

// HitungSaldo menghitung saldo seorang akun agen. Untuk validasi pencairan,
// panggil dengan *sql.Tx setelah mengunci baris agen (7.6).
func HitungSaldo(ctx context.Context, q RowQuerier, akunID int64) (Saldo, error) {
	var matang, tertahan, pencairan, cashback, cashbackDipakai float64
	err := q.QueryRowContext(ctx, qSaldo, akunID, akunID, akunID, akunID, akunID).
		Scan(&matang, &tertahan, &pencairan, &cashback, &cashbackDipakai)
	if err != nil {
		return Saldo{}, fmt.Errorf("komisi: hitung saldo: %w", err)
	}
	return Saldo{
		Tersedia:       matang - pencairan,
		Tertahan:       tertahan,
		KreditCashback: cashback - cashbackDipakai,
	}, nil
}
