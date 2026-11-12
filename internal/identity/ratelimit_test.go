package identity

import (
	"sync"
	"sync/atomic"
	"testing"
)

// Regresi audit keamanan 2026-11-12: percobaan login admin dipesan atomik sebelum bcrypt, sehingga request
// paralel tidak bisa melampaui loginAttemptLimit sebelum kegagalannya tercatat.
func TestReserveLoginAttemptAtomikSaatParalel(t *testing.T) {
	h := NewHandler(nil)
	const n = 100
	var lolos atomic.Int32
	var wg sync.WaitGroup
	mulai := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-mulai
			if ok, _ := h.reserveLoginAttempt("ip|admin@hana.id"); ok {
				lolos.Add(1)
			}
		}()
	}
	close(mulai)
	wg.Wait()
	if got := lolos.Load(); got != loginAttemptLimit {
		t.Fatalf("lolos %d dari %d percobaan paralel, want tepat %d", got, n, loginAttemptLimit)
	}

	// Percobaan yang dilepas (galat server) mengembalikan satu slot; login berhasil menghapus riwayat.
	h.releaseLoginAttempt("ip|admin@hana.id")
	if ok, _ := h.reserveLoginAttempt("ip|admin@hana.id"); !ok {
		t.Fatal("setelah release satu slot harus tersedia")
	}
	h.recordSuccessLogin("ip|admin@hana.id")
	if ok, _ := h.reserveLoginAttempt("ip|admin@hana.id"); !ok {
		t.Fatal("setelah login berhasil batas harus kembali penuh")
	}
}
