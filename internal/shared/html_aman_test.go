package shared

import "testing"

func TestSanitasiHTML(t *testing.T) {
	cases := []struct{ nama, masuk, want string }{
		{"format editor dipertahankan", `<h2>Pasal 1</h2><p><strong>Agen</strong> wajib <em>jujur</em>.</p><ol><li>Satu</li></ol>`,
			`<h2>Pasal 1</h2><p><strong>Agen</strong> wajib <em>jujur</em>.</p><ol><li>Satu</li></ol>`},
		{"atribut dibuang", `<p class="x" onclick="alert(1)" style="color:red">Hai</p>`, `<p>Hai</p>`},
		{"tag tidak dikenal dibuang, isinya tetap", `<a href="javascript:alert(1)">klik</a><div>isi</div>`, `klikisi`},
		{"script dan isinya dihapus", `<p>a</p><script>alert(1)</script><style>p{}</style>`, `<p>a</p>`},
		{"teks biasa di-escape", `1 < 2 & 3 > 2`, `1 &lt; 2 &amp; 3 &gt; 2`},
		{"entitas tidak digandakan", `<p>A &amp; B</p>`, `<p>A &amp; B</p>`},
		{"br", `baris<br/>baru<BR>`, `baris<br>baru<br>`},
		{"komentar dibuang", `<!-- x --><p>a</p>`, `<p>a</p>`},
		{"teks lama tanpa tag", "1. Agen wajib.\n2. Ujroh dicatat.", "1. Agen wajib.\n2. Ujroh dicatat."},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			if got := SanitasiHTML(c.masuk); got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestHTMLKosong(t *testing.T) {
	if !HTMLKosong("<p></p>") || !HTMLKosong(" <p>  </p><br> ") {
		t.Fatal("editor kosong harus terdeteksi kosong")
	}
	if HTMLKosong("<p>a</p>") {
		t.Fatal("berisi teks tidak kosong")
	}
}
