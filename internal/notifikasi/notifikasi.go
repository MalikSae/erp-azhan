// Package notifikasi: kotak masuk notifikasi akun pelanggan di app (keputusan 16 November 2026).
//
// Notifikasi disimpan satu baris per akun saat kejadian terjadi. Paket lain memanggil Kirim di
// dalam transaksi yang sama dengan perubahan datanya; kegagalan menulis notifikasi hanya dicatat
// di log dan tidak membatalkan perubahan utama (lihat KirimAman). Pengingat berbasis waktu dan
// pengumuman dibuat oleh loop latar (pengingat.go) dengan kunci unik supaya tidak dobel.
package notifikasi

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Jenis notifikasi. Nilainya disimpan di kolom jenis dan dikirim apa adanya ke app.
const (
	JenisPembayaranDiterima  = "pembayaran_diterima"
	JenisPembayaranDitolak   = "pembayaran_ditolak"
	JenisBookingLunas        = "booking_lunas"
	JenisBookingBatal        = "booking_batal"
	JenisHoldSeat            = "hold_seat"
	JenisPelunasan           = "pelunasan"
	JenisDokumenDiverifikasi = "dokumen_diverifikasi"
	JenisDokumenDitolak      = "dokumen_ditolak"
	JenisManasik             = "manasik"
	JenisManasikBesok        = "manasik_besok"
	JenisPerlengkapanSiap    = "perlengkapan_siap"
	JenisBerangkat           = "berangkat"
	JenisPerubahanDisetujui  = "perubahan_disetujui"
	JenisPerubahanDitolak    = "perubahan_ditolak"
	JenisAgenDisetujui       = "agen_disetujui"
	JenisJamaahReferral      = "jamaah_referral"
	JenisJamaahAktivasi      = "jamaah_aktivasi"
	JenisKomisiTersedia      = "komisi_tersedia"
	JenisKomisiBatal         = "komisi_batal"
	JenisPencairanDiproses   = "pencairan_diproses"
	JenisPencairanDitolak    = "pencairan_ditolak"
	JenisAgenBinaan          = "agen_binaan"
	JenisProspekBaru         = "prospek_baru"
	JenisProspekHold         = "prospek_hold"
	JenisPengumuman          = "pengumuman"
	JenisTahapan             = "tahapan"
)

// Tujuan layar app saat notifikasi ditap.
const (
	RefBooking    = "booking"    // ref_id = booking id (detail pesanan)
	RefPerjalanan = "perjalanan" // ref_id = booking id (detail perjalanan)
	RefDompet     = "dompet"     // dompet agen
	RefAgen       = "agen"       // beranda agen
	RefPengumuman = "pengumuman" // ref_id = pengumuman id
)

// MasaSimpan: notifikasi lebih tua dari ini dihapus.
const MasaSimpan = 90 * 24 * time.Hour

// BatasDaftar: jumlah notifikasi terbanyak yang dikirim ke app sekali muat.
const BatasDaftar = 100

// ZonaWIB dipakai untuk jam pengingat dan tanggal di isi notifikasi.
var ZonaWIB = time.FixedZone("WIB", 7*3600)

var ErrTidakAda = errors.New("notifikasi tidak ditemukan")

// Execer cukup untuk *sql.DB dan *sql.Tx.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

