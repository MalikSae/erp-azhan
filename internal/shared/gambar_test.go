package shared

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/png"
	"testing"
)

// pngHeader membuat awal berkas PNG (signature + IHDR sah) yang mengaku berukuran w x h piksel.
func pngHeader(w, h uint32) []byte {
	var buf bytes.Buffer
	buf.Write([]byte("\x89PNG\r\n\x1a\n"))
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], w)
	binary.BigEndian.PutUint32(ihdr[4:], h)
	ihdr[8], ihdr[9] = 8, 6 // 8 bit, RGBA
	binary.Write(&buf, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	buf.Write(chunk)
	binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return buf.Bytes()
}

// Regresi audit keamanan 2026-11-12: gambar kecil yang mengaku 40000x40000 piksel ditolak tanpa didekode.
func TestCekDimensiGambarMenolakBomDekompresi(t *testing.T) {
	if err := CekDimensiGambar(bytes.NewReader(pngHeader(40000, 40000)), MaxImagePixels); !errors.Is(err, ErrGambarTerlaluBesar) {
		t.Fatalf("40000x40000: err=%v, want ErrGambarTerlaluBesar", err)
	}
	if err := CekDimensiGambar(bytes.NewReader(pngHeader(5000, 4000)), MaxLogoPixels); !errors.Is(err, ErrGambarTerlaluBesar) {
		t.Fatalf("logo 20 MP: err=%v, want ErrGambarTerlaluBesar", err)
	}
	var asli bytes.Buffer
	if err := png.Encode(&asli, image.NewRGBA(image.Rect(0, 0, 800, 600))); err != nil {
		t.Fatal(err)
	}
	if err := CekDimensiGambar(bytes.NewReader(asli.Bytes()), MaxLogoPixels); err != nil {
		t.Fatalf("gambar 800x600 harus lolos: %v", err)
	}
	if err := CekDimensiGambar(bytes.NewReader([]byte("bukan gambar")), MaxImagePixels); err == nil {
		t.Fatal("bukan gambar harus ditolak")
	}
}
