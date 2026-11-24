package customer

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"erp-azhan/api/internal/testdb"
	"github.com/go-sql-driver/mysql"
)

const pepperUji = "pepper-uji-aktivasi-0123456789-abcdefghij"

func TestValidNIK(t *testing.T) {
	for nik, want := range map[string]bool{
		"3273010101900001": true, "": false, "123": false, "32730101019000010": false, "327301010190000a": false, " 3273010101900001": false,
	} {
		if got := ValidNIK(nik); got != want {
			t.Errorf("ValidNIK(%q) = %v, mau %v", nik, got, want)
		}
	}
}

func TestKonfigAktivasiDariEnv(t *testing.T) {
	cases := []struct {
		nama, aktif, pepper string
		wantErr, wantAktif  bool
	}{
		{"mati bawaan", "", "", false, false},
		{"mati walau pepper pendek", "false", "pendek", false, false},
		{"aktif tanpa pepper gagal start", "true", "", true, true},
		{"aktif pepper 31 byte gagal start", "true", strings.Repeat("a", 31), true, true},
		{"aktif pepper 32 byte lolos", "true", strings.Repeat("a", 32), false, true},
		{"angka 1 dihitung aktif", "1", strings.Repeat("a", 40), false, true},
	}
	for _, c := range cases {
		t.Run(c.nama, func(t *testing.T) {
			t.Setenv("AKTIVASI_JAMAAH_AKTIF", c.aktif)
			t.Setenv("AKTIVASI_PEPPER", c.pepper)
			cfg, err := KonfigAktivasiDariEnv()
			if (err != nil) != c.wantErr || cfg.Aktif != c.wantAktif {
				t.Fatalf("err=%v aktif=%v", err, cfg.Aktif)
			}
		})
	}
}

func TestMapDuplicateJamaahTertaut(t *testing.T) {
	dup := func(key string) error {
		return &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'x' for key '" + key + "'"}
	}
	if err := mapDuplicate(dup("customer_accounts.uq_customer_jamaah"), "op"); !errors.Is(err, ErrJamaahTertaut) || errors.Is(err, ErrEmailTaken) {
		t.Fatalf("uq_customer_jamaah = %v", err)
	}
	if err := mapDuplicate(dup("customer_accounts.uq_customer_brand_hp"), "op"); !errors.Is(err, ErrPhoneTaken) {
		t.Fatalf("hp = %v", err)
	}
	if err := mapDuplicate(dup("customer_accounts.uq_customer_brand_email"), "op"); !errors.Is(err, ErrEmailTaken) {
		t.Fatalf("email = %v", err)
	}
}

func TestAktivasiFlagMati404(t *testing.T) {
	h := NewAktivasiHandler(&Aktivasi{}, nil, AktivasiConfig{})
	rec := httptest.NewRecorder()
	h.Publik(rec, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("flag mati: %d", rec.Code)
	}
}

func TestAktivasiPepperPendekDitolak(t *testing.T) {
	a := NewAktivasi(nil, nil, []byte("pendek"))
	if _, err := a.Jalankan(context.Background(), AktivasiInput{BrandID: 1, KodeBooking: "AB1234", NIK: "3273010101900001"}); !errors.Is(err, errPepperPendek) {
		t.Fatalf("err = %v", err)
	}
}

// ---- fixture ----

type fxAkt struct {
	t        *testing.T
	db       *sql.DB
	brand    int64
	sched    int64
	bookings []int64
	jamaah   []int64
	akun     []int64
	mulai    time.Time
	seq      int64
}

func newFxAkt(t *testing.T) *fxAkt {
	t.Helper()
	db := testdb.DB(t)
	f := &fxAkt{t: t, db: db, mulai: time.Now().Add(-time.Minute)}
	if err := db.QueryRow(`SELECT s.id, s.brand_id FROM schedules s JOIN brands br ON br.id = s.brand_id
		WHERE br.kode_brand IS NOT NULL ORDER BY s.id LIMIT 1`).Scan(&f.sched, &f.brand); err != nil {
		t.Fatal(err)
	}
	f.resetHitung()
	t.Cleanup(f.bersihkan)
	return f
}

func (f *fxAkt) resetHitung() {
	if _, err := f.db.Exec(`DELETE FROM jamaah_aktivasi_hitung`); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fxAkt) ids(s []int64) string {
	if len(s) == 0 {
		return "0"
	}
	p := make([]string, len(s))
	for i, v := range s {
		p[i] = strconv.FormatInt(v, 10)
	}
	return strings.Join(p, ",")
}

func (f *fxAkt) bersihkan() {
	ex := func(q string) {
		if _, err := f.db.Exec(q); err != nil {
			f.t.Logf("bersihkan %q: %v", q, err)
		}
	}
	// Akun hasil aktivasi (id tidak diketahui test) ikut dicari lewat jamaah yang tertaut.
	rows, err := f.db.Query(`SELECT id FROM customer_accounts WHERE jamaah_id IN (` + f.ids(f.jamaah) + `)`)
	if err == nil {
		for rows.Next() {
			var id int64
			_ = rows.Scan(&id)
			f.akun = append(f.akun, id)
		}
		rows.Close()
	}
	ex(`DELETE FROM jamaah_aktivasi_hitung`)
	ex(`DELETE FROM jamaah_aktivasi_log WHERE brand_id = ` + strconv.FormatInt(f.brand, 10) + ` AND dibuat_pada >= '` + f.mulai.Format("2006-01-02 15:04:05") + `'`)
	ex(`DELETE FROM booking_pax WHERE booking_id IN (` + f.ids(f.bookings) + `)`)
	ex(`DELETE FROM bookings WHERE id IN (` + f.ids(f.bookings) + `)`)
	ex(`UPDATE jamaah SET direkrut_oleh_akun_id = NULL, dibuat_oleh_akun_id = NULL WHERE id IN (` + f.ids(f.jamaah) + `)`)
	ex(`DELETE FROM jamaah WHERE id IN (` + f.ids(f.jamaah) + `)`)
	ex(`UPDATE customer_accounts SET direkrut_oleh_akun_id = NULL, upline_akun_id = NULL, jamaah_id = NULL WHERE id IN (` + f.ids(f.akun) + `)`)
	ex(`DELETE FROM notifikasi WHERE akun_id IN (` + f.ids(f.akun) + `)`)
	ex(`DELETE FROM akun_kaitan_log WHERE akun_id IN (` + f.ids(f.akun) + `)`)
	ex(`DELETE FROM customer_accounts WHERE id IN (` + f.ids(f.akun) + `)`)
}

