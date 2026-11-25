package testdb

import (
	"context"
	"database/sql"
	"strings"
	"testing"
)

// CloneSchedule menyalin satu jadwal (keberangkatan lain dengan data sama) dan mengembalikan id barunya.
func CloneSchedule(t *testing.T, tx *sql.Tx, scheduleID int64) int64 {
	t.Helper()
	rows, err := tx.QueryContext(context.Background(), `
		SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'schedules' AND COLUMN_NAME <> 'id' AND EXTRA NOT LIKE '%GENERATED%'
		ORDER BY ORDINAL_POSITION`)
	if err != nil {
		t.Fatalf("kolom schedules: %v", err)
	}
	var cols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatal(err)
		}
		cols = append(cols, "`"+c+"`")
	}
	rows.Close()
	list := strings.Join(cols, ",")
	res := mustExec(t, tx, "INSERT INTO schedules ("+list+") SELECT "+list+" FROM schedules WHERE id = ?", scheduleID)
	id, err := res.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	return id
}

// seatDipegang: seat yang dipegang booking jadwal (is_seat_blocked, bukan batal).
func seatDipegang(t *testing.T, tx *sql.Tx, scheduleID int64) (total, sisa, dipegang int) {
	t.Helper()
	if err := tx.QueryRowContext(context.Background(), `
		SELECT s.seat_total, s.seat_sisa,
		       COALESCE((SELECT SUM(b.seat_count) FROM bookings b WHERE b.schedule_id=s.id AND b.is_seat_blocked=1 AND b.status<>'batal'),0)
		FROM schedules s WHERE s.id=?`, scheduleID).Scan(&total, &sisa, &dipegang); err != nil {
		t.Fatalf("baca seat jadwal: %v", err)
	}
	return
}

// SamakanSisa mengisi seat_sisa = seat_total − seat dipegang booking. Fixture lain menambah
// kuota tanpa booking; panggil ini sebelum menguji rumus sisa seat.
func SamakanSisa(t *testing.T, tx *sql.Tx, scheduleID int64) {
	t.Helper()
	total, _, dipegang := seatDipegang(t, tx, scheduleID)
	mustExec(t, tx, `UPDATE schedules SET seat_sisa=? WHERE id=?`, total-dipegang, scheduleID)
}

// CekRumusSeat menggagalkan test bila seat_sisa ≠ seat_total − seat dipegang booking
// (tidak ada booking di luar sistem).
func CekRumusSeat(t *testing.T, tx *sql.Tx, scheduleID int64, langkah string) {
	t.Helper()
	total, sisa, dipegang := seatDipegang(t, tx, scheduleID)
	if sisa != total-dipegang {
		t.Fatalf("%s: seat_sisa=%d, want seat_total(%d) − dipegang(%d) = %d", langkah, sisa, total, dipegang, total-dipegang)
	}
}
