package customer

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Masuk dengan Google (keputusan 16 November 2026). App mengirim ID token dari Google Sign-In; ERP memeriksa
// tanda tangannya dengan kunci publik Google, penerbit, masa berlaku, dan audience (client ID OAuth app yang
// terdaftar di GOOGLE_CLIENT_IDS, dipisah koma). Email wajib terverifikasi oleh Google.

// ProfilGoogle: isi ID token yang dipakai ERP.
type ProfilGoogle struct {
	Sub   string
	Email string
	Nama  string
}

// VerifikatorGoogle memeriksa ID token. Diganti di test.
type VerifikatorGoogle interface {
	Verifikasi(ctx context.Context, idToken string) (*ProfilGoogle, error)
}

var (
	ErrGoogleTidakDikonfigurasi = errors.New("masuk dengan Google belum dikonfigurasi")
	ErrTokenGoogle              = errors.New("token Google tidak valid")
)

const googleCertsURL = "https://www.googleapis.com/oauth2/v3/certs"

var googleIssuer = map[string]bool{"accounts.google.com": true, "https://accounts.google.com": true}

type verifikatorGoogle struct {
	clientIDs []string
	http      *http.Client

	mu     sync.Mutex
	kunci  map[string]*rsa.PublicKey
	sampai time.Time
}

// NewVerifikatorGoogle membaca GOOGLE_CLIENT_IDS. Tanpa nilai, Verifikasi selalu ErrGoogleTidakDikonfigurasi.
func NewVerifikatorGoogle() VerifikatorGoogle {
	var ids []string
	for _, s := range strings.Split(os.Getenv("GOOGLE_CLIENT_IDS"), ",") {
		if s = strings.TrimSpace(s); s != "" {
			ids = append(ids, s)
		}
	}
	return &verifikatorGoogle{clientIDs: ids, http: &http.Client{Timeout: 10 * time.Second}}
}

type klaimGoogle struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"` // Google mengirim boolean; versi lama mengirim "true"
	Name          string `json:"name"`
	jwt.RegisteredClaims
}

func (k *klaimGoogle) emailTerverifikasi() bool {
	switch v := k.EmailVerified.(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}
	return false
}

func (v *verifikatorGoogle) Verifikasi(ctx context.Context, idToken string) (*ProfilGoogle, error) {
	if len(v.clientIDs) == 0 {
		return nil, ErrGoogleTidakDikonfigurasi
	}
	var klaim klaimGoogle
	_, err := jwt.ParseWithClaims(idToken, &klaim, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return v.kunciPublik(ctx, kid)
	}, jwt.WithValidMethods([]string{"RS256"}), jwt.WithExpirationRequired(), jwt.WithLeeway(time.Minute))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTokenGoogle, err)
	}
	return profilDariKlaim(&klaim, v.clientIDs)
}

// profilDariKlaim memeriksa penerbit, audience, dan email terverifikasi pada klaim yang tanda tangannya sah.
func profilDariKlaim(k *klaimGoogle, clientIDs []string) (*ProfilGoogle, error) {
	iss, _ := k.GetIssuer()
	if !googleIssuer[iss] {
		return nil, fmt.Errorf("%w: penerbit %q", ErrTokenGoogle, iss)
	}
	aud, _ := k.GetAudience()
	cocok := false
	for _, a := range aud {
		for _, id := range clientIDs {
			if a == id {
				cocok = true
			}
		}
	}
	if !cocok {
		return nil, fmt.Errorf("%w: audience", ErrTokenGoogle)
	}
	if k.Subject == "" || k.Email == "" || !k.emailTerverifikasi() {
		return nil, fmt.Errorf("%w: email belum terverifikasi", ErrTokenGoogle)
	}
	return &ProfilGoogle{Sub: k.Subject, Email: strings.TrimSpace(k.Email), Nama: strings.TrimSpace(k.Name)}, nil
}

var maxAge = regexp.MustCompile(`max-age=(\d+)`)

// kunciPublik mengambil kunci RSA Google sesuai kid, dengan cache sesuai Cache-Control. kid yang tidak
// dikenal memicu pengambilan ulang (Google merotasi kunci).
func (v *verifikatorGoogle) kunciPublik(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if k, ok := v.kunci[kid]; ok && time.Now().Before(v.sampai) {
		return k, nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, googleCertsURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := v.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ambil kunci Google: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ambil kunci Google: status %d", res.StatusCode)
	}
	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(res.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("baca kunci Google: %w", err)
	}
	kunci := map[string]*rsa.PublicKey{}
	for _, k := range jwks.Keys {
		if k.Kty != "RSA" {
			continue
		}
		n, err1 := base64.RawURLEncoding.DecodeString(k.N)
		e, err2 := base64.RawURLEncoding.DecodeString(k.E)
		if err1 != nil || err2 != nil {
			continue
		}
		kunci[k.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(n), E: int(new(big.Int).SetBytes(e).Int64())}
	}
	umur := time.Hour
	if m := maxAge.FindStringSubmatch(res.Header.Get("Cache-Control")); m != nil {
		if s, err := strconv.Atoi(m[1]); err == nil && s > 0 {
			umur = time.Duration(s) * time.Second
		}
	}
	v.kunci, v.sampai = kunci, time.Now().Add(umur)
	k, ok := kunci[kid]
	if !ok {
		return nil, fmt.Errorf("kunci Google %q tidak dikenal", kid)
	}
	return k, nil
}
