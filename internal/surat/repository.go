package surat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrPengaturanBelumLengkap: kode surat atau penandatangan brand belum diisi di dashboard.
var ErrPengaturanBelumLengkap = errors.New("pengaturan surat brand belum lengkap")

// Pengaturan adalah isian surat per brand (tabel brand_surat). Nilai kosong berarti belum diisi.
type Pengaturan struct {
	BrandID              uint64 `json:"brand_id"`
	NamaPerusahaan       string `json:"nama_perusahaan"`
	KodeSurat            string `json:"kode_surat"`
	PenandatanganNama    string `json:"penandatangan_nama"`
	PenandatanganJabatan string `json:"penandatangan_jabatan"`
	KopURL               string `json:"kop_url"`
	TandaTanganURL       string `json:"tanda_tangan_url"`
	StempelURL           string `json:"stempel_url"`
	NomorTerakhir        uint32 `json:"nomor_terakhir"`
}

// Lengkap: surat hanya bisa dibuat bila kode surat dan nama penandatangan ada (nomor dan tanda tangan butuh keduanya).
func (p Pengaturan) Lengkap() bool {
	return strings.TrimSpace(p.KodeSurat) != "" && strings.TrimSpace(p.PenandatanganNama) != ""
}

// Terbitan adalah surat yang sudah bernomor untuk satu booking, jamaah, dan jenis.
type Terbitan struct {
	Nomor    string
	TerbitAt time.Time
}

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func ns(v sql.NullString) string { return strings.TrimSpace(v.String) }

// Pengaturan membaca isian surat brand; brand yang belum punya baris mengembalikan isian kosong.
func (r *Repository) Pengaturan(ctx context.Context, brandID uint64) (Pengaturan, error) {
	p := Pengaturan{BrandID: brandID}
	var nama, kode, ttdNama, jabatan, kop, ttd, stempel sql.NullString
	err := r.db.QueryRowContext(ctx, `SELECT nama_perusahaan, kode_surat, penandatangan_nama, penandatangan_jabatan, kop_url, tanda_tangan_url, stempel_url, nomor_terakhir FROM brand_surat WHERE brand_id=?`, brandID).
		Scan(&nama, &kode, &ttdNama, &jabatan, &kop, &ttd, &stempel, &p.NomorTerakhir)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if err != nil {
		return p, fmt.Errorf("surat.Pengaturan: %w", err)
	}
	p.NamaPerusahaan, p.KodeSurat, p.PenandatanganNama, p.PenandatanganJabatan = ns(nama), ns(kode), ns(ttdNama), ns(jabatan)
	p.KopURL, p.TandaTanganURL, p.StempelURL = ns(kop), ns(ttd), ns(stempel)
	return p, nil
}

func kosongNull(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.TrimSpace(s)
}

