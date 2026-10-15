package perlengkapan

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
)

// Buku besar stok perlengkapan per brand. Tidak ada gudang pusat: barang yang dibeli
// langsung dicatat masuk ke stok brand tujuan.
//   pembelian  : barang masuk ke stok brand (vendor, harga, catatan)
//   distribusi : keluar dari stok karena diserahkan ke jamaah (otomatis dari booking)
//   retur      : masuk kembali karena penyerahan dibatalkan (otomatis)
//   koreksi    : penyesuaian manual (stok opname, barang rusak/hilang)

var ErrStokKurang = errors.New("stok tidak boleh kurang dari 0")

// MutasiRow adalah satu baris riwayat pergerakan stok.
type MutasiRow struct {
	ID          uint64    `json:"id"`
	ItemID      uint64    `json:"perlengkapan_item_id"`
	ItemNama    string    `json:"item_nama"`
	BrandID     uint64    `json:"brand_id"`
	Lokasi      string    `json:"lokasi"` // nama brand
	Jenis       string    `json:"jenis"`
	Qty         int       `json:"qty"`
	StokSetelah int       `json:"stok_setelah"`
	HargaSatuan *float64  `json:"harga_satuan"`
	Vendor      *string   `json:"vendor"`
	Catatan     *string   `json:"catatan"`
	Referensi   *string   `json:"referensi"`
	Oleh        *string   `json:"oleh"`
	CreatedAt   time.Time `json:"created_at"`
}

type execQuerier interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// lockStok mengunci baris stok brand dan mengembalikan saldo saat ini.
func lockStok(ctx context.Context, tx execQuerier, itemID, brandID uint64) (int, error) {
	if _, err := tx.ExecContext(ctx,
		`INSERT IGNORE INTO perlengkapan_stok (brand_id, perlengkapan_item_id, stok_tersedia) VALUES (?, ?, 0)`, brandID, itemID); err != nil {
		return 0, err
	}
	var cur int
	err := tx.QueryRowContext(ctx,
		`SELECT stok_tersedia FROM perlengkapan_stok WHERE brand_id = ? AND perlengkapan_item_id = ? FOR UPDATE`, brandID, itemID).Scan(&cur)
	return cur, err
}

func setStok(ctx context.Context, tx execQuerier, itemID, brandID uint64, value int) error {
	_, err := tx.ExecContext(ctx,
		`UPDATE perlengkapan_stok SET stok_tersedia = ? WHERE brand_id = ? AND perlengkapan_item_id = ?`, value, brandID, itemID)
	return err
}

type mutasiInput struct {
	ItemID    uint64
	BrandID   uint64
	Jenis     string
	Qty       int
	Setelah   int
	Harga     *float64
	Vendor    *string
	Catatan   *string
	RefTipe   *string
	RefID     *uint64
	CreatedBy *int64
}

func insertMutasi(ctx context.Context, tx execQuerier, m mutasiInput) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO perlengkapan_mutasi
		  (perlengkapan_item_id, brand_id, jenis, qty, stok_setelah, harga_satuan, vendor, catatan, referensi_tipe, referensi_id, created_by)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		m.ItemID, m.BrandID, m.Jenis, m.Qty, m.Setelah, m.Harga, m.Vendor, m.Catatan, m.RefTipe, m.RefID, m.CreatedBy)
	return err
}

// ApplyBrandDelta mengubah stok brand sebesar delta dalam transaksi pemanggil dan mencatatnya
// di buku besar. Dipakai penyerahan/pembatalan perlengkapan pada booking.
func ApplyBrandDelta(ctx context.Context, tx *sql.Tx, brandID, itemID uint64, delta int, jenis, refTipe string, refID uint64) error {
	cur, err := lockStok(ctx, tx, itemID, brandID)
	if err != nil {
		return err
	}
	next := cur + delta
	if next < 0 {
		return ErrStokKurang
	}
	if err := setStok(ctx, tx, itemID, brandID, next); err != nil {
		return err
	}
	return insertMutasi(ctx, tx, mutasiInput{
		ItemID: itemID, BrandID: brandID, Jenis: jenis, Qty: delta, Setelah: next,
		RefTipe: &refTipe, RefID: &refID,
	})
}

