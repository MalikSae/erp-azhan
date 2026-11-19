package selfbooking

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"erp-azhan/api/internal/customer"
	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

// bacaPax mengembalikan jamaah_id semua pax booking, urut pembuatan.
func bacaPax(t *testing.T, tx *sql.Tx, bookingID int64) []int64 {
	t.Helper()
	rows, err := tx.Query(`SELECT jamaah_id FROM booking_pax WHERE booking_id=? ORDER BY id`, bookingID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}
	return ids
}

// akunPemesanUji: agen aktif ber-NIK (upline opsional) atau akun biasa.
func akunPemesanUji(t *testing.T, tx *sql.Tx, brand int64, agen bool, upline int64) (akun int64, nik string) {
	t.Helper()
	nik = nikUji()
	if !agen {
		return testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunBiasa", Direkrut: upline}), nik
	}
	akun = testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenPemesan", StatusAgen: "aktif", Direkrut: upline, Upline: upline})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nik, akun)
	return akun, nik
}

// Satu aturan kunci untuk semua jalur: pemesan agen aktif mengunci ke dirinya, penanda saya_sendiri dan pembuat
// agen terisi; pemesan biasa mengikuti upline akun. Mode dari status akun, bukan jalur.
func TestKunciPemesanAgenDanBiasa(t *testing.T) {
	for _, ikut := range []bool{false, true} {
		t.Run(fmt.Sprintf("ikut-%v", ikut), func(t *testing.T) { kunciPemesan(t, ikut) })
	}
}

func kunciPemesan(t *testing.T, ikut bool) {
	tx, sched, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	up := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineAkun", StatusAgen: "aktif"})

	// Agen aktif.
	agen, nikAgen := akunPemesanUji(t, tx, brand, true, 0)
	pic := PICInput{NamaLengkap: "Pax Agen (uji)", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUji())}
	if ikut {
		pic = PICInput{NamaLengkap: "AgenPemesan (uji)", JenisKelamin: "L", RoomType: "Quad"}
		testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AgenPemesan", NIK: nikAgen})
	}
	req := BookingRequest{ScheduleID: sched, PemesanTidakIkut: !ikut, PIC: pic,
		Anggota: []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Anggota", JenisKelamin: "P", RoomType: quad(), NIK: nikp(nikUji())}}}
	resp, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: agen})
	if err != nil {
		t.Fatalf("agen ikut=%v: %v", ikut, err)
	}
	var pembuat sql.NullInt64
	var penanda sql.NullInt64
	tx.QueryRow(`SELECT dibuat_oleh_agen_akun_id FROM bookings WHERE id=?`, resp.Booking.BookingID).Scan(&pembuat)
	tx.QueryRow(`SELECT saya_sendiri_akun_id FROM booking_pax WHERE booking_id=? ORDER BY id LIMIT 1`, resp.Booking.BookingID).Scan(&penanda)
	if pembuat.Int64 != agen {
		t.Fatalf("agen ikut=%v: dibuat_oleh_agen_akun_id = %v, want %d", ikut, pembuat, agen)
	}
	if ikut != (penanda.Int64 == agen) {
		t.Fatalf("agen ikut=%v: saya_sendiri_akun_id = %v", ikut, penanda)
	}
	pax := bacaPax(t, tx, resp.Booking.BookingID)
	if st, by, sumber := kaitan(t, tx, pax[1]); st != "terikat_agen" || by != agen || sumber != "jalur1" {
		t.Fatalf("agen ikut=%v: anggota %s/%d/%s, want terikat_agen/%d/jalur1", ikut, st, by, sumber, agen)
	}
	if !ikut {
		if st, by, _ := kaitan(t, tx, pax[0]); st != "terikat_agen" || by != agen {
			t.Fatalf("pax pertama %s/%d, want terikat ke agen %d", st, by, agen)
		}
	}

	// Akun biasa dengan upline akun.
	biasa, _ := akunPemesanUji(t, tx, brand, false, up)
	req2 := BookingRequest{ScheduleID: sched, PemesanTidakIkut: true, PIC: PICInput{NamaLengkap: "Pax Biasa", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUji())}}
	resp2, err := processBookingTx(ctx, tx, brand, req2, inisiator{accountID: biasa})
	if err != nil {
		t.Fatalf("biasa: %v", err)
	}
	tx.QueryRow(`SELECT dibuat_oleh_agen_akun_id FROM bookings WHERE id=?`, resp2.Booking.BookingID).Scan(&pembuat)
	tx.QueryRow(`SELECT saya_sendiri_akun_id FROM booking_pax WHERE booking_id=?`, resp2.Booking.BookingID).Scan(&penanda)
	if pembuat.Valid || penanda.Valid {
		t.Fatalf("akun biasa: pembuat=%v penanda=%v, want NULL", pembuat, penanda)
	}
	if st, by, sumber := kaitan(t, tx, resp2.Jamaah.ID); st != "terikat_agen" || by != up || sumber != "jalur2" {
		t.Fatalf("akun biasa: %s/%d/%s, want terikat_agen/%d/jalur2", st, by, sumber, up)
	}
}

