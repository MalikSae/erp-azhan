package customer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"erp-azhan/api/internal/mailer"
	"erp-azhan/api/internal/testdb"
)

type captureMail struct {
	mu   sync.Mutex
	msgs []mailer.Message
}

func (c *captureMail) Send(_ context.Context, m mailer.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.msgs = append(c.msgs, m)
	return nil
}

func (c *captureMail) last(to string) (mailer.Message, bool) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		for i := len(c.msgs) - 1; i >= 0; i-- {
			if c.msgs[i].To == to {
				m := c.msgs[i]
				c.mu.Unlock()
				return m, true
			}
		}
		c.mu.Unlock()
		time.Sleep(20 * time.Millisecond)
	}
	return mailer.Message{}, false
}

func tokenFrom(t *testing.T, text string) string {
	t.Helper()
	i := strings.Index(text, "token=")
	if i < 0 {
		t.Fatalf("token tidak ada di email: %s", text)
	}
	rest := text[i+len("token="):]
	if j := strings.IndexAny(rest, " \n\r"); j >= 0 {
		rest = rest[:j]
	}
	return rest
}

func post(h http.HandlerFunc, body any, token string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.RemoteAddr = "203.0.113.9:1234"
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestAccountLifecycle(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()

	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO brand_domains (brand_id, domain) VALUES (?, 'akun-test.azhan.test')`, brandID); err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("uji-%d@example.com", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal=?`, email)
		_, _ = db.Exec(`DELETE FROM brand_domains WHERE domain='akun-test.azhan.test'`)
	})

	// Captcha opsional: tanpa secret nonaktif.
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	mail := &captureMail{}
	h := NewHandler(NewRepository(db), mail)
	t.Setenv("SITE_URL_TEMPLATE", "http://{domain}")

	// Daftar
	rec := post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Budi Uji", "email": email, "no_hp": "081234567890", "password": "rahasia-123"}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body)
	}
	var reg authResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	if reg.SessionToken == "" {
		t.Fatalf("register harus memberi sesi: %+v", reg)
	}

	// Email sama (huruf besar) ditolak
	if rec = post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Budi Dua", "email": strings.ToUpper(email), "no_hp": "081234567891", "password": "rahasia-123"}, ""); rec.Code != http.StatusConflict {
		t.Fatalf("email ganda harus 409, dapat %d", rec.Code)
	}

	// Tanpa verifikasi email: pendaftaran tidak mengirim email apa pun dan akun langsung bisa dipakai.
	if _, kirim := mail.last(email); kirim {
		t.Fatal("pendaftaran tidak boleh mengirim email verifikasi")
	}
	if acc, err := h.repo.AccountBySession(ctx, reg.SessionToken); err != nil || acc.Email != email {
		t.Fatalf("sesi harus langsung berlaku: %v %+v", err, acc)
	}

	// Login: salah vs benar
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "salah-salah"}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("password salah harus 401, dapat %d", rec.Code)
	}
	rec = post(h.Login, map[string]any{"brand_id": brandID, "email": strings.ToUpper(email), "password": "rahasia-123"}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("login: %d %s", rec.Code, rec.Body)
	}
	var login authResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &login)

	// Lupa password -> reset -> sesi lama mati, password baru berlaku
	if rec = post(h.ForgotPassword, map[string]any{"brand_id": brandID, "email": email}, ""); rec.Code != http.StatusOK {
		t.Fatalf("forgot: %d", rec.Code)
	}
	var resetMsg mailer.Message
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if m, ok := mail.last(email); ok && strings.Contains(m.Subject, "Atur ulang") {
			resetMsg = m
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if resetMsg.Text == "" {
		t.Fatal("email reset tidak terkirim")
	}
	if rec = post(h.ResetPassword, map[string]string{"token": tokenFrom(t, resetMsg.Text), "password": "baru-12345"}, ""); rec.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", rec.Code, rec.Body)
	}
	if _, err := h.repo.AccountBySession(ctx, login.SessionToken); err == nil {
		t.Fatal("sesi lama harus dicabut setelah reset password")
	}
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "rahasia-123"}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatal("password lama tidak boleh berlaku")
	}
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "baru-12345"}, ""); rec.Code != http.StatusOK {
		t.Fatal("password baru harus berlaku")
	}

	// Forgot untuk email tak dikenal: jawaban sama, tidak membocorkan
	if rec = post(h.ForgotPassword, map[string]any{"brand_id": brandID, "email": "tidak-ada@example.com"}, ""); rec.Code != http.StatusOK {
		t.Fatalf("forgot email tak dikenal harus tetap 200, dapat %d", rec.Code)
	}
}

