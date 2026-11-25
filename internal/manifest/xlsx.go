package manifest

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

// kolom adalah satu kolom tabel pax di sebuah sheet.
type kolom struct {
	judul string
	lebar float64
	isi   func(no int, p Pax) string
}

// sheetManifest adalah satu sheet: nama, kolom, dan baris judul grup (judul di atas beberapa kolom).
type sheetManifest struct {
	nama  string
	kolom []kolom
}

func kolNo() kolom           { return kolom{"NO", 5, func(no int, _ Pax) string { return strconv.Itoa(no) }} }
func kolNama(j string) kolom { return kolom{j, 34, func(_ int, p Pax) string { return namaPaspor(p) }} }
func kolLP(j string) kolom   { return kolom{j, 8, func(_ int, p Pax) string { return p.JenisKelamin }} }
func kolKotaLahir() kolom {
	return kolom{"BIRTH CITY", 18, func(_ int, p Pax) string { return strings.ToUpper(p.TempatLahir) }}
}
func kolTglLahir() kolom {
	return kolom{"BIRTH DATE", 14, func(_ int, p Pax) string { return tanggalManifest(p.TanggalLahir) }}
}
func kolNoPaspor() kolom {
	return kolom{"PASSPORT NO", 14, func(_ int, p Pax) string { return strings.ToUpper(p.NoPaspor) }}
}
func kolTerbit() kolom {
	return kolom{"DATE OF ISSUE", 14, func(_ int, p Pax) string { return tanggalManifest(p.PasporTerbit) }}
}
func kolHabis() kolom {
	return kolom{"DATE OF EXPIRE", 14, func(_ int, p Pax) string { return tanggalManifest(p.PasporHabis) }}
}
func kolKantor(j string) kolom {
	return kolom{j, 20, func(_ int, p Pax) string { return strings.ToUpper(p.KantorPaspor) }}
}

// kolKosong: kolom yang belum diisi ERP di Fase 1 (relasi, keterangan seat, mahram); admin melengkapi di Excel.
func kolKosong(j string, lebar float64) kolom {
	return kolom{j, lebar, func(int, Pax) string { return "" }}
}

func daftarSheet(m *Manifest) []sheetManifest {
	blokSeat := "BLOK SEAT"
	if kata := strings.Fields(strings.ToUpper(m.Paket.Maskapai)); len(kata) > 0 {
		blokSeat += " " + kata[0]
	}
	identitas := []kolom{kolNo(), kolNama("NAME OF PASSPORT"), kolLP("L/P"), kolKotaLahir(), kolTglLahir(),
		kolNoPaspor(), kolTerbit(), kolHabis(), kolKantor("ISSUE OF CITY")}
	salin := func(tambah ...kolom) []kolom { return append(append([]kolom{}, identitas...), tambah...) }
	return []sheetManifest{
		{potongNamaSheet(blokSeat), salin(kolKosong("RELATION", 14), kolKosong("KETERANGAN", 24))},
		{"MANIFEST", salin(kolKosong("RELATION", 14),
			kolom{"JENIS TIKET", 12, func(_ int, p Pax) string { return p.JenisTiket }})},
		{"VISA", []kolom{kolNo(), kolNoPaspor(), kolNama("FULL NAME"), kolLP("GENDER"), kolKosong("MAHRAM", 30),
			kolKosong("RELATION", 14), kolTglLahir(), kolKotaLahir(), kolKantor("ISSUING OFFICE"), kolTerbit(), kolHabis()}},
		{"ASURANSI", []kolom{kolNo(), kolNama("NAME OF PASSPORT"), kolLP("L/P"),
			kolom{"USIA", 7, func(_ int, p Pax) string {
				if p.Usia == nil {
					return ""
				}
				return strconv.Itoa(*p.Usia)
			}},
			kolom{"NIK", 20, func(_ int, p Pax) string { return p.NIK }},
			kolKotaLahir(), kolTglLahir(), kolNoPaspor(), kolTerbit(), kolHabis(), kolKantor("ISSUE OF CITY")}},
	}
}

