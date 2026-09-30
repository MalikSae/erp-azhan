package media

import (
	"context"
	"database/sql"
	"erp-azhan/api/internal/identity"
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func protectedPath(raw string) (string, string, bool) {
	parts := strings.Split(strings.TrimPrefix(raw, "/"), "/")
	if len(parts) == 5 && parts[0] == "api" && parts[1] == "admin" && parts[2] == "media" {
		parts = parts[3:]
	} else if len(parts) == 3 && parts[0] == "uploads" {
		parts = parts[1:]
	} else {
		return "", "", false
	}
	_, ok := protectedCategories[parts[0]]
	if !ok || !safeFilename(parts[1]) {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func ValidatePortalUpload(ctx context.Context, db *sql.DB, raw string, id int64) error {
	category, name, ok := protectedPath(raw)
	if !ok || category != "dokumen-jamaah" {
		return errors.New("berkas harus berasal dari unggahan akun Anda")
	}
	canonical := "/api/admin/media/" + category + "/" + name
	var owned bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_uploads WHERE path=? AND jamaah_id=?)`, canonical, id).Scan(&owned); err != nil {
		return err
	}
	if !owned {
		return errors.New("berkas bukan unggahan akun Anda")
	}
	info, err := os.Stat(filepath.Join("uploads", category, name))
	if err != nil || info.IsDir() {
		return errors.New("berkas unggahan tidak ditemukan")
	}
	return nil
}

func (h *Handler) ServeDocument(w http.ResponseWriter, r *http.Request) {
	var raw string
	err := h.db.QueryRowContext(r.Context(), `SELECT file_url FROM dokumen_jamaah WHERE id=? AND jamaah_id=?`, chi.URLParam(r, "id"), identity.GetPortalJamaahID(r.Context())).Scan(&raw)
	category, name, ok := protectedPath(raw)
	if err != nil || !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filepath.Join("uploads", category, name))
}

func (h *Handler) canReadAdminFile(r *http.Request, category, name string) bool {
	brand := identity.GetBrandID(r.Context())
	if brand == nil {
		return identity.GetAdminUserID(r.Context()) > 0
	}
	canonical := "/api/admin/media/" + category + "/" + name
	legacy := "/uploads/" + category + "/" + name
	var allowed bool
	err := h.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM media_uploads WHERE path=? AND brand_id=?) OR EXISTS(SELECT 1 FROM dokumen_jamaah d JOIN jamaah j ON j.id=d.jamaah_id WHERE d.file_url IN (?,?) AND j.brand_id=?) OR EXISTS(SELECT 1 FROM payments p JOIN bookings b ON b.id=p.booking_id JOIN schedules s ON s.id=b.schedule_id WHERE p.bukti_url IN (?,?) AND s.brand_id=?)`, canonical, *brand, canonical, legacy, *brand, canonical, legacy, *brand).Scan(&allowed)
	return err == nil && allowed
}
