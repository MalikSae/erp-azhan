package media

import (
	"context"
	"database/sql"
	"erp-azhan/api/internal/identity"
	"fmt"
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

// ValidateCustomerUpload memastikan berkas (mis. bukti transfer) adalah unggahan akun
// pelanggan itu sendiri dan masih ada di disk.
//
// Galat untuk pengguna berupa *UploadError (pesannya aman ditampilkan); galat database dibungkus biasa
// supaya pemanggil mencatatnya dan menjawab pesan umum, bukan teks galat driver.
func ValidateCustomerUpload(ctx context.Context, db *sql.DB, raw string, accountID int64) error {
	category, name, ok := protectedPath(raw)
	if !ok || category != "dokumen-jamaah" {
		return &UploadError{"Berkas harus berasal dari unggahan akun Anda."}
	}
	canonical := "/api/admin/media/" + category + "/" + name
	var owned bool
	if err := db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM media_uploads WHERE path=? AND customer_account_id=?)`, canonical, accountID).Scan(&owned); err != nil {
		return fmt.Errorf("media.ValidateCustomerUpload: %w", err)
	}
	if !owned {
		return &UploadError{"Berkas bukan unggahan akun Anda."}
	}
	info, err := os.Stat(filepath.Join("uploads", category, name))
	if err != nil || info.IsDir() {
		return &UploadError{"Berkas unggahan tidak ditemukan."}
	}
	return nil
}

// ValidateAdminUpload memastikan dokumen yang disimpan admin menunjuk ke berkas
// unggahan resmi (JB-11): kategori dokumen-jamaah, diunggah lewat jalur admin, dan
// berkasnya ada di disk. Yang boleh ditautkan hanya unggahan admin itu sendiri
// (tanpa brand, atau brand jamaah) atau unggahan admin lain dari brand jamaah.
// Unggahan akun pelanggan, portal jamaah lama, dan brand lain ditolak: setelah
// tertaut, berkas ikut terbaca oleh admin brand jamaah (canReadAdminFile).
// URL eksternal dan path buatan ditolak.
func ValidateAdminUpload(ctx context.Context, db *sql.DB, raw string, jamaahID, adminUserID int64) error {
	category, name, ok := protectedPath(raw)
	if !ok || category != "dokumen-jamaah" {
		return &UploadError{"Berkas harus diunggah melalui formulir dokumen jamaah."}
	}
	canonical := "/api/admin/media/" + category + "/" + name
	var owned bool
	if err := db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM media_uploads m JOIN jamaah j ON j.id = ?
			WHERE m.path = ? AND m.admin_user_id IS NOT NULL
			  AND (m.brand_id = j.brand_id OR (m.brand_id IS NULL AND m.admin_user_id = ?))
		)`, jamaahID, canonical, adminUserID).Scan(&owned); err != nil {
		return fmt.Errorf("media.ValidateAdminUpload: %w", err)
	}
	if !owned {
		return &UploadError{"Berkas bukan unggahan brand jamaah ini."}
	}
	info, err := os.Stat(filepath.Join("uploads", category, name))
	if err != nil || info.IsDir() {
		return &UploadError{"Berkas unggahan tidak ditemukan."}
	}
	return nil
}

// UploadError adalah penolakan berkas yang pesannya aman ditampilkan ke pengguna.
type UploadError struct{ Message string }

func (e *UploadError) Error() string { return e.Message }

func (h *Handler) canReadAdminFile(r *http.Request, category, name string) bool {
	brand := identity.GetBrandID(r.Context())
	if brand == nil {
		return identity.GetAdminUserID(r.Context()) > 0
	}
	canonical := "/api/admin/media/" + category + "/" + name
	legacy := "/uploads/" + category + "/" + name
	var allowed bool
	err := h.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM media_uploads WHERE path=? AND brand_id=?) OR EXISTS(SELECT 1 FROM dokumen_jamaah d JOIN jamaah j ON j.id=d.jamaah_id WHERE d.file_url IN (?,?) AND j.brand_id=?) OR EXISTS(SELECT 1 FROM dokumen_jamaah_versions v JOIN dokumen_jamaah d ON d.id=v.dokumen_id JOIN jamaah j ON j.id=d.jamaah_id WHERE v.file_url IN (?,?) AND j.brand_id=?) OR EXISTS(SELECT 1 FROM payments p JOIN bookings b ON b.id=p.booking_id JOIN schedules s ON s.id=b.schedule_id WHERE p.bukti_url IN (?,?) AND s.brand_id=?)`, canonical, *brand, canonical, legacy, *brand, canonical, legacy, *brand, canonical, legacy, *brand).Scan(&allowed)
	return err == nil && allowed
}

// ValidateAdminPaymentProof memastikan bukti pembayaran yang dicatat admin menunjuk ke berkas
// unggahan admin resmi (kategori payment-proofs), bukan URL bebas. Admin brand hanya boleh memakai
// berkas yang tercatat untuk brand-nya; super admin (brandID nil) boleh memakai unggahan admin mana pun.
// URL eksternal (https://...), path buatan, dan berkas yang tidak ada ditolak.
func ValidateAdminPaymentProof(ctx context.Context, db *sql.DB, raw string, brandID *int64) error {
	return ValidateAdminProof(ctx, db, raw, brandID, "payment-proofs")
}

// ValidateAdminProof sama dengan ValidateAdminPaymentProof untuk kategori yang diizinkan (mis. bukti bayar
// agen boleh berupa dokumen-jamaah atau payment-proofs). Dipakai setiap kolom bukti yang diisi admin, supaya
// URL bukti tidak bisa menunjuk berkas brand lain (canReadAdminFile mempercayai payments.bukti_url).
func ValidateAdminProof(ctx context.Context, db *sql.DB, raw string, brandID *int64, categories ...string) error {
	category, name, ok := protectedPath(raw)
	allowed := false
	for _, c := range categories {
		if ok && category == c {
			allowed = true
		}
	}
	if !allowed {
		return &UploadError{"Bukti pembayaran harus diunggah melalui formulir pembayaran."}
	}
	canonical := "/api/admin/media/" + category + "/" + name
	query := `SELECT EXISTS(SELECT 1 FROM media_uploads WHERE path=? AND admin_user_id IS NOT NULL`
	args := []any{canonical}
	if brandID != nil {
		query += ` AND brand_id=?`
		args = append(args, *brandID)
	}
	query += `)`
	var owned bool
	if err := db.QueryRowContext(ctx, query, args...).Scan(&owned); err != nil {
		return fmt.Errorf("media.ValidateAdminPaymentProof: %w", err)
	}
	if !owned {
		return &UploadError{"Bukti pembayaran bukan unggahan resmi brand ini."}
	}
	info, err := os.Stat(filepath.Join("uploads", category, name))
	if err != nil || info.IsDir() {
		return &UploadError{"Berkas bukti pembayaran tidak ditemukan."}
	}
	return nil
}
