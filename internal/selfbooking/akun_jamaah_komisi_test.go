package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"erp-azhan/api/internal/agen"
	"erp-azhan/api/internal/customer"
	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/testdb"
)

const (
	komisiLangsungUji  = 1_000_000
	bonusPembinaanUji  = 300_000
	biayaAgenUji       = 500_000
	catatanBebasUjiAkt = "dibebaskan"
)

// jadwalKedua menyiapkan jadwal lain di brand yang sama (keberangkatan berbeda) yang bisa dipesan.
func jadwalKedua(t *testing.T, tx *sql.Tx, brand, pertama int64) int64 {
	t.Helper()
	var id int64
	if err := tx.QueryRow(`SELECT id FROM schedules WHERE brand_id=? AND id<>? ORDER BY id LIMIT 1`, brand, pertama).Scan(&id); err != nil {
		t.Fatalf("jadwal kedua: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE schedules SET status='published', berangkat_tanggal='2099-02-01', minimal_dp=0, seat_sisa=100,
		harga_quad=1000, harga_triple=1000, harga_double=1000 WHERE id=?`, id)
	return id
}

func setKomisiJadwal(t *testing.T, tx *sql.Tx, ids ...int64) {
	t.Helper()
	for _, id := range ids {
		testdb.Exec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=?, nominal_bonus_pembinaan=? WHERE id=?`, komisiLangsungUji, bonusPembinaanUji, id)
	}
}

// lunasKomisi menandai booking lunas dan menjalankan hook komisi seperti jalur produksi.
func lunasKomisi(t *testing.T, tx *sql.Tx, bookingID int64) {
	t.Helper()
	ctx := context.Background()
	if err := komisi.SnapshotNominal(ctx, tx, bookingID); err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	testdb.Exec(t, tx, `UPDATE bookings SET status='lunas' WHERE id=?`, bookingID)
	if err := komisi.ProcessBookingLunas(ctx, tx, bookingID); err != nil {
		t.Fatalf("ProcessBookingLunas: %v", err)
	}
}

// ledgerBooking: baris komisi aktif sebuah booking sebagai "penerima:jenis:nominal", terurut.
func ledgerBooking(t *testing.T, tx *sql.Tx, bookingID int64, nama map[int64]string) []string {
	t.Helper()
	rows, err := tx.Query(`SELECT akun_penerima_id, jenis, nominal FROM transaksi_komisi WHERE booking_id=? AND dibatalkan_at IS NULL`, bookingID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id int64
		var jenis string
		var nominal float64
		if err := rows.Scan(&id, &jenis, &nominal); err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprintf("%s:%s:%.0f", nama[id], jenis, nominal))
	}
	sort.Strings(out)
	return out
}

func wantLedger(t *testing.T, label string, got []string, want ...string) {
	t.Helper()
	sort.Strings(want)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("%s:\n got  %v\n want %v", label, got, want)
	}
}

// aktivasiUji meniru transaksi simpan aktivasi (customer.simpanTx): akun baru tertaut ke jamaah, upline dari kunci.
func aktivasiUji(t *testing.T, tx *sql.Tx, brand int64, nama string, jamaahID int64) int64 {
	t.Helper()
	ctx := context.Background()
	u := nikUji()
	akun, err := customer.NewRepository(nil).CreateTx(ctx, tx, brand, nama, "akt"+u+"@uji.test", "0878"+u[4:12], "x", 0, jamaahID, customer.Biodata{})
	if err != nil {
		t.Fatalf("CreateTx: %v", err)
	}
	if err := agen.IkatAkunDariKunciTx(ctx, tx, akun, jamaahID); err != nil {
		t.Fatalf("IkatAkunDariKunciTx: %v", err)
	}
	return akun
}

var ajuanUji = agen.AjukanRequest{Domisili: "Kota Bekasi", JenisKelamin: "L", TanggalLahir: "1990-05-12", SetujuSyaratKetentuan: true, SetujuMoU: true}

