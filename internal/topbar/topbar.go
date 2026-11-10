// Package topbar melayani dua fitur topbar dashboard office: pencarian global
// (paket, booking, jamaah) dan ringkasan pekerjaan yang menunggu keputusan (lonceng).
//
// Kedua rute hanya mensyaratkan login admin. Izin diperiksa per jenis di sini: jenis yang
// tidak diizinkan peran dikembalikan kosong, bukan 403, supaya satu permintaan cukup untuk
// semua peran. Admin yang terikat satu brand hanya melihat data brand-nya.
package topbar

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"unicode/utf8"
)

const (
	// PanjangMinimal adalah jumlah karakter terpendek yang dicari; di bawahnya hasil kosong.
	PanjangMinimal = 2
	// BatasHasil adalah jumlah hasil terbanyak per jenis.
	BatasHasil = 5
	// kata kunci dipotong supaya pola LIKE tidak membengkak.
	panjangMaksimal = 80
)

// Izin yang menentukan jenis hasil dan antrean yang boleh dilihat.
const (
	IzinPaket           = "schedule.view"
	IzinBooking         = "booking.view"
	IzinJamaah          = "jamaah.view"
	IzinPerubahanJamaah = "jamaah.perubahan.approve"
	IzinPencairan       = "agen.payout.approve"
	IzinPengajuanAgen   = "agen.approve"
	IzinPembayaran      = "payment.verify"
)

// Akses adalah hak pemanggil: brand yang mengikatnya (nil = lintas brand) dan izin perannya.
type Akses struct {
	BrandID *int64
	Izin    []string
}

