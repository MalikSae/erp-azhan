package customer

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// Audit keamanan 2026-11-26 (Q1): daftar dengan sandi tidak memverifikasi email. Bila akun itu kemudian ditautkan ke
// akun Google dengan email yang sama, sandi pendaftar sebelumnya (bisa orang lain) tidak boleh lagi berlaku.
func TestTautanGoogleMencabutSandiAkunBelumTerverifikasi(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	s := time.Now().UnixNano()
	email := fmt.Sprintf("tautan-google-%d@example.com", s)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE s FROM customer_sessions s JOIN customer_accounts a ON a.id = s.account_id WHERE a.email_normal = ?`, email)
		_, _ = db.Exec(`DELETE t FROM customer_tokens t JOIN customer_accounts a ON a.id = t.account_id WHERE a.email_normal = ?`, email)
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal = ?`, email)
	})
	h := NewHandler(NewRepository(db), &captureMail{})

	// Pendaftar pertama memakai email itu dengan sandinya sendiri, lalu punya sesi dan tautan reset aktif.
	rec := post(h.RegisterApp, map[string]any{"brand_id": brandID, "nama_lengkap": "Pendaftar Awal", "email": email,
		"no_hp": fmt.Sprintf("0816%08d", s%100000000), "password": "sandi-pendaftar-1"}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("daftar: %d %s", rec.Code, rec.Body)
	}
	var akunID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM customer_accounts WHERE email_normal = ?`, email).Scan(&akunID); err != nil {
		t.Fatal(err)
	}
	if _, err := h.repo.IssueToken(ctx, akunID, PurposeResetPass, time.Hour); err != nil {
		t.Fatal(err)
	}

	// Pemilik email masuk dengan Google: akun ditautkan.
	h.google = googlePalsu{&ProfilGoogle{Sub: fmt.Sprintf("sub-tautan-%d", s), Email: email, Nama: "Pemilik Email"}}
	rec = post(h.LoginGoogle, map[string]any{"brand_id": brandID, "id_token": "t"}, "")
	if rec.Code != http.StatusOK || lengkapiDari(rec.Body.Bytes()) {
		t.Fatalf("masuk google: %d %s", rec.Code, rec.Body)
	}

	// Sandi lama tidak berlaku lagi.
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "sandi-pendaftar-1"}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("sandi pendaftar awal masih berlaku setelah ditautkan ke Google: %d %s", rec.Code, rec.Body)
	}
	var sesiAktif, tokenAktif int
	var terverifikasi bool
	if err := db.QueryRowContext(ctx, `
		SELECT (SELECT COUNT(*) FROM customer_sessions WHERE account_id = ? AND revoked_at IS NULL),
		       (SELECT COUNT(*) FROM customer_tokens WHERE account_id = ? AND used_at IS NULL),
		       email_verified_at IS NOT NULL
		FROM customer_accounts WHERE id = ?`, akunID, akunID, akunID).Scan(&sesiAktif, &tokenAktif, &terverifikasi); err != nil {
		t.Fatal(err)
	}
	// Satu sesi aktif: sesi Google yang baru dibuat. Sesi pendaftar awal dan tautan reset sudah dicabut.
	if sesiAktif != 1 || tokenAktif != 0 || !terverifikasi {
		t.Fatalf("sesi aktif=%d (harus 1), token aktif=%d (harus 0), terverifikasi=%v", sesiAktif, tokenAktif, terverifikasi)
	}

	// Masuk Google berikutnya (akun sudah terverifikasi) tidak mencabut apa pun lagi; sandi baru bisa dibuat.
	rec = post(h.LoginGoogle, map[string]any{"brand_id": brandID, "id_token": "t"}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("masuk google kedua: %d %s", rec.Code, rec.Body)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM customer_sessions WHERE account_id = ? AND revoked_at IS NULL`, akunID).Scan(&sesiAktif); err != nil || sesiAktif != 2 {
		t.Fatalf("masuk google kedua mencabut sesi: aktif=%d %v", sesiAktif, err)
	}
}
