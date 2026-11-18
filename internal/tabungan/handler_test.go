package tabungan

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/testdb"
	"github.com/go-chi/chi/v5"
)

// Akun uji dibuat dengan commit karena handler membuka transaksinya sendiri; cleanup menghapus akun
// dan ON DELETE CASCADE menghapus data tabungannya.
func buatAkun(t *testing.T, db *sql.DB) (akun, brand int64) {
	t.Helper()
	if err := db.QueryRow(`SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brand); err != nil {
		t.Fatal(err)
	}
	u := strconv.FormatInt(time.Now().UnixNano(), 10)
	email := "tab-" + u + "@uji.test"
	res, err := db.Exec(`INSERT INTO customer_accounts (brand_id, nama_lengkap, email, email_normal, no_hp, password_hash, status_agen)
		VALUES (?,?,?,?,?,?, 'tidak_aktif')`, brand, "Tabungan Uji", email, email, "0877"+u[len(u)-9:], "x")
	if err != nil {
		t.Fatal(err)
	}
	akun, _ = res.LastInsertId()
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM perangkat_push WHERE akun_id = ?`, akun)
		_, _ = db.Exec(`DELETE FROM customer_accounts WHERE id = ?`, akun)
	})
	return akun, brand
}

// router meniru grup /api/account: akun diambil dari header X-Akun, tanpa header ditolak 401.
func router(db *sql.DB) http.Handler {
	h := NewHandler(db)
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			id, err := strconv.ParseInt(req.Header.Get("X-Akun"), 10, 64)
			if err != nil || id <= 0 {
				writeError(w, http.StatusUnauthorized, "Masuk dulu.")
				return
			}
			next.ServeHTTP(w, req.WithContext(identity.WithCustomer(req.Context(), id, 1)))
		})
	})
	r.Get("/tabungan", h.Ambil)
	r.Put("/tabungan/target", h.SimpanTarget)
	r.Delete("/tabungan/target", h.HapusTarget)
	r.Post("/tabungan/catatan", h.TambahCatatan)
	r.Delete("/tabungan/catatan/{id}", h.HapusCatatan)
	r.Put("/perangkat-push", h.SimpanPerangkat)
	r.Delete("/perangkat-push", h.HapusPerangkat)
	return r
}

func panggil(t *testing.T, rt http.Handler, akun int64, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if akun > 0 {
		req.Header.Set("X-Akun", strconv.FormatInt(akun, 10))
	}
	rec := httptest.NewRecorder()
	rt.ServeHTTP(rec, req)
	return rec
}

func baca(t *testing.T, rec *httptest.ResponseRecorder) Respons {
	t.Helper()
	var r Respons
	if err := json.Unmarshal(rec.Body.Bytes(), &r); err != nil {
		t.Fatalf("balasan bukan JSON: %v (%s)", err, rec.Body.String())
	}
	return r
}