// Jamaah diri agen (NIK = nik_agen, atau jamaah tertaut ke akun) dikunci ke upline_akun_id agen, aktif maupun
// nonaktif. Jamaah lain tetap ke agen.
func TestKunciDiriAgenKeUplineAktifDanNonaktif(t *testing.T) {
	for _, statusUpline := range []string{"aktif", "nonaktif"} {
		for _, tertaut := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-tertaut-%v", statusUpline, tertaut), func(t *testing.T) { kunciDiriAgen(t, statusUpline, tertaut) })
		}
	}
}

func kunciDiriAgen(t *testing.T, statusUpline string, tertaut bool) {
	tx, sched, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	up := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Upline", StatusAgen: statusUpline})
	x, nikX := akunPemesanUji(t, tx, brand, true, up)
	diri := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AgenPemesan", NIK: nikX})
	if tertaut {
		// Tertaut ke akun: NIK profil agen bukan NIK jamaah ini; jamaah dikenali lewat tautan.
		testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=NULL, jamaah_id=? WHERE id=?`, diri, x)
	}
	req := BookingRequest{ScheduleID: sched, PIC: PICInput{NamaLengkap: "AgenPemesan (uji)", JenisKelamin: "L", RoomType: "Quad"}}
	resp, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: x})
	if err != nil {
		t.Fatalf("upline %s tertaut=%v: %v", statusUpline, tertaut, err)
	}
	if resp.Jamaah.ID != diri {
		t.Fatalf("jamaah = %d, want %d", resp.Jamaah.ID, diri)
	}
	if st, by, _ := kaitan(t, tx, diri); st != "terikat_agen" || by != up {
		t.Fatalf("upline %s tertaut=%v: diri %s/%d, want terikat_agen/%d", statusUpline, tertaut, st, by, up)
	}
}

// Jamaah yang sudah terkunci agen A tidak pindah ke agen B yang memesannya, lewat jalur apa pun.
func TestKunciTidakPindah(t *testing.T) {
	tx, sched, brand, b := setupAgenBooking(t)
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenA", StatusAgen: "aktif"})
	nik := nikUji()
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Milik A", Direkrut: a, NIK: nik})
	for _, tidakIkut := range []bool{true, true} {
		req := BookingRequest{ScheduleID: sched, PemesanTidakIkut: tidakIkut, PIC: PICInput{NamaLengkap: "Milik A (uji)", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nik)}}
		resp, err := processBookingTx(context.Background(), tx, brand, req, inisiator{accountID: b})
		if err != nil {
			t.Fatal(err)
		}
		testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, resp.Booking.BookingID)
		if st, by, _ := kaitan(t, tx, j); st != "terikat_agen" || by != a {
			t.Fatalf("kunci pindah: %s/%d, want terikat_agen/%d", st, by, a)
		}
	}
}

// Batas menurut status akun: agen aktif memakai batas hold seat agen tanpa batas 3 booking per hari; akun biasa
// tetap kena batas harian dan 9 seat per jadwal.
func TestBatasMenurutStatusAkun(t *testing.T) {
	tx, sched, brand, agen := setupAgenBooking(t)
	ctx := context.Background()
	booking := func(akun int64, seat int) error {
		req := BookingRequest{ScheduleID: sched, PemesanTidakIkut: true, PIC: PICInput{NamaLengkap: "Pax", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUji())}}
		for i := 1; i < seat; i++ {
			req.Anggota = append(req.Anggota, AnggotaInput{PaxType: "reguler", NamaLengkap: "Anggota", JenisKelamin: "L", RoomType: quad(), NIK: nikp(nikUji())})
		}
		_, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: akun})
		return err
	}
	// Agen: 4 booking x 5 seat = 20 seat (batas 20 per jadwal), lebih dari 3 booking per hari.
	for i := 0; i < 4; i++ {
		if err := booking(agen, 5); err != nil {
			t.Fatalf("agen booking %d: %v", i+1, err)
		}
	}
	if err := booking(agen, 1); !errors.Is(err, ErrBatasSeatAgen) {
		t.Fatalf("agen lewat 20 seat: %v, want ErrBatasSeatAgen", err)
	}
	// Akun biasa: 9 seat per jadwal.
	biasa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Biasa"})
	if err := booking(biasa, 5); err != nil {
		t.Fatalf("biasa 5 seat: %v", err)
	}
	if err := booking(biasa, 5); !errors.Is(err, ErrBatasSeatAkun) {
		t.Fatalf("biasa 10 seat: %v, want ErrBatasSeatAkun", err)
	}
	// Akun biasa: batas 3 booking per 24 jam.
	harian := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Harian"})
	for i := 0; i < 3; i++ {
		if err := booking(harian, 1); err != nil {
			t.Fatalf("harian %d: %v", i+1, err)
		}
	}
	if err := booking(harian, 1); !errors.Is(err, ErrBatasBookingHarian) {
		t.Fatalf("booking ke-4: %v, want ErrBatasBookingHarian", err)
	}
}

// request_key yang sama dari akun yang sama adalah jawaban ulang (jalur tidak ikut sidik jari). Akun yang
// berubah jadi agen antar retry dengan payload berbeda mendapat ErrRequestConflict.
func TestIdempotensiLintasJalurDanAkunJadiAgen(t *testing.T) {
	tx, sched, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	akun, nik := akunPemesanUji(t, tx, brand, false, 0)
	testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AkunBiasa", NIK: nik})
	req := BookingRequest{RequestKey: strings.Repeat("b", 40), TermsVersion: TermsVersion, TermsAccepted: true, ScheduleID: sched,
		PIC: PICInput{NamaLengkap: "AkunBiasa (uji)", NoHP: nomorUji(77), JenisKelamin: "L", RoomType: "Quad", NIK: &nik}}
	r1, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: akun})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: akun})
	if err != nil || !r2.Replayed || r2.Booking.BookingCode != r1.Booking.BookingCode {
		t.Fatalf("retry: %+v %v, want jawaban ulang", r2, err)
	}
	testdb.Exec(t, tx, `UPDATE customer_accounts SET status_agen='aktif', nik_agen=? WHERE id=?`, nik, akun)
	req.PIC.NIK = nil // app tidak lagi mengirim NIK untuk agen yang ikut berangkat
	if _, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: akun}); !errors.Is(err, ErrRequestConflict) {
		t.Fatalf("akun jadi agen antar retry: %v, want ErrRequestConflict", err)
	}
}

// jamaah_id (Jamaah Saya) hanya untuk agen aktif: akun biasa ditolak, agen harus cocok NIK dan nama.
func TestJamaahIDHanyaAgenAktif(t *testing.T) {
	tx, sched, brand, agen := setupAgenBooking(t)
	ctx := context.Background()
	nik := nikUji()
	milik := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Milik", Direkrut: agen, NIK: nik})
	req := BookingRequest{ScheduleID: sched, PemesanTidakIkut: true, PIC: PICInput{JamaahID: &milik, NamaLengkap: "Milik (uji)", JenisKelamin: "L", RoomType: "Quad", NIK: &nik}}
	biasa := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Biasa"})
	if _, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: biasa}); !errors.Is(err, ErrInputTidakValid) {
		t.Fatalf("akun biasa: %v, want ErrInputTidakValid", err)
	}
	resp, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: agen})
	if err != nil || resp.Jamaah.ID != milik {
		t.Fatalf("agen aktif: %v id=%v", err, resp)
	}
}

// Skenario acuan: Ibrahim (agen) mendaftarkan Ahmad; Ahmad menjadi agen (upline Ibrahim) lalu memesan dirinya
// dan istrinya di jadwal lain. Komisi: Ahmad -> Ibrahim, istri -> Ahmad, pembinaan istri -> Ibrahim.
func TestSkenarioIbrahimAhmadIstri(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	b1, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', nominal_komisi_langsung=1000000, nominal_bonus_pembinaan=300000 WHERE id=?`, b1)
	ibrahim := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Ibrahim", StatusAgen: "aktif"})
	nikAhmad := nikUji()

	// B1: Ibrahim mendaftarkan Ahmad (pemesan tidak ikut). Ahmad dikunci ke Ibrahim.
	r1, err := processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: b1, PemesanTidakIkut: true,
		PIC: PICInput{NamaLengkap: "Ahmad", JenisKelamin: "L", RoomType: "Quad", NIK: &nikAhmad}}, inisiator{accountID: ibrahim})
	if err != nil {
		t.Fatalf("B1: %v", err)
	}
	jAhmad := r1.Jamaah.ID
	if st, by, _ := kaitan(t, tx, jAhmad); st != "terikat_agen" || by != ibrahim {
		t.Fatalf("kunci Ahmad %s/%d, want terikat/%d", st, by, ibrahim)
	}
	var n int
	tx.QueryRow(`SELECT COUNT(*) FROM transaksi_komisi WHERE booking_id=?`, r1.Booking.BookingID).Scan(&n)
	if n != 0 {
		t.Fatalf("komisi tercatat saat daftar: %d", n)
	}
	ledger := func(bookingID int64) map[int64]map[string]float64 {
		testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, bookingID)
		if err := komisi.SnapshotNominal(ctx, tx, bookingID); err != nil {
			t.Fatal(err)
		}
		if err := komisi.ProcessBookingLunas(ctx, tx, bookingID); err != nil {
			t.Fatal(err)
		}
		rows, err := tx.Query(`SELECT akun_penerima_id, jenis, SUM(nominal) FROM transaksi_komisi WHERE booking_id=? GROUP BY akun_penerima_id, jenis`, bookingID)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		out := map[int64]map[string]float64{}
		for rows.Next() {
			var id int64
			var jenis string
			var v float64
			rows.Scan(&id, &jenis, &v)
			if out[id] == nil {
				out[id] = map[string]float64{}
			}
			out[id][jenis] = v
		}
		return out
	}
	// Ahmad belum jadi agen: B1 lunas, komisi langsung ke Ibrahim.
	got := ledger(r1.Booking.BookingID)
	if got[ibrahim]["langsung"] != 1_000_000 || len(got) != 1 {
		t.Fatalf("ledger B1 = %v, want hanya Ibrahim 1.000.000", got)
	}
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE id=?`, r1.Booking.BookingID)

	// Ahmad punya akun agen aktif: upline Ibrahim, nik_agen = NIK jamaahnya.
	ahmad := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AhmadAkun", StatusAgen: "aktif", Direkrut: ibrahim, Upline: ibrahim})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=?, jamaah_id=? WHERE id=?`, nikAhmad, jAhmad, ahmad) // tautan jamaah seperti hasil aktivasi (E2)

	// B2: Ahmad memesan dirinya dan istri.
	r2, err := processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: b1,
		PIC:     PICInput{NamaLengkap: "Ahmad", JenisKelamin: "L", RoomType: "Quad"},
		Anggota: []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Istri Ahmad", JenisKelamin: "P", RoomType: quad(), NIK: nikp(nikUji())}}},
		inisiator{accountID: ahmad})
	if err != nil {
		t.Fatalf("B2: %v", err)
	}
	if r2.Jamaah.ID != jAhmad {
		t.Fatalf("pax Ahmad = %d, want %d", r2.Jamaah.ID, jAhmad)
	}
	pax := bacaPax(t, tx, r2.Booking.BookingID)
	if st, by, _ := kaitan(t, tx, jAhmad); st != "terikat_agen" || by != ibrahim {
		t.Fatalf("Ahmad pindah kunci: %s/%d", st, by)
	}
	if st, by, _ := kaitan(t, tx, pax[1]); st != "terikat_agen" || by != ahmad {
		t.Fatalf("istri %s/%d, want terikat/%d", st, by, ahmad)
	}
	got = ledger(r2.Booking.BookingID)
	// B1 (jadwal sama, batal setelah lunas) sudah membayar komisi langsung Ahmad ke Ibrahim: B2 tidak membayar lagi (ADR 0004).
	if got[ibrahim]["langsung"] != 0 || got[ibrahim]["pembinaan"] != 300_000 || got[ahmad]["langsung"] != 1_000_000 || got[ahmad]["pembinaan"] != 0 {
		t.Fatalf("ledger B2 = %v, want Ibrahim pembinaan 300rb saja, Ahmad langsung 1jt", got)
	}
}

