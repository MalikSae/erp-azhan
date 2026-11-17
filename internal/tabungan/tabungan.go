package tabungan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

// ErrTidakAda: baris tidak ada atau bukan milik akun. Handler membalas 404 tanpa membedakan keduanya.
var ErrTidakAda = errors.New("tabungan: tidak ada")

// ErrTanpaTarget: catatan ditambah saat akun belum punya target.
var ErrTanpaTarget = errors.New("tabungan: belum ada target")

// Querier dipenuhi *sql.DB dan *sql.Tx.
type Querier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func indeksBulan(t time.Time) int { return t.Year()*12 + int(t.Month()) - 1 }

func parseBulan(s string) (time.Time, bool) {
	if len(s) != 7 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01", s)
	return t, err == nil
}

// ValidasiTarget memeriksa batas kontrak dan merapikan teks. Mengembalikan pesan galat, kosong bila lolos.
func ValidasiTarget(t *Target, now time.Time) string {
	t.NamaPaket = strings.TrimSpace(t.NamaPaket)
	switch t.Jenis {
	case "paket":
		if t.ScheduleID == nil || *t.ScheduleID <= 0 {
			return "Pilih paket."
		}
	case "perkiraan":
		if t.ScheduleID != nil {
			return "Data target tidak valid."
		}
	default:
		return "Jenis target tidak valid."
	}
	if utf8.RuneCountInString(t.NamaPaket) > MaksNamaPaket {
		return "Nama paket maksimal 150 karakter."
	}
	bulan, ok := parseBulan(t.BulanBerangkat)
	if !ok {
		return "Bulan berangkat tidak valid."
	}
	selisih := indeksBulan(bulan) - indeksBulan(now.In(ZonaWIB))
	if selisih < 0 {
		return "Bulan berangkat tidak boleh lewat."
	}
	if selisih > MaksBulanKeDepan {
		return "Bulan berangkat maksimal 36 bulan lagi."
	}
	if t.JumlahDewasa < 1 || t.JumlahDewasa > MaksDewasa {
		return "Jumlah dewasa 1 sampai 20."
	}
	if t.JumlahBayi < 0 || t.JumlahBayi > MaksBayi {
		return "Jumlah bayi 0 sampai 10."
	}
	if !TipeKamar[t.TipeKamar] {
		return "Pilih tipe kamar."
	}
	if t.TargetTotal < MinTarget || t.TargetTotal > MaksTarget {
		return "Target total Rp1.000.000 sampai Rp2.000.000.000."
	}
	if t.DPMinimal < 0 || t.DPMinimal > t.TargetTotal {
		return "DP tidak boleh melebihi target total."
	}
	if t.PengingatTanggal < 1 || t.PengingatTanggal > 28 {
		return "Tanggal pengingat 1 sampai 28."
	}
	if t.PengingatJam < 0 || t.PengingatJam > 23 {
		return "Jam pengingat 0 sampai 23."
	}
	return ""
}

// ValidasiCatatan memeriksa jumlah dan tanggal. Tanggal kosong berarti hari ini WIB.
// Mengembalikan tanggal yang dipakai dan pesan galat.
func ValidasiCatatan(jumlah int64, tanggal string, now time.Time) (string, string) {
	if jumlah < MinCatatan || jumlah > MaksCatatan {
		return "", "Jumlah Rp1.000 sampai Rp1.000.000.000."
	}
	hariIni := now.In(ZonaWIB).Format("2006-01-02")
	if tanggal == "" {
		return hariIni, ""
	}
	d, err := time.Parse("2006-01-02", tanggal)
	if err != nil || d.Format("2006-01-02") != tanggal {
		return "", "Tanggal tidak valid."
	}
	if tanggal > hariIni {
		return "", "Tanggal tidak boleh lewat hari ini."
	}
	if d.Before(TanggalAwalCatatan) {
		return "", "Tanggal tidak boleh sebelum 2020."
	}
	return tanggal, ""
}

// Hitung menghitung ringkasan dari target dan angka catatan. bulanIni adalah jumlah catatan di bulan berjalan WIB.
func Hitung(t Target, tercatat, bulanIni int64, now time.Time) Ringkasan {
	r := Ringkasan{Tercatat: tercatat, TercatatBulanIni: bulanIni}
	r.Sisa = t.TargetTotal - tercatat
	if r.Sisa < 0 {
		r.Sisa = 0
	}
	if t.TargetTotal > 0 {
		p := tercatat * 100 / t.TargetTotal
		if p > 100 {
			p = 100
		}
		r.Persen = int(p)
	}
	r.SisaBulan = 1
	if bulan, ok := parseBulan(t.BulanBerangkat); ok {
		if s := indeksBulan(bulan) - indeksBulan(now.In(ZonaWIB)); s > 1 {
			r.SisaBulan = s
		}
	}
	if r.Sisa > 0 {
		n := int64(r.SisaBulan)
		perBulan := (r.Sisa + n - 1) / n
		r.PerBulan = (perBulan + KelipatanPerBulan - 1) / KelipatanPerBulan * KelipatanPerBulan
	}
	r.DPTercapai = tercatat >= t.DPMinimal
	r.Tercapai = tercatat >= t.TargetTotal
	return r
}