// buatJadwal menyisipkan jadwal uji (commit) dan menghapusnya saat cleanup.
func buatJadwal(t *testing.T, db *sql.DB, status string, hariKeDepan int) int64 {
	t.Helper()
	u := strconv.FormatInt(time.Now().UnixNano(), 10)
	b := time.Now().In(ZonaWIB).AddDate(0, 0, hariKeDepan)
	var brand int64
	if err := db.QueryRow(`SELECT id FROM brands ORDER BY id LIMIT 1`).Scan(&brand); err != nil {
		t.Fatal(err)
	}
	res, err := db.Exec(`INSERT INTO schedules (brand_id, jadwal_nama, seat_total, seat_sisa, berangkat_tanggal, pulang_tanggal,
		harga_quad, harga_triple, harga_double, status) VALUES (?,?,?,?,?,?,?,?,?,?)`,
		brand, "Jadwal Uji Tabungan "+u, 10, 10, b.Format("2006-01-02"), b.AddDate(0, 0, 9).Format("2006-01-02"),
		30_000_000, 31_000_000, 32_000_000, status)
	if err != nil {
		t.Fatal(err)
	}
	id, _ := res.LastInsertId()
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM schedules WHERE id = ?`, id) })
	return id
}

func bodyTarget(t *testing.T, db *sql.DB, total int64) map[string]any {
	t.Helper()
	sched := buatJadwal(t, db, "published", 90)
	bulan := time.Now().In(ZonaWIB).AddDate(0, 6, 0).Format("2006-01")
	return map[string]any{
		"jenis": "paket", "schedule_id": sched, "nama_paket": "Paket Uji", "bulan_berangkat": bulan,
		"jumlah_dewasa": 2, "jumlah_bayi": 1, "tipe_kamar": "Quad", "target_total": total, "dp_minimal": 10_000_000,
		"pengingat_aktif": true, "pengingat_tanggal": 5, "pengingat_jam": 9,
	}
}

func TestTanpaSesiDitolak(t *testing.T) {
	db := testdb.DB(t)
	rt := router(db)
	for _, c := range []struct{ m, p string }{
		{"GET", "/tabungan"}, {"PUT", "/tabungan/target"}, {"DELETE", "/tabungan/target"},
		{"POST", "/tabungan/catatan"}, {"DELETE", "/tabungan/catatan/1"},
		{"PUT", "/perangkat-push"}, {"DELETE", "/perangkat-push"},
	} {
		if rec := panggil(t, rt, 0, c.m, c.p, nil); rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s: status %d, mau 401", c.m, c.p, rec.Code)
		}
	}
}

func TestAlurTargetDanCatatan(t *testing.T) {
	db := testdb.DB(t)
	rt := router(db)
	a, _ := buatAkun(t, db)

	rec := panggil(t, rt, a, "GET", "/tabungan", nil)
	if r := baca(t, rec); rec.Code != 200 || r.Target != nil || r.Ringkasan != nil || r.Catatan == nil || len(r.Catatan) != 0 {
		t.Fatalf("awal: %d %s", rec.Code, rec.Body.String())
	}

	// Catatan tanpa target: 409.
	if rec := panggil(t, rt, a, "POST", "/tabungan/catatan", map[string]any{"jumlah": 50000}); rec.Code != http.StatusConflict {
		t.Fatalf("tanpa target: %d, mau 409", rec.Code)
	}

	// Target ditolak bila ada batas terlanggar, paket tidak ada, atau JSON rusak.
	buruk := bodyTarget(t, db, 500)
	if rec := panggil(t, rt, a, "PUT", "/tabungan/target", buruk); rec.Code != 400 {
		t.Fatalf("target 500: %d", rec.Code)
	}
	tidakAda := bodyTarget(t, db, 60_000_000)
	tidakAda["schedule_id"] = 2_000_000_000
	if rec := panggil(t, rt, a, "PUT", "/tabungan/target", tidakAda); rec.Code != 400 {
		t.Fatalf("paket tidak ada: %d", rec.Code)
	}

	rec = panggil(t, rt, a, "PUT", "/tabungan/target", bodyTarget(t, db, 60_000_000))
	r := baca(t, rec)
	if rec.Code != 200 || r.Target == nil || r.Ringkasan == nil || r.Target.DibuatAt == "" || r.Target.TipeKamar != "Quad" {
		t.Fatalf("simpan target: %d %s", rec.Code, rec.Body.String())
	}
	if r.Ringkasan.Sisa != 60_000_000 || r.Ringkasan.SisaBulan != 6 || r.Ringkasan.DPTercapai {
		t.Fatalf("ringkasan awal: %+v", r.Ringkasan)
	}

	// Catatan: 201, tanggal default hari ini WIB, terbaru dulu.
	rec = panggil(t, rt, a, "POST", "/tabungan/catatan", map[string]any{"jumlah": 2_000_000})
	if rec.Code != http.StatusCreated {
		t.Fatalf("catatan: %d %s", rec.Code, rec.Body.String())
	}
	hariIni := time.Now().In(ZonaWIB).Format("2006-01-02")
	r = baca(t, rec)
	if len(r.Catatan) != 1 || r.Catatan[0].Tanggal != hariIni || r.Ringkasan.Tercatat != 2_000_000 || r.Ringkasan.TercatatBulanIni != 2_000_000 {
		t.Fatalf("setelah catatan: %+v %+v", r.Catatan, r.Ringkasan)
	}
	rec = panggil(t, rt, a, "POST", "/tabungan/catatan", map[string]any{"jumlah": 9_000_000, "tanggal": "2026-01-02"})
	r = baca(t, rec)
	if rec.Code != 201 || len(r.Catatan) != 2 || r.Catatan[0].Tanggal != hariIni || r.Catatan[1].Tanggal != "2026-01-02" {
		t.Fatalf("urutan: %d %+v", rec.Code, r.Catatan)
	}
	if r.Ringkasan.Tercatat != 11_000_000 || r.Ringkasan.TercatatBulanIni != 2_000_000 || !r.Ringkasan.DPTercapai {
		t.Fatalf("ringkasan: %+v", r.Ringkasan)
	}
	if rec := panggil(t, rt, a, "POST", "/tabungan/catatan", map[string]any{"jumlah": 100}); rec.Code != 400 {
		t.Fatalf("jumlah kecil: %d", rec.Code)
	}
	if rec := panggil(t, rt, a, "POST", "/tabungan/catatan", map[string]any{"jumlah": 5000, "tanggal": "2999-01-01"}); rec.Code != 400 {
		t.Fatalf("tanggal depan: %d", rec.Code)
	}

	// Ganti target tidak menghapus catatan.
	rec = panggil(t, rt, a, "PUT", "/tabungan/target", bodyTarget(t, db, 40_000_000))
	r = baca(t, rec)
	if rec.Code != 200 || r.Target.TargetTotal != 40_000_000 || len(r.Catatan) != 2 || r.Ringkasan.Tercatat != 11_000_000 {
		t.Fatalf("ganti target: %d %s", rec.Code, rec.Body.String())
	}
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM tabungan_target WHERE akun_id = ?`, a).Scan(&n)
	if n != 1 {
		t.Fatalf("satu target per akun, ada %d", n)
	}

	// Hapus catatan.
	id := r.Catatan[0].ID
	rec = panggil(t, rt, a, "DELETE", fmt.Sprintf("/tabungan/catatan/%d", id), nil)
	r = baca(t, rec)
	if rec.Code != 200 || len(r.Catatan) != 1 || r.Ringkasan.Tercatat != 9_000_000 {
		t.Fatalf("hapus catatan: %d %s", rec.Code, rec.Body.String())
	}

	// Hapus target menghapus semua catatan.
	if rec := panggil(t, rt, a, "DELETE", "/tabungan/target", nil); rec.Code != http.StatusNoContent {
		t.Fatalf("hapus target: %d", rec.Code)
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM tabungan_catatan WHERE akun_id = ?`, a).Scan(&n)
	if n != 0 {
		t.Fatalf("catatan tersisa %d", n)
	}
	r = baca(t, panggil(t, rt, a, "GET", "/tabungan", nil))
	if r.Target != nil || len(r.Catatan) != 0 {
		t.Fatalf("setelah hapus: %+v", r)
	}
}

// Jadwal draft, arsip, atau sudah berangkat ditolak dengan pesan yang sama seperti id yang tidak ada.
func TestJadwalTidakTayangDitolak(t *testing.T) {
	db := testdb.DB(t)
	rt := router(db)
	a, _ := buatAkun(t, db)
	b := bodyTarget(t, db, 30_000_000)
	b["schedule_id"] = 999_999_999
	rec := panggil(t, rt, a, "PUT", "/tabungan/target", b)
	if rec.Code != 400 {
		t.Fatalf("id tidak ada: %d", rec.Code)
	}
	pesanTidakAda := rec.Body.String()
	for nama, id := range map[string]int64{
		"draft":    buatJadwal(t, db, "draft", 90),
		"archived": buatJadwal(t, db, "archived", 90),
		"lewat":    buatJadwal(t, db, "published", -5),
	} {
		b["schedule_id"] = id
		rec := panggil(t, rt, a, "PUT", "/tabungan/target", b)
		if rec.Code != 400 || rec.Body.String() != pesanTidakAda {
			t.Fatalf("%s: %d %s (want %s)", nama, rec.Code, rec.Body.String(), pesanTidakAda)
		}
	}
	b["schedule_id"] = buatJadwal(t, db, "published", 0)
	if rec := panggil(t, rt, a, "PUT", "/tabungan/target", b); rec.Code != 200 {
		t.Fatalf("berangkat hari ini sah: %d %s", rec.Code, rec.Body.String())
	}
}

// Paket yang dihapus admin: target dibaca sebagai perkiraan (nama_paket tetap) dan bisa disimpan ulang.
func TestPaketDihapusJadiPerkiraan(t *testing.T) {
	db := testdb.DB(t)
	rt := router(db)
	a, _ := buatAkun(t, db)
	b := bodyTarget(t, db, 30_000_000)
	id := b["schedule_id"].(int64)
	if rec := panggil(t, rt, a, "PUT", "/tabungan/target", b); rec.Code != 200 {
		t.Fatalf("simpan: %d %s", rec.Code, rec.Body.String())
	}
	if _, err := db.Exec(`DELETE FROM schedules WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	rec := panggil(t, rt, a, "GET", "/tabungan", nil)
	r := baca(t, rec)
	if r.Target == nil || r.Target.Jenis != "perkiraan" || r.Target.ScheduleID != nil || r.Target.NamaPaket != "Paket Uji" {
		t.Fatalf("setelah hapus: %s", rec.Body.String())
	}
	var kirim map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &kirim)
	balik, _ := kirim["target"].(map[string]any)
	delete(balik, "dibuat_at")
	if rec := panggil(t, rt, a, "PUT", "/tabungan/target", balik); rec.Code != 200 {
		t.Fatalf("simpan ulang: %d %s", rec.Code, rec.Body.String())
	}
}

