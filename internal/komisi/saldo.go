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

// Saldo seorang jamaah/agen (agen-azhan.md 7.7 dan 7.13, keputusan D4 & D6).
type Saldo struct {
	// Tersedia: komisi (tanpa cashback) yang booking sumbernya sudah berangkat,
	// dikurangi pencairan yang disetujui maupun masih pending.
	Tersedia float64 `json:"saldo_tersedia"`
	// Tertahan: komisi (tanpa cashback) yang booking sumbernya belum berangkat.
	Tertahan float64 `json:"saldo_tertahan"`
	// KreditCashback: cashback yang belum dipakai. Tidak bisa dicairkan,
	// hanya untuk potongan booking berikutnya.
	KreditCashback float64 `json:"kredit_cashback"`
}

const qSaldo = `
SELECT
  COALESCE((SELECT SUM(tk.nominal) FROM transaksi_komisi tk
            JOIN bookings b ON b.id = tk.booking_id
            JOIN schedules s ON s.id = b.schedule_id
            WHERE tk.jamaah_penerima_id = ? AND tk.jenis <> 'cashback'
              AND s.berangkat_tanggal <= CURRENT_DATE), 0),
  COALESCE((SELECT SUM(tk.nominal) FROM transaksi_komisi tk
            JOIN bookings b ON b.id = tk.booking_id
            JOIN schedules s ON s.id = b.schedule_id
            WHERE tk.jamaah_penerima_id = ? AND tk.jenis <> 'cashback'
              AND s.berangkat_tanggal > CURRENT_DATE), 0),
  COALESCE((SELECT SUM(nominal_diajukan) FROM pengajuan_pencairan
            WHERE jamaah_agen_id = ? AND status IN ('disetujui','pending')), 0),
  COALESCE((SELECT SUM(nominal) FROM transaksi_komisi
            WHERE jamaah_penerima_id = ? AND jenis = 'cashback'), 0),
  COALESCE((SELECT SUM(nominal) FROM pemakaian_cashback WHERE jamaah_id = ?), 0)`

// HitungSaldo menghitung saldo seorang jamaah/agen. Untuk validasi pencairan,
// panggil dengan *sql.Tx setelah mengunci baris agen (7.6).
func HitungSaldo(ctx context.Context, q RowQuerier, jamaahID int64) (Saldo, error) {
	var matang, tertahan, pencairan, cashback, cashbackDipakai float64
	err := q.QueryRowContext(ctx, qSaldo, jamaahID, jamaahID, jamaahID, jamaahID, jamaahID).
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