func TestLoginRateLimit(t *testing.T) {
	db := testdb.DB(t)
	h := NewHandler(NewRepository(db), &captureMail{})
	for i := 0; i < loginKeyLimit; i++ {
		if rec := post(h.Login, map[string]any{"brand_id": 1, "email": "x@example.com", "password": "salah-salah"}, ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("percobaan %d: %d", i+1, rec.Code)
		}
	}
	if rec := post(h.Login, map[string]any{"brand_id": 1, "email": "x@example.com", "password": "salah-salah"}, ""); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("percobaan ke-%d harus 429, dapat %d", loginKeyLimit+1, rec.Code)
	}
}

// Nomor WhatsApp akun unik per brand, dengan penyamaan format (0812..., 62812..., +62 812...).
func TestRegisterNomorWhatsAppUnik(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	tail := time.Now().UnixNano() % 100000000
	emailA := fmt.Sprintf("hp-a-%d@example.com", time.Now().UnixNano())
	emailB := fmt.Sprintf("hp-b-%d@example.com", time.Now().UnixNano())
	hpA := fmt.Sprintf("0899%08d", tail)
	hpB := fmt.Sprintf("+62 899-%08d", tail) // nomor yang sama, format lain
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal IN (?,?)`, emailA, emailB)
	})
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	h := NewHandler(NewRepository(db), &captureMail{})

	rec := post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Akun Satu", "email": emailA, "no_hp": hpA, "password": "rahasia-123"}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("akun pertama: %d %s", rec.Code, rec.Body)
	}
	rec = post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Akun Dua", "email": emailB, "no_hp": hpB, "password": "rahasia-123"}, "")
	if rec.Code != http.StatusConflict || !strings.Contains(rec.Body.String(), "Nomor WhatsApp sudah terdaftar") {
		t.Fatalf("nomor sama beda format harus 409 dengan pesan nomor, dapat %d %s", rec.Code, rec.Body)
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM customer_accounts WHERE email_normal=?`, emailB).Scan(&n)
	if n != 0 {
		t.Fatal("akun kedua tidak boleh terbentuk")
	}
}

