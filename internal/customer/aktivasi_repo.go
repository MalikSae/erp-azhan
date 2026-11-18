package customer

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Penghitung percobaan aktivasi yang atomik di database (migrasi 154). Slot diambil SEBELUM evaluasi lewat satu
// INSERT .. ON DUPLICATE KEY UPDATE, jadi permintaan paralel tidak bisa melewati batas bersamaan; slot dikembalikan
// saat aktivasi berhasil. Jendela waktu tetap per jenis batas.

type jenisBatas struct {
	jenis string
	lebar time.Duration
	batas int
}

// batasPasangan (kode+NIK) adalah pengaman utama: kegagalan satu pasangan tidak mengunci kode atau NIK orang lain.
// batasKode24 dan batasNIK24 sengaja tinggi: hanya menahan tebakan terdistribusi dan menjadi alarm saat tercapai.
// batasBrand hanya alarm (log), tidak menolak atau memperketat apa pun.
var (
	batasIP        = jenisBatas{"ip", 15 * time.Minute, 5}
	batasPerangkat = jenisBatas{"dev", 15 * time.Minute, 5}
	batasPasangan  = jenisBatas{"pasangan", 15 * time.Minute, 5}
	batasKode24    = jenisBatas{"kode_24", 24 * time.Hour, 100}
	batasNIK24     = jenisBatas{"nik_24", 24 * time.Hour, 100}
	batasBrand     = jenisBatas{"brand", time.Hour, 300}
	batasSimpanIP  = jenisBatas{"simpan_ip", time.Hour, 10}
)

// RetryAfterAktivasi: nilai Retry-After tetap untuk semua 429 aktivasi, supaya jenis batas yang terpicu tidak terbaca.
const RetryAfterAktivasi = 15 * time.Minute

// NormalisasiIPAktivasi menyeragamkan IP untuk penghitung: IPv4 apa adanya, IPv6 dipotong ke blok /64.
func NormalisasiIPAktivasi(ip string) string {
	host := strings.TrimSpace(ip)
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	parsed := net.ParseIP(host)
	if parsed == nil {
		return ip
	}
	if v4 := parsed.To4(); v4 != nil {
		return v4.String()
	}
	return parsed.Mask(net.CIDRMask(64, 128)).String()
}

// slotBatas: satu slot yang sudah diambil, untuk dikembalikan saat berhasil.
type slotBatas struct {
	jenis   string
	kunci   string
	jendela int64
}

type pembatas struct {
	db     *sql.DB
	pepper []byte
	now    func() time.Time
}

// hash menghasilkan HMAC-SHA256 heksadesimal (64 karakter). Nilai mentah (kode, NIK, IP, perangkat) tidak pernah disimpan.
func (p *pembatas) hash(jenis string, brandID int64, nilai string) string {
	m := hmac.New(sha256.New, p.pepper)
	m.Write([]byte(jenis + "|" + strconv.FormatInt(brandID, 10) + "|" + nilai))
	return hex.EncodeToString(m.Sum(nil))
}

func (p *pembatas) jendela(b jenisBatas) (nomor int64, akhir time.Time) {
	detik := int64(b.lebar / time.Second)
	nomor = p.now().Unix() / detik
	return nomor, time.Unix((nomor+1)*detik, 0)
}

// ambil menaikkan penghitung satu langkah. ok=false bila batas terlampaui (slot tetap terhitung sebagai percobaan);
// tunggu: sisa waktu jendela. urutan: nilai penghitung setelah dinaikkan.
func (p *pembatas) ambil(ctx context.Context, b jenisBatas, kunciHash string) (s slotBatas, urutan int, ok bool, tunggu time.Duration, err error) {
	nomor, akhir := p.jendela(b)
	res, err := p.db.ExecContext(ctx, `
		INSERT INTO jamaah_aktivasi_hitung (kunci_hash, jenis, jendela, jumlah, kedaluwarsa_at) VALUES (?,?,?,1,?)
		ON DUPLICATE KEY UPDATE jumlah = LAST_INSERT_ID(jumlah + 1)`, kunciHash, b.jenis, nomor, akhir.Add(time.Hour))
	if err != nil {
		return s, 0, false, 0, fmt.Errorf("aktivasi: ambil slot %s: %w", b.jenis, err)
	}
	id, _ := res.LastInsertId()
	urutan = int(id)
	if urutan == 0 {
		urutan = 1
	}
	tunggu = akhir.Sub(p.now())
	if tunggu < time.Second {
		tunggu = time.Second
	}
	return slotBatas{jenis: b.jenis, kunci: kunciHash, jendela: nomor}, urutan, urutan <= b.batas, tunggu, nil
}

