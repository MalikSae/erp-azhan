package shared

import (
	"html"
	"regexp"
	"strings"
)

// Teks berformat dari editor admin (S&K agen, MoU agen) disimpan sebagai HTML terbatas. Hanya tag di bawah yang
// dipertahankan, tanpa atribut apa pun; tag lain dibuang dan isi <script>/<style> dihapus. Hasilnya aman
// ditampilkan di app dan web karena tidak ada atribut (onclick, href, style) atau tag aktif yang lolos.
var tagHTMLAman = map[string]bool{
	"p": true, "br": true, "h2": true, "h3": true, "strong": true, "b": true, "em": true, "i": true,
	"u": true, "ul": true, "ol": true, "li": true, "blockquote": true,
}

var (
	polaTagHTML     = regexp.MustCompile(`<(/?)([a-zA-Z][a-zA-Z0-9]*)\b[^>]*>`)
	polaBlokBerbaya = regexp.MustCompile(`(?is)<(script|style|iframe|object|embed)\b.*?</(script|style|iframe|object|embed)\s*>`)
	polaKomentar    = regexp.MustCompile(`(?s)<!--.*?-->`)
)

// SanitasiHTML merapikan HTML dari editor admin ke daftar tag aman. Teks di luar tag di-escape ulang supaya tanda
// "<" yang bukan tag tidak menjadi markup.
func SanitasiHTML(masukan string) string {
	s := polaKomentar.ReplaceAllString(masukan, "")
	s = polaBlokBerbaya.ReplaceAllString(s, "")
	var out strings.Builder
	sisa := s
	for {
		loc := polaTagHTML.FindStringSubmatchIndex(sisa)
		if loc == nil {
			out.WriteString(escapeTeks(sisa))
			break
		}
		out.WriteString(escapeTeks(sisa[:loc[0]]))
		tutup := sisa[loc[2]:loc[3]] == "/"
		nama := strings.ToLower(sisa[loc[4]:loc[5]])
		if tagHTMLAman[nama] {
			switch {
			case nama == "br":
				out.WriteString("<br>")
			case tutup:
				out.WriteString("</" + nama + ">")
			default:
				out.WriteString("<" + nama + ">")
			}
		}
		sisa = sisa[loc[1]:]
	}
	return strings.TrimSpace(out.String())
}

// escapeTeks meng-escape teks biasa tanpa menggandakan entitas yang sudah ada (editor mengirim &amp;, &lt;, dll.).
func escapeTeks(t string) string {
	return html.EscapeString(html.UnescapeString(t))
}

// HTMLKosong melaporkan HTML yang tidak berisi teks (mis. "<p></p>" dari editor kosong).
func HTMLKosong(s string) bool {
	return strings.TrimSpace(html.UnescapeString(polaTagHTML.ReplaceAllString(s, ""))) == ""
}