// Ubah data akun: nama, email, dan nomor WhatsApp. Email baru membatalkan verifikasi dan mengirim tautan baru;
// nomor yang dipakai akun lain ditolak.
func TestUpdateMe(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO brand_domains (brand_id, domain) VALUES (?, 'ubah-test.azhan.test')`, brandID); err != nil {
		t.Fatal(err)
	}
	tail := time.Now().UnixNano() % 100000000
	stamp := time.Now().UnixNano()
	emailA := fmt.Sprintf("ubah-a-%d@example.com", stamp)
	emailA2 := fmt.Sprintf("ubah-a2-%d@example.com", stamp)
	emailB := fmt.Sprintf("ubah-b-%d@example.com", stamp)
	hpA := fmt.Sprintf("0898%08d", tail)
	hpA2 := fmt.Sprintf("0897%08d", tail)
	hpB := fmt.Sprintf("0896%08d", tail)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal IN (?,?,?)`, emailA, emailA2, emailB)
		_, _ = db.Exec(`DELETE FROM brand_domains WHERE domain='ubah-test.azhan.test'`)
	})
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	t.Setenv("SITE_URL_TEMPLATE", "http://{domain}")
	mail := &captureMail{}
	h := NewHandler(NewRepository(db), mail)
	patch := h.RequireCustomer(http.HandlerFunc(h.UpdateMe))

	var tokA string
	for _, a := range []struct{ email, hp string }{{emailA, hpA}, {emailB, hpB}} {
		rec := post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Akun Uji", "email": a.email, "no_hp": a.hp, "password": "rahasia-123"}, "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("daftar %s: %d %s", a.email, rec.Code, rec.Body)
		}
		if a.email == emailA {
			var reg authResponse
			_ = json.Unmarshal(rec.Body.Bytes(), &reg)
			tokA = reg.SessionToken
		}
	}
	idA, _ := h.repo.AccountBySession(ctx, tokA)

	call := func(body map[string]any) *httptest.ResponseRecorder {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPatch, "/", bytes.NewReader(b))
		req.RemoteAddr = "203.0.113.9:1234"
		req.Header.Set("Authorization", "Bearer "+tokA)
		rec := httptest.NewRecorder()
		patch.ServeHTTP(rec, req)
		return rec
	}

	// Nama dan nomor berubah, email sama.
	if rec := call(map[string]any{"nama_lengkap": "Nama Baru", "email": emailA, "no_hp": hpA2}); rec.Code != http.StatusOK {
		t.Fatalf("ubah nama dan nomor: %d %s", rec.Code, rec.Body)
	}
	got, _ := h.repo.GetByID(ctx, idA.ID)
	if got.NamaLengkap != "Nama Baru" || got.NoHP != hpA2 {
		t.Fatalf("data tidak sesuai: %+v", got)
	}

	// Nomor akun lain ditolak.
	if rec := call(map[string]any{"nama_lengkap": "Nama Baru", "email": emailA, "no_hp": hpB}); rec.Code != http.StatusConflict {
		t.Fatalf("nomor akun lain harus 409, dapat %d", rec.Code)
	}
	// Email akun lain ditolak.
	if rec := call(map[string]any{"nama_lengkap": "Nama Baru", "email": emailB, "no_hp": hpA2}); rec.Code != http.StatusConflict {
		t.Fatalf("email akun lain harus 409, dapat %d", rec.Code)
	}
	// Format tidak valid ditolak.
	if rec := call(map[string]any{"nama_lengkap": "Nama Baru", "email": "bukan-email", "no_hp": hpA2}); rec.Code != http.StatusBadRequest {
		t.Fatalf("email tidak valid harus 400, dapat %d", rec.Code)
	}

	// Email baru langsung berlaku, tanpa verifikasi ulang dan tanpa email apa pun.
	if rec := call(map[string]any{"nama_lengkap": "Nama Baru", "email": emailA2, "no_hp": hpA2}); rec.Code != http.StatusOK {
		t.Fatalf("ubah email: %d %s", rec.Code, rec.Body)
	}
	got, _ = h.repo.GetByID(ctx, idA.ID)
	if got.Email != emailA2 {
		t.Fatalf("email harus berubah: %+v", got)
	}
	if _, ok := mail.last(emailA2); ok {
		t.Fatal("ganti email tidak boleh mengirim email verifikasi")
	}

	// Kota domisili dan tanggal lahir: opsional. Tidak dikirim = tidak berubah, kosong = dihapus.
	dasar := map[string]any{"nama_lengkap": "Nama Baru", "email": emailA2, "no_hp": hpA2}
	withExtra := func(extra map[string]any) map[string]any {
		m := map[string]any{}
		for k, v := range dasar {
			m[k] = v
		}
		for k, v := range extra {
			m[k] = v
		}
		return m
	}
	if rec := call(withExtra(map[string]any{"domisili": "Kota Bandung", "tanggal_lahir": "1990-05-12"})); rec.Code != http.StatusOK {
		t.Fatalf("isi domisili dan tanggal lahir: %d %s", rec.Code, rec.Body)
	}
	got, _ = h.repo.GetByID(ctx, idA.ID)
	if got.Domisili == nil || *got.Domisili != "Kota Bandung" || got.TanggalLahir == nil || *got.TanggalLahir != "1990-05-12" {
		t.Fatalf("domisili dan tanggal lahir tidak tersimpan: %+v / %+v", got.Domisili, got.TanggalLahir)
	}
	if rec := call(dasar); rec.Code != http.StatusOK {
		t.Fatalf("ubah tanpa field tambahan: %d %s", rec.Code, rec.Body)
	}
	got, _ = h.repo.GetByID(ctx, idA.ID)
	if got.Domisili == nil || got.TanggalLahir == nil {
		t.Fatal("field yang tidak dikirim tidak boleh terhapus")
	}
	for _, tgl := range []string{"12-05-1990", "2999-01-01", "1800-01-01", "bukan"} {
		if rec := call(withExtra(map[string]any{"tanggal_lahir": tgl})); rec.Code != http.StatusBadRequest {
			t.Fatalf("tanggal lahir %q harus 400, dapat %d", tgl, rec.Code)
		}
	}
	if rec := call(withExtra(map[string]any{"domisili": strings.Repeat("k", 101)})); rec.Code != http.StatusBadRequest {
		t.Fatalf("domisili 101 karakter harus 400, dapat %d", rec.Code)
	}
	if rec := call(withExtra(map[string]any{"domisili": "", "tanggal_lahir": ""})); rec.Code != http.StatusOK {
		t.Fatalf("kosongkan: %d %s", rec.Code, rec.Body)
	}
	got, _ = h.repo.GetByID(ctx, idA.ID)
	if got.Domisili != nil || got.TanggalLahir != nil {
		t.Fatalf("kosong harus menghapus: %+v / %+v", got.Domisili, got.TanggalLahir)
	}
}