// kembalikan mengurangi satu slot yang sudah diambil.
func (p *pembatas) kembalikan(ctx context.Context, s slotBatas) error {
	_, err := p.db.ExecContext(ctx, `
		UPDATE jamaah_aktivasi_hitung SET jumlah = IF(jumlah > 0, jumlah - 1, 0)
		WHERE kunci_hash = ? AND jenis = ? AND jendela = ?`, s.kunci, s.jenis, s.jendela)
	if err != nil {
		return fmt.Errorf("aktivasi: kembalikan slot %s: %w", s.jenis, err)
	}
	return nil
}

// entriLogAktivasi: satu baris jamaah_aktivasi_log. Hanya hash, tanpa NIK atau kode mentah.
type entriLogAktivasi struct {
	BrandID    int64
	Aksi       string
	JamaahID   int64
	AkunID     int64
	KodeHash   string
	NIKHash    string
	IPHash     string
	DeviceHash string
	Alasan     string
	// Hanya aksi admin (lepas_admin).
	OlehAdminID int64
	Catatan     string
}

type execer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func catatLogAktivasi(ctx context.Context, q execer, e entriLogAktivasi) error {
	_, err := q.ExecContext(ctx, `
		INSERT INTO jamaah_aktivasi_log (brand_id, aksi, jamaah_id, akun_id, kode_hash, nik_hash, ip_hash, device_hash, alasan, oleh_admin_id, catatan)
		VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?)`,
		e.BrandID, e.Aksi, nullableAgen(e.JamaahID), nullableAgen(e.AkunID), e.KodeHash, e.NIKHash, e.IPHash, e.DeviceHash, e.Alasan,
		nullableAgen(e.OlehAdminID), e.Catatan)
	if err != nil {
		return fmt.Errorf("aktivasi: tulis log: %w", err)
	}
	return nil
}

// Batas tulis log: kegagalan di kode yang sama ditulis paling banyak maxLogGagalPerKode per 24 jam supaya penyerang
// tidak membengkakkan tabel. Penghitung atomik tetap jalan; hanya baris log yang dilewati.
const maxLogGagalPerKode = 100

// bolehTulisLogGagal: false bila kode ini sudah punya maxLogGagalPerKode baris kegagalan dalam 24 jam terakhir.
func bolehTulisLogGagal(ctx context.Context, q rowQueryer, kodeHash string) bool {
	if kodeHash == "" {
		return true
	}
	var n int
	err := q.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM jamaah_aktivasi_log
		WHERE kode_hash = ? AND aksi IN ('cek_gagal','kode_dikunci') AND dibuat_pada >= DATE_SUB(NOW(), INTERVAL 24 HOUR)`, kodeHash).Scan(&n)
	return err != nil || n < maxLogGagalPerKode
}

type rowQueryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Masa simpan: penghitung dibuang setelah kedaluwarsa_at; log kegagalan 90 hari; jejak aktivasi dan
// lepas_admin 2 tahun.
const (
	simpanLogGagalHari = 90
	simpanLogJejakHari = 730
	batasHapusPerBatch = 5000
)

// HasilBersih: jumlah baris yang dibuang pembersih.
type HasilBersih struct{ Hitung, LogGagal, LogJejak int64 }

// BersihkanAktivasi membuang penghitung kedaluwarsa dan log lama, bertahap supaya tidak mengunci tabel lama.
func BersihkanAktivasi(ctx context.Context, db *sql.DB) (HasilBersih, error) {
	var h HasilBersih
	var err error
	if h.Hitung, err = hapusBertahap(ctx, db, `DELETE FROM jamaah_aktivasi_hitung WHERE kedaluwarsa_at < NOW() LIMIT ?`); err != nil {
		return h, err
	}
	if h.LogGagal, err = hapusBertahap(ctx, db, fmt.Sprintf(`DELETE FROM jamaah_aktivasi_log
		WHERE aksi IN ('cek_gagal','cek_ok','kode_dikunci') AND dibuat_pada < DATE_SUB(NOW(), INTERVAL %d DAY) LIMIT ?`, simpanLogGagalHari)); err != nil {
		return h, err
	}
	if h.LogJejak, err = hapusBertahap(ctx, db, fmt.Sprintf(`DELETE FROM jamaah_aktivasi_log
		WHERE dibuat_pada < DATE_SUB(NOW(), INTERVAL %d DAY) LIMIT ?`, simpanLogJejakHari)); err != nil {
		return h, err
	}
	return h, nil
}

func hapusBertahap(ctx context.Context, db *sql.DB, q string) (int64, error) {
	var total int64
	for {
		res, err := db.ExecContext(ctx, q, batasHapusPerBatch)
		if err != nil {
			return total, fmt.Errorf("aktivasi: bersihkan: %w", err)
		}
		n, _ := res.RowsAffected()
		total += n
		if n < batasHapusPerBatch {
			return total, nil
		}
	}
}