func (r *Repository) ensureRefs(ctx context.Context, tx execQuerier, itemID, brandID uint64) error {
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM perlengkapan_items WHERE id = ?`, itemID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM brands WHERE id = ?`, brandID).Scan(&n); err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// Pembelian: barang masuk ke stok brand.
func (r *Repository) Pembelian(ctx context.Context, userID *int64, itemID, brandID uint64, qty int, harga *float64, vendor, catatan *string) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := r.ensureRefs(ctx, tx, itemID, brandID); err != nil {
		return 0, err
	}
	cur, err := lockStok(ctx, tx, itemID, brandID)
	if err != nil {
		return 0, err
	}
	next := cur + qty
	if err := setStok(ctx, tx, itemID, brandID, next); err != nil {
		return 0, err
	}
	if err := insertMutasi(ctx, tx, mutasiInput{
		ItemID: itemID, BrandID: brandID, Jenis: "pembelian", Qty: qty, Setelah: next,
		Harga: harga, Vendor: vendor, Catatan: catatan, CreatedBy: userID,
	}); err != nil {
		return 0, err
	}
	return next, tx.Commit()
}

// Koreksi: set stok brand ke angka baru, selisihnya dicatat.
func (r *Repository) Koreksi(ctx context.Context, userID *int64, itemID, brandID uint64, stokBaru int, catatan *string) (int, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if err := r.ensureRefs(ctx, tx, itemID, brandID); err != nil {
		return 0, err
	}
	cur, err := lockStok(ctx, tx, itemID, brandID)
	if err != nil {
		return 0, err
	}
	if stokBaru == cur {
		return cur, nil
	}
	if err := setStok(ctx, tx, itemID, brandID, stokBaru); err != nil {
		return 0, err
	}
	if err := insertMutasi(ctx, tx, mutasiInput{
		ItemID: itemID, BrandID: brandID, Jenis: "koreksi", Qty: stokBaru - cur, Setelah: stokBaru, Catatan: catatan, CreatedBy: userID,
	}); err != nil {
		return 0, err
	}
	return stokBaru, tx.Commit()
}

// ListMutasi mengembalikan riwayat terbaru (maks 300), bisa disaring per item / brand / jenis.
func (r *Repository) ListMutasi(ctx context.Context, itemID, brandID *uint64, jenis string, beforeID uint64, limit int) ([]MutasiRow, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := `
		SELECT m.id, m.perlengkapan_item_id, i.nama, m.brand_id, b.name, m.jenis, m.qty, m.stok_setelah,
		       m.harga_satuan, m.vendor, m.catatan,
		       CASE WHEN m.referensi_tipe = 'booking_pax'
		            THEN CONCAT(IF(m.jenis = 'retur', 'Dibatalkan: ', 'Jamaah: '),
		                        COALESCE(j.nama_lengkap, CONCAT('#', m.referensi_id)),
		                        IF(bk.id_booking IS NULL, '', CONCAT(' · ', bk.id_booking)))
		       END,
		       u.display_name, m.created_at
		FROM perlengkapan_mutasi m
		JOIN perlengkapan_items i ON i.id = m.perlengkapan_item_id
		JOIN brands b ON b.id = m.brand_id
		LEFT JOIN admin_users u ON u.id = m.created_by
		LEFT JOIN booking_pax bp ON m.referensi_tipe = 'booking_pax' AND bp.id = m.referensi_id
		LEFT JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN bookings bk ON bk.id = bp.booking_id
		WHERE 1=1`
	var args []any
	if itemID != nil {
		q += " AND m.perlengkapan_item_id = ?"
		args = append(args, *itemID)
	}
	if brandID != nil {
		q += " AND m.brand_id = ?"
		args = append(args, *brandID)
	}
	if jenis != "" {
		q += " AND m.jenis = ?"
		args = append(args, jenis)
	}
	if beforeID > 0 {
		q += " AND m.id < ?"
		args = append(args, beforeID)
	}
	q += " ORDER BY m.id DESC LIMIT ?"
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.ListMutasi: %w", err)
	}
	defer rows.Close()
	out := make([]MutasiRow, 0)
	for rows.Next() {
		var m MutasiRow
		if err := rows.Scan(&m.ID, &m.ItemID, &m.ItemNama, &m.BrandID, &m.Lokasi, &m.Jenis, &m.Qty, &m.StokSetelah,
			&m.HargaSatuan, &m.Vendor, &m.Catatan, &m.Referensi, &m.Oleh, &m.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ─── Handler ──────────────────────────────────────────────────────────────────

type pembelianRequest struct {
	ItemID      uint64   `json:"perlengkapan_item_id"`
	BrandID     uint64   `json:"brand_id"`
	Qty         int      `json:"qty"`
	HargaSatuan *float64 `json:"harga_satuan"`
	Vendor      *string  `json:"vendor"`
	Catatan     *string  `json:"catatan"`
}

type koreksiRequest struct {
	ItemID   uint64  `json:"perlengkapan_item_id"`
	BrandID  uint64  `json:"brand_id"`
	StokBaru *int    `json:"stok_baru"`
	Catatan  *string `json:"catatan"`
}

func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// brandDiizinkan: admin terikat brand hanya boleh mengubah stok brand-nya sendiri.
func (h *Handler) brandDiizinkan(w http.ResponseWriter, r *http.Request, brandID uint64) bool {
	if scoped := identity.GetBrandID(r.Context()); scoped != nil && uint64(*scoped) != brandID {
		h.sendError(w, http.StatusForbidden, "Anda hanya dapat mengelola stok brand Anda sendiri")
		return false
	}
	return true
}

func (h *Handler) mutasiError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		h.sendError(w, http.StatusNotFound, "Item atau brand tidak ditemukan")
	case errors.Is(err, ErrStokKurang):
		h.sendError(w, http.StatusBadRequest, err.Error())
	default:
		h.sendError(w, http.StatusInternalServerError, "Gagal mencatat pergerakan stok")
	}
}