func (f *fxAkt) tx(fn func(tx *sql.Tx)) {
	f.t.Helper()
	tx, err := f.db.Begin()
	if err != nil {
		f.t.Fatal(err)
	}
	fn(tx)
	if err := tx.Commit(); err != nil {
		f.t.Fatal(err)
	}
}

func (f *fxAkt) agen(status string) int64 {
	var id int64
	f.tx(func(tx *sql.Tx) {
		id = testdb.NewAkun(f.t, tx, f.brand, testdb.AkunOpts{Nama: "AgenAkt", StatusAgen: status})
	})
	f.akun = append(f.akun, id)
	return id
}

func (f *fxAkt) akunBiasa(direkrut int64) int64 {
	var id int64
	f.tx(func(tx *sql.Tx) {
		id = testdb.NewAkun(f.t, tx, f.brand, testdb.AkunOpts{Nama: "AkunAkt", Direkrut: direkrut})
	})
	f.akun = append(f.akun, id)
	return id
}

func (f *fxAkt) nikBaru() string {
	f.seq++
	return fmt.Sprintf("98%014d", (time.Now().UnixNano()/1000+f.seq)%100000000000000)
}

// booking membuat satu jamaah ber-NIK dalam satu booking berstatus tertentu.
type bk struct {
	kode      string
	nik       string
	jamaahID  int64
	bookingID int64
	paxID     int64
}

// kunci: akun agen pemilik kunci jamaah (0 = belum ditentukan). tanpaAgen: kaitan tanpa_agen.
func (f *fxAkt) booking(status string, kunci int64, tanpaAgen bool) bk {
	f.t.Helper()
	var r bk
	r.nik = f.nikBaru()
	f.tx(func(tx *sql.Tx) {
		r.jamaahID = testdb.NewJamaah(f.t, tx, f.brand, testdb.JamaahOpts{Nama: "Jamaah Aktivasi", Direkrut: kunci, TanpaAgen: tanpaAgen, NIK: r.nik})
		testdb.Exec(f.t, tx, `UPDATE jamaah SET jenis_kelamin='P', tanggal_lahir='1990-02-03' WHERE id=?`, r.jamaahID)
		var pax []int64
		r.bookingID, pax = testdb.NewBookingPax(f.t, tx, f.sched, status, 1000, testdb.Pax{JamaahID: r.jamaahID})
		r.paxID = pax[0]
		if err := tx.QueryRow(`SELECT id_booking FROM bookings WHERE id=?`, r.bookingID).Scan(&r.kode); err != nil {
			f.t.Fatal(err)
		}
	})
	f.jamaah = append(f.jamaah, r.jamaahID)
	f.bookings = append(f.bookings, r.bookingID)
	return r
}

func (f *fxAkt) core() (*Aktivasi, *int32) {
	a := NewAktivasi(f.db, NewRepository(f.db), []byte(pepperUji))
	var jeda int32
	a.jedaGagal = func() { atomic.AddInt32(&jeda, 1) }
	return a, &jeda
}

func (f *fxAkt) handler(a *Aktivasi) *AktivasiHandler {
	return NewAktivasiHandler(a, a.repo, AktivasiConfig{Aktif: true, Pepper: []byte(pepperUji)})
}

var ipSeq int64

func ipBaru() string {
	n := atomic.AddInt64(&ipSeq, 1)
	return fmt.Sprintf("198.51.%d.%d:4000", (n/250)%250, n%250+1)
}

func kirim(fn http.HandlerFunc, ip, perangkat string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b))
	req.RemoteAddr = ip
	if perangkat != "" {
		req.Header.Set("X-Perangkat-Id", perangkat)
	}
	rec := httptest.NewRecorder()
	fn(rec, req)
	return rec
}

func (f *fxAkt) emailHP() (string, string) {
	f.seq++
	n := time.Now().UnixNano() + f.seq
	return fmt.Sprintf("akt-%d@uji.test", n), fmt.Sprintf("0856%08d", n%100000000)
}

func (f *fxAkt) simpanBody(b bk) map[string]any {
	email, hp := f.emailHP()
	return map[string]any{"brand_id": f.brand, "kode_booking": b.kode, "nik": b.nik, "simpan": true,
		"no_hp": hp, "email": email, "password": "rahasia-123"}
}

func (f *fxAkt) cekBody(b bk) map[string]any {
	return map[string]any{"brand_id": f.brand, "kode_booking": b.kode, "nik": b.nik}
}

func (f *fxAkt) akunDariJamaah(jamaahID int64) (id int64, direkrut sql.NullInt64) {
	err := f.db.QueryRow(`SELECT id, direkrut_oleh_akun_id FROM customer_accounts WHERE jamaah_id=?`, jamaahID).Scan(&id, &direkrut)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, sql.NullInt64{}
	}
	if err != nil {
		f.t.Fatal(err)
	}
	return id, direkrut
}

func (f *fxAkt) hitungJumlah(jenis string) int {
	var n sql.NullInt64
	if err := f.db.QueryRow(`SELECT SUM(jumlah) FROM jamaah_aktivasi_hitung WHERE jenis=?`, jenis).Scan(&n); err != nil {
		f.t.Fatal(err)
	}
	return int(n.Int64)
}

func (f *fxAkt) logKode(a *Aktivasi, kode string) (aksi, alasan []string) {
	rows, err := f.db.Query(`SELECT aksi, alasan FROM jamaah_aktivasi_log WHERE kode_hash=? ORDER BY id`, a.batas.hash("kode", f.brand, kode))
	if err != nil {
		f.t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var x, y string
		_ = rows.Scan(&x, &y)
		aksi, alasan = append(aksi, x), append(alasan, y)
	}
	return
}

// ---- cek dan simpan publik ----

