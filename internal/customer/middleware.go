package customer

import (
	"context"
	"errors"
	"log"
	"net/http"
	"strings"

	"erp-azhan/api/internal/identity"
)

type ctxKey int

const (
	accountKey ctxKey = iota
	tokenKey
)

// FromContext mengembalikan akun yang sedang login (nil bila tidak ada).
func FromContext(ctx context.Context) *Account {
	a, _ := ctx.Value(accountKey).(*Account)
	return a
}

// ContextWithAccount menaruh akun di context seperti RequireCustomer (dipakai test handler lain).
func ContextWithAccount(ctx context.Context, acc *Account) context.Context {
	ctx = context.WithValue(ctx, accountKey, acc)
	return identity.WithCustomer(ctx, acc.ID, acc.BrandID)
}

// TokenFromContext mengembalikan token sesi mentah dari request ini.
func TokenFromContext(ctx context.Context) string {
	s, _ := ctx.Value(tokenKey).(string)
	return s
}

func bearer(r *http.Request) string {
	h := strings.TrimSpace(r.Header.Get("Authorization"))
	if len(h) > 7 && strings.EqualFold(h[:7], "Bearer ") {
		return strings.TrimSpace(h[7:])
	}
	return ""
}

// RequireCustomer menolak request tanpa sesi akun yang sah. Token dikirim server Next
// (dari cookie httpOnly) sebagai header Authorization.
func (h *Handler) RequireCustomer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := bearer(r)
		acc, err := h.repo.AccountBySession(r.Context(), raw)
		if errors.Is(err, ErrSessionInvalid) {
			writeError(w, http.StatusUnauthorized, "sesi berakhir, silakan masuk kembali")
			return
		}
		if err != nil {
			// Gangguan database bukan sesi berakhir: app menghapus token tersimpan bila menerima 401,
			// jadi galat sementara dijawab 500 supaya pengguna tidak ikut keluar.
			log.Printf("[ERROR] customer sesi: %v", err)
			writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
			return
		}
		ctx := context.WithValue(r.Context(), accountKey, acc)
		ctx = context.WithValue(ctx, tokenKey, raw)
		ctx = identity.WithCustomer(ctx, acc.ID, acc.BrandID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
