package media

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	_ "golang.org/x/image/webp"
	_ "image/gif" // dekoder sumber

	"golang.org/x/image/draw"
)

// Varian gambar publik: /uploads/{category}/{file}?w=400 (juga 600, 800, 1200) menyajikan salinan selebar 400 px. Dibuat sekali saat
// pertama diminta lalu disimpan di uploads/_varian (folder ini tidak bisa diakses langsung lewat rute publik).
// URL di API tidak berubah; klien yang menambahkan ?w= sesuai ukuran tampil.

var lebarVarian = map[int]struct{}{400: {}, 600: {}, 800: {}, 1200: {}}

// Hanya foto dan poster; logo sudah kecil.
var kategoriVarian = map[string]struct{}{
	"schedule-brosur": {}, "banner-promosi": {}, "haramain": {}, "ulasan-haramain": {}, "hotel-photos": {},
}

const (
	folderVarian = "_varian"
	// Sumber di atas ini disajikan apa adanya: mendekodenya butuh terlalu banyak memori untuk permintaan publik.
	maxPikselSumberVarian = 40_000_000
	kualitasJPEG          = 75
	// Varian yang tidak sampai 15% lebih kecil dari asli tidak dipakai (poster WebP sering sudah efisien).
	rasioVarianMaks = 0.85
)

var (
	errTanpaVarian = errors.New("tanpa varian")
	// Paling banyak dua gambar didekode bersamaan.
	slotVarian  = make(chan struct{}, 2)
	kunciVarian sync.Map // path varian -> *sync.Mutex, agar satu varian tidak dibuat dua kali bersamaan
)

// lebarDiminta membaca ?w=; 0 berarti tanpa varian.
func lebarDiminta(raw string) int {
	w, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	if _, ok := lebarVarian[w]; !ok {
		return 0
	}
	return w
}

// pathVarian mengembalikan berkas varian di disk, membuatnya bila belum ada. errTanpaVarian berarti sajikan
// berkas asli (gambar sudah lebih kecil, format tidak didukung, atau sumber terlalu besar).
func pathVarian(root, category, filename string, lebar int) (string, error) {
	if _, ok := kategoriVarian[category]; !ok {
		return "", errTanpaVarian
	}
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
		return "", errTanpaVarian
	}
	dasar := strings.TrimSuffix(filename, filepath.Ext(filename))
	dir := filepath.Join(root, folderVarian, category)
	for _, e := range []string{".jpg", ".png"} {
		p := filepath.Join(dir, fmt.Sprintf("%s-w%d%s", dasar, lebar, e))
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	// Penanda: sumber tidak perlu varian untuk lebar ini (sudah kecil / tidak bisa didekode).
	penanda := filepath.Join(dir, fmt.Sprintf("%s-w%d.asli", dasar, lebar))
	if _, err := os.Stat(penanda); err == nil {
		return "", errTanpaVarian
	}

	m, _ := kunciVarian.LoadOrStore(penanda, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()
	// Permintaan lain mungkin sudah membuatnya selagi menunggu kunci.
	for _, e := range []string{".jpg", ".png"} {
		p := filepath.Join(dir, fmt.Sprintf("%s-w%d%s", dasar, lebar, e))
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	if _, err := os.Stat(penanda); err == nil {
		return "", errTanpaVarian
	}

	slotVarian <- struct{}{}
	defer func() { <-slotVarian }()

	isi, ekstensi, err := buatVarian(filepath.Join(root, category, filename), lebar)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	if errors.Is(err, errTanpaVarian) {
		_ = os.WriteFile(penanda, nil, 0o644)
		return "", errTanpaVarian
	}
	if err != nil {
		return "", err
	}
	tujuan := filepath.Join(dir, fmt.Sprintf("%s-w%d%s", dasar, lebar, ekstensi))
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return "", err
	}
	if _, err := tmp.Write(isi); err != nil {
		tmp.Close()
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	if err := os.Rename(tmp.Name(), tujuan); err != nil {
		_ = os.Remove(tmp.Name())
		return "", err
	}
	return tujuan, nil
}

// buatVarian mengecilkan gambar ke lebar tertentu. Gambar tanpa transparansi menjadi JPEG, sisanya PNG.
func buatVarian(sumber string, lebar int) ([]byte, string, error) {
	data, err := os.ReadFile(sumber)
	if err != nil {
		return nil, "", err
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", errTanpaVarian
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxPikselSumberVarian || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, "", errTanpaVarian
	}
	orientasi := orientasiEXIF(data)
	w, h := cfg.Width, cfg.Height
	if orientasi >= 5 { // diputar 90°: lebar tampil = tinggi berkas
		w, h = h, w
	}
	if w <= lebar {
		return nil, "", errTanpaVarian
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", errTanpaVarian
	}
	src = terapkanOrientasi(src, orientasi)

	tinggi := max(1, int(float64(h)*float64(lebar)/float64(w)+0.5))
	dst := image.NewNRGBA(image.Rect(0, 0, lebar, tinggi))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Src, nil)

	var buf bytes.Buffer
	ekstensi := ".jpg"
	if dst.Opaque() {
		err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: kualitasJPEG})
	} else {
		ekstensi = ".png"
		err = (&png.Encoder{CompressionLevel: png.BestCompression}).Encode(&buf, dst)
	}
	if err != nil {
		return nil, "", err
	}
	if float64(buf.Len()) > float64(len(data))*rasioVarianMaks {
		return nil, "", errTanpaVarian
	}
	return buf.Bytes(), ekstensi, nil
}

