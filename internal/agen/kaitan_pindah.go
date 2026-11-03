package agen

import (
	"context"
	"database/sql"
	"fmt"

	"erp-azhan/api/internal/komisi"
)

// pindahkanKomisiTx memindahkan komisi yang sudah tercatat atas satu jamaah ke agen baru: komisi langsung ke
// agen baru, bonus pembinaan ke upline agen baru. Bonus pembinaan tanpa penerima (agen baru tanpa upline)
// dihapus; keduanya dihitung di log kaitan. Saldo agen lama tidak boleh negatif sesudahnya (sudah dicairkan).
func pindahkanKomisiTx(ctx context.Context, tx *sql.Tx, jamaahID, agenBaru int64) (dipindah, dihapus int, err error) {
	var upline sql.NullInt64
	if err = tx.QueryRowContext(ctx, `SELECT upline_akun_id FROM customer_accounts WHERE id = ?`, agenBaru).Scan(&upline); err != nil {
		return 0, 0, fmt.Errorf("agen: baca upline agen baru: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id, akun_penerima_id, jenis FROM transaksi_komisi
		WHERE jamaah_sumber_id = ? AND jenis IN ('langsung','pembinaan') FOR UPDATE`, jamaahID)
	if err != nil {
		return 0, 0, fmt.Errorf("agen: baca komisi jamaah: %w", err)
	}
	type baris struct {
		id, penerima int64
		jenis        string
	}
	var semua []baris
	for rows.Next() {
		var b baris
		if err = rows.Scan(&b.id, &b.penerima, &b.jenis); err != nil {
			rows.Close()
			return 0, 0, err
		}
		semua = append(semua, b)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return 0, 0, err
	}
	lama := map[int64]bool{}
	for _, b := range semua {
		target := agenBaru
		if b.jenis == "pembinaan" {
			if !upline.Valid {
				if _, err = tx.ExecContext(ctx, `DELETE FROM transaksi_komisi WHERE id = ?`, b.id); err != nil {
					return 0, 0, fmt.Errorf("agen: hapus komisi pembinaan: %w", err)
				}
				lama[b.penerima] = true
				dihapus++
				continue
			}
			target = upline.Int64
		}
		if target == b.penerima {
			continue
		}
		if _, err = tx.ExecContext(ctx, `UPDATE transaksi_komisi SET akun_penerima_id = ? WHERE id = ?`, target, b.id); err != nil {
			return 0, 0, fmt.Errorf("agen: pindahkan komisi: %w", err)
		}
		lama[b.penerima] = true
		dipindah++
	}
	for id := range lama {
		if _, err = tx.ExecContext(ctx, `SELECT id FROM customer_accounts WHERE id = ? FOR UPDATE`, id); err != nil {
			return 0, 0, err
		}
		s, err := komisi.HitungSaldo(ctx, tx, id)
		if err != nil {
			return 0, 0, err
		}
		if s.Tersedia < 0 {
			return 0, 0, ErrKomisiSudahDicairkan
		}
	}
	return dipindah, dihapus, nil
}
