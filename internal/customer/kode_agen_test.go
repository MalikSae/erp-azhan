package customer

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

type fixtureKodeAgen struct {
	db        *sql.DB
	brandID   int64
	brandLain int64
	sfx       string
	n         int
}

func newFixtureKodeAgen(t *testing.T) *fixtureKodeAgen {
	t.Helper()
	db := testdb.DB(t)
	f := &fixtureKodeAgen{db: db, sfx: fmt.Sprintf("%d", time.Now().UnixNano())}
	rows, err := db.QueryContext(context.Background(), `SELECT id FROM brands ORDER BY id LIMIT 2`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		t.Fatal("tidak ada brand")
	}
	f.brandID = ids[0]
	if len(ids) > 1 {
		f.brandLain = ids[1]
	}
	t.Cleanup(func() {
		pola := "kodeagen-%-" + f.sfx + "@example.com"
		_, _ = db.Exec(`DELETE s FROM customer_sessions s JOIN customer_accounts a ON a.id = s.account_id WHERE a.email_normal LIKE ?`, pola)
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal LIKE ? AND direkrut_oleh_akun_id IS NOT NULL`, pola)
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal LIKE ?`, pola)
	})
	return f
}

func (f *fixtureKodeAgen) email(k string) string {
	return fmt.Sprintf("kodeagen-%s-%s@example.com", k, f.sfx)
}

func (f *fixtureKodeAgen) hp() string {
	f.n++
	return fmt.Sprintf("0818%04d%04d", f.n, time.Now().UnixNano()%10000)
}

