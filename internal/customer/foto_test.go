package customer

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

const fotoA = "/api/admin/media/dokumen-jamaah/aaaaaaaa-0000-0000-0000-000000000001.jpg"
const fotoB = "/api/admin/media/dokumen-jamaah/aaaaaaaa-0000-0000-0000-000000000002.png"

func fotoAkun(t *testing.T, tx *sql.Tx, id int64) sql.NullString {
	t.Helper()
	var f sql.NullString
	must(t, "baca foto", tx.QueryRow(`SELECT foto_url FROM customer_accounts WHERE id=?`, id).Scan(&f))
	return f
}

func TestSimpanFotoAkunHanyaBerkasMilikSendiri(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PemilikFoto"})
	b := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunLain"})
	testdb.Exec(t, tx, `INSERT INTO media_uploads (path, brand_id, customer_account_id) VALUES (?, ?, ?)`, fotoA, brand, a)
	testdb.Exec(t, tx, `INSERT INTO media_uploads (path, brand_id, customer_account_id) VALUES (?, ?, ?)`, fotoB, brand, b)

	must(t, "simpan foto sendiri", simpanFotoAkun(ctx, tx, a, fotoA))
	if f := fotoAkun(t, tx, a); f.String != fotoA {
		t.Fatalf("foto = %v", f)
	}
	for nama, url := range map[string]string{
		"milik akun lain":     fotoB,
		"url karangan":        "/api/admin/media/dokumen-jamaah/00000000-0000-0000-0000-000000000000.jpg",
		"bukan gambar":        "/api/admin/media/dokumen-jamaah/aaaaaaaa-0000-0000-0000-000000000001.pdf",
		"bukan berkas portal": "https://contoh.test/foto.jpg",
	} {
		if err := simpanFotoAkun(ctx, tx, a, url); !errors.Is(err, ErrFotoTidakValid) {
			t.Errorf("%s: err = %v, want ErrFotoTidakValid", nama, err)
		}
	}
	if f := fotoAkun(t, tx, a); f.String != fotoA {
		t.Fatalf("foto tidak boleh berubah saat ditolak: %v", f)
	}

	// Kosong = hapus foto.
	must(t, "hapus foto", simpanFotoAkun(ctx, tx, a, ""))
	if f := fotoAkun(t, tx, a); f.Valid {
		t.Fatalf("foto harus NULL setelah dihapus: %v", f)
	}
}

func TestBerkasFotoDariURL(t *testing.T) {
	if p, ok := berkasFoto(fotoA); !ok || p != "uploads/dokumen-jamaah/aaaaaaaa-0000-0000-0000-000000000001.jpg" {
		t.Fatalf("berkasFoto = %q %v", p, ok)
	}
	for _, bad := range []string{"", "/etc/passwd", "/api/admin/media/dokumen-jamaah/../../x.jpg", "/api/admin/media/payment-proofs/aaaaaaaa-0000-0000-0000-000000000001.jpg"} {
		if _, ok := berkasFoto(bad); ok {
			t.Errorf("berkasFoto(%q) harus ditolak", bad)
		}
	}
}