// akunAgenHandler: akun agen aktif untuk uji handler.
func akunAgenHandler() *customer.Account {
	a := *akunHandlerUji
	a.ID, a.StatusAgen = 91, "aktif"
	return &a
}

func postKe(h *Handler, fn func(http.ResponseWriter, *http.Request), path, body string, akun *customer.Account) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.RemoteAddr = "198.51.100.9:5555"
	req = req.WithContext(customer.ContextWithAccount(req.Context(), akun))
	rec := httptest.NewRecorder()
	fn(rec, req)
	return rec
}

// Alias portal agen: payload lama diterjemahkan (saya_sendiri -> pemesan ikut; bawaan tidak ikut), jalur app
// bawaan ikut. Akun yang bukan agen aktif ditolak 403 di alias.
func TestAliasPortalAgenMenerjemahkanPayload(t *testing.T) {
	h := newTestHandler(t, "", 0)
	var got []BookingRequest
	h.book = func(_ context.Context, _, _ int64, req BookingRequest) (*BookingResponse, error) {
		got = append(got, req)
		return &BookingResponse{Status: "success"}, nil
	}
	agen := akunAgenHandler()
	body := func(sendiri string) string {
		return `{"schedule_id":10,"pic":{"nama_lengkap":"Uji","jenis_kelamin":"L","room_type":"Quad","nik":"3201010101010001"` + sendiri + `},"request_key":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","terms_version":"` + TermsVersion + `","terms_accepted":true,"expected_total":30000000,"expected_dp":5000000}`
	}
	if rec := postKe(h, h.CreateBookingAgen, "/api/portal/agen/bookings", body(""), agen); rec.Code != http.StatusOK {
		t.Fatalf("alias default: %d %s", rec.Code, rec.Body.String())
	}
	if rec := postKe(h, h.CreateBookingAgen, "/api/portal/agen/bookings", body(`,"saya_sendiri":true`), agen); rec.Code != http.StatusOK {
		t.Fatalf("alias saya sendiri: %d %s", rec.Code, rec.Body.String())
	}
	if rec := postKe(h, h.CreateBookingApp, "/api/account/bookings-app", body(""), agen); rec.Code != http.StatusOK {
		t.Fatalf("app default: %d %s", rec.Code, rec.Body.String())
	}
	if len(got) != 3 || !got[0].PemesanTidakIkut || got[1].PemesanTidakIkut || got[2].PemesanTidakIkut {
		t.Fatalf("pemesan_tidak_ikut per jalur = %v %v %v, want true false false", got[0].PemesanTidakIkut, got[1].PemesanTidakIkut, got[2].PemesanTidakIkut)
	}
	if got[1].PIC.SayaSendiri {
		t.Fatal("saya_sendiri harus dinetralkan setelah diterjemahkan")
	}
	if rec := postKe(h, h.CreateBookingAgen, "/api/portal/agen/bookings", body(""), akunHandlerUji); rec.Code != http.StatusForbidden {
		t.Fatalf("alias akun biasa: %d %s, want 403", rec.Code, rec.Body.String())
	}
}

