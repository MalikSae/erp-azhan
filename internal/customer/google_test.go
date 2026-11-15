package customer

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"erp-azhan/api/internal/testdb"
)

const clientUji = "uji.apps.googleusercontent.com"

func tokenUji(t *testing.T, kunci *rsa.PrivateKey, kid string, ubah func(jwt.MapClaims)) string {
	t.Helper()
	klaim := jwt.MapClaims{
		"iss": "https://accounts.google.com", "aud": clientUji, "sub": "1234567890",
		"email": "pengguna@gmail.com", "email_verified": true, "name": "Pengguna Google",
		"exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(),
	}
	if ubah != nil {
		ubah(klaim)
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, klaim)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(kunci)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestVerifikasiTokenGoogle(t *testing.T) {
	kunci, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	lain, _ := rsa.GenerateKey(rand.Reader, 2048)
	v := &verifikatorGoogle{clientIDs: []string{clientUji},
		kunci: map[string]*rsa.PublicKey{"k1": &kunci.PublicKey}, sampai: time.Now().Add(time.Hour)}
	ctx := context.Background()

	p, err := v.Verifikasi(ctx, tokenUji(t, kunci, "k1", nil))
	if err != nil || p.Sub != "1234567890" || p.Email != "pengguna@gmail.com" || p.Nama != "Pengguna Google" {
		t.Fatalf("token sah: %+v %v", p, err)
	}
	for nama, tok := range map[string]string{
		"audience lain":      tokenUji(t, kunci, "k1", func(m jwt.MapClaims) { m["aud"] = "app-lain" }),
		"penerbit lain":      tokenUji(t, kunci, "k1", func(m jwt.MapClaims) { m["iss"] = "https://contoh.id" }),
		"kedaluwarsa":        tokenUji(t, kunci, "k1", func(m jwt.MapClaims) { m["exp"] = time.Now().Add(-time.Hour).Unix() }),
		"email belum verif":  tokenUji(t, kunci, "k1", func(m jwt.MapClaims) { m["email_verified"] = false }),
		"tanda tangan palsu": tokenUji(t, lain, "k1", nil),
	} {
		if _, err := v.Verifikasi(ctx, tok); !errors.Is(err, ErrTokenGoogle) {
			t.Errorf("%s: harus ditolak (%v)", nama, err)
		}
	}
	if _, err := (&verifikatorGoogle{}).Verifikasi(ctx, "x"); !errors.Is(err, ErrGoogleTidakDikonfigurasi) {
		t.Errorf("tanpa client id: %v", err)
	}
}

type googlePalsu struct{ p *ProfilGoogle }

func (g googlePalsu) Verifikasi(context.Context, string) (*ProfilGoogle, error) {
	if g.p == nil {
		return nil, ErrTokenGoogle
	}
	return g.p, nil
}

func TestMasukDanDaftarGoogle(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().UnixNano()
	emailBaru := fmt.Sprintf("google-baru-%d@example.com", suffix)
	emailLama := fmt.Sprintf("google-lama-%d@example.com", suffix)
	hp := fmt.Sprintf("0815%08d", suffix%100000000)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE s FROM customer_sessions s JOIN customer_accounts a ON a.id = s.account_id WHERE a.email_normal IN (?, ?)`, emailBaru, emailLama)
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal IN (?, ?)`, emailBaru, emailLama)
	})
	h := NewHandler(NewRepository(db), &captureMail{})
	sub := fmt.Sprintf("sub-%d", suffix)
	h.google = googlePalsu{&ProfilGoogle{Sub: sub, Email: emailBaru, Nama: "Pengguna Baru"}}

	// Belum terdaftar: perlu lengkapi data.
	rec := post(h.LoginGoogle, map[string]any{"brand_id": brandID, "id_token": "t"}, "")
	var lengkapi googleLengkapi
	_ = json.Unmarshal(rec.Body.Bytes(), &lengkapi)
	if rec.Code != http.StatusOK || !lengkapi.PerluLengkapi || lengkapi.Email != emailBaru || lengkapi.NamaLengkap != "Pengguna Baru" {
		t.Fatalf("login akun baru: %d %s", rec.Code, rec.Body)
	}
	// Lengkapi data: WA wajib.
	if rec = post(h.DaftarGoogle, map[string]any{"brand_id": brandID, "id_token": "t"}, ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("daftar tanpa WA: %d %s", rec.Code, rec.Body)
	}
	rec = post(h.DaftarGoogle, map[string]any{"brand_id": brandID, "id_token": "t", "no_hp": hp,
		"jenis_kelamin": "L", "domisili": "Bandung", "tanggal_lahir": "1990-05-01"}, "")
	var auth struct {
		Account      Account `json:"account"`
		SessionToken string  `json:"session_token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &auth)
	if rec.Code != http.StatusCreated || auth.SessionToken == "" || auth.Account.PunyaPassword || auth.Account.Email != emailBaru {
		t.Fatalf("daftar google: %d %s", rec.Code, rec.Body)
	}
	// Masuk lagi: langsung sesi.
	rec = post(h.LoginGoogle, map[string]any{"brand_id": brandID, "id_token": "t"}, "")
	if rec.Code != http.StatusOK || !json.Valid(rec.Body.Bytes()) || lengkapiDari(rec.Body.Bytes()) {
		t.Fatalf("login akun google: %d %s", rec.Code, rec.Body)
	}
	// Login email/password gagal untuk akun tanpa sandi.
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": emailBaru, "password": ""}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("login tanpa sandi: %d", rec.Code)
	}
	// Buat sandi sekali, lalu bisa login dengan email.
	buat := func(w http.ResponseWriter, r *http.Request) { h.RequireCustomer(http.HandlerFunc(h.BuatPassword)).ServeHTTP(w, r) }
	if rec = post(buat, map[string]any{"password_baru": "rahasia-123"}, auth.SessionToken); rec.Code != http.StatusOK {
		t.Fatalf("buat sandi: %d %s", rec.Code, rec.Body)
	}
	if rec = post(buat, map[string]any{"password_baru": "rahasia-456"}, auth.SessionToken); rec.Code != http.StatusConflict {
		t.Fatalf("buat sandi kedua: %d %s", rec.Code, rec.Body)
	}
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": emailBaru, "password": "rahasia-123"}, ""); rec.Code != http.StatusOK {
		t.Fatalf("login setelah buat sandi: %d %s", rec.Code, rec.Body)
	}

	// Akun lama (email/password) dengan email yang sama: masuk dan tertaut otomatis.
	if rec = post(h.RegisterApp, map[string]any{"brand_id": brandID, "nama_lengkap": "Akun Lama", "email": emailLama,
		"no_hp": hp + "1", "password": "rahasia-123"}, ""); rec.Code != http.StatusCreated {
		t.Fatalf("daftar akun lama: %d %s", rec.Code, rec.Body)
	}
	subLama := sub + "-lama"
	h.google = googlePalsu{&ProfilGoogle{Sub: subLama, Email: emailLama, Nama: "Akun Lama"}}
	if rec = post(h.LoginGoogle, map[string]any{"brand_id": brandID, "id_token": "t"}, ""); rec.Code != http.StatusOK || lengkapiDari(rec.Body.Bytes()) {
		t.Fatalf("login google akun lama: %d %s", rec.Code, rec.Body)
	}
	var tertaut string
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(google_sub, '') FROM customer_accounts WHERE email_normal = ?`, emailLama).Scan(&tertaut); err != nil || tertaut != subLama {
		t.Fatalf("tautan akun lama: %q %v", tertaut, err)
	}
	// Akun Google lain dengan email yang sama ditolak.
	h.google = googlePalsu{&ProfilGoogle{Sub: sub + "-penyusup", Email: emailLama, Nama: "X"}}
	if rec = post(h.LoginGoogle, map[string]any{"brand_id": brandID, "id_token": "t"}, ""); rec.Code != http.StatusConflict {
		t.Fatalf("akun google lain: %d %s", rec.Code, rec.Body)
	}
	// Token tidak sah.
	h.google = googlePalsu{}
	if rec = post(h.LoginGoogle, map[string]any{"brand_id": brandID, "id_token": "t"}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token tidak sah: %d %s", rec.Code, rec.Body)
	}
}

func lengkapiDari(b []byte) bool {
	var l googleLengkapi
	_ = json.Unmarshal(b, &l)
	return l.PerluLengkapi
}
