// Package pengumuman: pengumuman untuk pelanggan di dashboard akun. Admin menulisnya per brand dengan masa tayang, dan boleh menujukannya
// ke satu jadwal keberangkatan; pelanggan hanya melihat yang masih tayang dan sesuai booking mereka.
package pengumuman

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"
	"unicode/utf8"
)

type Pengumuman struct {
	ID         int64  `json:"id"`
	BrandID    int64  `json:"brand_id"`
	ScheduleID *int64 `json:"schedule_id"`
	JadwalNama string `json:"jadwal_nama"`
	Judul      string `json:"judul"`
	Isi        string `json:"isi"`
	Mulai      string `json:"mulai"`
	Akhir      string `json:"akhir"`
	Status     string `json:"status"` // akan_tayang | tayang | berakhir
}

// Input: isian dari admin. BrandID dipakai admin lintas brand (holding); admin satu brand selalu memakai brand-nya.
type Input struct {
	BrandID    *int64 `json:"brand_id"`
	ScheduleID *int64 `json:"schedule_id"`
	Judul      string `json:"judul"`
	Isi        string `json:"isi"`
	Mulai      string `json:"mulai"`
	Akhir      string `json:"akhir"`
}

type ErrValidasi struct{ Pesan string }

func (e *ErrValidasi) Error() string { return e.Pesan }

var ErrTidakAda = errors.New("pengumuman tidak ditemukan")

const (
	maksJudul = 120
	maksIsi   = 1000
	tglFormat = "2006-01-02"
)

type dbq interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

// hariIni: tanggal kalender WIB (zona aplikasi), sebagai YYYY-MM-DD.
func hariIni(sekarang time.Time) string {
	return sekarang.In(time.FixedZone("WIB", 7*3600)).Format(tglFormat)
}

func status(mulai, akhir, hari string) string {
	switch {
	case hari < mulai:
		return "akan_tayang"
	case hari > akhir:
		return "berakhir"
	}
	return "tayang"
}

// validasi memeriksa isian dan merapikannya (spasi dipangkas). Mengembalikan pesan untuk admin, atau "".
func (in *Input) validasi() string {
	in.Judul, in.Isi = strings.TrimSpace(in.Judul), strings.TrimSpace(in.Isi)
	if n := utf8.RuneCountInString(in.Judul); n < 3 || n > maksJudul {
		return "judul wajib diisi (3 sampai 120 karakter)"
	}
	if n := utf8.RuneCountInString(in.Isi); n < 3 || n > maksIsi {
		return "isi wajib diisi (3 sampai 1000 karakter)"
	}
	m, err1 := time.Parse(tglFormat, in.Mulai)
	a, err2 := time.Parse(tglFormat, in.Akhir)
	if err1 != nil || err2 != nil {
		return "tanggal mulai dan akhir tayang wajib diisi (YYYY-MM-DD)"
	}
	if a.Before(m) {
		return "tanggal akhir tayang tidak boleh sebelum tanggal mulai"
	}
	return ""
}

const kolom = `p.id, p.brand_id, p.schedule_id, COALESCE(s.jadwal_nama,''), p.judul, p.isi,
	DATE_FORMAT(p.mulai_tayang,'%Y-%m-%d'), DATE_FORMAT(p.akhir_tayang,'%Y-%m-%d')`

func scan(rows interface{ Scan(...any) error }, hari string) (*Pengumuman, error) {
	var p Pengumuman
	var sched sql.NullInt64
	if err := rows.Scan(&p.ID, &p.BrandID, &sched, &p.JadwalNama, &p.Judul, &p.Isi, &p.Mulai, &p.Akhir); err != nil {
		return nil, err
	}
	if sched.Valid {
		p.ScheduleID = &sched.Int64
	}
	p.Status = status(p.Mulai, p.Akhir, hari)
	return &p, nil
}

