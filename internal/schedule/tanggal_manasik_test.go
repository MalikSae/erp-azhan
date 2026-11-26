package schedule

import (
	"context"
	"errors"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit Manasik M11: tanggal berangkat paket bermanasik tidak boleh dimajukan ke tanggal manasik atau sebelumnya,
// termasuk paket tanpa booking aktif.
func TestUbahTanggalBerangkatTerhadapManasik(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	sched, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-02-01"})
	testdb.Exec(t, tx, `UPDATE bookings SET status='batal' WHERE schedule_id=?`, sched)
	testdb.Exec(t, tx, `DELETE FROM manasik_jadwal WHERE schedule_id=?`, sched)
	res, err := tx.ExecContext(ctx, `INSERT INTO manasik (brand_id, judul, tanggal, jam_mulai, jam_selesai, tempat, alamat)
		VALUES (?, 'Manasik Uji', '2099-01-15', '08:00', '12:00', 'Aula', 'Jl. Uji')`, brand)
	if err != nil {
		t.Fatal(err)
	}
	mid, _ := res.LastInsertId()
	testdb.Exec(t, tx, `INSERT INTO manasik_jadwal (schedule_id, manasik_id) VALUES (?, ?)`, sched, mid)

	ubah := func(baru string) error {
		return ubahTanggalBerangkatTx(ctx, tx, sched, "2099-02-01", &ScheduleInput{BerangkatTanggal: baru})
	}
	for _, baru := range []string{"2099-01-15", "2099-01-10"} {
		if err := ubah(baru); !errors.Is(err, ErrConflict) {
			t.Errorf("berangkat %s (manasik 2099-01-15): %v", baru, err)
		}
	}
	if err := ubah("2099-01-16"); err != nil {
		t.Errorf("berangkat setelah manasik: %v", err)
	}
}
