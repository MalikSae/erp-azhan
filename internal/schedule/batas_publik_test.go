package schedule

import (
	"testing"
	"time"
)

// Paket tampil dan bisa dipesan hanya bila berangkat 30 hari atau lebih dari hari ini (WIB).
func TestBatasPublikWIB(t *testing.T) {
	want := time.Now().In(time.FixedZone("WIB", 7*60*60)).AddDate(0, 0, 30).Format("2006-01-02")
	if got := batasPublikWIB(); got != want {
		t.Fatalf("batasPublikWIB() = %s, want %s", got, want)
	}
	if "2000-01-01" >= batasPublikWIB() {
		t.Fatal("tanggal lampau harus berada di bawah batas")
	}
}
