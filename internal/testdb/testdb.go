// Package testdb menyediakan fixture untuk integration test yang butuh MySQL.
//
// Test memakai database dari .env (biasanya erp_azhan_dev) di dalam satu
// transaksi yang SELALU di-rollback, jadi tidak meninggalkan data. Test
// dilewati kecuali ERP_TEST_DB=1:
//
//	$env:ERP_TEST_DB = '1'; go test ./internal/...
package testdb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"erp-azhan/api/internal/shared"
	"github.com/joho/godotenv"
)

// Tx membuka koneksi dan transaksi yang di-rollback saat test selesai.
func Tx(t *testing.T) (*sql.DB, *sql.Tx) {
	t.Helper()
	if os.Getenv("ERP_TEST_DB") != "1" {
		t.Skip("integration test dilewati: set ERP_TEST_DB=1 untuk menjalankan")
	}
	_, file, _, _ := runtime.Caller(0)
	_ = godotenv.Load(filepath.Join(filepath.Dir(file), "..", "..", ".env"))

	db, err := shared.NewDB(shared.LoadConfig())
	if err != nil {
		t.Fatalf("koneksi database: %v", err)
	}
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("begin tx: %v", err)
	}
	t.Cleanup(func() {
		_ = tx.Rollback()
		_ = db.Close()
	})
	// Exercise the new checkout schema without applying a persistent migration
	// or backfilling existing bookings. A connection-local temporary table shadows
	// any real table and disappears when this fixture closes its connection.
	migration, err := os.ReadFile(filepath.Join(filepath.Dir(file), "..", "..", "migrations", "065_booking_checkout.sql"))
	if err != nil {
		t.Fatal(err)
	}
	source := string(migration)
	start := strings.Index(source, "CREATE TABLE")
	end := strings.Index(source[start:], ";") + start
	ddl := strings.Replace(source[start:end], "CREATE TABLE IF NOT EXISTS", "CREATE TEMPORARY TABLE", 1)
	ddl = strings.Replace(ddl, ",\n FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE", "", 1)
	if _, err = tx.ExecContext(context.Background(), ddl); err != nil {
		t.Fatalf("temporary checkout fixture: %v", err)
	}
	return db, tx
}

// Booking adalah fixture booking satu pax di jadwal pertama yang brand-nya
// punya kode_brand. Minimal DP jadwal di-set ke MinimalDP dan kuota kursi
// ditambah 100 (semua di dalam transaksi, ikut di-rollback).
type Booking struct {
	ID         int64
	ScheduleID int64
	JamaahID   int64
	PaxID      int64
}

type BookingOpts struct {
	Status     string  // baru | dp | lunas | batal
	TotalHarga float64 // total tagihan
	MinimalDP  float64 // minimal DP per pax (schedules.minimal_dp)
	Blocked    bool    // is_seat_blocked
}

func NewBooking(t *testing.T, tx *sql.Tx, o BookingOpts) Booking {
	t.Helper()
	ctx := context.Background()
	var b Booking
	var brandID int64
	if err := tx.QueryRowContext(ctx, `
		SELECT s.id, s.brand_id FROM schedules s JOIN brands br ON br.id = s.brand_id
		WHERE br.kode_brand IS NOT NULL ORDER BY s.id LIMIT 1`).Scan(&b.ScheduleID, &brandID); err != nil {
		t.Fatalf("fixture schedule: %v", err)
	}
	mustExec(t, tx, `UPDATE schedules SET minimal_dp=?, seat_sisa=seat_sisa+100 WHERE id=?`, o.MinimalDP, b.ScheduleID)

	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1_000_000_000)
	res := mustExec(t, tx, `INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap) VALUES (?,?,?,?)`,
		brandID, "TST-"+suffix, "T"+suffix[len(suffix)-5:], "Jamaah Uji "+suffix)
	b.JamaahID, _ = res.LastInsertId()

	res = mustExec(t, tx, `INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, seat_count, status, is_seat_blocked, total_harga)
		VALUES (?,?,?,1,?,?,?)`, "TS"+suffix[len(suffix)-4:], b.ScheduleID, b.JamaahID, o.Status, o.Blocked, o.TotalHarga)
	b.ID, _ = res.LastInsertId()

	res = mustExec(t, tx, `INSERT INTO booking_pax (booking_id, jamaah_id, pax_type, room_type, harga_pax, counts_for_seat, pax_status)
		VALUES (?,?,'reguler','Quad',?,TRUE,'aktif')`, b.ID, b.JamaahID, o.TotalHarga)
	b.PaxID, _ = res.LastInsertId()
	return b
}

var seq int64

// uniq menghasilkan akhiran unik untuk kolom UNIQUE global (id_jamaah,
// kode_jamaah, id_booking) di dalam satu proses test.
func uniq() string {
	seq++
	return fmt.Sprintf("%05d", (time.Now().UnixNano()/1000+seq)%100000)
}

// ScheduleOpts mengatur jadwal fixture. Nominal nil = tidak ikut program Syiar.
type ScheduleOpts struct {
	KomisiLangsung   *float64
	BonusPembinaan   *float64
	BerangkatTanggal string // YYYY-MM-DD
}