// orientasiEXIF membaca tag Orientation (1-8) dari JPEG; 1 bila tidak ada. Foto ponsel sering disimpan miring
// dengan tag ini, dan browser memutarnya saat tampil, jadi varian harus ikut diputar.
func orientasiEXIF(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}
	i := 2
	for i+4 <= len(data) {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 { // awal data gambar: tidak ada EXIF
			return 1
		}
		size := int(binary.BigEndian.Uint16(data[i+2:]))
		if size < 2 || i+2+size > len(data) {
			return 1
		}
		seg := data[i+4 : i+2+size]
		if marker == 0xE1 && len(seg) > 14 && string(seg[:6]) == "Exif\x00\x00" {
			return orientasiTIFF(seg[6:])
		}
		i += 2 + size
	}
	return 1
}

func orientasiTIFF(t []byte) int {
	if len(t) < 8 {
		return 1
	}
	var bo binary.ByteOrder
	switch string(t[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 1
	}
	ifd := int(bo.Uint32(t[4:]))
	if ifd+2 > len(t) {
		return 1
	}
	n := int(bo.Uint16(t[ifd:]))
	for k := 0; k < n; k++ {
		e := ifd + 2 + k*12
		if e+12 > len(t) {
			return 1
		}
		if bo.Uint16(t[e:]) == 0x0112 {
			v := int(bo.Uint16(t[e+8:]))
			if v >= 1 && v <= 8 {
				return v
			}
			return 1
		}
	}
	return 1
}

// terapkanOrientasi memutar/membalik gambar sesuai tag EXIF agar hasilnya tegak.
func terapkanOrientasi(src image.Image, o int) image.Image {
	if o <= 1 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewNRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var dx, dy int
			switch o {
			case 2:
				dx, dy = w-1-x, y
			case 3:
				dx, dy = w-1-x, h-1-y
			case 4:
				dx, dy = x, h-1-y
			case 5:
				dx, dy = y, x
			case 6:
				dx, dy = h-1-y, x
			case 7:
				dx, dy = h-1-y, w-1-x
			case 8:
				dx, dy = y, w-1-x
			}
			dst.Set(dx, dy, src.At(b.Min.X+x, b.Min.Y+y))
		}
	}
	return dst
}