// Kontrak respons alias portal agen: bentuk lama, tanpa peringatan.
func TestAliasPortalAgenBentukResponsLama(t *testing.T) {
	h := newTestHandler(t, "", 0)
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		room := "Quad"
		return &BookingResponse{Status: "success",
			Booking: BookingSummary{BookingID: 1, InvoiceToken: "tok", BookingCode: "HNA-1", TotalHarga: 1, MinimalDP: 1, SeatHoldExpiresAt: "2099-01-01T00:00:00Z",
				PaxSummary: []PaxSummary{{Nama: "A", PaxType: "reguler", RoomType: &room, Harga: 1}}},
			Jamaah:       JamaahInfo{ID: 2, IDJamaah: "J-2"},
			BankAccounts: []BankAccountInfo{{ID: 3, BankName: "BCA", AccountNumber: "1", AccountHolder: "PT"}}}, nil
	}
	body := `{"schedule_id":10,"pic":{"nama_lengkap":"Uji","jenis_kelamin":"L","room_type":"Quad","nik":"3201010101010001"},"request_key":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","terms_version":"` + TermsVersion + `","terms_accepted":true,"expected_total":30000000,"expected_dp":5000000}`
	rec := postKe(h, h.CreateBookingAgen, "/api/portal/agen/bookings", body, akunAgenHandler())
	if rec.Code != http.StatusOK {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &top); err != nil {
		t.Fatal(err)
	}
	kunci := func(m map[string]json.RawMessage) []string {
		var ks []string
		for k := range m {
			ks = append(ks, k)
		}
		return ks
	}
	if len(top) != 4 {
		t.Fatalf("kunci atas = %v, want status, booking, jamaah, bank_accounts", kunci(top))
	}
	for _, k := range []string{"status", "booking", "jamaah", "bank_accounts"} {
		if _, ok := top[k]; !ok {
			t.Fatalf("kunci %q hilang: %v", k, kunci(top))
		}
	}
	if _, ok := top["peringatan"]; ok {
		t.Fatal("peringatan tidak boleh ada di respons")
	}
	var booking map[string]json.RawMessage
	json.Unmarshal(top["booking"], &booking)
	for _, k := range []string{"booking_id", "invoice_token", "booking_code", "total_harga", "minimal_dp", "seat_hold_expires_at", "pax_summary"} {
		if _, ok := booking[k]; !ok {
			t.Fatalf("booking.%s hilang", k)
		}
	}
	if len(booking) != 7 {
		t.Fatalf("kunci booking = %v", kunci(booking))
	}
	var jamaah map[string]json.RawMessage
	json.Unmarshal(top["jamaah"], &jamaah)
	if _, ok := jamaah["id"]; !ok || len(jamaah) != 2 {
		t.Fatalf("kunci jamaah = %v", kunci(jamaah))
	}
}