// Schedule memakai jadwal pertama yang brand-nya punya kode_brand, lalu
// mengatur nominal komisi, tanggal berangkat, minimal DP 0, dan menambah kuota.
func Schedule(t *testing.T, tx *sql.Tx, o ScheduleOpts) (scheduleID, brandID int64) {
	t.Helper()
	if err := tx.QueryRowContext(context.Background(), `
		SELECT s.id, s.brand_id FROM schedules s JOIN brands br ON br.id = s.brand_id
		WHERE br.kode_brand IS NOT NULL ORDER BY s.id LIMIT 1`).Scan(&scheduleID, &brandID); err != nil {
		t.Fatalf("fixture schedule: %v", err)
	}
	mustExec(t, tx, `UPDATE schedules SET nominal_komisi_langsung=?, nominal_bonus_pembinaan=?,
		berangkat_tanggal=?, minimal_dp=0, seat_sisa=seat_sisa+100 WHERE id=?`,
		o.KomisiLangsung, o.BonusPembinaan, o.BerangkatTanggal, scheduleID)
	return scheduleID, brandID
}

// JamaahOpts mengatur jamaah fixture. Direkrut/Upline 0 = NULL.
type JamaahOpts struct {
	Nama       string
	StatusAgen string // default tidak_aktif
	Direkrut   int64
	Upline     int64
	TanpaAgen  bool
}

func NewJamaah(t *testing.T, tx *sql.Tx, brandID int64, o JamaahOpts) int64 {
	t.Helper()
	if o.StatusAgen == "" {
		o.StatusAgen = "tidak_aktif"
	}
	kaitan := "belum_ditentukan"
	var direkrut, upline any
	if o.Direkrut > 0 {
		direkrut, kaitan = o.Direkrut, "terikat_agen"
	} else if o.TanpaAgen {
		kaitan = "tanpa_agen"
	}
	if o.Upline > 0 {
		upline = o.Upline
	}
	u := uniq()
	res := mustExec(t, tx, `INSERT INTO jamaah (brand_id, id_jamaah, kode_jamaah, nama_lengkap,
		status_agen, direkrut_oleh_jamaah_id, upline_jamaah_id, kaitan_status) VALUES (?,?,?,?,?,?,?,?)`,
		brandID, "TST-"+u, "T"+u, o.Nama+" (uji)", o.StatusAgen, direkrut, upline, kaitan)
	id, _ := res.LastInsertId()
	return id
}

// Pax dalam booking fixture.
type Pax struct {
	JamaahID int64
	Batal    bool
}

// NewBookingPax membuat booking berisi beberapa pax di jadwal tertentu.
// Mengembalikan ID booking dan ID booking_pax sesuai urutan input.
func NewBookingPax(t *testing.T, tx *sql.Tx, scheduleID int64, status string, total float64, paxes ...Pax) (int64, []int64) {
	t.Helper()
	res := mustExec(t, tx, `INSERT INTO bookings (id_booking, schedule_id, pic_jamaah_id, seat_count, status, is_seat_blocked, total_harga)
		VALUES (?,?,?,?,?,TRUE,?)`, "TB"+uniq()[1:], scheduleID, paxes[0].JamaahID, len(paxes), status, total)
	bookingID, _ := res.LastInsertId()
	var paxIDs []int64
	for _, p := range paxes {
		st := "aktif"
		if p.Batal {
			st = "batal"
		}
		r := mustExec(t, tx, `INSERT INTO booking_pax (booking_id, jamaah_id, pax_type, room_type, harga_pax, counts_for_seat, pax_status)
			VALUES (?,?,'reguler','Quad',?,TRUE,?)`, bookingID, p.JamaahID, total/float64(len(paxes)), st)
		id, _ := r.LastInsertId()
		paxIDs = append(paxIDs, id)
	}
	return bookingID, paxIDs
}

// AddPayment menambah pembayaran dengan status tertentu, mengembalikan ID-nya.
func AddPayment(t *testing.T, tx *sql.Tx, bookingID int64, jumlah float64, status string) int64 {
	t.Helper()
	res := mustExec(t, tx, `INSERT INTO payments (booking_id, jumlah, status) VALUES (?,?,?)`, bookingID, jumlah, status)
	id, _ := res.LastInsertId()
	return id
}

// Status membaca status booking saat ini.
func Status(t *testing.T, tx *sql.Tx, bookingID int64) string {
	t.Helper()
	var s string
	if err := tx.QueryRowContext(context.Background(), `SELECT status FROM bookings WHERE id=?`, bookingID).Scan(&s); err != nil {
		t.Fatalf("baca status: %v", err)
	}
	return s
}

// Exec menjalankan query di transaksi test.
func Exec(t *testing.T, tx *sql.Tx, q string, args ...any) {
	t.Helper()
	mustExec(t, tx, q, args...)
}

func mustExec(t *testing.T, tx *sql.Tx, q string, args ...any) sql.Result {
	t.Helper()
	res, err := tx.ExecContext(context.Background(), q, args...)
	if err != nil {
		t.Fatalf("exec %q: %v", q, err)
	}
	return res
}