// Daftar (admin): semua pengumuman brand (brandID nil = semua brand), yang terbaru di atas.
func Daftar(ctx context.Context, q dbq, brandID *int64, sekarang time.Time) ([]Pengumuman, error) {
	query := `SELECT ` + kolom + ` FROM pengumuman p LEFT JOIN schedules s ON s.id = p.schedule_id WHERE 1=1`
	var args []any
	if brandID != nil {
		query += ` AND p.brand_id = ?`
		args = append(args, *brandID)
	}
	rows, err := q.QueryContext(ctx, query+` ORDER BY p.mulai_tayang DESC, p.id DESC LIMIT 500`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	hari := hariIni(sekarang)
	out := []Pengumuman{}
	for rows.Next() {
		p, err := scan(rows, hari)
		if err != nil {
			return nil, err
		}
		out = append(out, *p)
	}
	return out, rows.Err()
}

// jadwalMilikBrand memastikan jadwal ada dan milik brand yang sama.
func jadwalMilikBrand(ctx context.Context, q dbq, jadwalID, brandID int64) error {
	var n int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM schedules WHERE id=? AND brand_id=?`, jadwalID, brandID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return &ErrValidasi{Pesan: "jadwal keberangkatan tidak ditemukan di brand ini"}
	}
	return nil
}

func nullInt(p *int64) any {
	if p == nil {
		return nil
	}
	return *p
}

// Buat menyimpan pengumuman baru. brandID sudah ditentukan pemanggil (brand admin, atau pilihan admin holding).
func Buat(ctx context.Context, q dbq, brandID, adminID int64, in Input) (int64, error) {
	if msg := in.validasi(); msg != "" {
		return 0, &ErrValidasi{Pesan: msg}
	}
	if in.ScheduleID != nil {
		if err := jadwalMilikBrand(ctx, q, *in.ScheduleID, brandID); err != nil {
			return 0, err
		}
	}
	var admin any
	if adminID > 0 {
		admin = adminID
	}
	res, err := q.ExecContext(ctx, `INSERT INTO pengumuman (brand_id, schedule_id, judul, isi, mulai_tayang, akhir_tayang, dibuat_oleh) VALUES (?,?,?,?,?,?,?)`,
		brandID, nullInt(in.ScheduleID), in.Judul, in.Isi, in.Mulai, in.Akhir, admin)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Ubah memperbarui pengumuman; brand pengumuman tidak berubah. scope: brand admin (nil = lintas brand).
func Ubah(ctx context.Context, q dbq, id int64, scope *int64, in Input) error {
	if msg := in.validasi(); msg != "" {
		return &ErrValidasi{Pesan: msg}
	}
	var brandID int64
	if err := q.QueryRowContext(ctx, `SELECT brand_id FROM pengumuman WHERE id=?`, id).Scan(&brandID); errors.Is(err, sql.ErrNoRows) {
		return ErrTidakAda
	} else if err != nil {
		return err
	}
	if scope != nil && *scope != brandID {
		return ErrTidakAda
	}
	if in.ScheduleID != nil {
		if err := jadwalMilikBrand(ctx, q, *in.ScheduleID, brandID); err != nil {
			return err
		}
	}
	_, err := q.ExecContext(ctx, `UPDATE pengumuman SET schedule_id=?, judul=?, isi=?, mulai_tayang=?, akhir_tayang=? WHERE id=?`,
		nullInt(in.ScheduleID), in.Judul, in.Isi, in.Mulai, in.Akhir, id)
	return err
}

// Hapus menghapus pengumuman (dalam brand scope admin).
func Hapus(ctx context.Context, q dbq, id int64, scope *int64) error {
	query := `DELETE FROM pengumuman WHERE id=?`
	args := []any{id}
	if scope != nil {
		query += ` AND brand_id=?`
		args = append(args, *scope)
	}
	res, err := q.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTidakAda
	}
	return nil
}

// UntukAkun: pengumuman yang masih tayang pada sekarang dan sesuai akun: tanpa jadwal untuk semua pelanggan brand, dengan jadwal hanya bila
// akun punya booking (selain draft dan batal) di jadwal itu. Terbaru di atas; limit <= 0 berarti semua. total = jumlah seluruhnya.
func UntukAkun(ctx context.Context, q dbq, brandID, akunID int64, limit int, sekarang time.Time) (items []Pengumuman, total int, err error) {
	hari := hariIni(sekarang)
	const sesuai = ` FROM pengumuman p LEFT JOIN schedules s ON s.id = p.schedule_id
		WHERE p.brand_id = ? AND ? BETWEEN p.mulai_tayang AND p.akhir_tayang
		  AND (p.schedule_id IS NULL OR p.schedule_id IN (
		       SELECT b.schedule_id FROM bookings b WHERE b.customer_account_id = ? AND b.status NOT IN ('draft','batal')))`
	if err = q.QueryRowContext(ctx, `SELECT COUNT(*)`+sesuai, brandID, hari, akunID).Scan(&total); err != nil {
		return nil, 0, err
	}
	query := `SELECT ` + kolom + sesuai + ` ORDER BY p.mulai_tayang DESC, p.id DESC`
	args := []any{brandID, hari, akunID}
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items = []Pengumuman{}
	for rows.Next() {
		p, err := scan(rows, hari)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, *p)
	}
	return items, total, rows.Err()
}
