package notifikasi

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"
)

// KirimPengumuman membuat notifikasi untuk pengumuman yang sudah mulai tayang dan belum dikirim.
// Sasaran sama dengan pengumuman.UntukAkun: semua akun brand, atau akun yang punya booking aktif
// di jadwal sasaran. Kunci "pengumuman:<id>" mencegah dobel bila loop berjalan ulang.
func KirimPengumuman(ctx context.Context, db Querier, now time.Time) error {
	hari := now.In(ZonaWIB).Format("2006-01-02")
	rows, err := db.QueryContext(ctx, `SELECT id, brand_id, schedule_id, judul, isi FROM pengumuman
		WHERE notif_dikirim_at IS NULL AND mulai_tayang <= ? AND akhir_tayang >= ?`, hari, hari)
	if err != nil {
		return err
	}
	type item struct {
		id, brand int64
		jadwal    sql.NullInt64
		judul     string
		isi       string
	}
	var daftar []item
	for rows.Next() {
		var it item
		if err := rows.Scan(&it.id, &it.brand, &it.jadwal, &it.judul, &it.isi); err != nil {
			rows.Close()
			return err
		}
		daftar = append(daftar, it)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, p := range daftar {
		sasaran := `SELECT a.id AS id FROM customer_accounts a WHERE a.brand_id = ? AND a.dihapus_at IS NULL`
		args := []any{p.brand}
		if p.jadwal.Valid {
			sasaran = `SELECT DISTINCT b.customer_account_id AS id FROM bookings b
				JOIN customer_accounts a ON a.id = b.customer_account_id AND a.dihapus_at IS NULL
				WHERE b.schedule_id = ? AND b.status NOT IN ('draft','batal')`
			args = []any{p.jadwal.Int64}
		}
		_, err := db.ExecContext(ctx, `INSERT IGNORE INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, ref_id, pengumuman_id, kunci)
			SELECT x.id, ?, ?, ?, ?, ?, ?, ? FROM (`+sasaran+`) x`,
			append([]any{JenisPengumuman, potong(p.judul, 150), ringkas(p.isi, 300), RefPengumuman, p.id, p.id, "pengumuman:" + strconv.FormatInt(p.id, 10)}, args...)...)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `UPDATE pengumuman SET notif_dikirim_at = NOW(), diubah_at = diubah_at WHERE id = ?`, p.id); err != nil {
			return err
		}
	}
	return nil
}

// Pengumuman: isi lengkap pengumuman untuk layar detail app, hanya bila akun menerima notifikasinya.
type Pengumuman struct {
	ID    int64  `json:"id"`
	Judul string `json:"judul"`
	Isi   string `json:"isi"`
	Mulai string `json:"mulai"`
}

func AmbilPengumuman(ctx context.Context, q Querier, akunID, id int64) (*Pengumuman, error) {
	var p Pengumuman
	var mulai time.Time
	err := q.QueryRowContext(ctx, `SELECT p.id, p.judul, p.isi, p.mulai_tayang FROM pengumuman p
		WHERE p.id = ? AND EXISTS (SELECT 1 FROM notifikasi n WHERE n.pengumuman_id = p.id AND n.akun_id = ?)`, id, akunID).
		Scan(&p.ID, &p.Judul, &p.Isi, &mulai)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrTidakAda
	}
	if err != nil {
		return nil, err
	}
	p.Mulai = mulai.Format("2006-01-02")
	return &p, nil
}
