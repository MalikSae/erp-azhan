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

// Allow mengembalikan false jika key sudah mencapai batas di jendela aktif.
func (l *WindowLimiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	h, ok := l.hits[key]
	if !ok {
		return true
	}
	if time.Since(h.first) > l.window {
		delete(l.hits, key)
		return true
	}
	return h.count < l.limit
}

// Record mencatat satu kejadian untuk key.
func (l *WindowLimiter) Record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Bersihkan entri kedaluwarsa agar map tidak tumbuh tanpa batas.
	if len(l.hits) > 10000 {
		for k, h := range l.hits {
			if time.Since(h.first) > l.window {
				delete(l.hits, k)
			}
		}
	}
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
