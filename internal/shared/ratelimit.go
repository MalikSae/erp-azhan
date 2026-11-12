package shared

import (
	"sync"
	"time"
)

type windowHit struct {
	count int
	first time.Time
}

// WindowLimiter membatasi jumlah kejadian per key dalam jendela waktu tetap.
// Disimpan di memori; reset saat server restart.
type WindowLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*windowHit
}

func NewWindowLimiter(limit int, window time.Duration) *WindowLimiter {
	return &WindowLimiter{limit: limit, window: window, hits: make(map[string]*windowHit)}
}

// Take memeriksa batas DAN mencatat satu kejadian dalam satu kunci. Pakai ini, bukan pasangan Allow lalu
// Record: di antara keduanya, request paralel bisa lolos Allow semua sebelum satu pun tercatat (mis. selama
// bcrypt berjalan), sehingga batas bisa dilampaui. Mengembalikan false (tanpa mencatat) bila batas tercapai.
// Kejadian yang ternyata tidak perlu dihitung (mis. login berhasil) dikembalikan lewat Refund atau Reset.
func (l *WindowLimiter) Take(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked()
	h, ok := l.hits[key]
	if !ok || time.Since(h.first) > l.window {
		l.hits[key] = &windowHit{count: 1, first: time.Now()}
		return true
	}
	if h.count >= l.limit {
		return false
	}
	h.count++
	return true
}

// Refund mengembalikan satu kejadian yang sudah diambil lewat Take (mis. percobaan yang ternyata berhasil).
func (l *WindowLimiter) Refund(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	h, ok := l.hits[key]
	if !ok {
		return
	}
	if h.count <= 1 || time.Since(h.first) > l.window {
		delete(l.hits, key)
		return
	}
	h.count--
}

// pruneLocked membersihkan entri kedaluwarsa agar map tidak tumbuh tanpa batas. Pemanggil memegang l.mu.
func (l *WindowLimiter) pruneLocked() {
	if len(l.hits) <= 10000 {
		return
	}
	for k, h := range l.hits {
		if time.Since(h.first) > l.window {
			delete(l.hits, k)
		}
	}
}

// Record mencatat satu kejadian untuk key tanpa memeriksa batas. Untuk memeriksa sekaligus mencatat, pakai Take.
func (l *WindowLimiter) Record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pruneLocked()
	h, ok := l.hits[key]
	if !ok || time.Since(h.first) > l.window {
		l.hits[key] = &windowHit{count: 1, first: time.Now()}
		return
	}
	h.count++
}

// Reset menghapus catatan key (mis. setelah login berhasil).
func (l *WindowLimiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}