// Satu limiter: percobaan identitas gagal dari dua jalur dihitung dalam satu kunci per akun; agen aktif tidak
// kena batas booking harian 3; akun biasa tetap kena.
func TestHandlerSatuLimiterDanBatasHarianAgen(t *testing.T) {
	h := newTestHandler(t, "", 0)
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		return nil, errNIKBeda(1)
	}
	agen := akunAgenHandler()
	for i := 0; i < identitasGagalAccountLimit; i++ {
		fn, path := h.CreateBookingApp, "/api/account/bookings-app"
		if i%2 == 1 {
			fn, path = h.CreateBookingAgen, "/api/portal/agen/bookings"
		}
		if rec := postKe(h, fn, path, bodyCheckout, agen); rec.Code != http.StatusBadRequest {
			t.Fatalf("percobaan %d: %d %s", i+1, rec.Code, rec.Body.String())
		}
	}
	rec := postKe(h, h.CreateBookingApp, "/api/account/bookings-app", bodyCheckout, agen)
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), KodeIdentitasTerlaluSering) {
		t.Fatalf("got %d %s, want 429 identitas", rec.Code, rec.Body.String())
	}

	h2 := newTestHandler(t, "", 0)
	h2.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		return &BookingResponse{Status: "success"}, nil
	}
	for i := 0; i < bookingAccountSuccessLimit+3; i++ {
		if rec := postKe(h2, h2.CreateBookingApp, "/api/account/bookings-app", bodyCheckout, agen); rec.Code != http.StatusOK {
			t.Fatalf("agen booking %d: %d %s", i+1, rec.Code, rec.Body.String())
		}
	}
	for i := 0; i < bookingAccountSuccessLimit; i++ {
		if rec := postKe(h2, h2.CreateBookingApp, "/api/account/bookings-app", bodyCheckout, akunHandlerUji); rec.Code != http.StatusOK {
			t.Fatalf("biasa booking %d: %d %s", i+1, rec.Code, rec.Body.String())
		}
	}
	if rec := postKe(h2, h2.CreateBookingApp, "/api/account/bookings-app", bodyCheckout, akunHandlerUji); rec.Code != http.StatusTooManyRequests {
		t.Fatalf("biasa booking ke-4: %d %s, want 429", rec.Code, rec.Body.String())
	}
}

