package selfbooking

import (
	"sync"
	"time"
)

// windowLimiter membatasi jumlah kejadian per key dalam jendela waktu tetap.
// Disimpan di memori; reset saat server restart.
type windowLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	hits   map[string]*checkAttempt
}

func newWindowLimiter(limit int, window time.Duration) *windowLimiter {
	return &windowLimiter{limit: limit, window: window, hits: make(map[string]*checkAttempt)}
}

// allow mengembalikan false jika key sudah mencapai batas di jendela aktif.
func (l *windowLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	attempt, exists := l.hits[key]
	if !exists {
		return true
	}
	if time.Since(attempt.FirstFail) > l.window {
		delete(l.hits, key)
		return true
	}
	return attempt.Count < l.limit
}

// record mencatat satu kejadian untuk key.
func (l *windowLimiter) record(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Bersihkan entri kedaluwarsa agar map tidak tumbuh tanpa batas.
	if len(l.hits) > 10000 {
		for k, a := range l.hits {
			if time.Since(a.FirstFail) > l.window {
				delete(l.hits, k)
			}
		}
	}

	attempt, exists := l.hits[key]
	if !exists || time.Since(attempt.FirstFail) > l.window {
		l.hits[key] = &checkAttempt{Count: 1, FirstFail: time.Now()}
		return
	}
	attempt.Count++
}