// Ganti password dari dalam akun: wajib password saat ini, sesi lain dicabut, sesi yang dipakai tetap aktif,
// pemilik diberi tahu lewat email, dan percobaan salah dibatasi.
func TestChangePassword(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT IGNORE INTO brand_domains (brand_id, domain) VALUES (?, 'sandi-test.azhan.test')`, brandID); err != nil {
		t.Fatal(err)
	}
	email := fmt.Sprintf("sandi-%d@example.com", time.Now().UnixNano())
	hp := fmt.Sprintf("0895%08d", time.Now().UnixNano()%100000000)
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal=?`, email)
		_, _ = db.Exec(`DELETE FROM brand_domains WHERE domain='sandi-test.azhan.test'`)
	})
	t.Setenv("TURNSTILE_SECRET_KEY", "")
	t.Setenv("SITE_URL_TEMPLATE", "http://{domain}")
	mail := &captureMail{}
	h := NewHandler(NewRepository(db), mail)
	change := h.RequireCustomer(http.HandlerFunc(h.ChangePassword))

	rec := post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Akun Sandi", "email": email, "no_hp": hp, "password": "rahasia-123"}, "")
	if rec.Code != http.StatusCreated {
		t.Fatalf("daftar: %d %s", rec.Code, rec.Body)
	}
	var reg authResponse
	_ = json.Unmarshal(rec.Body.Bytes(), &reg)
	var login authResponse
	rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "rahasia-123"}, "")
	_ = json.Unmarshal(rec.Body.Bytes(), &login) // sesi kedua = "perangkat lain"
	idA, _ := h.repo.AccountBySession(ctx, reg.SessionToken)

	call := func(body map[string]any) int {
		b, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
		req.RemoteAddr = "203.0.113.9:1234"
		req.Header.Set("Authorization", "Bearer "+reg.SessionToken)
		rr := httptest.NewRecorder()
		change.ServeHTTP(rr, req)
		return rr.Code
	}

	if c := call(map[string]any{"password_saat_ini": "salah-salah", "password_baru": "baru-12345"}); c != http.StatusBadRequest {
		t.Fatalf("password saat ini salah harus 400, dapat %d", c)
	}
	if c := call(map[string]any{"password_saat_ini": "rahasia-123", "password_baru": "pendek"}); c != http.StatusBadRequest {
		t.Fatalf("password baru pendek harus 400, dapat %d", c)
	}
	if c := call(map[string]any{"password_saat_ini": "rahasia-123", "password_baru": "rahasia-123"}); c != http.StatusBadRequest {
		t.Fatalf("password sama harus 400, dapat %d", c)
	}
	resetToken, err := h.repo.IssueToken(ctx, idA.ID, PurposeResetPass, ResetTokenTTL)
	if err != nil {
		t.Fatal(err)
	}
	if c := call(map[string]any{"password_saat_ini": "rahasia-123", "password_baru": "baru-12345"}); c != http.StatusOK {
		t.Fatalf("ganti password harus 200, dapat %d", c)
	}
	if _, err := h.repo.ConsumeToken(ctx, resetToken, PurposeResetPass); err == nil {
		t.Fatal("tautan reset lama harus dicabut setelah ganti password")
	}
	if _, err := h.repo.AccountBySession(ctx, reg.SessionToken); err != nil {
		t.Fatal("sesi yang dipakai harus tetap aktif")
	}
	if _, err := h.repo.AccountBySession(ctx, login.SessionToken); err == nil {
		t.Fatal("sesi lain harus dicabut")
	}
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "rahasia-123"}, ""); rec.Code != http.StatusUnauthorized {
		t.Fatal("password lama tidak boleh berlaku")
	}
	if rec = post(h.Login, map[string]any{"brand_id": brandID, "email": email, "password": "baru-12345"}, ""); rec.Code != http.StatusOK {
		t.Fatal("password baru harus berlaku")
	}
	if m, ok := mail.last(email); !ok || !strings.Contains(m.Subject, "diubah") {
		t.Fatalf("email pemberitahuan tidak terkirim: %+v", m)
	}

	for i := 0; i < passwordFailLimit; i++ {
		call(map[string]any{"password_saat_ini": "salah-salah", "password_baru": "baru-67890"})
	}
	if c := call(map[string]any{"password_saat_ini": "baru-12345", "password_baru": "baru-67890"}); c != http.StatusTooManyRequests {
		t.Fatalf("setelah %d percobaan salah harus 429, dapat %d", passwordFailLimit, c)
	}
}