// jamaah_id dari akun biasa ditolak di gerbang handler.
func TestHandlerJamaahIDAkunBiasaDitolak(t *testing.T) {
	h := newTestHandler(t, "", 0)
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		t.Fatal("tidak boleh sampai ke booking")
		return nil, nil
	}
	body := strings.Replace(bodyCheckout, `"pic":{`, `"pemesan_tidak_ikut":true,"pic":{"jamaah_id":5,`, 1)
	if rec := postKe(h, h.CreateBookingApp, "/api/account/bookings-app", body, akunHandlerUji); rec.Code != http.StatusBadRequest {
		t.Fatalf("got %d %s, want 400", rec.Code, rec.Body.String())
	}
}

// Email konfirmasi dikirim untuk semua pemesan, termasuk agen aktif dan alias portal.
func TestEmailKonfirmasiUntukAgen(t *testing.T) {
	m := &mailTiruan{}
	h := handlerEmail(t, m, false)
	agen := akunAgenHandler()
	postKe(h, h.CreateBookingApp, "/api/account/bookings-app", bodyCheckout, agen)
	postKe(h, h.CreateBookingAgen, "/api/portal/agen/bookings", strings.Replace(bodyCheckout, `"request_key":"aaaa`, `"request_key":"bbbb`, 1), agen)
	h.mailWG.Wait()
	if m.count() != 2 {
		t.Fatalf("email terkirim %d, want 2", m.count())
	}
}