func potongNamaSheet(s string) string {
	if r := []rune(s); len(r) > 31 {
		return string(r[:31])
	}
	return s
}

// judulManifest: "MANIFEST GROUP 14 SEPTEMBER 2026 <NAMA PAKET>".
func judulManifest(p Paket) string {
	bagian := []string{"MANIFEST GROUP"}
	if t, ok := parseTanggal(p.BerangkatTanggal); ok {
		bagian = append(bagian, fmt.Sprintf("%d %s %d", t.Day(), strings.ToUpper(t.Month().String()), t.Year()))
	}
	bagian = append(bagian, strings.ToUpper(strings.TrimSpace(p.Nama)))
	return strings.Join(bagian, " ")
}

func namaMaskapai(p Paket) string {
	n := strings.ToUpper(strings.TrimSpace(p.Maskapai))
	if p.KodeMaskapai != "" {
		n += " (" + strings.ToUpper(p.KodeMaskapai) + ")"
	}
	return n
}

var karakterNamaFile = regexp.MustCompile(`[^A-Za-z0-9 ._()*+-]+`)

// NamaFile: "<tanggal berangkat> <nama paket>.xlsx", aman untuk header unduhan.
func NamaFile(p Paket) string {
	tgl := ""
	if t, ok := parseTanggal(p.BerangkatTanggal); ok {
		tgl = fmt.Sprintf("%d %s %d ", t.Day(), bulanInggris[t.Month()-1], t.Year())
	}
	nama := strings.Join(strings.Fields(karakterNamaFile.ReplaceAllString(strings.ToUpper(p.Nama), " ")), " ")
	if nama == "" {
		nama = "MANIFEST"
	}
	return tgl + nama + ".xlsx"
}

const (
	lebarBlokKanan = 50 // cukup untuk "Flight Detail: " dan satu baris detail penerbangan
	lebarKolomMaks = 45
)

// tulisHeader menulis satu baris header "Label: nilai" (label tebal) dalam satu sel. Baris lanjutan (lanjutan=true)
// memakai label yang sama berwarna putih supaya nilainya sejajar persis dengan baris di atasnya.
func tulisHeader(f *excelize.File, sheet, cell, label, nilai string, lanjutan bool) {
	warna := ""
	if lanjutan {
		warna = "FFFFFF"
	}
	_ = f.SetCellRichText(sheet, cell, []excelize.RichTextRun{
		{Text: label + ": ", Font: &excelize.Font{Bold: true, Family: "Calibri", Size: 11, Color: warna}},
		{Text: nilai, Font: &excelize.Font{Family: "Calibri", Size: 11}},
	})
}

// lebarKolom: lebar bawaan kolom, diperlebar mengikuti isi terpanjang (maks lebarKolomMaks).
func lebarKolom(s sheetManifest, pax []Pax) []float64 {
	out := make([]float64, len(s.kolom))
	for c, k := range s.kolom {
		out[c] = k.lebar
		for r, p := range pax {
			if n := float64(len([]rune(k.isi(r+1, p))) + 2); n > out[c] {
				out[c] = min(n, lebarKolomMaks)
			}
		}
	}
	return out
}

// kolomDariAkhir: kolom awal (1-based) rentang di ujung kanan yang lebar gabungannya mencapai minimal.
func kolomDariAkhir(lebar []float64, minimal float64) int {
	return kolomDariAkhirSebelum(lebar, len(lebar)+1, minimal)
}

// kolomDariAkhirSebelum: kolom awal (1-based) rentang yang berakhir tepat sebelum kolom batas.
func kolomDariAkhirSebelum(lebar []float64, batas int, minimal float64) int {
	total := 0.0
	for c := batas - 2; c >= 0; c-- {
		total += lebar[c]
		if total >= minimal {
			return c + 1
		}
	}
	return 1
}