func (f *fixtureKodeAgen) agen(t *testing.T, brandID int64, k, kode, status string) int64 {
	t.Helper()
	res, err := f.db.Exec(`INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash, status_agen, kode_referral)
		VALUES (?,?,?,?,?,'x',?,?)`, brandID, "Agen "+k, f.email(k), f.email(k), f.hp(), status, kode)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func (f *fixtureKodeAgen) kode(p string) string {
	return strings.ToUpper(p + f.sfx[len(f.sfx)-8:])
}

// upline membaca direkrut_oleh_akun_id akun; ada=false bila akun tidak dibuat.
func (f *fixtureKodeAgen) upline(t *testing.T, k string) (id int64, ada bool) {
	t.Helper()
	var d sql.NullInt64
	err := f.db.QueryRow(`SELECT direkrut_oleh_akun_id FROM customer_accounts WHERE email_normal = ?`, f.email(k)).Scan(&d)
	if err == sql.ErrNoRows {
		return 0, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return d.Int64, true
}

func (f *fixtureKodeAgen) body(k string, tambah map[string]any) map[string]any {
	b := map[string]any{"brand_id": f.brandID, "nama_lengkap": "Akun Uji", "email": f.email(k), "no_hp": f.hp(), "password": "rahasia-123"}
	for key, v := range tambah {
		b[key] = v
	}
	return b
}

func TestRegisterAppKodeAgenKetik(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	f := newFixtureKodeAgen(t)
	a := f.agen(t, f.brandID, "a", f.kode("A"), "aktif")
	b := f.agen(t, f.brandID, "b", f.kode("B"), "aktif")
	f.agen(t, f.brandID, "off", f.kode("O"), "nonaktif")
	var lain string
	if f.brandLain > 0 {
		lain = f.kode("L")
		f.agen(t, f.brandLain, "lain", lain, "aktif")
	}
	h := NewHandler(NewRepository(f.db), &captureMail{})

	t.Run("kode ketik menang atas kode referral otomatis", func(t *testing.T) {
		rec := post(h.RegisterApp, f.body("menang", map[string]any{"kode_agen": strings.ToLower(f.kode("A")), "kode_referral": f.kode("B")}), "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		if got, _ := f.upline(t, "menang"); got != a {
			t.Fatalf("upline = %d, want %d", got, a)
		}
	})

	t.Run("kode ketik kosong memakai kode otomatis", func(t *testing.T) {
		for i, kosong := range []string{"", "   "} {
			k := fmt.Sprintf("otomatis%d", i)
			rec := post(h.RegisterApp, f.body(k, map[string]any{"kode_agen": kosong, "kode_referral": f.kode("B")}), "")
			if rec.Code != http.StatusCreated {
				t.Fatalf("%d %s", rec.Code, rec.Body)
			}
			if got, _ := f.upline(t, k); got != b {
				t.Fatalf("upline = %d, want %d", got, b)
			}
		}
	})

	t.Run("kode otomatis tidak valid diabaikan diam-diam", func(t *testing.T) {
		rec := post(h.RegisterApp, f.body("otomatissalah", map[string]any{"kode_referral": "TIDAKADA999"}), "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		if got, ada := f.upline(t, "otomatissalah"); !ada || got != 0 {
			t.Fatalf("upline = %d ada=%v, want akun tanpa upline", got, ada)
		}
	})

	t.Run("kode ketik salah ditahan dan akun tidak dibuat", func(t *testing.T) {
		kasus := map[string]string{"tidakada": "TIDAKADA999", "nonaktif": f.kode("O")}
		if lain != "" {
			kasus["brandlain"] = lain
		}
		for nama, kode := range kasus {
			k := "salah" + nama
			// kode_referral otomatis yang valid tidak boleh menyelamatkan kode ketik yang salah.
			rec := post(h.RegisterApp, f.body(k, map[string]any{"kode_agen": kode, "kode_referral": f.kode("A")}), "")
			if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"Kode agen tidak ditemukan"`) {
				t.Fatalf("%s: %d %s", nama, rec.Code, rec.Body)
			}
			if strings.Contains(rec.Body.String(), "session_token") {
				t.Fatalf("%s: sesi tercipta: %s", nama, rec.Body)
			}
			if _, ada := f.upline(t, k); ada {
				t.Fatalf("%s: akun tercipta", nama)
			}
		}
	})
}

func TestRegisterWebDanAgenMengabaikanKodeAgen(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	f := newFixtureKodeAgen(t)
	f.agen(t, f.brandID, "a", f.kode("A"), "aktif")
	b := f.agen(t, f.brandID, "b", f.kode("B"), "aktif")
	h := NewHandler(NewRepository(f.db), &captureMail{})

	for nama, fn := range map[string]http.HandlerFunc{"web": h.Register, "agen": h.RegisterAgen} {
		// kode_agen valid tidak mengikat; kode_agen salah tidak menahan.
		rec := post(fn, f.body("valid"+nama, map[string]any{"kode_agen": f.kode("A"), "kode_referral": f.kode("B")}), "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s: %d %s", nama, rec.Code, rec.Body)
		}
		if got, _ := f.upline(t, "valid"+nama); got != b {
			t.Fatalf("%s: upline = %d, want %d (kode_referral, bukan kode_agen)", nama, got, b)
		}
		rec = post(fn, f.body("salah"+nama, map[string]any{"kode_agen": "TIDAKADA999"}), "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("%s kode_agen salah: %d %s", nama, rec.Code, rec.Body)
		}
		if got, ada := f.upline(t, "salah"+nama); !ada || got != 0 {
			t.Fatalf("%s: upline = %d ada=%v, want tanpa upline", nama, got, ada)
		}
	}
}

func TestDaftarGoogleKodeAgenKetik(t *testing.T) {
	f := newFixtureKodeAgen(t)
	a := f.agen(t, f.brandID, "a", f.kode("A"), "aktif")
	b := f.agen(t, f.brandID, "b", f.kode("B"), "aktif")
	f.agen(t, f.brandID, "off", f.kode("O"), "nonaktif")
	h := NewHandler(NewRepository(f.db), &captureMail{})

	daftar := func(k string, tambah map[string]any) (int, string) {
		h.google = googlePalsu{&ProfilGoogle{Sub: "sub-" + k + f.sfx, Email: f.email(k), Nama: "Pengguna Google"}}
		body := map[string]any{"brand_id": f.brandID, "id_token": "t", "no_hp": f.hp()}
		for key, v := range tambah {
			body[key] = v
		}
		rec := post(h.DaftarGoogle, body, "")
		return rec.Code, rec.Body.String()
	}

	if code, body := daftar("menang", map[string]any{"kode_agen": f.kode("A"), "kode_referral": f.kode("B")}); code != http.StatusCreated {
		t.Fatalf("menang: %d %s", code, body)
	}
	if got, _ := f.upline(t, "menang"); got != a {
		t.Fatalf("menang: upline = %d, want %d", got, a)
	}

	if code, body := daftar("otomatis", map[string]any{"kode_referral": f.kode("B")}); code != http.StatusCreated {
		t.Fatalf("otomatis: %d %s", code, body)
	}
	if got, _ := f.upline(t, "otomatis"); got != b {
		t.Fatalf("otomatis: upline = %d, want %d", got, b)
	}

	if code, body := daftar("otomatissalah", map[string]any{"kode_referral": "TIDAKADA999"}); code != http.StatusCreated {
		t.Fatalf("otomatis salah: %d %s", code, body)
	}
	if got, ada := f.upline(t, "otomatissalah"); !ada || got != 0 {
		t.Fatalf("otomatis salah: upline = %d ada=%v", got, ada)
	}

	for nama, kode := range map[string]string{"tidakada": "TIDAKADA999", "nonaktif": f.kode("O")} {
		code, body := daftar("salah"+nama, map[string]any{"kode_agen": kode, "kode_referral": f.kode("A")})
		if code != http.StatusBadRequest || !strings.Contains(body, `"Kode agen tidak ditemukan"`) {
			t.Fatalf("%s: %d %s", nama, code, body)
		}
		if _, ada := f.upline(t, "salah"+nama); ada {
			t.Fatalf("%s: akun tercipta", nama)
		}
	}
}