func TestPerkiraanTanpaJadwal(t *testing.T) {
	db := testdb.DB(t)
	rt := router(db)
	a, _ := buatAkun(t, db)
	b := bodyTarget(t, db, 30_000_000)
	b["jenis"] = "perkiraan"
	b["schedule_id"] = nil
	b["nama_paket"] = ""
	rec := panggil(t, rt, a, "PUT", "/tabungan/target", b)
	r := baca(t, rec)
	if rec.Code != 200 || r.Target == nil || r.Target.ScheduleID != nil || r.Target.Jenis != "perkiraan" {
		t.Fatalf("perkiraan: %d %s", rec.Code, rec.Body.String())
	}
}

// Akun B tidak bisa melihat atau menghapus catatan akun A; balasan 404 sama dengan catatan yang tidak ada.
func TestIDORCatatan(t *testing.T) {
	db := testdb.DB(t)
	rt := router(db)
	a, _ := buatAkun(t, db)
	b, _ := buatAkun(t, db)
	panggil(t, rt, a, "PUT", "/tabungan/target", bodyTarget(t, db, 60_000_000))
	panggil(t, rt, b, "PUT", "/tabungan/target", bodyTarget(t, db, 60_000_000))
	rec := panggil(t, rt, a, "POST", "/tabungan/catatan", map[string]any{"jumlah": 3_000_000})
	idA := baca(t, rec).Catatan[0].ID

	rec = panggil(t, rt, b, "DELETE", fmt.Sprintf("/tabungan/catatan/%d", idA), nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("B hapus catatan A: %d, mau 404", rec.Code)
	}
	tidakAda := panggil(t, rt, b, "DELETE", "/tabungan/catatan/2000000000", nil)
	if tidakAda.Code != http.StatusNotFound || tidakAda.Body.String() != rec.Body.String() {
		t.Fatalf("balasan harus sama: %d %q vs %q", tidakAda.Code, tidakAda.Body.String(), rec.Body.String())
	}
	rb := baca(t, panggil(t, rt, b, "GET", "/tabungan", nil))
	if len(rb.Catatan) != 0 || rb.Ringkasan.Tercatat != 0 {
		t.Fatalf("B melihat catatan A: %+v", rb.Catatan)
	}
	ra := baca(t, panggil(t, rt, a, "GET", "/tabungan", nil))
	if len(ra.Catatan) != 1 {
		t.Fatalf("catatan A hilang: %+v", ra.Catatan)
	}
	// Hapus target B tidak menyentuh catatan A.
	panggil(t, rt, b, "DELETE", "/tabungan/target", nil)
	ra = baca(t, panggil(t, rt, a, "GET", "/tabungan", nil))
	if len(ra.Catatan) != 1 || ra.Target == nil {
		t.Fatalf("hapus target B merusak A: %+v", ra)
	}
	if rec := panggil(t, rt, a, "DELETE", fmt.Sprintf("/tabungan/catatan/%d", idA), nil); rec.Code != 200 {
		t.Fatalf("A hapus catatan sendiri: %d", rec.Code)
	}
}