// Ibrahim (agen aktif) memesan Ahmad; Ahmad aktivasi, ajukan agen (otomatis aktif), lalu memesan untuk dirinya dan istri.
// Komisi: pendaftaran Ahmad ke Ibrahim (B1 dan B2, keberangkatan berbeda), istri ke Ahmad, pembinaan istri ke Ibrahim.
func TestKomisiAkunJamaahUjungKeUjung(t *testing.T) {
	tx, sched1, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	sched2 := jadwalKedua(t, tx, brand, sched1)
	setKomisiJadwal(t, tx, sched1, sched2)
	testdb.Exec(t, tx, `UPDATE brands SET biaya_pendaftaran_agen=? WHERE id=?`, biayaAgenUji, brand)

	ibrahim := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Ibrahim", StatusAgen: "aktif"})
	nama := map[int64]string{ibrahim: "Ibrahim"}
	nikA := nikUji()

	// 1. Ibrahim memesankan Ahmad: jamaah Ahmad dikunci ke Ibrahim.
	r1, err := processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: sched1, PemesanTidakIkut: true,
		PIC: PICInput{NamaLengkap: "Ahmad Fulan", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikA)}}, inisiator{accountID: ibrahim})
	if err != nil {
		t.Fatalf("B1: %v", err)
	}
	b1, jamaahA := r1.Booking.BookingID, r1.Jamaah.ID
	if st, by, _ := kaitan(t, tx, jamaahA); st != "terikat_agen" || by != ibrahim {
		t.Fatalf("jamaah Ahmad = %s/%d, want terikat ke Ibrahim", st, by)
	}

	// Tanpa aktivasi: pax Ahmad tidak berkomisi untuk Ahmad (komisi hanya ke Ibrahim saat lunas, dicek di bawah).
	testdb.Exec(t, tx, `UPDATE bookings SET status='dp' WHERE id=?`, b1)

	// 2. Ahmad aktivasi (E2): akun tertaut, direkrut = Ibrahim.
	ahmad := aktivasiUji(t, tx, brand, "Ahmad Fulan", jamaahA)
	nama[ahmad] = "Ahmad"
	var direkrut sql.NullInt64
	tx.QueryRow(`SELECT direkrut_oleh_akun_id FROM customer_accounts WHERE id=?`, ahmad).Scan(&direkrut)
	if direkrut.Int64 != ibrahim {
		t.Fatalf("direkrut Ahmad = %d, want Ibrahim", direkrut.Int64)
	}

	// 3. Ahmad ajukan agen: akun jamaah (B1 dp, terikat, upline = pemilik kunci), biaya bebas, aktif, nik_agen = NIK jamaah.
	if err := agen.AjukanTx(ctx, tx, ahmad, ajuanUji); err != nil {
		t.Fatalf("ajukan: %v", err)
	}
	var status string
	var nikAgen sql.NullString
	var upline sql.NullInt64
	tx.QueryRow(`SELECT status_agen, nik_agen, upline_akun_id FROM customer_accounts WHERE id=?`, ahmad).Scan(&status, &nikAgen, &upline)
	if status != "aktif" || nikAgen.String != nikA || upline.Int64 != ibrahim {
		t.Fatalf("Ahmad = %s nik=%q upline=%d, want aktif, NIK jamaah, upline Ibrahim", status, nikAgen.String, upline.Int64)
	}
	var bayar string
	tx.QueryRow(`SELECT status FROM pembayaran_pendaftaran_agen WHERE akun_id=?`, ahmad).Scan(&bayar)
	if bayar != catatanBebasUjiAkt {
		t.Fatalf("pembayaran = %s, want %s", bayar, catatanBebasUjiAkt)
	}

	// 4. Ahmad memesan untuk dirinya dan istri di keberangkatan lain.
	istriNIK := nikUji()
	r2, err := processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: sched2,
		PIC:     PICInput{NamaLengkap: "Ahmad Fulan", JenisKelamin: "L", RoomType: "Quad"},
		Anggota: []AnggotaInput{{PaxType: "reguler", NamaLengkap: "Istri Ahmad", JenisKelamin: "P", RoomType: quad(), NIK: nikp(istriNIK)}}},
		inisiator{accountID: ahmad})
	if err != nil {
		t.Fatalf("B2: %v", err)
	}
	b2 := r2.Booking.BookingID
	if r2.Jamaah.ID != jamaahA {
		t.Fatalf("pax diri B2 = jamaah %d, want jamaah tertaut %d", r2.Jamaah.ID, jamaahA)
	}
	pax := bacaPax(t, tx, b2)
	istri := pax[1]
	if st, by, _ := kaitan(t, tx, istri); st != "terikat_agen" || by != ahmad {
		t.Fatalf("istri = %s/%d, want terikat ke Ahmad", st, by)
	}
	if st, by, _ := kaitan(t, tx, jamaahA); st != "terikat_agen" || by != ibrahim {
		t.Fatalf("jamaah Ahmad setelah B2 = %s/%d, want tetap Ibrahim", st, by)
	}

	// Lunas B1 dan B2.
	lunasKomisi(t, tx, b1)
	lunasKomisi(t, tx, b2)
	wantLedger(t, "B1 pendaftaran Ahmad", ledgerBooking(t, tx, b1, nama), "Ibrahim:langsung:1000000")
	wantLedger(t, "B2 Ahmad dan istri", ledgerBooking(t, tx, b2, nama),
		"Ibrahim:langsung:1000000", // Ahmad sendiri: keberangkatan lain, komisi langsung kedua ke pemilik kunci
		"Ahmad:langsung:1000000",   // istri
		"Ibrahim:pembinaan:300000") // pembinaan istri ke upline Ahmad

	// Lunas ulang tidak menambah baris; repeat dan cashback tetap mati.
	if err := komisi.ProcessBookingLunas(ctx, tx, b2); err != nil {
		t.Fatal(err)
	}
	wantLedger(t, "B2 lunas ulang", ledgerBooking(t, tx, b2, nama),
		"Ibrahim:langsung:1000000", "Ahmad:langsung:1000000", "Ibrahim:pembinaan:300000")

	// Duplikat pada keberangkatan sama ditolak: Ahmad sudah ada di B1 (sched1).
	_, err = processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: sched1,
		PIC: PICInput{NamaLengkap: "Ahmad Fulan", JenisKelamin: "L", RoomType: "Quad"}}, inisiator{accountID: ahmad})
	if !errors.Is(err, ErrDuplicate) {
		t.Fatalf("booking sama keberangkatan: err=%v, want ErrDuplicate", err)
	}
}

// Tanpa aktivasi: akun yang bukan tertaut tidak mengubah komisi pax Ahmad; komisi tetap ke pemilik kunci.
func TestKomisiPaxTanpaAktivasiKePemilikKunci(t *testing.T) {
	tx, sched, brand, _ := setupAgenBooking(t)
	ctx := context.Background()
	setKomisiJadwal(t, tx, sched)
	ibrahim := testdb.NewAkun(t, tx, brand, testdb.AkunOpts{Nama: "Ibrahim", StatusAgen: "aktif"})
	nama := map[int64]string{ibrahim: "Ibrahim"}
	r, err := processBookingTx(ctx, tx, brand, BookingRequest{ScheduleID: sched, PemesanTidakIkut: true,
		PIC: PICInput{NamaLengkap: "Ahmad Fulan", JenisKelamin: "L", RoomType: "Quad", NIK: nikp(nikUji())}}, inisiator{accountID: ibrahim})
	if err != nil {
		t.Fatal(err)
	}
	lunasKomisi(t, tx, r.Booking.BookingID)
	wantLedger(t, "tanpa aktivasi", ledgerBooking(t, tx, r.Booking.BookingID, nama), "Ibrahim:langsung:1000000")
}
