package identity

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
	"github.com/google/uuid"
)

func adminUji(t *testing.T, tx *sql.Tx) int64 {
	t.Helper()
	res, err := tx.Exec(`INSERT INTO admin_users (email, password_hash) VALUES (?, 'x')`,
		"rotasi-"+uuid.NewString()[:8]+"@uji.test")
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	return id
}

func tokenUji(t *testing.T, tx *sql.Tx, userID int64) string {
	t.Helper()
	jti := uuid.NewString()
	testdb.Exec(t, tx, `INSERT INTO admin_refresh_tokens (jti, admin_user_id, expires_at) VALUES (?, ?, ?)`,
		jti, userID, time.Now().Add(time.Hour).UTC())
	return jti
}

type statusToken struct {
	dicabut, diganti bool
}

func cekToken(t *testing.T, tx *sql.Tx, jti string) (statusToken, bool) {
	t.Helper()
	var s statusToken
	err := tx.QueryRow(`SELECT revoked_at IS NOT NULL, diganti_at IS NOT NULL FROM admin_refresh_tokens WHERE jti = ?`, jti).
		Scan(&s.dicabut, &s.diganti)
	if errors.Is(err, sql.ErrNoRows) {
		return s, false
	}
	if err != nil {
		t.Fatal(err)
	}
	return s, true
}

// Temuan audit: refresh token admin dulu bisa dipakai berulang 7 hari. Sekarang setiap refresh mengganti token.
func TestRotasiRefreshToken(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour)
	user := adminUji(t, tx)

	lama := tokenUji(t, tx, user)
	baru := uuid.NewString()
	if err := rotasiRefreshTx(ctx, tx, lama, user, baru, exp); err != nil {
		t.Fatalf("rotasi pertama: %v", err)
	}
	if s, _ := cekToken(t, tx, lama); !s.dicabut || !s.diganti {
		t.Fatalf("token lama harus dicabut karena rotasi, dapat %+v", s)
	}
	if s, ada := cekToken(t, tx, baru); !ada || s.dicabut {
		t.Fatalf("token baru harus tercatat aktif (ada=%v, %+v)", ada, s)
	}

	// Tab lain memakai token lama dalam masa tenggang: tetap dapat token baru, token lain tidak dicabut.
	baru2 := uuid.NewString()
	if err := rotasiRefreshTx(ctx, tx, lama, user, baru2, exp); err != nil {
		t.Fatalf("rotasi dalam masa tenggang: %v", err)
	}
	if s, _ := cekToken(t, tx, baru); s.dicabut {
		t.Fatal("masa tenggang tidak boleh mencabut token tab lain")
	}

	// Token lama dipakai lagi setelah masa tenggang: dianggap bocor, semua token user dicabut.
	testdb.Exec(t, tx, `UPDATE admin_refresh_tokens SET diganti_at = NOW() - INTERVAL 5 MINUTE WHERE jti = ?`, lama)
	err := rotasiRefreshTx(ctx, tx, lama, user, uuid.NewString(), exp)
	if !errors.Is(err, ErrRefreshDipakaiUlang) {
		t.Fatalf("pakai ulang setelah tenggang: err=%v, want ErrRefreshDipakaiUlang", err)
	}
	for _, jti := range []string{baru, baru2} {
		if s, _ := cekToken(t, tx, jti); !s.dicabut {
			t.Fatalf("token %s harus ikut dicabut saat pakai ulang terdeteksi", jti)
		}
	}
}

func TestRotasiRefreshTokenMenolakTokenTidakBerlaku(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	exp := time.Now().Add(time.Hour)
	user := adminUji(t, tx)
	lain := adminUji(t, tx)

	logout := tokenUji(t, tx, user)
	testdb.Exec(t, tx, `UPDATE admin_refresh_tokens SET revoked_at = NOW() WHERE jti = ?`, logout)
	kedaluwarsa := tokenUji(t, tx, user)
	testdb.Exec(t, tx, `UPDATE admin_refresh_tokens SET expires_at = NOW() - INTERVAL 1 MINUTE WHERE jti = ?`, kedaluwarsa)
	milikLain := tokenUji(t, tx, lain)

	kasus := map[string]string{
		"dicabut logout":  logout,
		"kedaluwarsa":     kedaluwarsa,
		"milik user lain": milikLain,
		"tidak tercatat":  uuid.NewString(),
	}
	for nama, jti := range kasus {
		baru := uuid.NewString()
		if err := rotasiRefreshTx(ctx, tx, jti, user, baru, exp); !errors.Is(err, ErrRefreshTidakValid) {
			t.Errorf("%s: err=%v, want ErrRefreshTidakValid", nama, err)
		}
		if _, ada := cekToken(t, tx, baru); ada {
			t.Errorf("%s: token baru tidak boleh tercatat", nama)
		}
	}
}

// Temuan audit (INFO): user.admin hanya boleh dipakai token holding, walau role brand keliru diberi izin itu.
func TestRequireHoldingPermission(t *testing.T) {
	t.Setenv("JWT_SECRET", secretUji)
	SetUserStatusLoader(func(context.Context, int64) (bool, int, error) { return true, 0, nil })
	t.Cleanup(func() { SetUserStatusLoader(nil) })

	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })
	h := RequireAuth(RequireHoldingPermission("user.admin")(ok))
	brand := int64(2)
	run := func(id int64, brandID *int64, perms []string) int {
		tok, err := GenerateAccessTokenWithAccess(id, brandID, "admin", "", nil, perms, 0)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/admin/users", nil)
		req.Header.Set("Authorization", "Bearer "+tok)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec.Code
	}
	if c := run(101, nil, []string{"user.admin"}); c != http.StatusOK {
		t.Fatalf("holding dengan user.admin: %d, want 200", c)
	}
	if c := run(102, &brand, []string{"user.admin"}); c != http.StatusForbidden {
		t.Fatalf("brand dengan user.admin: %d, want 403", c)
	}
	if c := run(103, nil, []string{"booking.view"}); c != http.StatusForbidden {
		t.Fatalf("holding tanpa user.admin: %d, want 403", c)
	}
}