// SimpanPengaturan menyimpan isian surat brand (dibuat bila belum ada).
func (r *Repository) SimpanPengaturan(ctx context.Context, p Pengaturan) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO brand_surat (brand_id, nama_perusahaan, kode_surat, penandatangan_nama, penandatangan_jabatan, kop_url, tanda_tangan_url, stempel_url, nomor_terakhir)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE nama_perusahaan=VALUES(nama_perusahaan), kode_surat=VALUES(kode_surat), penandatangan_nama=VALUES(penandatangan_nama),
			penandatangan_jabatan=VALUES(penandatangan_jabatan), kop_url=VALUES(kop_url), tanda_tangan_url=VALUES(tanda_tangan_url),
			stempel_url=VALUES(stempel_url), nomor_terakhir=VALUES(nomor_terakhir)`,
		p.BrandID, kosongNull(p.NamaPerusahaan), kosongNull(p.KodeSurat), kosongNull(p.PenandatanganNama), kosongNull(p.PenandatanganJabatan),
		kosongNull(p.KopURL), kosongNull(p.TandaTanganURL), kosongNull(p.StempelURL), p.NomorTerakhir)
	if err != nil {
		return fmt.Errorf("surat.SimpanPengaturan: %w", err)
	}
	return nil
}

// Terbit mengembalikan surat bernomor untuk booking, jamaah, dan jenis. Unduhan pertama mengambil nomor urut berikutnya milik brand
// (dikunci dalam transaksi) dan menyimpannya; unduhan berikutnya memakai nomor dan tanggal terbit yang sama dan hanya menambah hitungan.
func (r *Repository) Terbit(ctx context.Context, brandID uint64, bookingID, jamaahID int64, jenis string, sekarang time.Time) (Terbitan, error) {
	if KodeJenis(jenis) == "" {
		return Terbitan{}, fmt.Errorf("surat.Terbit: jenis %q tidak dikenal", jenis)
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Terbitan{}, fmt.Errorf("surat.Terbit: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var t Terbitan
	err = tx.QueryRowContext(ctx, `SELECT nomor, terbit_at FROM surat_jamaah WHERE booking_id=? AND jamaah_id=? AND jenis=? FOR UPDATE`, bookingID, jamaahID, jenis).Scan(&t.Nomor, &t.TerbitAt)
	switch {
	case err == nil:
		if _, err := tx.ExecContext(ctx, `UPDATE surat_jamaah SET jumlah_unduh=jumlah_unduh+1, unduh_terakhir_at=? WHERE booking_id=? AND jamaah_id=? AND jenis=?`, sekarang, bookingID, jamaahID, jenis); err != nil {
			return Terbitan{}, fmt.Errorf("surat.Terbit hitung: %w", err)
		}
		return t, tx.Commit()
	case !errors.Is(err, sql.ErrNoRows):
		return Terbitan{}, fmt.Errorf("surat.Terbit baca: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO brand_surat (brand_id) VALUES (?)`, brandID); err != nil {
		return Terbitan{}, fmt.Errorf("surat.Terbit brand: %w", err)
	}
	var kode sql.NullString
	var penandatangan sql.NullString
	var terakhir uint32
	if err := tx.QueryRowContext(ctx, `SELECT kode_surat, penandatangan_nama, nomor_terakhir FROM brand_surat WHERE brand_id=? FOR UPDATE`, brandID).Scan(&kode, &penandatangan, &terakhir); err != nil {
		return Terbitan{}, fmt.Errorf("surat.Terbit kunci: %w", err)
	}
	if ns(kode) == "" || ns(penandatangan) == "" {
		return Terbitan{}, ErrPengaturanBelumLengkap
	}
	urut := terakhir + 1
	nomor := FormatNomor(urut, ns(kode), jenis, sekarang)
	if _, err := tx.ExecContext(ctx, `INSERT INTO surat_jamaah (booking_id, jamaah_id, jenis, nomor_urut, nomor, terbit_at, unduh_terakhir_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		bookingID, jamaahID, jenis, urut, nomor, sekarang, sekarang); err != nil {
		return Terbitan{}, fmt.Errorf("surat.Terbit simpan: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE brand_surat SET nomor_terakhir=? WHERE brand_id=?`, urut, brandID); err != nil {
		return Terbitan{}, fmt.Errorf("surat.Terbit urut: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Terbitan{}, fmt.Errorf("surat.Terbit commit: %w", err)
	}
	return Terbitan{Nomor: nomor, TerbitAt: sekarang}, nil
}

// Nomor yang sudah terbit untuk booking (jenis dan jamaah), untuk ditampilkan di daftar surat. Kunci: "jamaahID|jenis".
func (r *Repository) NomorBooking(ctx context.Context, bookingID int64) (map[string]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT jamaah_id, jenis, nomor FROM surat_jamaah WHERE booking_id=?`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("surat.NomorBooking: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var jid int64
		var jenis, nomor string
		if err := rows.Scan(&jid, &jenis, &nomor); err != nil {
			return nil, fmt.Errorf("surat.NomorBooking scan: %w", err)
		}
		out[fmt.Sprintf("%d|%s", jid, jenis)] = nomor
	}
	return out, rows.Err()
}