func TestAktivasiCekDanSimpanPublik(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	K := f.agen("aktif")
	lain := f.agen("aktif")
	b := f.booking("dp", K, false)

	body := f.cekBody(b)
	body["kode_booking"] = strings.ToLower(b.kode)
	rec := kirim(h.Publik, ipBaru(), "", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("cek: %d %s", rec.Code, rec.Body)
	}
	var cek map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &cek)
	if len(cek) != 3 || cek["nama_lengkap"] != "Jamaah Aktivasi (uji)" || cek["jenis_kelamin"] != "P" || cek["tanggal_lahir"] != "1990-02-03" {
		t.Fatalf("isi cek = %v", cek)
	}
	if _, ada := cek["kolom_kode_agen"]; ada || strings.Contains(rec.Body.String(), b.nik) {
		t.Fatalf("cek membocorkan data: %s", rec.Body)
	}
	if id, _ := f.akunDariJamaah(b.jamaahID); id != 0 {
		t.Fatal("cek tidak boleh membuat akun")
	}

	var kodeLain string
	_ = f.db.QueryRow(`SELECT kode_referral FROM customer_accounts WHERE id=?`, lain).Scan(&kodeLain)
	sb := f.simpanBody(b)
	sb["kode_referral"] = kodeLain
	sb["kode_agen"] = "BUKAN1"
	rec = kirim(h.Publik, ipBaru(), "", sb)
	if rec.Code != http.StatusCreated {
		t.Fatalf("simpan: %d %s", rec.Code, rec.Body)
	}
	var out struct {
		Account      Account `json:"account"`
		SessionToken string  `json:"session_token"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &out)
	if out.SessionToken == "" || out.Account.ID == 0 {
		t.Fatalf("respons simpan = %s", rec.Body)
	}
	f.akun = append(f.akun, out.Account.ID)

	var jamaahID, direkrut sql.NullInt64
	var nama, jk, lahir string
	if err := f.db.QueryRow(`SELECT jamaah_id, direkrut_oleh_akun_id, nama_lengkap, jenis_kelamin, DATE_FORMAT(tanggal_lahir,'%Y-%m-%d')
		FROM customer_accounts WHERE id=?`, out.Account.ID).Scan(&jamaahID, &direkrut, &nama, &jk, &lahir); err != nil {
		t.Fatal(err)
	}
	if jamaahID.Int64 != b.jamaahID || direkrut.Int64 != K || nama != "Jamaah Aktivasi (uji)" || jk != "P" || lahir != "1990-02-03" {
		t.Fatalf("akun: jamaah=%v direkrut=%v nama=%q jk=%q lahir=%q", jamaahID, direkrut, nama, jk, lahir)
	}
	var kaitan string
	var kunci int64
	_ = f.db.QueryRow(`SELECT kaitan_status, direkrut_oleh_akun_id FROM jamaah WHERE id=?`, b.jamaahID).Scan(&kaitan, &kunci)
	if kaitan != "terikat_agen" || kunci != K {
		t.Fatalf("kunci jamaah berubah: %s %d", kaitan, kunci)
	}
	var alasanLog string
	var oleh sql.NullInt64
	if err := f.db.QueryRow(`SELECT alasan, diganti_oleh FROM akun_kaitan_log WHERE akun_id=?`, out.Account.ID).Scan(&alasanLog, &oleh); err != nil {
		t.Fatal(err)
	}
	if alasanLog != "aktivasi akun" || oleh.Valid {
		t.Fatalf("log kaitan = %q %v", alasanLog, oleh)
	}
	var isi string
	if err := f.db.QueryRow(`SELECT isi FROM notifikasi WHERE akun_id=? AND jenis='jamaah_aktivasi'`, K).Scan(&isi); err != nil {
		t.Fatalf("notifikasi pemilik kunci: %v", err)
	}
	if isi != "Jamaah Jamaah Aktivasi (uji) sudah mengaktifkan akun." {
		t.Fatalf("isi notifikasi = %q", isi)
	}
	aksi, _ := f.logKode(a, b.kode)
	if len(aksi) != 1 || aksi[0] != "aktivasi" {
		t.Fatalf("log aktivasi = %v", aksi)
	}
	if n := f.hitungJumlah("ip"); n != 0 {
		t.Fatalf("slot IP tidak dikembalikan saat sukses: %d", n)
	}

	// Aktivasi kedua untuk jamaah yang sama: galat generik.
	rec = kirim(h.Publik, ipBaru(), "", f.simpanBody(b))
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), KodeGalatAktivasiGagal) {
		t.Fatalf("aktivasi kedua: %d %s", rec.Code, rec.Body)
	}
}

func TestAktivasiValidasiIsianTanpaMenyentuhPenghitung(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	body := f.simpanBody(b)
	body["email"] = "bukan-email"
	rec := kirim(h.Publik, ipBaru(), "", body)
	if rec.Code != http.StatusBadRequest || strings.Contains(rec.Body.String(), KodeGalatAktivasiGagal) {
		t.Fatalf("email salah: %d %s", rec.Code, rec.Body)
	}
	if n := f.hitungJumlah("ip") + f.hitungJumlah("simpan_ip"); n != 0 {
		t.Fatalf("penghitung tersentuh: %d", n)
	}
}

// ---- galat generik ----

func TestAktivasiSemuaSebabGagalSama(t *testing.T) {
	f := newFxAkt(t)
	a, jeda := f.core()
	h := f.handler(a)
	K := f.agen("aktif")

	ok := f.booking("dp", K, false)
	batal := f.booking("batal", K, false)
	bayi := f.booking("dp", K, false)
	testdb.Exec(t, mustTx(t, f), `UPDATE booking_pax SET pax_type='infant' WHERE id=?`, bayi.paxID)
	commitLast(t, f)
	tertaut := f.booking("dp", K, false)
	akunLama := f.akunBiasa(0)
	testdb.Exec(t, mustTx(t, f), `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, tertaut.jamaahID, akunLama)
	commitLast(t, f)

	type kasus struct {
		nama, alasan string
		fn           http.HandlerFunc
		body         map[string]any
		logAda       bool
	}
	nikLain := f.nikBaru()
	kasuss := []kasus{
		{"kode salah", "", h.Publik, map[string]any{"brand_id": f.brand, "kode_booking": "QQQQQ9", "nik": ok.nik}, false},
		{"NIK salah", "nik_salah", h.Publik, map[string]any{"brand_id": f.brand, "kode_booking": ok.kode, "nik": nikLain}, true},
		{"NIK jamaah lain", "bukan_peserta", h.Publik, map[string]any{"brand_id": f.brand, "kode_booking": ok.kode, "nik": bayi.nik}, true},
		{"booking batal", "booking_batal", h.Publik, f.cekBody(batal), true},
		{"bayi", "bayi", h.Publik, f.cekBody(bayi), true},
		{"brand lain", "", h.Publik, map[string]any{"brand_id": f.brand + 1000, "kode_booking": ok.kode, "nik": ok.nik}, false},
		{"sudah tertaut", "sudah_tertaut", h.Publik, f.cekBody(tertaut), true},
		{"format NIK salah", "", h.Publik, map[string]any{"brand_id": f.brand, "kode_booking": ok.kode, "nik": "123"}, false},
	}
	var pertama *httptest.ResponseRecorder
	for _, c := range kasuss {
		kodeUji, _ := c.body["kode_booking"].(string)
		sebelum, _ := f.logKode(a, strings.ToUpper(kodeUji))
		rec := kirim(c.fn, ipBaru(), "", c.body)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("%s: status %d %s", c.nama, rec.Code, rec.Body)
		}
		if pertama == nil {
			pertama = rec
		} else if rec.Body.String() != pertama.Body.String() || rec.Header().Get("Content-Type") != pertama.Header().Get("Content-Type") {
			t.Fatalf("%s: body beda\n%s\n%s", c.nama, rec.Body, pertama.Body)
		}
		if !strings.Contains(rec.Body.String(), KodeGalatAktivasiGagal) || strings.Contains(rec.Body.String(), "nik") && strings.Contains(rec.Body.String(), "NIK salah") {
			t.Fatalf("%s: body %s", c.nama, rec.Body)
		}
		if rec.Header().Get("Retry-After") != "" {
			t.Fatalf("%s: Retry-After pada galat generik", c.nama)
		}
		if bodyKode, _ := c.body["kode_booking"].(string); c.logAda {
			_, alasan := f.logKode(a, bodyKode)
			if len(alasan) == 0 || alasan[len(alasan)-1] != c.alasan {
				t.Errorf("%s: alasan log = %v, mau %q", c.nama, alasan, c.alasan)
			}
		} else if sesudah, _ := f.logKode(a, strings.ToUpper(bodyKode)); len(sesudah) != len(sebelum) {
			t.Errorf("%s: kegagalan di kode yang tidak ada tidak boleh ditulis di log: %v", c.nama, sesudah)
		}
	}
	if int(atomic.LoadInt32(jeda)) != len(kasuss) {
		t.Fatalf("jeda acak terpanggil %d kali, mau %d", *jeda, len(kasuss))
	}
}

