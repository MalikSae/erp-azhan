package identity

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// UserStatusLoader membaca status terkini admin user (aktif dan perm_version). User yang tidak
// ada dikembalikan sebagai tidak aktif tanpa error.
type UserStatusLoader func(ctx context.Context, adminUserID int64) (active bool, permVersion int, err error)

// userStatusTTL: pencabutan akun/izin berlaku paling lambat selama ini (bukan menunggu access token
// kedaluwarsa, maks. 60 menit).
const userStatusTTL = 30 * time.Second

type userStatus struct {
	active    bool
	version   int
	fetchedAt time.Time
}

type userStatusCache struct {
	mu      sync.Mutex
	loader  UserStatusLoader
	ttl     time.Duration
	entries map[int64]userStatus
}

var statusCache = &userStatusCache{ttl: userStatusTTL, entries: map[int64]userStatus{}}

// SetUserStatusLoader memasang pemeriksa status untuk RequireAuth (dipanggil main.go). Tanpa
// loader (nil), RequireAuth hanya memvalidasi token.
func SetUserStatusLoader(l UserStatusLoader) {
	statusCache.mu.Lock()
	defer statusCache.mu.Unlock()
	statusCache.loader = l
	statusCache.entries = map[int64]userStatus{}
}

var errStatusUnavailable = errors.New("status pengguna tidak dapat diperiksa")

func (c *userStatusCache) load(ctx context.Context, l UserStatusLoader, id int64) (userStatus, error) {
	active, ver, err := l(ctx, id)
	if err != nil {
		return userStatus{}, err
	}
	s := userStatus{active: active, version: ver, fetchedAt: time.Now()}
	c.mu.Lock()
	if len(c.entries) > 10000 {
		for k, e := range c.entries {
			if time.Since(e.fetchedAt) > c.ttl {
				delete(c.entries, k)
			}
		}
	}
	c.entries[id] = s
	c.mu.Unlock()
	return s, nil
}

// allow memeriksa token terhadap status user: akun nonaktif/terhapus ditolak; token yang membawa
// perm_version (pver) berbeda dengan database ditolak agar klien me-refresh dan mendapat izin terbaru.
// Token lama tanpa pver (0) hanya diperiksa status aktifnya.
func (c *userStatusCache) allow(ctx context.Context, claims *AccessClaims) (bool, error) {
	c.mu.Lock()
	l := c.loader
	s, ok := c.entries[claims.AdminUserID]
	c.mu.Unlock()
	if l == nil {
		return true, nil
	}
	fresh := false
	if !ok || time.Since(s.fetchedAt) > c.ttl {
		var err error
		if s, err = c.load(ctx, l, claims.AdminUserID); err != nil {
			return false, fmt.Errorf("%w: %v", errStatusUnavailable, err)
		}
		fresh = true
	}
	// Token lebih baru dari cache (izin baru saja berubah lalu klien refresh): muat ulang sekali.
	if !fresh && claims.PermVersion > s.version {
		var err error
		if s, err = c.load(ctx, l, claims.AdminUserID); err != nil {
			return false, fmt.Errorf("%w: %v", errStatusUnavailable, err)
		}
	}
	if !s.active {
		return false, nil
	}
	if claims.PermVersion > 0 && claims.PermVersion != s.version {
		return false, nil
	}
	return true, nil
}

// UserStatus memenuhi UserStatusLoader dari tabel admin_users.
func (r *Repository) UserStatus(ctx context.Context, id int64) (bool, int, error) {
	var active bool
	var ver int
	err := r.db.QueryRowContext(ctx, `SELECT is_active, perm_version FROM admin_users WHERE id=?`, id).Scan(&active, &ver)
	if errors.Is(err, sql.ErrNoRows) {
		return false, 0, nil
	}
	if err != nil {
		log.Printf("[ERROR] identity.UserStatus: %v", err)
		return false, 0, err
	}
	return active, ver, nil
}
