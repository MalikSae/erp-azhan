package agen

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/testdb"
	"github.com/go-chi/chi/v5"
)

// Rentang ujroh publik (K23): min dan max dari jadwal tayang yang ikut program; jadwal tanpa nominal,
// tidak tayang, atau sudah berangkat tidak dihitung.
func TestRentangUjrohDariJadwalTayang(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	s1, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{KomisiLangsung: rp(750_000), BonusPembinaan: rp(200_000), BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=NULL, nominal_bonus_pembinaan=NULL WHERE brand_id=? AND id<>?`, brand, s1)
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, s1)

	u, err := rentangUjroh(ctx, tx, brand)
	must(t, "rentang", err)
	if u.JumlahJadwal != 1 || u.KomisiLangsung == nil || u.KomisiLangsung.Min != 750_000 || u.KomisiLangsung.Max != 750_000 ||
		u.BonusPembinaan == nil || u.BonusPembinaan.Max != 200_000 {
		t.Fatalf("satu jadwal: %+v", u)
	}

	// Jadwal kedua dengan nominal lebih besar memperlebar rentang; tanpa pembinaan tidak mengubah rentang pembinaan.
	res, err := tx.Exec(`UPDATE schedules SET status='published', berangkat_tanggal='2099-02-01', nominal_komisi_langsung=1500000, nominal_bonus_pembinaan=NULL
		WHERE brand_id=? AND id<>? ORDER BY id LIMIT 1`, brand, s1)
	must(t, "jadwal kedua", err)
	if n, _ := res.RowsAffected(); n == 0 {
		t.Skip("butuh minimal dua jadwal di brand fixture")
	}
	u, err = rentangUjroh(ctx, tx, brand)
	must(t, "rentang 2", err)
	if u.JumlahJadwal != 2 || u.KomisiLangsung.Min != 750_000 || u.KomisiLangsung.Max != 1_500_000 || u.BonusPembinaan.Min != 200_000 || u.BonusPembinaan.Max != 200_000 {
		t.Fatalf("dua jadwal: %+v", u)
	}

	// Tidak tayang atau sudah berangkat: tidak dihitung, rentang kosong.
	testdb.Exec(t, tx, `UPDATE schedules SET status='draft' WHERE brand_id=?`, brand)
	u, err = rentangUjroh(ctx, tx, brand)
	must(t, "rentang 3", err)
	if u.JumlahJadwal != 0 || u.KomisiLangsung != nil || u.BonusPembinaan != nil {
		t.Fatalf("tanpa jadwal tayang: %+v", u)
	}
	if _, err := rentangUjroh(ctx, tx, brand+999_999); err != ErrNotFound {
		t.Fatalf("brand tidak ada: %v", err)
	}
}

// mediaMilikAkun: foto dan bukti transfer harus hasil unggahan akun itu sendiri.
func TestMediaMilikAkun(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	_, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Pengunggah"})
	lain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Lain"})
	url := "/api/admin/media/dokumen-jamaah/aaaaaaaa-2222-3333-4444-555555555555.jpg"
	testdb.Exec(t, tx, `INSERT INTO media_uploads(path, brand_id, customer_account_id) VALUES (?,?,?)`, url, brand, a)
	for _, c := range []struct {
		akun int64
		url  string
		want bool
	}{{a, url, true}, {lain, url, false}, {a, url + "x", false}} {
		got, err := mediaMilikAkun(ctx, tx, c.akun, c.url)
		must(t, "cek", err)
		if got != c.want {
			t.Fatalf("mediaMilikAkun(%d,%s) = %v, want %v", c.akun, c.url, got, c.want)
		}
	}
}

// Akses portal: status terbuka untuk akun mana pun, fitur agen hanya untuk agen aktif (nonaktif ditolak, layar A4).
func TestPortalAgenAksesHanyaAgenAktif(t *testing.T) {
	db := testdb.DB(t)
	var brandID int64
	must(t, "brand", db.QueryRow(`SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brandID))
	u := time.Now().UnixNano()
	buat := func(k, status string, n int) int64 {
		email := fmt.Sprintf("portal-%s-%d@example.com", k, u)
		res, err := db.Exec(`INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash, status_agen)
			VALUES (?,?,?,?,?,'x',?)`, brandID, "Portal "+k, email, email, fmt.Sprintf("0855%02d%06d", n, u%1000000), status)
		if err != nil {
			t.Fatal(err)
		}
		id, _ := res.LastInsertId()
		t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM customer_accounts WHERE id=?`, id) })
		return id
	}
	aktif, nonaktif, biasa := buat("aktif", "aktif", 1), buat("nonaktif", "nonaktif", 2), buat("biasa", "tidak_aktif", 3)

	h := NewHandler(NewRepository(db))
	router := func(akun int64) http.Handler {
		r := chi.NewRouter()
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				next.ServeHTTP(w, req.WithContext(identity.WithCustomer(req.Context(), akun, brandID)))
			})
		})
		r.Get("/", h.GetStatus)
		r.Post("/pengajuan", h.Ajukan)
		r.Group(func(r chi.Router) {
			r.Use(h.RequireAgenAktif)
			r.Get("/dashboard", h.Dashboard)
			r.Get("/komisi", h.RiwayatKomisiPortal)
			r.Get("/pencairan", h.GetPencairan)
		})
		return r
	}
	do := func(akun int64, method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		router(akun).ServeHTTP(rec, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
		return rec
	}

	for _, c := range []struct {
		akun int64
		path string
		want int
	}{
		{aktif, "/", 200}, {biasa, "/", 200}, {nonaktif, "/", 200},
		{aktif, "/dashboard", 200}, {aktif, "/komisi", 200}, {aktif, "/pencairan", 200},
		{nonaktif, "/dashboard", 403}, {nonaktif, "/pencairan", 403},
		{biasa, "/dashboard", 403}, {biasa, "/komisi", 403},
	} {
		if rec := do(c.akun, "GET", c.path, ""); rec.Code != c.want {
			t.Fatalf("akun %d GET %s = %d %s, want %d", c.akun, c.path, rec.Code, rec.Body, c.want)
		}
	}

	// Ajukan dengan foto yang bukan hasil unggahan akun ini ditolak (URL karangan).
	body, _ := json.Marshal(map[string]any{
		"foto_agen_url": "/api/admin/media/dokumen-jamaah/00000000-0000-0000-0000-000000000000.jpg", "domisili": "Bekasi", "setuju_syarat_ketentuan": true,
	})
	if rec := do(biasa, "POST", "/pengajuan", string(body)); rec.Code != 400 {
		t.Fatalf("ajukan foto karangan = %d %s, want 400", rec.Code, rec.Body)
	}
}
