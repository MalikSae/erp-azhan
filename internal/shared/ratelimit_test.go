package shared

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Regresi audit keamanan 2026-11-12: pemeriksaan dan pencatatan dulu terpisah (Allow lalu Record), sehingga
// request paralel lolos semua sebelum satu pun tercatat. Take harus atomik: dari banyak goroutine serentak,
// hanya `limit` yang lolos.
func TestWindowLimiterTakeAtomikSaatParalel(t *testing.T) {
	const limit, n = 5, 200
	l := NewWindowLimiter(limit, time.Minute)
	var lolos atomic.Int32
	var wg sync.WaitGroup
	mulai := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-mulai
			if l.Take("k") {
				lolos.Add(1)
			}
		}()
	}
	close(mulai)
	wg.Wait()
	if got := lolos.Load(); got != limit {
		t.Fatalf("lolos %d dari %d request paralel, want tepat %d", got, n, limit)
	}
}

func TestWindowLimiterRefundDanReset(t *testing.T) {
	l := NewWindowLimiter(2, time.Minute)
	if !l.Take("k") || !l.Take("k") {
		t.Fatal("dua slot pertama harus lolos")
	}
	if l.Take("k") {
		t.Fatal("slot ketiga harus ditolak")
	}
	l.Refund("k")
	if !l.Take("k") {
		t.Fatal("setelah Refund satu slot harus tersedia lagi")
	}
	l.Reset("k")
	if !l.Take("k") || !l.Take("k") {
		t.Fatal("setelah Reset batas harus kembali penuh")
	}
	l.Refund("tidak-ada") // tidak boleh panik

	// Jendela kedaluwarsa: Take membuka jendela baru.
	pendek := NewWindowLimiter(1, 10*time.Millisecond)
	if !pendek.Take("k") || pendek.Take("k") {
		t.Fatal("batas 1 per jendela")
	}
	time.Sleep(20 * time.Millisecond)
	if !pendek.Take("k") {
		t.Fatal("jendela baru harus lolos")
	}
}