// mustTx / commitLast: tx sementara untuk memakai helper testdb yang menerima *sql.Tx.
var txTerakhir = map[*fxAkt]*sql.Tx{}

func mustTx(t *testing.T, f *fxAkt) *sql.Tx {
	t.Helper()
	tx, err := f.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	txTerakhir[f] = tx
	return tx
}

func commitLast(t *testing.T, f *fxAkt) {
	t.Helper()
	if err := txTerakhir[f].Commit(); err != nil {
		t.Fatal(err)
	}
}

// ---- batas ----

func TestAktivasiBatasIPDanRetryAfter(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	ip := "203.0.113.50:1111"
	for i := 0; i < 5; i++ {
		body := f.cekBody(b)
		body["nik"] = f.nikBaru()
		if rec := kirim(h.Publik, ip, "", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	rec := kirim(h.Publik, ip, "", f.cekBody(b))
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), KodeGalatAktivasiTerlaluSering) {
		t.Fatalf("percobaan ke-6: %d %s", rec.Code, rec.Body)
	}
	if ra, _ := strconv.Atoi(rec.Header().Get("Retry-After")); ra != 900 {
		t.Fatalf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
	// IP lain tidak ikut terkunci dan data benar tetap lolos.
	if rec := kirim(h.Publik, ipBaru(), "", f.cekBody(b)); rec.Code != http.StatusOK {
		t.Fatalf("IP lain: %d %s", rec.Code, rec.Body)
	}
}

func TestAktivasiBatasPerangkat(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	for i := 0; i < 5; i++ {
		body := f.cekBody(b)
		body["nik"] = f.nikBaru()
		if rec := kirim(h.Publik, ipBaru(), "dev-uji-1", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan %d: %d", i+1, rec.Code)
		}
	}
	if rec := kirim(h.Publik, ipBaru(), "dev-uji-1", f.cekBody(b)); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("perangkat ke-6: %d %s", rec.Code, rec.Body)
	}
	if rec := kirim(h.Publik, ipBaru(), "dev-uji-2", f.cekBody(b)); rec.Code != http.StatusOK {
		t.Fatalf("perangkat lain: %d", rec.Code)
	}
}

func TestNormalisasiIPAktivasi(t *testing.T) {
	cases := map[string]string{
		"203.0.113.9":                      "203.0.113.9",
		"203.0.113.9:4000":                 "203.0.113.9",
		"2001:db8:1:2:aaaa:bbbb:cccc:dddd": "2001:db8:1:2::",
		"2001:db8:1:2:1:2:3:4":             "2001:db8:1:2::",
		"[2001:db8:1:2::5]:8080":           "2001:db8:1:2::",
		"::ffff:203.0.113.9":               "203.0.113.9",
		"-":                                "-",
	}
	for in, want := range cases {
		if got := NormalisasiIPAktivasi(in); got != want {
			t.Errorf("NormalisasiIPAktivasi(%q) = %q, mau %q", in, got, want)
		}
	}
}

