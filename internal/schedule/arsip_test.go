package schedule

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Paket published yang berangkatnya sudah lewat diarsipkan; yang belum berangkat dan yang bukan published tidak disentuh.
func TestArsipkanLampau(t *testing.T) {
	db, tx := testdb.Tx(t)
	_ = db
	lampau, _ := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2000-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, lampau)

	n, err := arsipkanLampau(context.Background(), tx)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("paket lampau tidak diarsipkan (rows=%d)", n)
	}
	var status string
	if err := tx.QueryRowContext(context.Background(), `SELECT status FROM schedules WHERE id=?`, lampau).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "archived" {
		t.Fatalf("status = %q, want archived", status)
	}

	depan, _ := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-01-01"})
	testdb.Exec(t, tx, `UPDATE schedules SET status='published' WHERE id=?`, depan)
	if _, err := arsipkanLampau(context.Background(), tx); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(`SELECT status FROM schedules WHERE id=?`, depan).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "published" {
		t.Fatalf("paket yang belum berangkat berubah jadi %q", status)
	}
}
