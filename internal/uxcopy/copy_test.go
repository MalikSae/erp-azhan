// Package uxcopy menjaga pesan galat API tetap sesuai docs/ux-copy.md.
// Pesan galat dari handler tampil langsung di dashboard dan aplikasi.
package uxcopy

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Panggilan helper galat dengan pesan literal. Pesan yang disambung (+) dilewati.
// Galat sentinel internal/akses dan konstanta pesannya ikut diperiksa: handler pengguna
// mengirimnya apa adanya lewat err.Error().
var panggilan = regexp.MustCompile(`(?m)(?:[wW]riteError|sendError|writeErrorCode)\(w,[^"\n]*?(?:fmt\.Sprintf\()?"((?:[^"\\\n]|\\.)*)"([^\n]*)`)
var pesanAkses = regexp.MustCompile(`(?m)(?:errors\.New\(|ErrConcurrentMsg = )"((?:[^"\\\n]|\\.)*)"([^\n]*)`)

var larangan = []struct {
	pola  *regexp.Regexp
	saran string
}{
	{regexp.MustCompile(`\b[a-z]+_[a-z_]+\b`), "nama field teknis; pakai istilah pengguna (brand, paket, jamaah)"},
	{regexp.MustCompile(`(?i)\bsilakan\b`), `tanpa "silakan"; langsung kata kerja ("Coba lagi.")`},
	{regexp.MustCompile(`(?i)\bpassword\b`), `pakai "kata sandi"`},
	{regexp.MustCompile(`(?i)terjadi kesalahan`), `sebut masalahnya, mis. "Permintaan gagal diproses. Coba lagi."`},
	{regexp.MustCompile(`(?i)\b(request body|payload|token|rfc3339|jwt|turnstile|logout)\b`), "istilah teknis; pakai bahasa pengguna"},
	{regexp.MustCompile(`(?i)\bmaaf\b`), "tanpa permintaan maaf"},
	{regexp.MustCompile(`[A-Za-z]!`), "tanpa tanda seru"},
	{regexp.MustCompile(`(?i)\b(peserta|penumpang|kursi|pesanan|pemesanan|reservasi|blokir|cicilan|uang muka)\b`), "ikuti istilah di GLOSSARY.md"},
}

func TestPesanGalatMengikutiPanduanUXCopy(t *testing.T) {
	akar := filepath.Join("..")
	var temuan []string
	err := filepath.WalkDir(akar, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		cocok := panggilan.FindAllStringSubmatchIndex(string(b), -1)
		if filepath.Base(filepath.Dir(p)) == "akses" {
			cocok = append(cocok, pesanAkses.FindAllStringSubmatchIndex(string(b), -1)...)
		}
		for _, m := range cocok {
			pesan := string(b)[m[2]:m[3]]
			sisa := string(b)[m[4]:m[5]]
			baris := strings.Count(string(b)[:m[0]], "\n") + 1
			lokasi := p + ":" + itoa(baris) + " " + `"` + pesan + `"`
			if pesan == "" || strings.Contains(sisa, "+") && strings.HasPrefix(strings.TrimSpace(sisa), "+") {
				continue
			}
			if c := pesan[0]; c >= 'a' && c <= 'z' {
				temuan = append(temuan, lokasi+" — awali dengan huruf kapital")
			}
			if !strings.HasSuffix(pesan, ".") && !strings.HasSuffix(pesan, "?") && !strings.HasSuffix(pesan, ":") {
				temuan = append(temuan, lokasi+" — akhiri dengan titik")
			}
			for _, l := range larangan {
				if l.pola.MatchString(pesan) {
					temuan = append(temuan, lokasi+" — "+l.saran)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(temuan) > 0 {
		t.Errorf("%d pelanggaran docs/ux-copy.md:\n%s", len(temuan), strings.Join(temuan, "\n"))
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