func TestAktivasiBatasIPv6SeBlok64(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	for i := 0; i < 5; i++ {
		body := f.cekBody(b)
		body["nik"] = f.nikBaru()
		ip := fmt.Sprintf("[2001:db8:77:9::%x]:5000", i+1)
		if rec := kirim(h.Publik, ip, "", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	body := f.cekBody(b)
	body["nik"] = f.nikBaru()
	if rec := kirim(h.Publik, "[2001:db8:77:9::ffff]:5000", "", body); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("alamat lain di blok /64 yang sama lolos: %d %s", rec.Code, rec.Body)
	}
	if rec := kirim(h.Publik, "[2001:db8:77:a::1]:5000", "", f.cekBody(b)); rec.Code != http.StatusOK {
		t.Fatalf("blok lain: %d %s", rec.Code, rec.Body)
	}
}

func TestAktivasiBatasPasanganPengamanUtama(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	salah := f.nikBaru()
	// Satu pasangan kode+NIK salah dari banyak IP dan perangkat: terkunci setelah 5.
	for i := 0; i < 5; i++ {
		body := f.cekBody(b)
		body["nik"] = salah
		if rec := kirim(h.Publik, ipBaru(), fmt.Sprint("dev-p-", i), body); rec.Code != http.StatusBadRequest {
			t.Fatalf("pasangan percobaan %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	body := f.cekBody(b)
	body["nik"] = salah
	rec := kirim(h.Publik, ipBaru(), "dev-p-baru", body)
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("pasangan ke-6: %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Retry-After") != "900" {
		t.Fatalf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
	// Pemilik kode yang benar tidak ikut terkunci oleh tebakan pihak lain pada kodenya.
	if rec := kirim(h.Publik, ipBaru(), "", f.cekBody(b)); rec.Code != http.StatusOK {
		t.Fatalf("pasangan benar ikut terkunci: %d %s", rec.Code, rec.Body)
	}
	// NIK korban dengan kode acak dari banyak IP: pasangan korban tidak terkunci.
	for i := 0; i < 15; i++ {
		body := map[string]any{"brand_id": f.brand, "kode_booking": fmt.Sprintf("ZQ%04d", 100+i), "nik": b.nik}
		if rec := kirim(h.Publik, ipBaru(), "", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("NIK percobaan %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	if rec := kirim(h.Publik, ipBaru(), "", f.cekBody(b)); rec.Code != http.StatusOK {
		t.Fatalf("NIK korban terkunci oleh kode acak: %d %s", rec.Code, rec.Body)
	}
}

func TestAktivasiBatasKodeHarianDanResetAdmin(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	// Tebakan NIK terdistribusi pada satu kode yang ada: terkunci setelah batas harian.
	for i := 0; i < batasKode24.batas; i++ {
		body := f.cekBody(b)
		body["nik"] = f.nikBaru()
		if rec := kirim(h.Publik, ipBaru(), fmt.Sprint("dev-k-", i), body); rec.Code != http.StatusBadRequest {
			t.Fatalf("kode percobaan %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	rec := kirim(h.Publik, ipBaru(), "dev-k-baru", f.cekBody(b))
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("kode terkunci: %d %s", rec.Code, rec.Body)
	}
	if rec.Header().Get("Retry-After") != "900" {
		t.Fatalf("Retry-After = %q", rec.Header().Get("Retry-After"))
	}
	aksi, _ := f.logKode(a, b.kode)
	var dikunci int
	for _, x := range aksi {
		if x == "kode_dikunci" {
			dikunci++
		}
	}
	if dikunci != 1 {
		t.Fatalf("log kode_dikunci = %d, mau 1", dikunci)
	}

	// Reset admin membuka kunci dan tercatat.
	reset := func(body map[string]any) *httptest.ResponseRecorder {
		raw, _ := json.Marshal(body)
		req := httptest.NewRequest(http.MethodPost, "/?brand_id="+fmt.Sprint(f.brand), bytes.NewReader(raw))
		w := httptest.NewRecorder()
		h.ResetHitung(w, req)
		return w
	}
	if rec := reset(map[string]any{}); rec.Code != http.StatusBadRequest {
		t.Fatalf("reset kosong: %d", rec.Code)
	}
	if rec := reset(map[string]any{"kode_booking": b.kode, "nik": b.nik}); rec.Code != http.StatusBadRequest {
		t.Fatalf("reset ganda: %d", rec.Code)
	}
	if rec := reset(map[string]any{"kode_booking": b.kode}); rec.Code != http.StatusOK {
		t.Fatalf("reset: %d %s", rec.Code, rec.Body)
	}
	if rec := kirim(h.Publik, ipBaru(), "", f.cekBody(b)); rec.Code != http.StatusOK {
		t.Fatalf("setelah reset: %d %s", rec.Code, rec.Body)
	}
	var n int
	_ = f.db.QueryRow(`SELECT COUNT(*) FROM jamaah_aktivasi_log WHERE aksi='reset_admin' AND kode_hash=?`, a.batas.hash("kode", f.brand, b.kode)).Scan(&n)
	if n != 1 {
		t.Fatalf("log reset_admin = %d", n)
	}
}

func TestAktivasiBatasNIKHarianDanResetAdmin(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	for i := 0; i < batasNIK24.batas; i++ {
		body := map[string]any{"brand_id": f.brand, "kode_booking": fmt.Sprintf("ZN%04d", i), "nik": b.nik}
		if rec := kirim(h.Publik, ipBaru(), "", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("NIK percobaan %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	if rec := kirim(h.Publik, ipBaru(), "", f.cekBody(b)); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("NIK terkunci: %d %s", rec.Code, rec.Body)
	}
	raw, _ := json.Marshal(map[string]any{"nik": b.nik})
	w := httptest.NewRecorder()
	h.ResetHitung(w, httptest.NewRequest(http.MethodPost, "/?brand_id="+fmt.Sprint(f.brand), bytes.NewReader(raw)))
	if w.Code != http.StatusOK {
		t.Fatalf("reset NIK: %d %s", w.Code, w.Body)
	}
	if rec := kirim(h.Publik, ipBaru(), "", f.cekBody(b)); rec.Code != http.StatusOK {
		t.Fatalf("setelah reset NIK: %d %s", rec.Code, rec.Body)
	}
}

// Kode yang tidak ada diperlakukan sama dengan kode yang ada: terkunci di hitungan yang sama, Retry-After sama.
func TestAktivasiKodeTidakAdaSamaDenganKodeAda(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	for i := 0; i < batasKode24.batas; i++ {
		body := map[string]any{"brand_id": f.brand, "kode_booking": "ZK0001", "nik": f.nikBaru()}
		if rec := kirim(h.Publik, ipBaru(), "", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	body := map[string]any{"brand_id": f.brand, "kode_booking": "ZK0001", "nik": f.nikBaru()}
	rec := kirim(h.Publik, ipBaru(), "", body)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "900" {
		t.Fatalf("kode tidak ada: %d Retry-After=%q", rec.Code, rec.Header().Get("Retry-After"))
	}
}

func TestAktivasiKodeAcakTidakMengunciKodeOrang(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	for i := 0; i < 60; i++ {
		body := map[string]any{"brand_id": f.brand, "kode_booking": fmt.Sprintf("YX%04d", i), "nik": f.nikBaru()}
		if rec := kirim(h.Publik, ipBaru(), "", body); rec.Code != http.StatusBadRequest {
			t.Fatalf("tebakan %d: %d", i, rec.Code)
		}
	}
	if rec := kirim(h.Publik, ipBaru(), "", f.cekBody(b)); rec.Code != http.StatusOK {
		t.Fatalf("kode orang terkunci oleh tebakan acak: %d %s", rec.Code, rec.Body)
	}
}

// Alarm brand hanya log: batas IP pengguna lain tidak diperketat.
func TestAktivasiAlarmBrandTidakMemperketatIP(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	b := f.booking("dp", 0, false)
	ctx := context.Background()
	for i := 0; i < batasBrand.batas+5; i++ {
		_, err := a.Jalankan(ctx, AktivasiInput{BrandID: f.brand, KodeBooking: fmt.Sprintf("WV%04d", i), NIK: f.nikBaru(), IP: fmt.Sprint("10.9.", i/250, ".", i%250)})
		if !errors.Is(err, ErrAktivasiGagal) {
			t.Fatalf("percobaan %d: %v", i, err)
		}
	}
	ip := "10.77.0.1"
	for i := 0; i < batasIP.batas; i++ {
		if _, err := a.Jalankan(ctx, AktivasiInput{BrandID: f.brand, KodeBooking: b.kode, NIK: f.nikBaru(), IP: ip}); !errors.Is(err, ErrAktivasiGagal) {
			t.Fatalf("IP percobaan %d: %v", i, err)
		}
	}
	var sering *ErrAktivasiTerlaluSering
	if _, err := a.Jalankan(ctx, AktivasiInput{BrandID: f.brand, KodeBooking: b.kode, NIK: f.nikBaru(), IP: ip}); !errors.As(err, &sering) {
		t.Fatalf("IP percobaan ke-%d: %v", batasIP.batas+1, err)
	}
}

func TestAktivasiSlotDikembalikanSaatSukses(t *testing.T) {
	f := newFxAkt(t)
	a, jeda := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	ip := "203.0.113.77:1"
	for i := 0; i < 8; i++ {
		if rec := kirim(h.Publik, ip, "dev-ok", f.cekBody(b)); rec.Code != http.StatusOK {
			t.Fatalf("cek %d: %d %s", i+1, rec.Code, rec.Body)
		}
	}
	if *jeda != 0 {
		t.Fatalf("jeda acak terpanggil di jalur sukses: %d", *jeda)
	}
	for _, jenis := range []string{"ip", "dev", "pasangan", "nik_24", "kode_24"} {
		if n := f.hitungJumlah(jenis); n != 0 {
			t.Errorf("penghitung %s = %d setelah semua sukses", jenis, n)
		}
	}
}

// ---- kode agen (diabaikan sejak 2026-11-24) ----

func TestAktivasiKodeAgen(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	K := f.agen("aktif")
	A := f.agen("aktif")
	var kodeA string
	_ = f.db.QueryRow(`SELECT kode_referral FROM customer_accounts WHERE id=?`, A).Scan(&kodeA)

	// Keputusan 2026-11-24: aktivasi tidak menerima kode agen maupun kode referral. Akun dari jamaah tanpa agen
	// lahir tanpa upline; hanya admin yang mengaitkannya. Kode yang tetap dikirim klien lama diabaikan tanpa galat.
	tanpaUpline := func(t *testing.T, b bk, sb map[string]any) {
		t.Helper()
		rec := kirim(h.Publik, ipBaru(), "", sb)
		if rec.Code != http.StatusCreated {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		id, direkrut := f.akunDariJamaah(b.jamaahID)
		var kaitan string
		var dir sql.NullInt64
		_ = f.db.QueryRow(`SELECT kaitan_status, direkrut_oleh_akun_id FROM jamaah WHERE id=?`, b.jamaahID).Scan(&kaitan, &dir)
		if id == 0 || direkrut.Valid || kaitan != "tanpa_agen" || dir.Valid {
			t.Fatalf("akun=%d direkrut=%v kaitan=%s jamaah.direkrut=%v", id, direkrut, kaitan, dir)
		}
	}
	t.Run("tanpa agen: kode ketik diabaikan, akun tanpa upline", func(t *testing.T) {
		b := f.booking("dp", 0, false)
		sb := f.simpanBody(b)
		sb["kode_agen"] = kodeA
		tanpaUpline(t, b, sb)
	})
	t.Run("tanpa agen: kode ketik tidak valid juga diabaikan tanpa galat", func(t *testing.T) {
		b := f.booking("dp", 0, false)
		sb := f.simpanBody(b)
		sb["kode_agen"] = "ZZZZZZ"
		tanpaUpline(t, b, sb)
	})
	t.Run("tanpa agen pilihan admin: kode referral diabaikan", func(t *testing.T) {
		b := f.booking("dp", 0, true)
		sb := f.simpanBody(b)
		sb["kode_referral"] = kodeA
		tanpaUpline(t, b, sb)
	})
	t.Run("terikat_agen: kode diabaikan penuh, tanpa galat kode agen", func(t *testing.T) {
		b := f.booking("dp", K, false)
		sb := f.simpanBody(b)
		sb["kode_agen"] = "ZZZZZZ"
		sb["kode_referral"] = kodeA
		rec := kirim(h.Publik, ipBaru(), "", sb)
		if rec.Code != http.StatusCreated {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		_, direkrut := f.akunDariJamaah(b.jamaahID)
		if direkrut.Int64 != K {
			t.Fatalf("direkrut = %v, mau pemilik kunci %d", direkrut, K)
		}
	})
	t.Run("pemilik kunci nonaktif tetap mengikat", func(t *testing.T) {
		N := f.agen("nonaktif")
		b := f.booking("lunas", N, false)
		rec := kirim(h.Publik, ipBaru(), "", f.simpanBody(b))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		_, direkrut := f.akunDariJamaah(b.jamaahID)
		if direkrut.Int64 != N {
			t.Fatalf("direkrut = %v, mau %d", direkrut, N)
		}
	})
}

func TestAktivasiEmailDanWAGandaTanpaAkunYatim(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	ada := f.akunBiasa(0)
	var email, hp string
	_ = f.db.QueryRow(`SELECT email, no_hp FROM customer_accounts WHERE id=?`, ada).Scan(&email, &hp)
	b := f.booking("baru", 0, false)

	sb := f.simpanBody(b)
	sb["email"] = email
	rec := kirim(h.Publik, ipBaru(), "", sb)
	if rec.Code != http.StatusConflict {
		t.Fatalf("email ganda: %d %s", rec.Code, rec.Body)
	}
	sb = f.simpanBody(b)
	sb["no_hp"] = hp
	rec = kirim(h.Publik, ipBaru(), "", sb)
	if rec.Code != http.StatusConflict {
		t.Fatalf("WA ganda: %d %s", rec.Code, rec.Body)
	}
	if id, _ := f.akunDariJamaah(b.jamaahID); id != 0 {
		t.Fatal("akun yatim setelah 409")
	}
	for _, kunci := range []string{"uq_customer_jamaah"} {
		if strings.Contains(rec.Body.String(), kunci) {
			t.Fatal("nama kunci bocor")
		}
	}
	// Setelah 409 jamaah tetap bisa aktivasi dengan data lain (slot dikembalikan).
	if rec := kirim(h.Publik, ipBaru(), "", f.simpanBody(b)); rec.Code != http.StatusCreated {
		t.Fatalf("ulang: %d %s", rec.Code, rec.Body)
	}
}

// ---- upline akun baru dan akun yang sudah ada ----

func TestAktivasiAkunBaruUplineDariKunci(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	K := f.agen("aktif")
	Kn := f.agen("nonaktif")
	tertautKe := func(id int64) sql.NullInt64 {
		var d sql.NullInt64
		_ = f.db.QueryRow(`SELECT jamaah_id FROM customer_accounts WHERE id=?`, id).Scan(&d)
		return d
	}

	t.Run("upline akun baru = pemilik kunci, tercatat, pemilik kunci diberi tahu", func(t *testing.T) {
		b := f.booking("dp", K, false)
		rec := kirim(h.Publik, ipBaru(), "", f.simpanBody(b))
		if rec.Code != http.StatusCreated {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		akun, d := f.akunDariJamaah(b.jamaahID)
		if akun == 0 || d.Int64 != K {
			t.Fatalf("akun=%d direkrut=%v", akun, d)
		}
		var n int
		_ = f.db.QueryRow(`SELECT COUNT(*) FROM akun_kaitan_log WHERE akun_id=? AND alasan='aktivasi akun'`, akun).Scan(&n)
		if n != 1 {
			t.Fatalf("log kaitan = %d", n)
		}
		_ = f.db.QueryRow(`SELECT COUNT(*) FROM notifikasi WHERE akun_id=? AND jenis='jamaah_aktivasi' AND kunci=?`, K, fmt.Sprint("aktivasi:", b.jamaahID)).Scan(&n)
		if n != 1 {
			t.Fatalf("notifikasi pemilik kunci = %d", n)
		}
	})
	t.Run("pemilik kunci nonaktif tetap mengikat", func(t *testing.T) {
		b := f.booking("dp", Kn, false)
		if rec := kirim(h.Publik, ipBaru(), "", f.simpanBody(b)); rec.Code != http.StatusCreated {
			t.Fatalf("%d %s", rec.Code, rec.Body)
		}
		if _, d := f.akunDariJamaah(b.jamaahID); d.Int64 != Kn {
			t.Fatalf("direkrut=%v", d)
		}
	})
	t.Run("akun yang sudah terdaftar tidak bisa didaftarkan sebagai jamaah", func(t *testing.T) {
		b := f.booking("dp", K, false)
		lama := f.akunBiasa(0)
		var email, hp string
		if err := f.db.QueryRow(`SELECT email, no_hp FROM customer_accounts WHERE id=?`, lama).Scan(&email, &hp); err != nil {
			t.Fatal(err)
		}
		body := f.simpanBody(b)
		body["email"] = email
		if rec := kirim(h.Publik, ipBaru(), "", body); rec.Code != http.StatusConflict {
			t.Fatalf("email sama: %d %s", rec.Code, rec.Body)
		}
		body = f.simpanBody(b)
		body["no_hp"] = hp
		if rec := kirim(h.Publik, ipBaru(), "", body); rec.Code != http.StatusConflict {
			t.Fatalf("WA sama: %d %s", rec.Code, rec.Body)
		}
		var d sql.NullInt64
		_ = f.db.QueryRow(`SELECT direkrut_oleh_akun_id FROM customer_accounts WHERE id=?`, lama).Scan(&d)
		if tertautKe(lama).Valid || d.Valid {
			t.Fatalf("akun lama berubah: tertaut=%v direkrut=%v", tertautKe(lama), d)
		}
		if id, _ := f.akunDariJamaah(b.jamaahID); id != 0 {
			t.Fatal("jamaah tertaut walau pendaftaran ditolak")
		}
	})
}

func TestAktivasiHapusAkunMelepasJamaah(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	rec := kirim(h.Publik, ipBaru(), "", f.simpanBody(b))
	if rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	akun, _ := f.akunDariJamaah(b.jamaahID)
	f.akun = append(f.akun, akun)
	tx := mustTx(t, f)
	if err := anonimkanAkun(context.Background(), tx, akun); err != nil {
		t.Fatal(err)
	}
	commitLast(t, f)
	if id, _ := f.akunDariJamaah(b.jamaahID); id != 0 {
		t.Fatal("jamaah masih tertaut setelah akun dihapus")
	}
	rec = kirim(h.Publik, ipBaru(), "", f.simpanBody(b))
	if rec.Code != http.StatusCreated {
		t.Fatalf("akun lain setelah hapus: %d %s", rec.Code, rec.Body)
	}
}

func TestAktivasiLogTanpaDataMentah(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	h := f.handler(a)
	b := f.booking("dp", 0, false)
	ip := "203.0.113.88:9"
	body := f.cekBody(b)
	body["nik"] = f.nikBaru()
	kirim(h.Publik, ip, "dev-log", body)
	if rec := kirim(h.Publik, ipBaru(), "", f.simpanBody(b)); rec.Code != http.StatusCreated {
		t.Fatalf("%d %s", rec.Code, rec.Body)
	}
	rows, err := f.db.Query(`SELECT COALESCE(kode_hash,''), COALESCE(nik_hash,''), COALESCE(ip_hash,''), COALESCE(device_hash,''), alasan, catatan
		FROM jamaah_aktivasi_log WHERE jamaah_id=? OR kode_hash=?`, b.jamaahID, a.batas.hash("kode", f.brand, b.kode))
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
		var kode, nik, ipH, dev, alasan, catatan string
		_ = rows.Scan(&kode, &nik, &ipH, &dev, &alasan, &catatan)
		for _, v := range []string{kode, nik, ipH, dev} {
			if v != "" && len(v) != 64 {
				t.Errorf("kolom hash bukan 64 karakter: %q", v)
			}
		}
		for _, mentah := range []string{b.kode, b.nik, "203.0.113.88", "dev-log"} {
			if strings.Contains(kode+nik+ipH+dev+alasan+catatan, mentah) {
				t.Errorf("log memuat data mentah %q", mentah)
			}
		}
	}
	if n < 2 {
		t.Fatalf("baris log = %d", n)
	}
}

// ---- race ----

func TestAktivasiRaceDuaAktivasiParalelSatuJamaah(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	K := f.agen("aktif")
	b := f.booking("dp", K, false)
	var wg sync.WaitGroup
	var sukses, gagal int32
	const n = 5 // sama dengan batas pasangan kode+NIK
	for i := 0; i < n; i++ {
		email, hp := f.emailHP()
		wg.Add(1)
		go func(i int, email, hp string) {
			defer wg.Done()
			_, err := a.Jalankan(context.Background(), AktivasiInput{BrandID: f.brand, KodeBooking: b.kode, NIK: b.nik, Simpan: true,
				NoHP: hp, Email: email, Password: "rahasia-123", IP: fmt.Sprint("10.1.0.", i)})
			switch {
			case err == nil:
				atomic.AddInt32(&sukses, 1)
			case errors.Is(err, ErrAktivasiGagal):
				atomic.AddInt32(&gagal, 1)
			default:
				t.Errorf("galat tak terduga: %v", err)
			}
		}(i, email, hp)
	}
	wg.Wait()
	var akun int
	_ = f.db.QueryRow(`SELECT COUNT(*) FROM customer_accounts WHERE jamaah_id=?`, b.jamaahID).Scan(&akun)
	if sukses != 1 || gagal != n-1 || akun != 1 {
		t.Fatalf("sukses=%d gagal=%d akun=%d, mau 1, %d, 1", sukses, gagal, akun, n-1)
	}
}

func TestAktivasiRaceDenganPembatalanBooking(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	for i := 0; i < 6; i++ {
		b := f.booking("dp", 0, false)
		email, hp := f.emailHP()
		var wg sync.WaitGroup
		var err error
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err = a.Jalankan(context.Background(), AktivasiInput{BrandID: f.brand, KodeBooking: b.kode, NIK: b.nik, Simpan: true,
				NoHP: hp, Email: email, Password: "rahasia-123", IP: fmt.Sprint("10.3.0.", i)})
		}()
		go func() {
			defer wg.Done()
			tx, e := f.db.Begin()
			if e != nil {
				t.Error(e)
				return
			}
			_, _ = tx.Exec(`UPDATE bookings SET status='batal' WHERE id=?`, b.bookingID)
			_, _ = tx.Exec(`UPDATE booking_pax SET pax_status='batal' WHERE booking_id=?`, b.bookingID)
			_ = tx.Commit()
		}()
		wg.Wait()
		akun, _ := f.akunDariJamaah(b.jamaahID)
		if err != nil && !errors.Is(err, ErrAktivasiGagal) {
			t.Fatalf("putaran %d: galat tak terduga %v", i, err)
		}
		if (err == nil) != (akun != 0) {
			t.Fatalf("putaran %d: err=%v akun=%d tidak konsisten", i, err, akun)
		}
	}
}

func TestAktivasiRace500ParalelTerkunciTepatDiAmbang(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	b := f.booking("dp", 0, false)
	var gagal, sering int32
	var wg sync.WaitGroup
	const n = 500
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_, err := a.Jalankan(context.Background(), AktivasiInput{BrandID: f.brand, KodeBooking: b.kode,
				NIK: fmt.Sprintf("97%014d", i+1), IP: fmt.Sprint("172.16.", i/200, ".", i%200), Perangkat: fmt.Sprint("dev-race-", i)})
			var s *ErrAktivasiTerlaluSering
			switch {
			case errors.Is(err, ErrAktivasiGagal):
				atomic.AddInt32(&gagal, 1)
			case errors.As(err, &s):
				atomic.AddInt32(&sering, 1)
			default:
				t.Errorf("galat tak terduga: %v", err)
			}
		}(i)
	}
	wg.Wait()
	if gagal != int32(batasKode24.batas) || sering != n-int32(batasKode24.batas) {
		t.Fatalf("gagal=%d terkunci=%d, mau %d dan %d", gagal, sering, batasKode24.batas, n-batasKode24.batas)
	}
	if got := f.hitungJumlah("kode_jam"); got != n {
		t.Logf("penghitung kode_jam = %d", got)
	}
}

func TestAktivasi1000BerurutanTerkunciPerKode(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	b := f.booking("dp", 0, false)
	var gagal, sering int
	for i := 0; i < 1000; i++ {
		_, err := a.Jalankan(context.Background(), AktivasiInput{BrandID: f.brand, KodeBooking: b.kode,
			NIK: fmt.Sprintf("96%014d", i+1), IP: fmt.Sprint("172.20.", i/200, ".", i%200), Perangkat: fmt.Sprint("dev-seq-", i)})
		var s *ErrAktivasiTerlaluSering
		switch {
		case errors.Is(err, ErrAktivasiGagal):
			gagal++
		case errors.As(err, &s):
			sering++
		default:
			t.Fatalf("percobaan %d: %v", i, err)
		}
	}
	if gagal != batasKode24.batas || sering != 1000-batasKode24.batas {
		t.Fatalf("gagal=%d terkunci=%d", gagal, sering)
	}
	// Data benar pun tetap terkunci selama jendela berjalan.
	if _, err := a.Jalankan(context.Background(), AktivasiInput{BrandID: f.brand, KodeBooking: b.kode, NIK: b.nik, IP: "172.30.0.1"}); err == nil {
		t.Fatal("kode terkunci tetapi cek lolos")
	}
}

func TestPembatasAmbilAtomikBerurutan(t *testing.T) {
	f := newFxAkt(t)
	a, _ := f.core()
	ctx := context.Background()
	kunci := a.batas.hash("ip", 0, "uji-urut")
	var slots []slotBatas
	for i := 1; i <= 7; i++ {
		s, urutan, ok, tunggu, err := a.batas.ambil(ctx, batasIP, kunci)
		if err != nil {
			t.Fatal(err)
		}
		slots = append(slots, s)
		if urutan != i || ok != (i <= batasIP.batas) || tunggu < time.Second || tunggu > batasIP.lebar+time.Second {
			t.Fatalf("ambil %d: urutan=%d ok=%v tunggu=%v", i, urutan, ok, tunggu)
		}
	}
	for _, s := range slots[:3] {
		if err := a.batas.kembalikan(ctx, s); err != nil {
			t.Fatal(err)
		}
	}
	if _, urutan, ok, _, _ := a.batas.ambil(ctx, batasIP, kunci); urutan != 5 || !ok {
		t.Fatalf("setelah kembalikan: urutan=%d ok=%v", urutan, ok)
	}
}
