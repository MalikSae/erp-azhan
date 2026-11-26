package booking

import (
	"context"
	"testing"

	"erp-azhan/api/internal/testdb"
)

// Audit Manasik M7 (keputusan Q6): penanda "sudah manasik" hanya diisi absensi manasik, bukan endpoint progress pax.
func TestProgressPaxTolakManasik(t *testing.T) {
	_, tx := testdb.Tx(t)
	ctx := context.Background()
	jadwal, brand := testdb.Schedule(t, tx, testdb.ScheduleOpts{BerangkatTanggal: "2099-06-01"})
	j := testdb.NewJamaah(t, tx, brand, testdb.JamaahOpts{Nama: "JamaahProgresMnsQzp"})
	b, pax := testdb.NewBookingPax(t, tx, jadwal, "dp", 1000, testdb.Pax{JamaahID: j})

	if _, ok := AllowedPaxProgressFields["manasik"]; ok {
		t.Error("manasik masih ada di AllowedPaxProgressFields")
	}
	if err := simpanProgressPax(ctx, tx, b, pax[0], 0, map[string]bool{"manasik": true}); err == nil {
		t.Error("progress manasik dari endpoint progress pax harus ditolak")
	}
	var hadir bool
	if err := tx.QueryRowContext(ctx, `SELECT progress_manasik FROM booking_pax WHERE id=?`, pax[0]).Scan(&hadir); err != nil {
		t.Fatal(err)
	}
	if hadir {
		t.Error("penanda manasik terisi dari endpoint progress pax")
	}
}