// Galat agen dikembalikan menurut status pemesan: kode BATAS_SEAT_AGEN dengan sisa hanya untuk agen aktif.
func TestGalatBatasMengikutiStatusPemesan(t *testing.T) {
	rec := httptest.NewRecorder()
	tulisGalatBooking(rec, ErrBatasSeatAkun, false)
	if rec.Code != http.StatusTooManyRequests || !strings.Contains(rec.Body.String(), KodeBatasSeatAkun) || strings.Contains(rec.Body.String(), KodeBatasSeatAgen) {
		t.Fatalf("akun biasa: %d %s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	tulisGalatBooking(rec, ErrNIKAgenBelumAda, false)
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Lengkapi NIK di profil agen.") {
		t.Fatalf("nik agen: %d %s", rec.Code, rec.Body.String())
	}
}

// nik_agen tidak diverifikasi: agen A ber-nik_agen = NIK jamaah belum_ditentukan buatan pelanggan B memesan
// dengan ikut berangkat. Jamaah B tidak terkunci, saya_sendiri_akun_id tidak terisi, booking ditolak.
func TestNIKAgenMilikOrangLainBukanDiriAgen(t *testing.T) {
	tx, sched, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	b := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "PelangganB"})
	nikB := nikUji()
	korban := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Korban", NIK: nikB})
	testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id=?`, b, korban)
	up := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "UplineA", StatusAgen: "aktif"})
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenA", StatusAgen: "aktif", Direkrut: up, Upline: up})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET nik_agen=? WHERE id=?`, nikB, a)

	req := BookingRequest{ScheduleID: sched, PIC: PICInput{NamaLengkap: "Korban (uji)", JenisKelamin: "L", RoomType: "Quad"}}
	if _, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: a}); !errors.Is(err, ErrSayaSendiriBukanAnda) {
		t.Fatalf("err = %v, want ErrSayaSendiriBukanAnda", err)
	}
	if st, by, _ := kaitan(t, tx, korban); st != "belum_ditentukan" || by != 0 {
		t.Fatalf("jamaah korban %s/%d, want belum_ditentukan tanpa pemilik", st, by)
	}
	var n int
	tx.QueryRow(`SELECT COUNT(*) FROM booking_pax WHERE jamaah_id=? OR saya_sendiri_akun_id=?`, korban, a).Scan(&n)
	if n != 0 {
		t.Fatalf("pax tertulis %d, want 0", n)
	}
}

