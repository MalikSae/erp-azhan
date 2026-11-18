package itinerary

// Integration test itinerary khusus. Jalan hanya dengan ERP_TEST_DB=1 (testdb.DB melewati bila tidak).

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

func TestKhususSiklus(t *testing.T) {
	db := testdb.DB(t)
	ctx := context.Background()
	const paketID = 987654321 // tanpa FK, cukup id sembarang
	clean := func() { _, _ = db.Exec(`DELETE FROM itineraries WHERE schedule_id=?`, paketID) }
	clean()
	t.Cleanup(clean)

	days := []DayRequest{{Title: "H1", Location: "Jeddah", Activities: []Activity{{Text: "Tiba"}}}}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	id, err := SimpanKhusus(ctx, tx, paketID, "Paket Uji", days)
	if err != nil {
		t.Fatal(err)
	}
	// Upsert kedua memakai baris yang sama.
	id2, err := SimpanKhusus(ctx, tx, paketID, "Paket Uji 2", append(days, days[0]))
	if err != nil || id2 != id {
		t.Fatalf("upsert id=%d id2=%d err=%v", id, id2, err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(db)
	list, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range list {
		if it.ID == id {
			t.Fatal("itinerary pribadi muncul di daftar master")
		}
	}
	it, err := repo.GetByID(ctx, id)
	if err != nil || len(it.Days) != 2 {
		t.Fatalf("GetByID: %v days=%v", err, it)
	}

	if err := HapusKhusus(ctx, db, paketID, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.GetByID(ctx, id); err != ErrNotFound {
		t.Fatalf("setelah hapus: %v", err)
	}
}
