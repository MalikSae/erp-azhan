package notifikasi

import (
	"context"
	"strings"
)

// Notifikasi manasik (keputusan Q17 audit 2026-11-26): kotak masuk, bukan push. Penerima: akun pemesan dan akun yang
// tertaut ke jamaah (peran jamaah), untuk jamaah dewasa aktif di booking dp/lunas paket manasik. Satu notifikasi per
// akun per manasik untuk tiap jenis (kunci unik), ref_id = booking pertama akun itu di manasik tersebut.

// sqlTanggalManasik: tanggal pendek manasik, misalnya 12 Mar 2027.
const sqlTanggalManasik = `CONCAT(DAY(m.tanggal), ' ', ELT(MONTH(m.tanggal), 'Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des'), ' ', YEAR(m.tanggal))`

// sqlPenerimaManasik: (manasik_id, akun, booking_id) penerima notifikasi manasik. Saringan tambahan disisipkan lewat
// %SARING% pada kedua cabang.
const sqlPenerimaManasik = `
	SELECT mj.manasik_id, b.customer_account_id AS akun, b.id AS booking_id
	FROM manasik_jadwal mj
	JOIN bookings b ON b.schedule_id = mj.schedule_id
	JOIN booking_pax bp ON bp.booking_id = b.id
	WHERE b.status IN ('dp','lunas') AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant'
	  AND b.customer_account_id IS NOT NULL %SARING%
	UNION
	SELECT mj.manasik_id, a.id, b.id
	FROM manasik_jadwal mj
	JOIN bookings b ON b.schedule_id = mj.schedule_id
	JOIN booking_pax bp ON bp.booking_id = b.id
	JOIN customer_accounts a ON a.jamaah_id = bp.jamaah_id
	WHERE b.status IN ('dp','lunas') AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant' %SARING%`

func kirimManasik(ctx context.Context, q Execer, jenis, judul, isi, awalanKunci, saring string, args ...any) error {
	penerima := strings.ReplaceAll(sqlPenerimaManasik, "%SARING%", saring)
	// Argumen saringan dipakai dua kali (dua cabang UNION).
	semua := append([]any{jenis, judul, RefPerjalanan, awalanKunci}, args...)
	semua = append(semua, args...)
	_, err := q.ExecContext(ctx, `INSERT IGNORE INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, ref_id, kunci)
		SELECT x.akun, ?, ?, LEFT(`+isi+`, 300), ?, MIN(x.booking_id), CONCAT(?, m.id)
		FROM (`+penerima+`) x
		JOIN manasik m ON m.id = x.manasik_id
		JOIN customer_accounts a ON a.id = x.akun AND a.dihapus_at IS NULL
		GROUP BY x.akun, m.id, m.judul, m.tanggal, m.jam_mulai, m.tempat`, semua...)
	return err
}

// Manasik: manasik dibuat (scheduleIDs kosong = semua paket manasik) atau paket ditambahkan (hanya jamaah paket itu).
// Akun yang sudah diberi tahu tentang manasik ini tidak menerima lagi. Ubah manasik tidak memanggil fungsi ini.
func Manasik(ctx context.Context, q Execer, manasikID int64, scheduleIDs []int64) {
	saring, args := " AND mj.manasik_id = ?", []any{manasikID}
	if len(scheduleIDs) > 0 {
		saring += " AND mj.schedule_id IN (?" + strings.Repeat(",?", len(scheduleIDs)-1) + ")"
		for _, id := range scheduleIDs {
			args = append(args, id)
		}
	}
	err := kirimManasik(ctx, q, JenisManasik, "Jadwal manasik",
		`CONCAT(m.judul, ' · ', `+sqlTanggalManasik+`, ' · ', TIME_FORMAT(m.jam_mulai, '%H.%i'))`,
		"manasik:", saring, args...)
	catat(JenisManasik, err)
}

// manasikBesok: pengingat H-1, sekali per akun per manasik.
func manasikBesok(ctx context.Context, db Execer, hari string) error {
	return kirimManasik(ctx, db, JenisManasikBesok, "Manasik besok",
		`CONCAT(m.judul, ' · ', TIME_FORMAT(m.jam_mulai, '%H.%i'), ' · ', m.tempat)`,
		"manasik-besok:", " AND mj.manasik_id IN (SELECT id FROM manasik WHERE tanggal = DATE_ADD(?, INTERVAL 1 DAY))", hari)
}