type Querier interface {
	Execer
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Notif: satu notifikasi yang akan ditulis.
type Notif struct {
	Jenis   string
	Judul   string
	Isi     string
	RefTipe string
	RefID   int64
	// Kunci opsional; bila diisi, notifikasi dengan kunci yang sama untuk akun yang sama diabaikan.
	Kunci string
}

// Kirim menulis satu notifikasi untuk akun. Akun 0 diabaikan (booking tanpa akun).
func Kirim(ctx context.Context, q Execer, akunID int64, n Notif) error {
	if akunID <= 0 {
		return nil
	}
	_, err := q.ExecContext(ctx, `INSERT IGNORE INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, ref_id, kunci)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		akunID, n.Jenis, potong(n.Judul, 150), potong(n.Isi, 300), kosongNull(n.RefTipe), nolNull(n.RefID), kosongNull(n.Kunci))
	return err
}

// KirimAman memanggil Kirim dan hanya mencatat kegagalan di log, supaya perubahan utama tetap tersimpan.
func KirimAman(ctx context.Context, q Execer, akunID int64, n Notif) {
	if err := Kirim(ctx, q, akunID, n); err != nil {
		log.Printf("[WARN] notifikasi %s akun=%d: %v", n.Jenis, akunID, err)
	}
}

// Item: notifikasi untuk app.
type Item struct {
	ID      int64   `json:"id"`
	Jenis   string  `json:"jenis"`
	Judul   string  `json:"judul"`
	Isi     string  `json:"isi"`
	RefTipe *string `json:"ref_tipe"`
	RefID   *int64  `json:"ref_id"`
	Dibaca  bool    `json:"dibaca"`
	Dibuat  string  `json:"dibuat_at"` // YYYY-MM-DD HH:MM (WIB)
}

// Daftar: notifikasi akun terbaru lebih dulu, tidak lebih tua dari MasaSimpan. Notifikasi pengumuman
// memakai judul dan isi pengumuman terkini.
func Daftar(ctx context.Context, q Querier, akunID int64, now time.Time) ([]Item, error) {
	rows, err := q.QueryContext(ctx, `SELECT n.id, n.jenis, COALESCE(p.judul, n.judul), COALESCE(p.isi, n.isi),
			n.ref_tipe, n.ref_id, n.dibaca_at IS NOT NULL, n.dibuat_at
		FROM notifikasi n LEFT JOIN pengumuman p ON p.id = n.pengumuman_id
		WHERE n.akun_id = ? AND n.dibuat_at >= ?
		ORDER BY n.dibuat_at DESC, n.id DESC LIMIT ?`, akunID, batasLama(now), BatasDaftar)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		var it Item
		var refTipe sql.NullString
		var refID sql.NullInt64
		var dibuat time.Time
		if err := rows.Scan(&it.ID, &it.Jenis, &it.Judul, &it.Isi, &refTipe, &refID, &it.Dibaca, &dibuat); err != nil {
			return nil, err
		}
		if refTipe.Valid {
			it.RefTipe = &refTipe.String
		}
		if refID.Valid {
			it.RefID = &refID.Int64
		}
		if it.Jenis == JenisPengumuman {
			it.Isi = ringkas(it.Isi, 140)
		}
		it.Dibuat = dibuat.Format("2006-01-02 15:04")
		out = append(out, it)
	}
	return out, rows.Err()
}

// JumlahBelumDibaca untuk angka di lonceng.
func JumlahBelumDibaca(ctx context.Context, q Querier, akunID int64, now time.Time) (int, error) {
	var n int
	err := q.QueryRowContext(ctx, `SELECT COUNT(*) FROM notifikasi WHERE akun_id = ? AND dibaca_at IS NULL AND dibuat_at >= ?`,
		akunID, batasLama(now)).Scan(&n)
	return n, err
}

// TandaiDibaca menandai satu notifikasi milik akun.
func TandaiDibaca(ctx context.Context, q Querier, akunID, id int64) error {
	res, err := q.ExecContext(ctx, `UPDATE notifikasi SET dibaca_at = COALESCE(dibaca_at, NOW()) WHERE id = ? AND akun_id = ?`, id, akunID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		var ada int
		if err := q.QueryRowContext(ctx, `SELECT 1 FROM notifikasi WHERE id = ? AND akun_id = ?`, id, akunID).Scan(&ada); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrTidakAda
			}
			return err
		}
	}
	return nil
}

// TandaiSemuaDibaca menandai semua notifikasi akun.
func TandaiSemuaDibaca(ctx context.Context, q Execer, akunID int64) error {
	_, err := q.ExecContext(ctx, `UPDATE notifikasi SET dibaca_at = NOW() WHERE akun_id = ? AND dibaca_at IS NULL`, akunID)
	return err
}

// Bersihkan menghapus notifikasi yang lebih tua dari MasaSimpan.
func Bersihkan(ctx context.Context, q Execer, now time.Time) error {
	_, err := q.ExecContext(ctx, `DELETE FROM notifikasi WHERE dibuat_at < ?`, batasLama(now))
	return err
}

// Rupiah menulis nominal bulat dengan titik ribuan, misalnya Rp5.000.000.
func Rupiah(v float64) string {
	neg := v < 0
	if neg {
		v = -v
	}
	digit := strconv.FormatInt(int64(v+0.5), 10)
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	b.WriteString("Rp")
	for i, c := range digit {
		if i > 0 && (len(digit)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return b.String()
}

var bulan = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

// Tanggal menulis tanggal pendek, misalnya 12 Mar 2027.
func Tanggal(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + bulan[t.Month()-1] + " " + strconv.Itoa(t.Year())
}

// TanggalTeks mengubah "YYYY-MM-DD..." menjadi tanggal pendek; teks lain dikembalikan apa adanya.
func TanggalTeks(s string) string {
	if len(s) < 10 {
		return s
	}
	t, err := time.Parse("2006-01-02", s[:10])
	if err != nil {
		return s
	}
	return Tanggal(t)
}

// Gabung menyatukan bagian isi dengan " · ", melewati bagian kosong.
func Gabung(bagian ...string) string {
	isi := make([]string, 0, len(bagian))
	for _, b := range bagian {
		if b = strings.TrimSpace(b); b != "" {
			isi = append(isi, b)
		}
	}
	return strings.Join(isi, " · ")
}

func batasLama(now time.Time) time.Time { return now.Add(-MasaSimpan) }

func potong(s string, maks int) string {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) <= maks {
		return s
	}
	r := []rune(s)
	return string(r[:maks-1]) + "…"
}

// ringkas: satu baris pendek dari isi panjang (baris baru jadi spasi).
func ringkas(s string, maks int) string {
	return potong(strings.Join(strings.Fields(s), " "), maks)
}

func kosongNull(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func nolNull(v int64) any {
	if v == 0 {
		return nil
	}
	return v
}