// Ambil membaca target, ringkasan, dan 200 catatan terbaru milik akun.
func Ambil(ctx context.Context, q Querier, akun int64, now time.Time) (*Respons, error) {
	out := &Respons{Catatan: []Catatan{}}
	var tg TargetOut
	var sched sql.NullInt64
	var dibuat time.Time
	err := q.QueryRowContext(ctx, `
		SELECT jenis, schedule_id, nama_paket, bulan_berangkat, jumlah_dewasa, jumlah_bayi, tipe_kamar,
		       target_total, dp_minimal, pengingat_aktif, pengingat_tanggal, pengingat_jam, dibuat_at
		FROM tabungan_target WHERE akun_id = ?`, akun).Scan(
		&tg.Jenis, &sched, &tg.NamaPaket, &tg.BulanBerangkat, &tg.JumlahDewasa, &tg.JumlahBayi, &tg.TipeKamar,
		&tg.TargetTotal, &tg.DPMinimal, &tg.PengingatAktif, &tg.PengingatTanggal, &tg.PengingatJam, &dibuat)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("tabungan: baca target: %w", err)
	}
	if err == nil {
		if sched.Valid {
			v := sched.Int64
			tg.ScheduleID = &v
		} else if tg.Jenis == "paket" {
			// Paket draft yang dihapus admin meninggalkan schedule_id null; nama_paket tetap.
			tg.Jenis = "perkiraan"
		}
		tg.DibuatAt = dibuat.Format(time.RFC3339)
		out.Target = &tg

		awalBulan := now.In(ZonaWIB).Format("2006-01") + "-01"
		var tercatat, bulanIni int64
		if err := q.QueryRowContext(ctx, `
			SELECT COALESCE(SUM(jumlah), 0), COALESCE(SUM(CASE WHEN tanggal >= ? THEN jumlah ELSE 0 END), 0)
			FROM tabungan_catatan WHERE akun_id = ?`, awalBulan, akun).Scan(&tercatat, &bulanIni); err != nil {
			return nil, fmt.Errorf("tabungan: jumlah catatan: %w", err)
		}
		r := Hitung(tg.Target, tercatat, bulanIni, now)
		out.Ringkasan = &r
	}

	rows, err := q.QueryContext(ctx, `
		SELECT id, jumlah, DATE_FORMAT(tanggal, '%Y-%m-%d'), dibuat_at
		FROM tabungan_catatan WHERE akun_id = ?
		ORDER BY tanggal DESC, id DESC LIMIT ?`, akun, MaksDaftarCatatan)
	if err != nil {
		return nil, fmt.Errorf("tabungan: baca catatan: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var c Catatan
		var at time.Time
		if err := rows.Scan(&c.ID, &c.Jumlah, &c.Tanggal, &at); err != nil {
			return nil, fmt.Errorf("tabungan: baca catatan: %w", err)
		}
		c.DibuatAt = at.Format(time.RFC3339)
		out.Catatan = append(out.Catatan, c)
	}
	return out, rows.Err()
}

// JadwalTayang: apakah jadwal (paket) berstatus published dan belum berangkat (tanggal WIB).
func JadwalTayang(ctx context.Context, q Querier, id int64, now time.Time) (bool, error) {
	var n int
	if err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM schedules WHERE id = ? AND status = 'published' AND berangkat_tanggal >= ?`,
		id, now.In(ZonaWIB).Format("2006-01-02")).Scan(&n); err != nil {
		return false, fmt.Errorf("tabungan: cek jadwal: %w", err)
	}
	return n > 0, nil
}

// SimpanTarget: upsert satu target per akun. Catatan tidak disentuh.
func SimpanTarget(ctx context.Context, q Querier, akun int64, t Target) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO tabungan_target (akun_id, jenis, schedule_id, nama_paket, bulan_berangkat, jumlah_dewasa, jumlah_bayi,
			tipe_kamar, target_total, dp_minimal, pengingat_aktif, pengingat_tanggal, pengingat_jam)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON DUPLICATE KEY UPDATE jenis = VALUES(jenis), schedule_id = VALUES(schedule_id), nama_paket = VALUES(nama_paket),
			bulan_berangkat = VALUES(bulan_berangkat), jumlah_dewasa = VALUES(jumlah_dewasa), jumlah_bayi = VALUES(jumlah_bayi),
			tipe_kamar = VALUES(tipe_kamar), target_total = VALUES(target_total), dp_minimal = VALUES(dp_minimal),
			pengingat_aktif = VALUES(pengingat_aktif), pengingat_tanggal = VALUES(pengingat_tanggal),
			pengingat_jam = VALUES(pengingat_jam)`,
		akun, t.Jenis, t.ScheduleID, t.NamaPaket, t.BulanBerangkat, t.JumlahDewasa, t.JumlahBayi,
		t.TipeKamar, t.TargetTotal, t.DPMinimal, t.PengingatAktif, t.PengingatTanggal, t.PengingatJam)
	if err != nil {
		return fmt.Errorf("tabungan: simpan target: %w", err)
	}
	return nil
}

