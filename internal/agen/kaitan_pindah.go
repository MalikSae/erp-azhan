package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"erp-azhan/api/internal/komisi"
)

// pindahkanKomisiTx memindahkan komisi aktif seorang jamaah saat kaitannya diganti: komisi langsung ke agen
// baru, bonus pembinaan ke upline agen baru. Aturannya sama dengan saat lunas (spek 5.1): bonus pembinaan
// hanya untuk upline berstatus aktif. Bonus pembinaan tanpa penerima (agen baru tanpa upline, atau upline-nya
// nonaktif) ditandai batal dengan sebab ganti kaitan; barisnya tidak dihapus (K25). Jumlah yang dipindah dan
// yang dibatalkan dicatat di log kaitan. Baris yang sudah batal tetap pada penerima lamanya sebagai riwayat.
//
// Komisi yang sudah dicairkan agen lama tidak boleh dipindah, karena agen baru bisa mencairkannya lagi.
// Karena itu pemindahan ditolak bila ia sendiri menurunkan saldo tersedia penerima lama sampai di bawah nol.
// Saldo yang sudah negatif sebelumnya dan tidak berubah oleh pemindahan ini tidak menghalangi (K27).
func pindahkanKomisiTx(ctx context.Context, tx *sql.Tx, jamaahID, agenBaru int64) (dipindah, dibatalkan int, err error) {
	var upline sql.NullInt64
	err = tx.QueryRowContext(ctx, `
		SELECT u.id FROM customer_accounts a
		JOIN customer_accounts u ON u.id = a.upline_akun_id AND u.status_agen = 'aktif'
		WHERE a.id = ?`, agenBaru).Scan(&upline)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, 0, fmt.Errorf("agen: baca upline agen baru: %w", err)
	}
	rows, err := tx.QueryContext(ctx, `
		SELECT id, akun_penerima_id, jenis FROM transaksi_komisi
		WHERE jamaah_sumber_id = ? AND jenis IN ('langsung','pembinaan') AND dibatalkan_at IS NULL FOR UPDATE`, jamaahID)
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
			return 0, 0, fmt.Errorf("agen: baca komisi jamaah: %w", err)
		}
		semua = append(semua, b)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return 0, 0, fmt.Errorf("agen: baca komisi jamaah: %w", err)
	}

	// Saldo tersedia tiap penerima lama sebelum pemindahan, dengan baris akunnya terkunci.
	sebelum := map[int64]float64{}
	for _, b := range semua {
		if _, sudah := sebelum[b.penerima]; sudah {
			continue
		}
		var terkunci int64
		if err = tx.QueryRowContext(ctx, `SELECT id FROM customer_accounts WHERE id = ? FOR UPDATE`, b.penerima).Scan(&terkunci); err != nil {
			return 0, 0, fmt.Errorf("agen: kunci penerima lama: %w", err)
		}
		s, errSaldo := komisi.HitungSaldo(ctx, tx, b.penerima)
		if errSaldo != nil {
			return 0, 0, errSaldo
		}
		sebelum[b.penerima] = s.Tersedia
	}

	berubah := map[int64]bool{}
	for _, b := range semua {
		target := agenBaru
		if b.jenis == komisi.JenisPembinaan {
			if !upline.Valid {
				if _, err = tx.ExecContext(ctx, `UPDATE transaksi_komisi SET dibatalkan_at = NOW(), sebab_batal = ? WHERE id = ?`, komisi.SebabGantiKaitan, b.id); err != nil {
					return 0, 0, fmt.Errorf("agen: batalkan bonus pembinaan: %w", err)
				}
				berubah[b.penerima] = true
				dibatalkan++
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
		berubah[b.penerima] = true
		dipindah++
	}
	for id := range berubah {
		s, errSaldo := komisi.HitungSaldo(ctx, tx, id)
		if errSaldo != nil {
			return 0, 0, errSaldo
		}
		if s.Tersedia < 0 && s.Tersedia < sebelum[id] {
			return 0, 0, ErrKomisiSudahDicairkan
		}
	}
	return dipindah, dibatalkan, nil
}
