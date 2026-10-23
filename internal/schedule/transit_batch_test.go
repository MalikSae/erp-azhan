package schedule

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Butir 7g: hotel transit beberapa paket diambil dalam satu query, urutan per paket terjaga.
func TestTransitHotelsBySchedule(t *testing.T) {
	_, tx := testdb.Tx(t)
	var a, b int64
	if err := tx.QueryRow(`SELECT id FROM schedules ORDER BY id LIMIT 1`).Scan(&a); err != nil {
		t.Skip("butuh data jadwal")
	}
	if err := tx.QueryRow(`SELECT id FROM schedules WHERE id<>? ORDER BY id LIMIT 1`, a).Scan(&b); err != nil {
		t.Skip("butuh dua jadwal")
	}
	var h1, h2 int64
	if err := tx.QueryRow(`SELECT id FROM hotels ORDER BY id LIMIT 1`).Scan(&h1); err != nil {
		t.Skip("butuh data hotel")
	}
	if err := tx.QueryRow(`SELECT id FROM hotels WHERE id<>? ORDER BY id LIMIT 1`, h1).Scan(&h2); err != nil {
		t.Skip("butuh dua hotel")
	}
	testdb.Exec(t, tx, `DELETE FROM schedule_transit_hotels WHERE schedule_id IN (?,?)`, a, b)
	testdb.Exec(t, tx, `INSERT INTO schedule_transit_hotels (schedule_id, hotel_id, urutan) VALUES (?,?,2),(?,?,1),(?,?,1)`, a, h1, a, h2, b, h1)

	got, err := transitHotelsBySchedule(context.Background(), tx, []int64{a, b, -1})
	if err != nil {
		t.Fatal(err)
	}
	if len(got[a]) != 2 || got[a][0].HotelID != h2 || got[a][1].HotelID != h1 {
		t.Fatalf("jadwal %d: %+v, want [%d %d]", a, got[a], h2, h1)
	}
	if len(got[b]) != 1 || got[b][0].HotelID != h1 || len(got[-1]) != 0 {
		t.Fatalf("hasil salah: %+v", got)
	}
	if empty, err := transitHotelsBySchedule(context.Background(), tx, nil); err != nil || len(empty) != 0 {
		t.Fatalf("tanpa id: %v %v", empty, err)
	}
}
