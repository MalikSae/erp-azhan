package customer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
)

// Butir 6: nama disisipkan ke isi email, jadi hanya karakter nama yang diterima.
func TestValidNameMenolakInjeksiKonten(t *testing.T) {
	good := []string{"Budi", "Siti Nur'aini", "Abd. Rahman, S.Pd", "Jean-Pierre", "Muhammad ’Ali", "Zoë Ñúñez", "李小龍"}
	bad := []string{
		"", "B", strings.Repeat("a", 101),
		"Budi,\n\nAkun Anda akan diblokir: https://evil.example\n\n",
		"Budi https://evil.example", "Budi evil.example/login", "budi@evil.example",
		"Budi\tSantoso", "Budi\rSantoso", "Budi 123", "Budi <b>", "..", "-- ,",
	}
	for _, s := range good {
		if !ValidName(s) {
			t.Errorf("%q seharusnya valid", s)
		}
	}
	for _, s := range bad {
		if ValidName(s) {
			t.Errorf("%q seharusnya ditolak", s)
		}
	}
}

func TestMaskNIK(t *testing.T) {
	cases := map[string]string{"": "", "3174010101010001": "••••••••••••0001", "123": "•••"}
	for in, want := range cases {
		if got := MaskNIK(in); got != want {
			t.Errorf("MaskNIK(%q) = %q, want %q", in, got, want)
		}
	}
}

// Butir 1: /api/account/me tidak pernah mengirim NIK utuh.
func TestMeTidakMembocorkanNIK(t *testing.T) {
	_, tx := testdb.Tx(t)
	var brandID int64
	if err := tx.QueryRow(`SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	nik := "3174010101012222"
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	res, err := tx.Exec(`INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap, jenis_kelamin, nik) VALUES (?,?,?,?,?,?)`,
		brandID, "TST-ME-"+suffix, "M"+suffix[len(suffix)-5:], "Jamaah Me", "P", nik)
	if err != nil {
		t.Fatal(err)
	}
	jid, _ := res.LastInsertId()
	acc := &Account{ID: 1, BrandID: brandID, NamaLengkap: "Jamaah Me", Email: "me@example.com", JamaahID: &jid}

	raw, err := json.Marshal(map[string]any{"account": buildMeView(context.Background(), tx, acc)})
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if strings.Contains(body, nik) || strings.Contains(body, `"nik":`) {
		t.Fatalf("/me membocorkan NIK: %s", body)
	}
	if !strings.Contains(body, `"nik_masked":"••••••••••••2222"`) || !strings.Contains(body, `"jenis_kelamin":"P"`) {
		t.Fatalf("/me tanpa nik_masked/jenis_kelamin: %s", body)
	}
}

// Butir 7f: syarat batal mandiri diperiksa ulang di dalam transaksi (pembayaran yang masuk setelah
// pengecekan awal membatalkan pembatalan).
func TestCancelGuardPeriksaUlangDalamTransaksi(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b := testdb.NewBooking(t, tx, testdb.BookingOpts{Status: "baru", TotalHarga: 1000, Blocked: true})
	var brandID int64
	if err := tx.QueryRow(`SELECT brand_id FROM schedules WHERE id=?`, b.ScheduleID).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("batal-%d@example.com", time.Now().UnixNano())
	res, err := tx.Exec(`INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash) VALUES (?,?,?,?,?,'x')`,
		brandID, "Uji Batal", email, email, "081200000000")
	if err != nil {
		t.Fatal(err)
	}
	akun, _ := res.LastInsertId()
	testdb.Exec(t, tx, `UPDATE bookings SET customer_account_id=?, seat_hold_expires_at=? WHERE id=?`, akun, time.Now().Add(time.Hour), b.ID)

	if err := cancelGuard(akun, b.ID)(ctx, tx); err != nil {
		t.Fatalf("booking baru tanpa pembayaran harus bisa dibatalkan: %v", err)
	}
	if err := cancelGuard(akun+1, b.ID)(ctx, tx); err == nil {
		t.Fatal("akun lain lolos guard")
	}
	testdb.AddPayment(t, tx, b.ID, 500, "pending")
	if err := cancelGuard(akun, b.ID)(ctx, tx); err != errTidakBisaBatal {
		t.Fatalf("pembayaran yang baru masuk harus menggagalkan pembatalan: %v", err)
	}
}

// Butir 4: tanpa TURNSTILE_SECRET_KEY pendaftaran ditolak (fail closed), kecuali bypass dev eksplisit.
func TestRegisterCaptchaFailClosed(t *testing.T) {
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	for _, env := range []struct{ appEnv, allow string }{{"", ""}, {"production", "true"}, {"development", ""}, {"staging", "true"}} {
		t.Setenv("APP_ENV", env.appEnv)
		t.Setenv("ALLOW_DEV_CAPTCHA", env.allow)
		h := NewHandler(nil, &captureMail{}) // repo nil: request harus berhenti sebelum database
		rec := post(h.Register, map[string]any{"brand_id": 1, "nama_lengkap": "Budi", "email": "b@example.com", "no_hp": "081234567890", "password": "rahasia-123"}, "")
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("APP_ENV=%q ALLOW_DEV_CAPTCHA=%q: got %d %s, want 503", env.appEnv, env.allow, rec.Code, rec.Body.String())
		}
	}
}

// Butir 4: lupa password tanpa captcha dibatasi per email lintas IP; jawabannya tetap generik.
func TestForgotPasswordBatasPerEmailLintasIP(t *testing.T) {
	h := NewHandler(nil, &captureMail{})
	email := "korban@example.com"
	for i := 0; i < forgotEmailLimit; i++ {
		h.forgotEmail.Record("1|" + email)
	}
	// IP baru; repo nil akan panic bila tersentuh, jadi request harus berhenti di batas per email.
	rec := post(h.ForgotPassword, map[string]any{"brand_id": 1, "email": "  KORBAN@example.com "}, "")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "Jika email terdaftar") {
		t.Fatalf("got %d %s, want 200 generik", rec.Code, rec.Body.String())
	}
}
