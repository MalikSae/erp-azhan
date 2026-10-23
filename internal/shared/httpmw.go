package shared

import (
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
)

// invoiceTokenPath: token invoice (64 heks) berada di path; log tidak boleh memuatnya.
var invoiceTokenPath = regexp.MustCompile(`^(/api/public/invoice/)[^/?#]+`)

// sensitiveQueryKeys disamarkan bila muncul di query string yang tercatat di log.
var sensitiveQueryKeys = []string{"token", "invoice_token", "refresh_token", "access_token", "code"}

// RedactURI menyamarkan token pada path/query sebelum dicatat ke log akses.
func RedactURI(uri string) string {
	path, query, hasQuery := strings.Cut(uri, "?")
	path = invoiceTokenPath.ReplaceAllString(path, "${1}[disamarkan]")
	if !hasQuery {
		return path
	}
	values, err := url.ParseQuery(query)
	if err != nil {
		return path + "?[disamarkan]"
	}
	for _, k := range sensitiveQueryKeys {
		if _, ok := values[k]; ok {
			values.Set(k, "[disamarkan]")
		}
	}
	return path + "?" + values.Encode()
}

// redactingFormatter membungkus formatter log chi dan menyamarkan RequestURI.
type redactingFormatter struct{ inner middleware.LogFormatter }

func (f redactingFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	r2 := new(http.Request)
	*r2 = *r
	r2.RequestURI = RedactURI(r.RequestURI)
	if r.URL != nil {
		u := *r.URL
		u.Path = invoiceTokenPath.ReplaceAllString(u.Path, "${1}[disamarkan]")
		u.RawPath = ""
		if u.RawQuery != "" {
			_, q, _ := strings.Cut(RedactURI("/?"+u.RawQuery), "?")
			u.RawQuery = q
		}
		r2.URL = &u
	}
	return f.inner.NewLogEntry(r2)
}

// RedactingLogger adalah pengganti middleware.Logger yang menyamarkan token di URL.
func RedactingLogger(inner middleware.LogFormatter) func(http.Handler) http.Handler {
	return middleware.RequestLogger(redactingFormatter{inner: inner})
}

// statusCacheWriter menambahkan Cache-Control hanya untuk jawaban 200.
type statusCacheWriter struct {
	http.ResponseWriter
	value string
	wrote bool
}

func (w *statusCacheWriter) WriteHeader(code int) {
	if !w.wrote {
		w.wrote = true
		if code == http.StatusOK && w.Header().Get("Cache-Control") == "" {
			w.Header().Set("Cache-Control", w.value)
		}
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusCacheWriter) Write(b []byte) (int, error) {
	if !w.wrote {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// PublicCache memasang "Cache-Control: public, max-age=N" pada jawaban GET 200 yang tidak
// bergantung pada pengguna (API publik v1). Galat tidak di-cache.
func PublicCache(maxAgeSeconds int) func(http.Handler) http.Handler {
	value := "public, max-age=" + strconv.Itoa(maxAgeSeconds)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet && r.Method != http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(&statusCacheWriter{ResponseWriter: w, value: value}, r)
		})
	}
}