// HapusSemua menghapus target dan seluruh catatan akun.
func HapusSemua(ctx context.Context, q Querier, akun int64) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM tabungan_catatan WHERE akun_id = ?`, akun); err != nil {
		return fmt.Errorf("tabungan: hapus catatan: %w", err)
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM tabungan_target WHERE akun_id = ?`, akun); err != nil {
		return fmt.Errorf("tabungan: hapus target: %w", err)
	}
	return nil
}

// HapusDataAkun menghapus target, catatan, dan token push milik akun (dipanggil saat akun dianonimkan).
func HapusDataAkun(ctx context.Context, q Querier, akun int64) error {
	if err := HapusSemua(ctx, q, akun); err != nil {
		return err
	}
	if _, err := q.ExecContext(ctx, `DELETE FROM perangkat_push WHERE akun_id = ?`, akun); err != nil {
		return fmt.Errorf("tabungan: hapus perangkat: %w", err)
	}
	return nil
}

// TambahCatatan menulis satu catatan bila akun punya target (atomik); tanpa target ErrTanpaTarget.
func TambahCatatan(ctx context.Context, q Querier, akun, jumlah int64, tanggal string) error {
	res, err := q.ExecContext(ctx, `
		INSERT INTO tabungan_catatan (akun_id, jumlah, tanggal)
		SELECT ?, ?, ? FROM tabungan_target WHERE akun_id = ?`, akun, jumlah, tanggal, akun)
	if err != nil {
		return fmt.Errorf("tabungan: tambah catatan: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTanpaTarget
	}
	return nil
}

// HapusCatatan menghapus satu catatan milik akun; milik akun lain atau tidak ada: ErrTidakAda.
func HapusCatatan(ctx context.Context, q Querier, akun, id int64) error {
	res, err := q.ExecContext(ctx, `DELETE FROM tabungan_catatan WHERE id = ? AND akun_id = ?`, id, akun)
	if err != nil {
		return fmt.Errorf("tabungan: hapus catatan: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrTidakAda
	}
	return nil
}

var polaToken = regexp.MustCompile(`^Expo(nent)?PushToken\[[A-Za-z0-9_-]+\]$`)

// TokenValid: format token push Expo.
func TokenValid(token string) bool { return len(token) <= 255 && polaToken.MatchString(token) }

// SimpanPerangkat: upsert berdasarkan token (token milik akun lain pindah ke akun ini), lalu sisakan 10 terbaru.
func SimpanPerangkat(ctx context.Context, q Querier, akun int64, token, platform string) error {
	if _, err := q.ExecContext(ctx, `
		INSERT INTO perangkat_push (akun_id, token, platform) VALUES (?,?,?)
		ON DUPLICATE KEY UPDATE akun_id = VALUES(akun_id), platform = VALUES(platform), diperbarui_at = CURRENT_TIMESTAMP(3)`,
		akun, token, platform); err != nil {
		return fmt.Errorf("tabungan: simpan perangkat: %w", err)
	}
	rows, err := q.QueryContext(ctx, `
		SELECT id FROM perangkat_push WHERE akun_id = ?
		ORDER BY diperbarui_at DESC, id DESC LIMIT 18446744073709551615 OFFSET ?`, akun, MaksTokenPerAkun)
	if err != nil {
		return fmt.Errorf("tabungan: baca perangkat: %w", err)
	}
	var lebih []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("tabungan: baca perangkat: %w", err)
		}
		lebih = append(lebih, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range lebih {
		if _, err := q.ExecContext(ctx, `DELETE FROM perangkat_push WHERE id = ?`, id); err != nil {
			return fmt.Errorf("tabungan: hapus perangkat lama: %w", err)
		}
	}
	return nil
}

// HapusPerangkat menghapus token milik akun sendiri; token orang lain atau tidak ada diabaikan.
func HapusPerangkat(ctx context.Context, q Querier, akun int64, token string) error {
	if _, err := q.ExecContext(ctx, `DELETE FROM perangkat_push WHERE akun_id = ? AND token = ?`, akun, token); err != nil {
		return fmt.Errorf("tabungan: hapus perangkat: %w", err)
	}
	return nil
}