func adminID(r *http.Request) *int64 {
	id := identity.GetAdminUserID(r.Context())
	if id == 0 {
		return nil
	}
	return &id
}

// Pembelian: POST /api/admin/perlengkapan-mutasi/pembelian
func (h *Handler) Pembelian(w http.ResponseWriter, r *http.Request) {
	var req pembelianRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ItemID == 0 || req.BrandID == 0 || req.Qty < 1 {
		h.sendError(w, http.StatusBadRequest, "Item, brand, dan jumlah (minimal 1) wajib diisi")
		return
	}
	if req.HargaSatuan != nil && *req.HargaSatuan < 0 {
		h.sendError(w, http.StatusBadRequest, "Harga satuan tidak boleh negatif")
		return
	}
	if !h.brandDiizinkan(w, r, req.BrandID) {
		return
	}
	stok, err := h.repo.Pembelian(r.Context(), adminID(r), req.ItemID, req.BrandID, req.Qty, req.HargaSatuan, trimPtr(req.Vendor), trimPtr(req.Catatan))
	if err != nil {
		h.mutasiError(w, err)
		return
	}
	h.sendJSON(w, http.StatusCreated, map[string]int{"stok": stok})
}

// Koreksi: POST /api/admin/perlengkapan-mutasi/koreksi
func (h *Handler) Koreksi(w http.ResponseWriter, r *http.Request) {
	var req koreksiRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ItemID == 0 || req.BrandID == 0 || req.StokBaru == nil || *req.StokBaru < 0 {
		h.sendError(w, http.StatusBadRequest, "Item, brand, dan stok baru (0 atau lebih) wajib diisi")
		return
	}
	if trimPtr(req.Catatan) == nil {
		h.sendError(w, http.StatusBadRequest, "Alasan koreksi stok wajib diisi")
		return
	}
	if !h.brandDiizinkan(w, r, req.BrandID) {
		return
	}
	stok, err := h.repo.Koreksi(r.Context(), adminID(r), req.ItemID, req.BrandID, *req.StokBaru, trimPtr(req.Catatan))
	if err != nil {
		h.mutasiError(w, err)
		return
	}
	h.sendJSON(w, http.StatusOK, map[string]int{"stok": stok})
}

// ListMutasi: GET /api/admin/perlengkapan-mutasi?item_id=&brand_id=&jenis=&limit=&before_id=
// Halaman berikutnya: kirim before_id = id baris paling lama yang sudah diterima.
func (h *Handler) ListMutasi(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var itemID, brandID *uint64
	if v, err := strconv.ParseUint(q.Get("item_id"), 10, 64); err == nil && v > 0 {
		itemID = &v
	}
	if v, err := strconv.ParseUint(q.Get("brand_id"), 10, 64); err == nil && v > 0 {
		brandID = &v
	}
	// Admin terikat brand hanya boleh melihat riwayat brand-nya sendiri.
	if scoped := identity.GetBrandID(r.Context()); scoped != nil {
		b := uint64(*scoped)
		brandID = &b
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	beforeID, _ := strconv.ParseUint(q.Get("before_id"), 10, 64)
	rows, err := h.repo.ListMutasi(r.Context(), itemID, brandID, q.Get("jenis"), beforeID, limit)
	if err != nil {
		h.sendError(w, http.StatusInternalServerError, "Gagal memuat riwayat stok")
		return
	}
	h.sendJSON(w, http.StatusOK, rows)
}