func jumlahToken(t *testing.T, db *sql.DB, akun int64) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM perangkat_push WHERE akun_id = ?`, akun).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPerangkatPush(t *testing.T) {
	db := testdb.DB(t)
	rt := router(db)
	a, _ := buatAkun(t, db)
	b, _ := buatAkun(t, db)
	u := strconv.FormatInt(time.Now().UnixNano(), 10)
	tok := func(i int) string { return fmt.Sprintf("ExponentPushToken[uji%s-%d]", u, i) }
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM perangkat_push WHERE token LIKE ?`, "ExponentPushToken[uji"+u+"%") })

	// Format salah dan platform salah ditolak.
	for _, body := range []map[string]any{
		{"token": "bukan-token", "platform": "ios"},
		{"token": tok(0), "platform": "windows"},
		{"token": "", "platform": "ios"},
	} {
		if rec := panggil(t, rt, a, "PUT", "/perangkat-push", body); rec.Code != 400 {
			t.Fatalf("%v: %d, mau 400", body, rec.Code)
		}
	}
	if jumlahToken(t, db, a) != 0 {
		t.Fatal("token salah tersimpan")
	}

	// Upsert: dua kali token sama tetap satu baris.
	for i := 0; i < 2; i++ {
		if rec := panggil(t, rt, a, "PUT", "/perangkat-push", map[string]any{"token": tok(1), "platform": "ios"}); rec.Code != 204 {
			t.Fatalf("simpan: %d", rec.Code)
		}
	}
	if jumlahToken(t, db, a) != 1 {
		t.Fatalf("upsert: %d baris", jumlahToken(t, db, a))
	}

	// Token milik akun lain pindah ke akun ini.
	if rec := panggil(t, rt, b, "PUT", "/perangkat-push", map[string]any{"token": tok(1), "platform": "android"}); rec.Code != 204 {
		t.Fatalf("pindah: %d", rec.Code)
	}
	if jumlahToken(t, db, a) != 0 || jumlahToken(t, db, b) != 1 {
		t.Fatalf("pindah akun: a=%d b=%d", jumlahToken(t, db, a), jumlahToken(t, db, b))
	}

	// Akun A tidak bisa menghapus token milik B.
	if rec := panggil(t, rt, a, "DELETE", "/perangkat-push", map[string]any{"token": tok(1)}); rec.Code != 204 {
		t.Fatalf("hapus token orang: %d", rec.Code)
	}
	if jumlahToken(t, db, b) != 1 {
		t.Fatal("token B terhapus oleh A")
	}
	if rec := panggil(t, rt, b, "DELETE", "/perangkat-push", map[string]any{"token": tok(1)}); rec.Code != 204 || jumlahToken(t, db, b) != 0 {
		t.Fatalf("hapus token sendiri: %d", rec.Code)
	}

	// Maksimal 10 token per akun; yang terlama dihapus.
	for i := 1; i <= 11; i++ {
		if rec := panggil(t, rt, a, "PUT", "/perangkat-push", map[string]any{"token": tok(i), "platform": "ios"}); rec.Code != 204 {
			t.Fatalf("token %d: %d", i, rec.Code)
		}
	}
	if n := jumlahToken(t, db, a); n != MaksTokenPerAkun {
		t.Fatalf("batas token: %d, mau 10", n)
	}
	var ada int
	_ = db.QueryRow(`SELECT COUNT(*) FROM perangkat_push WHERE token = ?`, tok(1)).Scan(&ada)
	if ada != 0 {
		t.Fatal("token terlama tidak dihapus")
	}
	_ = db.QueryRow(`SELECT COUNT(*) FROM perangkat_push WHERE token = ?`, tok(11)).Scan(&ada)
	if ada != 1 {
		t.Fatal("token terbaru hilang")
	}
}

// Anonimkan akun menghapus target, catatan, dan token push lewat HapusDataAkun.
func TestHapusDataAkun(t *testing.T) {
	db := testdb.DB(t)
	rt := router(db)
	a, _ := buatAkun(t, db)
	u := strconv.FormatInt(time.Now().UnixNano(), 10)
	tok := "ExponentPushToken[hapus" + u + "]"
	t.Cleanup(func() { _, _ = db.Exec(`DELETE FROM perangkat_push WHERE token = ?`, tok) })
	panggil(t, rt, a, "PUT", "/tabungan/target", bodyTarget(t, db, 60_000_000))
	panggil(t, rt, a, "POST", "/tabungan/catatan", map[string]any{"jumlah": 1_000_000})
	panggil(t, rt, a, "PUT", "/perangkat-push", map[string]any{"token": tok, "platform": "ios"})

	if err := HapusDataAkun(context.Background(), db, a); err != nil {
		t.Fatal(err)
	}
	for _, tabel := range []string{"tabungan_target", "tabungan_catatan", "perangkat_push"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM `+tabel+` WHERE akun_id = ?`, a).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != 0 {
			t.Errorf("%s: %d baris tersisa", tabel, n)
		}
	}
}
