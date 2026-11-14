package customer

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

func TestBiodataDaftar(t *testing.T) {
	sekarang := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		nama              string
		jk, dom, lahir    string
		wantJK, wantPesan string
	}{
		{"kosong semua lolos", "", "", "", "", ""},
		{"jenis kelamin dirapikan", " p ", " Bandung ", "1990-05-01", "P", ""},
		{"jenis kelamin salah", "X", "", "", "", "jenis kelamin harus L atau P"},
		{"tanggal masa depan", "L", "", "2030-01-01", "", "tanggal lahir tidak valid"},
		{"tanggal salah format", "L", "", "01/05/1990", "", "tanggal lahir tidak valid"},
		{"domisili terlalu panjang", "L", string(make([]rune, 101)), "", "", "kota domisili maksimal 100 karakter"},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			bio, pesan := biodataDaftar(c.jk, c.dom, c.lahir, sekarang)
			if pesan != c.wantPesan {
				t.Fatalf("pesan = %q, want %q", pesan, c.wantPesan)
			}
			if pesan == "" && bio.JenisKelamin != c.wantJK {
				t.Fatalf("jenis kelamin = %q, want %q", bio.JenisKelamin, c.wantJK)
			}
		})
	}
}

// Daftar dari app: tanpa captcha walau captcha aktif, dan biodata opsional tersimpan di akun.
func TestRegisterAppTanpaCaptchaMenyimpanBiodata(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("app-uji-%d@example.com", suffix)
	hp := fmt.Sprintf("0814%08d", suffix%100000000)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal = ?`, email) })

	t.Setenv("TURNSTILE_SECRET_KEY", "rahasia-uji")
	h := NewHandler(NewRepository(db), &captureMail{})

	rec := post(h.RegisterApp, map[string]any{
		"brand_id": brandID, "nama_lengkap": "Pengguna App", "email": email, "no_hp": hp, "password": "rahasia-123",
		"jenis_kelamin": "p", "domisili": "Bandung", "tanggal_lahir": "1990-05-01",
	}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register app: %d %s", rec.Code, rec.Body)
	}
	var jk, dom, lahir sql.NullString
	if err := db.QueryRowContext(ctx,
		`SELECT jenis_kelamin, domisili, DATE_FORMAT(tanggal_lahir, '%Y-%m-%d') FROM customer_accounts WHERE email_normal = ?`, email,
	).Scan(&jk, &dom, &lahir); err != nil {
		t.Fatal(err)
	}
	if jk.String != "P" || dom.String != "Bandung" || lahir.String != "1990-05-01" {
		t.Fatalf("biodata = %q %q %q", jk.String, dom.String, lahir.String)
	}

	rec = post(h.RegisterApp, map[string]any{
		"brand_id": brandID, "nama_lengkap": "Pengguna App", "email": "dua-" + email, "no_hp": hp + "9", "password": "rahasia-123",
		"jenis_kelamin": "X",
	}, "")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("jenis kelamin salah: %d %s", rec.Code, rec.Body)
	}
}