// Pengikatan akun (spek 4.1): akun terikat ke agen pemilik kode referral saat akun dibuat. Kode yang dikirim
// route handler adalah kode klik terakhir sebelum daftar. Contoh: Z klik link W (hari 1), link X lalu daftar
// (hari 2), link Q (hari 3) -> akun Z terikat ke X. Klik Q sesudahnya tidak punya jalur apa pun ke backend.
func TestRegisterMengikatAkunKeAgenDariKodeReferral(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	var brandID int64
	if err := db.QueryRowContext(ctx, `SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID); err != nil {
		t.Fatal(err)
	}
	u := fmt.Sprint(time.Now().UnixNano())
	emails := map[string]string{}
	for _, k := range []string{"w", "x", "q", "nonaktif", "z", "z2", "z3", "z4"} {
		emails[k] = fmt.Sprintf("ref-%s-%s@example.com", k, u)
	}
	t.Cleanup(func() {
		// Anak dulu (akun yang diikat), baru agen.
		for _, k := range []string{"z", "z2", "z3", "z4"} {
			_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal=?`, emails[k])
		}
		for _, k := range []string{"w", "x", "q", "nonaktif"} {
			_, _ = db.Exec(`DELETE FROM customer_accounts WHERE email_normal=?`, emails[k])
		}
	})
	agen := func(k, kode, status string, n int) int64 {
		res, err := db.Exec(`INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash, status_agen, kode_referral)
			VALUES (?,?,?,?,?,'x',?,?)`, brandID, "Agen "+k, emails[k], emails[k], fmt.Sprintf("0877%02d%06d", n, time.Now().UnixNano()%1000000), status, kode)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		return id
	}
	kode := func(p string) string { return strings.ToUpper(p + u[len(u)-6:]) }
	_ = agen("w", kode("w"), "aktif", 1)
	x := agen("x", kode("x"), "aktif", 2)
	_ = agen("q", kode("q"), "aktif", 3)
	_ = agen("nonaktif", kode("n"), "nonaktif", 4)

	t.Setenv("TURNSTILE_SECRET_KEY", "")
	h := NewHandler(NewRepository(db), &captureMail{})
	daftar := func(k, kodeRef string, n int) int64 {
		rec := post(h.Register, map[string]any{"brand_id": brandID, "nama_lengkap": "Akun Uji", "email": emails[k],
			"no_hp": fmt.Sprintf("0866%02d%06d", n, time.Now().UnixNano()%1000000), "password": "rahasia-123", "kode_referral": kodeRef}, "")
		if rec.Code != http.StatusCreated {
			t.Fatalf("register %s: %d %s", k, rec.Code, rec.Body)
		}
		var id int64
		var direkrut *int64
		if err := db.QueryRow(`SELECT id, direkrut_oleh_akun_id FROM customer_accounts WHERE email_normal=?`, emails[k]).Scan(&id, &direkrut); err != nil {
			t.Fatal(err)
		}
		if direkrut != nil {
			return *direkrut
		}
		return 0
	}

	// Kode klik terakhir sebelum daftar (X) yang dipakai, huruf kecil dinormalisasi.
	if got := daftar("z", strings.ToLower(kode("x")), 1); got != x {
		t.Fatalf("akun Z terikat ke %d, want X (%d)", got, x)
	}
	// Tanpa kode, kode tidak dikenal, dan kode agen nonaktif: diabaikan diam-diam, akun tetap tanpa agen.
	if got := daftar("z2", "", 2); got != 0 {
		t.Fatalf("tanpa kode: terikat ke %d, want tanpa agen", got)
	}
	if got := daftar("z3", "TIDAKADA", 3); got != 0 {
		t.Fatalf("kode tidak dikenal: terikat ke %d, want tanpa agen", got)
	}
	if got := daftar("z4", kode("n"), 4); got != 0 {
		t.Fatalf("kode agen nonaktif: terikat ke %d, want tanpa agen", got)
	}
}
