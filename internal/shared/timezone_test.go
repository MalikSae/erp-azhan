package shared_test

import (
	"testing"
	"time"

	"erp-azhan/api/internal/shared"
	"erp-azhan/api/internal/testdb"
)

// NOW() di MySQL dan waktu yang dibaca driver Go harus menunjuk ke instant
// yang sama. Sebelumnya NOW() berzona sistem (WIB) tetapi dibaca sebagai UTC,
// sehingga tanggal tampil maju 7 jam di browser.
func TestZonaWaktuDBKonsisten(t *testing.T) {
	_, tx := testdb.Tx(t)
	var now time.Time
	var tz string
	if err := tx.QueryRow(`SELECT NOW(), @@session.time_zone`).Scan(&now, &tz); err != nil {
		t.Fatal(err)
	}
	if selisih := time.Since(now); selisih < -time.Minute || selisih > time.Minute {
		t.Fatalf("NOW() dibaca %v, selisih %v dari waktu sebenarnya (session time_zone %s)", now, selisih, tz)
	}
	loc := shared.LoadConfig().Location()
	if want := time.Now().In(loc).Format("-07:00"); tz != want {
		t.Fatalf("session time_zone = %s, want %s", tz, want)
	}

	// Waktu yang ditulis dari Go lalu dibaca lagi tetap instant yang sama.
	ditulis := time.Now().Add(3 * time.Hour).Truncate(time.Second)
	var dibaca time.Time
	if err := tx.QueryRow(`SELECT CAST(? AS DATETIME)`, ditulis).Scan(&dibaca); err != nil {
		t.Fatal(err)
	}
	if !dibaca.Equal(ditulis) {
		t.Fatalf("round trip DATETIME %v -> %v", ditulis, dibaca)
	}
}
