package customer

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// Audit keamanan 2026-11-26 (Q6): login ke akun tanpa sandi (Google) dulu menjawab jauh lebih cepat daripada email tak
// dikenal, karena bcrypt langsung gagal pada hash kosong. Kini keduanya menjalankan bcrypt penuh (cost 12, puluhan ms).
func TestLoginAkunTanpaSandiTetapMenjalankanBcrypt(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	s := time.Now().UnixNano()
	email := fmt.Sprintf("tanpa-sandi-%d@example.com", s)
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal = ?`, email) })
	h := NewHandler(NewRepository(db), &captureMail{})
	if _, err := h.repo.CreateGoogle(ctx, brandID, "Akun Google", email, fmt.Sprintf("0817%08d", s%100000000), fmt.Sprintf("sub-%d", s), "", Biodata{}); err != nil {
		t.Fatal(err)
	}

	mulai := time.Now()
	rec := post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "apa-saja-123"}, "")
	lama := time.Since(mulai)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login akun tanpa sandi: %d %s", rec.Code, rec.Body)
	}
	// bcrypt cost 12 memakan ratusan ms; perbandingan dengan hash tidak sah selesai dalam mikrodetik.
	if lama < 20*time.Millisecond {
		t.Fatalf("login akun tanpa sandi selesai dalam %v: bcrypt tidak dijalankan, waktu jawab membedakan akun Google", lama)
	}
}