// Jamaah tertaut ke akun lain bukan diri agen walau NIK-nya sama dengan nik_agen; jamaah buatan agen sendiri
// dengan NIK itu tetap diri.
func TestJamaahDiriAgenSyaratKepemilikan(t *testing.T) {
	tx, _, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	a, nik := akunPemesanUji(t, tx, brand, true, 0)
	p := pemesan{id: a, agen: true, nikAgen: nik}

	tertaut := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "Tertaut", NIK: nik})
	lain := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AkunLain"})
	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=? WHERE id=?`, tertaut, lain)
	if diri, err := jamaahDiriAgen(ctx, tx, p, tertaut); err != nil || diri {
		t.Fatalf("tertaut akun lain: diri=%v err=%v, want false", diri, err)
	}

	testdb.Exec(t, tx, `UPDATE customer_accounts SET jamaah_id=NULL WHERE id=?`, lain)
	testdb.Exec(t, tx, `UPDATE jamaah SET dibuat_oleh_akun_id=? WHERE id=?`, a, tertaut)
	if diri, err := jamaahDiriAgen(ctx, tx, p, tertaut); err != nil || !diri {
		t.Fatalf("buatan agen sendiri: diri=%v err=%v, want true", diri, err)
	}
}

// Agen aktif tanpa nik_agen lewat /bookings-app (ikut berangkat, bawaan app): ErrNIKAgenBelumAda dari booking,
// dipetakan handler menjadi 400 dengan teks "Lengkapi NIK di profil agen.".
func TestAgenTanpaNIKAgenLewatBookingApp(t *testing.T) {
	tx, sched, brand, _ := setupAgenBooking(t)
	a := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "AgenLama", StatusAgen: "aktif"})
	req := BookingRequest{ScheduleID: sched, PIC: PICInput{NamaLengkap: "AgenLama", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUji())}}
	if _, err := processBookingTx(context.Background(), tx, brand, req, inisiator{accountID: a}); !errors.Is(err, ErrNIKAgenBelumAda) {
		t.Fatalf("err = %v, want ErrNIKAgenBelumAda", err)
	}

	h := newTestHandler(t, "", 0)
	h.book = func(context.Context, int64, int64, BookingRequest) (*BookingResponse, error) {
		return nil, ErrNIKAgenBelumAda
	}
	rec := postKe(h, h.CreateBookingApp, "/api/account/bookings-app", bodyCheckout, akunAgenHandler())
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "Lengkapi NIK di profil agen.") {
		t.Fatalf("handler: %d %s, want 400 Lengkapi NIK di profil agen.", rec.Code, rec.Body.String())
	}
}

// Jawaban ulang hanya untuk booking yang masih hidup: booking batal atau holdnya dilepas ditolak (409 lewat handler).
func TestJawabanUlangBookingMati(t *testing.T) {
	for _, kasus := range []struct {
		nama, sql string
	}{
		{"batal", `UPDATE bookings SET status='batal' WHERE id=?`},
		{"hold dilepas", `UPDATE bookings SET is_seat_blocked=FALSE WHERE id=?`},
	} {
		t.Run(kasus.nama, func(t *testing.T) {
			tx, sched, brand, _ := setupAgenBooking(t)
			ctx := context.Background()
			akun, nik := akunPemesanUji(t, tx, brand, false, 0)
			testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "AkunBiasa", NIK: nik})
			req := BookingRequest{RequestKey: strings.Repeat("c", 40), TermsVersion: TermsVersion, TermsAccepted: true, ScheduleID: sched,
				PIC: PICInput{NamaLengkap: "AkunBiasa (uji)", NoHP: nomorUji(78), JenisKelamin: "L", RoomType: "Quad", NIK: &nik}}
			r1, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: akun})
			if err != nil {
				t.Fatal(err)
			}
			testdb.Exec(t, tx, kasus.sql, r1.Booking.BookingID)
			if _, err := processBookingTx(ctx, tx, brand, req, inisiator{accountID: akun}); !errors.Is(err, ErrBookingKedaluwarsa) {
				t.Fatalf("retry booking mati: %v, want ErrBookingKedaluwarsa", err)
			}
		})
	}
}
