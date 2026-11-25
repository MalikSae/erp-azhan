package manifest

import (
	"strings"
	"time"
)

const formatISO = "2006-01-02"

var bulanInggris = [...]string{"JAN", "FEB", "MAR", "APR", "MAY", "JUN", "JUL", "AUG", "SEP", "OCT", "NOV", "DEC"}

func parseTanggal(s string) (time.Time, bool) {
	if len(s) < 10 {
		return time.Time{}, false
	}
	t, err := time.Parse(formatISO, s[:10])
	return t, err == nil
}

// tanggalManifest memformat tanggal ISO menjadi "14-SEP-2026"; kosong bila tidak valid.
func tanggalManifest(s string) string {
	t, ok := parseTanggal(s)
	if !ok {
		return ""
	}
	return t.Format("02") + "-" + bulanInggris[t.Month()-1] + "-" + t.Format("2006")
}

// usiaPada menghitung usia penuh (tahun) pada tanggal acuan.
func usiaPada(lahir, acuan time.Time) int {
	u := acuan.Year() - lahir.Year()
	if acuan.Month() < lahir.Month() || (acuan.Month() == lahir.Month() && acuan.Day() < lahir.Day()) {
		u--
	}
	return u
}

// jenisTiket mengikuti kategori maskapai (Saudia: infant di bawah 2 tahun, child 2 sampai di bawah 12),
// dihitung pada tanggal berangkat. Tanpa tanggal lahir: infant bila pax infant, selain itu adult.
func jenisTiket(usia *int, tipePax string) string {
	if usia == nil {
		if tipePax == "infant" {
			return "INFANT"
		}
		return "ADULT"
	}
	switch {
	case *usia < 2:
		return "INFANT"
	case *usia < 12:
		return "CHILD"
	default:
		return "ADULT"
	}
}

// lamaHari menghitung lama paket, hari berangkat dan pulang ikut dihitung.
func lamaHari(berangkat, pulang string) int {
	b, ok1 := parseTanggal(berangkat)
	p, ok2 := parseTanggal(pulang)
	if !ok1 || !ok2 || p.Before(b) {
		return 0
	}
	return int(p.Sub(b).Hours()/24) + 1
}

// periksaPax mengisi usia, jenis tiket, dan daftar masalah satu pax.
func periksaPax(p *Pax, berangkat, pulang string) {
	p.Masalah = []string{}
	tglBerangkat, adaBerangkat := parseTanggal(berangkat)
	if lahir, ok := parseTanggal(p.TanggalLahir); ok && adaBerangkat {
		u := usiaPada(lahir, tglBerangkat)
		p.Usia = &u
	}
	p.JenisTiket = jenisTiket(p.Usia, p.TipePax)

	if strings.TrimSpace(p.NoPaspor) == "" {
		p.Masalah = append(p.Masalah, "Nomor paspor kosong")
	}
	habis, adaHabis := parseTanggal(p.PasporHabis)
	if !adaHabis {
		p.Masalah = append(p.Masalah, "Masa berlaku paspor kosong")
	} else {
		if terbit, ok := parseTanggal(p.PasporTerbit); ok && !habis.After(terbit) {
			p.Masalah = append(p.Masalah, "Tanggal paspor tidak sesuai")
		}
		if tglPulang, ok := parseTanggal(pulang); ok && habis.Before(tglPulang.AddDate(0, 6, 0)) {
			p.Masalah = append(p.Masalah, "Paspor habis kurang dari 6 bulan")
		}
	}
	if p.JenisKelamin == "" {
		p.Masalah = append(p.Masalah, "Jenis kelamin kosong")
	}
	if p.TanggalLahir == "" {
		p.Masalah = append(p.Masalah, "Tanggal lahir kosong")
	}
}

// namaPaspor memakai nama paspor bila ada, selain itu nama lengkap; huruf besar.
func namaPaspor(p Pax) string {
	n := strings.TrimSpace(p.NamaPaspor)
	if n == "" {
		n = p.Nama
	}
	return strings.ToUpper(strings.Join(strings.Fields(n), " "))
}

// rapikan menormalkan isian PNR dan detail penerbangan, lalu memvalidasinya.
func (in *InputInfo) rapikan() error {
	in.PNR = strings.ToUpper(strings.TrimSpace(in.PNR))
	baris := strings.Split(strings.ReplaceAll(in.DetailPenerbangan, "\r\n", "\n"), "\n")
	bersih := make([]string, 0, len(baris))
	for _, b := range baris {
		if b = strings.Join(strings.Fields(b), " "); b != "" {
			bersih = append(bersih, b)
		}
	}
	in.DetailPenerbangan = strings.Join(bersih, "\n")
	if len([]rune(in.PNR)) > maksPNR {
		return &ErrValidasi{"PNR terlalu panjang."}
	}
	if len([]rune(in.DetailPenerbangan)) > maksDetail {
		return &ErrValidasi{"Detail penerbangan terlalu panjang."}
	}
	return nil
}

// barisPenerbangan memecah detail penerbangan per baris (kosong = tidak ada baris).
func barisPenerbangan(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