func (a Akses) boleh(izin string) bool {
	for _, p := range a.Izin {
		if p == izin {
			return true
		}
	}
	return false
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type HasilPaket struct {
	ID               int64  `json:"id"`
	Nama             string `json:"nama"`
	BerangkatTanggal string `json:"berangkat_tanggal"`
	Status           string `json:"status"`
	Brand            string `json:"brand"`
}

type HasilBooking struct {
	ID          int64  `json:"id"`
	IDBooking   string `json:"id_booking"`
	NamaPemesan string `json:"nama_pemesan"`
	JadwalNama  string `json:"jadwal_nama"`
	Status      string `json:"status"`
	Brand       string `json:"brand"`
}

// HasilJamaah sengaja tidak memuat NIK dan nomor paspor: keduanya bisa dicari, tetapi tidak
// ditampilkan di panel hasil.
type HasilJamaah struct {
	ID          int64  `json:"id"`
	NamaLengkap string `json:"nama_lengkap"`
	IDJamaah    string `json:"id_jamaah"`
	NoHP        string `json:"no_hp"`
	Brand       string `json:"brand"`
}

type HasilCari struct {
	Paket   []HasilPaket   `json:"paket"`
	Booking []HasilBooking `json:"booking"`
	Jamaah  []HasilJamaah  `json:"jamaah"`
}

// polaLike membentuk pola "mengandung" dan menetralkan karakter khusus LIKE pada kata kunci.
func polaLike(kata string) string {
	kata = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(kata)
	return "%" + kata + "%"
}

// Cari mengembalikan paling banyak BatasHasil per jenis yang mengandung kata kunci.
func Cari(ctx context.Context, q querier, a Akses, kata string) (HasilCari, error) {
	h := HasilCari{Paket: []HasilPaket{}, Booking: []HasilBooking{}, Jamaah: []HasilJamaah{}}
	kata = strings.TrimSpace(kata)
	if utf8.RuneCountInString(kata) < PanjangMinimal {
		return h, nil
	}
	if r := []rune(kata); len(r) > panjangMaksimal {
		kata = string(r[:panjangMaksimal])
	}
	pola := polaLike(kata)

	if a.boleh(IzinPaket) {
		if err := cariPaket(ctx, q, a.BrandID, pola, &h.Paket); err != nil {
			return h, err
		}
	}
	if a.boleh(IzinBooking) {
		if err := cariBooking(ctx, q, a.BrandID, pola, &h.Booking); err != nil {
			return h, err
		}
	}
	if a.boleh(IzinJamaah) {
		if err := cariJamaah(ctx, q, a.BrandID, pola, &h.Jamaah); err != nil {
			return h, err
		}
	}
	return h, nil
}

func cariPaket(ctx context.Context, q querier, brandID *int64, pola string, out *[]HasilPaket) error {
	query := `
		SELECT s.id, s.jadwal_nama, COALESCE(DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d'), ''), s.status, br.name
		FROM schedules s JOIN brands br ON br.id = s.brand_id
		WHERE s.jadwal_nama LIKE ?`
	args := []any{pola}
	if brandID != nil {
		query += ` AND s.brand_id = ?`
		args = append(args, *brandID)
	}
	// Keberangkatan mendatang lebih dulu, yang terdekat di atas; sesudahnya yang sudah lewat, terbaru dulu.
	query += ` ORDER BY (s.berangkat_tanggal >= CURDATE()) DESC, ABS(DATEDIFF(s.berangkat_tanggal, CURDATE())) ASC, s.id DESC LIMIT ?`
	args = append(args, BatasHasil)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("topbar: cari paket: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var p HasilPaket
		if err := rows.Scan(&p.ID, &p.Nama, &p.BerangkatTanggal, &p.Status, &p.Brand); err != nil {
			return fmt.Errorf("topbar: cari paket scan: %w", err)
		}
		*out = append(*out, p)
	}
	return rows.Err()
}

func cariBooking(ctx context.Context, q querier, brandID *int64, pola string, out *[]HasilBooking) error {
	query := `
		SELECT b.id, COALESCE(b.id_booking, ''), COALESCE(j.nama_lengkap, ''), s.jadwal_nama, b.status, br.name
		FROM bookings b
		JOIN schedules s ON s.id = b.schedule_id
		JOIN brands br ON br.id = s.brand_id
		LEFT JOIN jamaah j ON j.id = b.pic_jamaah_id
		WHERE (b.id_booking LIKE ? OR j.nama_lengkap LIKE ?)`
	args := []any{pola, pola}
	if brandID != nil {
		query += ` AND s.brand_id = ?`
		args = append(args, *brandID)
	}
	query += ` ORDER BY b.created_at DESC, b.id DESC LIMIT ?`
	args = append(args, BatasHasil)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("topbar: cari booking: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var b HasilBooking
		if err := rows.Scan(&b.ID, &b.IDBooking, &b.NamaPemesan, &b.JadwalNama, &b.Status, &b.Brand); err != nil {
			return fmt.Errorf("topbar: cari booking scan: %w", err)
		}
		*out = append(*out, b)
	}
	return rows.Err()
}

func cariJamaah(ctx context.Context, q querier, brandID *int64, pola string, out *[]HasilJamaah) error {
	query := `
		SELECT j.id, j.nama_lengkap, COALESCE(j.id_jamaah, ''), COALESCE(j.no_hp, ''), br.name
		FROM jamaah j JOIN brands br ON br.id = j.brand_id
		WHERE (j.nama_lengkap LIKE ? OR j.id_jamaah LIKE ? OR j.kode_jamaah LIKE ?
		       OR j.no_hp LIKE ? OR j.nik LIKE ? OR j.no_paspor LIKE ?)`
	args := []any{pola, pola, pola, pola, pola, pola}
	if brandID != nil {
		query += ` AND j.brand_id = ?`
		args = append(args, *brandID)
	}
	query += ` ORDER BY j.created_at DESC, j.id DESC LIMIT ?`
	args = append(args, BatasHasil)
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("topbar: cari jamaah: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var j HasilJamaah
		if err := rows.Scan(&j.ID, &j.NamaLengkap, &j.IDJamaah, &j.NoHP, &j.Brand); err != nil {
			return fmt.Errorf("topbar: cari jamaah scan: %w", err)
		}
		*out = append(*out, j)
	}
	return rows.Err()
}

// Jenis antrean di lonceng.
const (
	JenisPerubahanJamaah = "perubahan_jamaah"
	JenisPencairan       = "pencairan_komisi"
	JenisPengajuanAgen   = "pengajuan_agen"
	JenisPembayaran      = "pembayaran"
)

// Antrean adalah satu jenis pekerjaan yang menunggu keputusan beserta jumlahnya.
type Antrean struct {
	Jenis  string `json:"jenis"`
	Jumlah int    `json:"jumlah"`
}

type Notifikasi struct {
	Antrean []Antrean `json:"antrean"`
	Total   int       `json:"total"`
}

// Syarat tiap antrean mengikuti halaman tujuannya, supaya angka di lonceng sama dengan isi
// daftar yang dibuka: perubahan jamaah (customer.ListAdmin), pencairan (agen.listPencairanAdmin),
// pengajuan agen (agen.listPengajuan), pembayaran (payment.ListAll).
var kueriAntrean = []struct {
	jenis, izin, sql, kolomBrand string
}{
	{JenisPerubahanJamaah, IzinPerubahanJamaah, `
		SELECT COUNT(*) FROM jamaah_perubahan p JOIN jamaah j ON j.id = p.jamaah_id
		WHERE p.status = 'menunggu'`, "j.brand_id"},
	{JenisPencairan, IzinPencairan, `
		SELECT COUNT(*) FROM pengajuan_pencairan p
		JOIN customer_accounts j ON j.id = p.akun_agen_id
		JOIN brands b ON b.id = p.brand_id
		WHERE p.status = 'pending'`, "p.brand_id"},
	{JenisPengajuanAgen, IzinPengajuanAgen, `
		SELECT COUNT(*) FROM customer_accounts j
		WHERE (j.status_agen = 'pengajuan'
		       OR EXISTS (
		         SELECT 1 FROM pembayaran_pendaftaran_agen p
		         WHERE p.id = (SELECT MAX(p2.id) FROM pembayaran_pendaftaran_agen p2 WHERE p2.akun_id = j.id)
		           AND p.keputusan_agen = 'disetujui' AND p.status NOT IN ('terverifikasi','dibebaskan') AND p.nominal_tagihan > 0))`, "j.brand_id"},
	{JenisPembayaran, IzinPembayaran, `
		SELECT COUNT(*) FROM payments p
		JOIN bookings b ON b.id = p.booking_id
		JOIN jamaah j ON j.id = b.pic_jamaah_id
		JOIN schedules s ON s.id = b.schedule_id
		WHERE b.status != 'draft' AND p.status = 'pending'`, "s.brand_id"},
}

// HitungNotifikasi menghitung antrean yang boleh diproses pemanggil. Antrean tanpa izin tidak
// ikut dikembalikan; antrean berizin tetap dikembalikan walau jumlahnya nol.
func HitungNotifikasi(ctx context.Context, q querier, a Akses) (Notifikasi, error) {
	n := Notifikasi{Antrean: []Antrean{}}
	for _, k := range kueriAntrean {
		if !a.boleh(k.izin) {
			continue
		}
		query, args := k.sql, []any{}
		if a.BrandID != nil {
			query += ` AND ` + k.kolomBrand + ` = ?`
			args = append(args, *a.BrandID)
		}
		var jumlah int
		if err := q.QueryRowContext(ctx, query, args...).Scan(&jumlah); err != nil {
			return n, fmt.Errorf("topbar: hitung %s: %w", k.jenis, err)
		}
		n.Antrean = append(n.Antrean, Antrean{Jenis: k.jenis, Jumlah: jumlah})
		n.Total += jumlah
	}
	return n, nil
}
