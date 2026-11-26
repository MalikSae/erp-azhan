package manasik

import (
	"context"
	"database/sql"
	"errors"
)

// Fungsi di file ini dipakai modul lain (schedule) supaya aturan manasik tidak ditulis ulang di luar modul manasik.

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// BentrokBerangkat: tanggal berangkat paket harus setelah tanggal manasiknya. Mengembalikan tanggal manasik
// (YYYY-MM-DD) bila tanggal berangkat baru sama dengan atau sebelum tanggal manasik; "" bila sah atau paket belum
// punya manasik. Pemanggil mengunci baris paket lebih dulu (FOR UPDATE); Ubah manasik mengunci bersama baris paket
// yang sama, sehingga keduanya tidak lolos bersamaan.
func BentrokBerangkat(ctx context.Context, q rowQuerier, scheduleID int64, berangkat string) (string, error) {
	var tanggal string
	err := q.QueryRowContext(ctx, `SELECT DATE_FORMAT(m.tanggal,'%Y-%m-%d') FROM manasik_jadwal mj
		JOIN manasik m ON m.id = mj.manasik_id WHERE mj.schedule_id = ?`, scheduleID).Scan(&tanggal)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if berangkat <= tanggal {
		return tanggal, nil
	}
	return "", nil
}
