package shared

import (
	"errors"
	"image"
	_ "image/gif"  // dekoder untuk DecodeConfig
	_ "image/jpeg" // dekoder untuk DecodeConfig
	_ "image/png"  // dekoder untuk DecodeConfig
	"io"

	_ "golang.org/x/image/webp" // dekoder untuk DecodeConfig
)

// Batas jumlah piksel gambar. Berkas 10 MB bisa mengaku berukuran 40000x40000 piksel; mendekodenya butuh
// sekitar 6 GB memori dan mematikan API untuk semua brand (audit keamanan 2026-11-12).
const (
	// MaxImagePixels berlaku untuk semua unggahan gambar: cukup untuk foto ponsel 48 MP.
	MaxImagePixels = 100_000_000
	// MaxLogoPixels berlaku untuk gambar yang didekode server (logo, kop, stempel, tanda tangan surat PDF).
	MaxLogoPixels = 16_000_000
)

// ErrGambarTerlaluBesar: dimensi gambar melebihi batas piksel atau tidak valid.
var ErrGambarTerlaluBesar = errors.New("dimensi gambar terlalu besar")

// CekDimensiGambar hanya membaca header gambar (tanpa mendekode piksel) dan menolak gambar yang lebar x
// tingginya melebihi maxPixels. Format yang tidak dikenal mengembalikan galat dari image.DecodeConfig.
func CekDimensiGambar(r io.Reader, maxPixels int64) error {
	cfg, _, err := image.DecodeConfig(r)
	if err != nil {
		return err
	}
	if cfg.Width <= 0 || cfg.Height <= 0 || int64(cfg.Width)*int64(cfg.Height) > maxPixels {
		return ErrGambarTerlaluBesar
	}
	return nil
}
