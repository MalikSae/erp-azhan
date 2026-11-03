package customer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// K2 dan K22: pendaftaran lewat form agen tanpa captcha dan tanpa email verifikasi; penolakan email atau nomor
// yang sudah ada dijawab satu pesan yang sama (bukan oracle).
func TestRegisterAgenTanpaCaptchaDanTanpaEmail(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	suffix := time.Now().UnixNano()
	email := fmt.Sprintf("agen-uji-%d@example.com", suffix)
	hp := fmt.Sprintf("0812%08d", suffix%100000000)
	hp2 := fmt.Sprintf("0813%08d", suffix%100000000)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal IN (?, ?)`, email, "dua-"+email) })

	t.Setenv("TURNSTILE_SECRET_KEY", "rahasia-uji") // captcha aktif: Register biasa tanpa token pasti ditolak
	mail := &captureMail{}
	h := NewHandler(NewRepository(db), mail)

	rec := post(h.RegisterAgen, map[string]any{"brand_id": brandID, "nama_lengkap": "Calon Agen", "email": email, "no_hp": hp, "password": "rahasia-123"}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register agen: %d %s", rec.Code, rec.Body)
	}
	var reg authResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	if reg.SessionToken == "" {
		t.Fatalf("harus memberi sesi: %+v", reg)
	}
	if _, kirim := mail.last(email); kirim {
		t.Fatal("pendaftaran agen tidak boleh mengirim email verifikasi")
	}
	if rec = post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Biasa", "email": "dua-" + email, "no_hp": hp2, "password": "rahasia-123"}, ""); rec.Code == http.StatusCreated {
		t.Fatal("Register biasa tetap wajib captcha")
	}

	emailSama := post(h.RegisterAgen, map[string]any{"brand_id": brandID, "nama_lengkap": "Dua", "email": email, "no_hp": hp2, "password": "rahasia-123"}, "")
	hpSama := post(h.RegisterAgen, map[string]any{"brand_id": brandID, "nama_lengkap": "Tiga", "email": "dua-" + email, "no_hp": hp, "password": "rahasia-123"}, "")
	if emailSama.Code != http.StatusConflict || hpSama.Code != http.StatusConflict {
		t.Fatalf("duplikat harus 409: email %d, nomor %d", emailSama.Code, hpSama.Code)
	}
	if emailSama.Body.String() != hpSama.Body.String() {
		t.Fatalf("pesan email dan nomor harus sama: %q vs %q", emailSama.Body, hpSama.Body)
	}
}