func sel(kol, baris int) string {
	nama, _ := excelize.CoordinatesToCellName(kol, baris)
	return nama
}

// BuatFile menyusun workbook manifest Fase 1.
func BuatFile(m *Manifest) (*excelize.File, error) {
	f := excelize.NewFile()
	judul, _ := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	garis := []excelize.Border{
		{Type: "left", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1},
		{Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1},
	}
	kepalaTabel, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true}, Border: garis,
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"D9D9D9"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
	})
	isiTabel, _ := f.NewStyle(&excelize.Style{Border: garis, Alignment: &excelize.Alignment{Vertical: "center"},
		NumFmt: 49}) // 49 = teks: NIK dan nomor paspor tidak berubah jadi angka.
	isiTengah, _ := f.NewStyle(&excelize.Style{Border: garis, NumFmt: 49,
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"}})

	penerbangan := barisPenerbangan(m.Paket.DetailPenerbangan)
	header := [][2]string{
		{"Flight Carrier", namaMaskapai(m.Paket)},
		{"Date of Departure", tanggalManifest(m.Paket.BerangkatTanggal)},
		{"Pax Number", fmt.Sprintf("%d PAX", len(m.Pax))},
		{"Package", fmt.Sprintf("%02d DAYS", m.LamaHari)},
	}
	barisKanan := max(len(penerbangan), 2)
	barisTabel := 3 + max(len(header), barisKanan) + 1

	for i, s := range daftarSheet(m) {
		nama := s.nama
		if i == 0 {
			if err := f.SetSheetName("Sheet1", nama); err != nil {
				return nil, err
			}
		} else if _, err := f.NewSheet(nama); err != nil {
			return nil, err
		}
		jumlahKolom := len(s.kolom)
		_ = f.SetCellStr(nama, "A1", judulManifest(m.Paket))
		_ = f.SetCellStyle(nama, "A1", "A1", judul)

		lebar := lebarKolom(s, m.Pax)
		for c, l := range lebar {
			col, _ := excelize.ColumnNumberToName(c + 1)
			_ = f.SetColWidth(nama, col, col, l)
		}

		// Header di atas tabel: satu sel per baris berisi label tebal lalu nilai, rata kiri. Blok kiri di kolom A,
		// blok kanan (PNR, penerbangan) di kolom yang menyisakan ruang cukup sampai ujung tabel.
		kolKanan := max(kolomDariAkhir(lebar, lebarBlokKanan), 4)
		for r, h := range header {
			tulisHeader(f, nama, sel(1, 3+r), h[0], h[1], false)
		}
		tulisHeader(f, nama, sel(kolKanan, 3), "PNR Code", m.Paket.PNR, false)
		if len(penerbangan) == 0 {
			penerbangan = []string{""}
		}
		for r, b := range penerbangan {
			tulisHeader(f, nama, sel(kolKanan, 4+r), "Flight Detail", b, r > 0)
		}

		for c, k := range s.kolom {
			_ = f.SetCellStr(nama, sel(c+1, barisTabel), k.judul)
		}
		_ = f.SetCellStyle(nama, sel(1, barisTabel), sel(jumlahKolom, barisTabel), kepalaTabel)
		_ = f.SetRowHeight(nama, barisTabel, 30)

		for r, p := range m.Pax {
			baris := barisTabel + 1 + r
			for c, k := range s.kolom {
				_ = f.SetCellStr(nama, sel(c+1, baris), k.isi(r+1, p))
			}
			_ = f.SetCellStyle(nama, sel(1, baris), sel(jumlahKolom, baris), isiTabel)
			_ = f.SetCellStyle(nama, sel(1, baris), sel(1, baris), isiTengah)
		}
		_ = f.SetPanes(nama, &excelize.Panes{Freeze: true, YSplit: barisTabel, TopLeftCell: sel(1, barisTabel+1), ActivePane: "bottomLeft"})
	}
	f.SetActiveSheet(0)
	return f, nil
}
