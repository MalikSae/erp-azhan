package perlengkapan

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/go-sql-driver/mysql"
)

var (
	ErrNotFound     = errors.New("item perlengkapan tidak ditemukan")
	ErrDuplicate    = errors.New("nama item perlengkapan sudah terdaftar")
	ErrInUse        = errors.New("tidak bisa dihapus, item masih dipakai di template set atau masih memiliki stok")
	ErrInvalidInput = errors.New("input data tidak valid")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// ─── Item Katalog Global ──────────────────────────────────────────────────────

func (r *Repository) ListItems(ctx context.Context) ([]PerlengkapanItem, error) {
	query := `
		SELECT i.id, i.nama, COALESCE(t.qty, 0) AS qty_per_set, i.created_at
		FROM perlengkapan_items i
		LEFT JOIN perlengkapan_set_template t ON i.id = t.perlengkapan_item_id
		ORDER BY i.nama ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.ListItems: %w", err)
	}
	defer rows.Close()

	items := make([]PerlengkapanItem, 0)
	for rows.Next() {
		var it PerlengkapanItem
		if err := rows.Scan(&it.ID, &it.Nama, &it.QtyPerSet, &it.CreatedAt); err != nil {
			return nil, fmt.Errorf("perlengkapan.ListItems scan: %w", err)
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *Repository) GetItemByID(ctx context.Context, id uint64) (*PerlengkapanItem, error) {
	query := `
		SELECT i.id, i.nama, COALESCE(t.qty, 0) AS qty_per_set, i.created_at
		FROM perlengkapan_items i
		LEFT JOIN perlengkapan_set_template t ON i.id = t.perlengkapan_item_id
		WHERE i.id = ?
	`
	var it PerlengkapanItem
	err := r.db.QueryRowContext(ctx, query, id).Scan(&it.ID, &it.Nama, &it.QtyPerSet, &it.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("perlengkapan.GetItemByID: %w", err)
	}
	return &it, nil
}

func (r *Repository) CreateItem(ctx context.Context, nama string, qtyPerSet int) (*PerlengkapanItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.CreateItem begin tx: %w", err)
	}
	defer tx.Rollback()

	queryItem := "INSERT INTO perlengkapan_items (nama) VALUES (?)"
	res, err := tx.ExecContext(ctx, queryItem, nama)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("perlengkapan.CreateItem insert: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.CreateItem lastInsertId: %w", err)
	}

	// Insert stok default 0 untuk semua brand yang ada
	queryStok := "INSERT INTO perlengkapan_stok (brand_id, perlengkapan_item_id, stok_tersedia) SELECT id, ?, 0 FROM brands"
	if _, err := tx.ExecContext(ctx, queryStok, id); err != nil {
		return nil, fmt.Errorf("perlengkapan.CreateItem init stok: %w", err)
	}

	// Jika qtyPerSet > 0, insert juga ke perlengkapan_set_template
	if qtyPerSet > 0 {
		querySet := "INSERT INTO perlengkapan_set_template (perlengkapan_item_id, qty) VALUES (?, ?)"
		if _, err := tx.ExecContext(ctx, querySet, id, qtyPerSet); err != nil {
			return nil, fmt.Errorf("perlengkapan.CreateItem insert set template: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("perlengkapan.CreateItem commit: %w", err)
	}

	return r.GetItemByID(ctx, uint64(id))
}

func (r *Repository) UpdateItem(ctx context.Context, id uint64, nama string, qtyPerSet int) (*PerlengkapanItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.UpdateItem begin tx: %w", err)
	}
	defer tx.Rollback()

	query := "UPDATE perlengkapan_items SET nama = ? WHERE id = ?"
	res, err := tx.ExecContext(ctx, query, nama, id)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return nil, ErrDuplicate
		}
		return nil, fmt.Errorf("perlengkapan.UpdateItem: %w", err)
	}

	rowsAff, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.UpdateItem rowsAffected: %w", err)
	}
	if rowsAff == 0 {
		// Cek apakah item memang ada
		if _, err := r.GetItemByID(ctx, id); err != nil {
			return nil, err
		}
	}

	// Upsert atau Delete template set
	if qtyPerSet > 0 {
		querySet := `
			INSERT INTO perlengkapan_set_template (perlengkapan_item_id, qty)
			VALUES (?, ?)
			ON DUPLICATE KEY UPDATE qty = VALUES(qty)
		`
		if _, err := tx.ExecContext(ctx, querySet, id, qtyPerSet); err != nil {
			return nil, fmt.Errorf("perlengkapan.UpdateItem upsert set template: %w", err)
		}
	} else {
		queryDelSet := "DELETE FROM perlengkapan_set_template WHERE perlengkapan_item_id = ?"
		if _, err := tx.ExecContext(ctx, queryDelSet, id); err != nil {
			return nil, fmt.Errorf("perlengkapan.UpdateItem delete set template: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("perlengkapan.UpdateItem commit: %w", err)
	}

	return r.GetItemByID(ctx, id)
}

func (r *Repository) DeleteItem(ctx context.Context, id uint64) error {
	// Cek apakah dipakai di set template
	var countSet int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM perlengkapan_set_template WHERE perlengkapan_item_id = ?", id).Scan(&countSet)
	if err != nil {
		return fmt.Errorf("perlengkapan.DeleteItem check template: %w", err)
	}
	if countSet > 0 {
		return ErrInUse
	}

	// Cek apakah ada brand yang memiliki stok > 0
	var countStok int
	err = r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM perlengkapan_stok WHERE perlengkapan_item_id = ? AND stok_tersedia > 0", id).Scan(&countStok)
	if err != nil {
		return fmt.Errorf("perlengkapan.DeleteItem check stok: %w", err)
	}
	if countStok > 0 {
		return ErrInUse
	}

	query := "DELETE FROM perlengkapan_items WHERE id = ?"
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1451 {
			return ErrInUse
		}
		return fmt.Errorf("perlengkapan.DeleteItem: %w", err)
	}

	rowsAff, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("perlengkapan.DeleteItem rowsAffected: %w", err)
	}
	if rowsAff == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Stok Per Brand ───────────────────────────────────────────────────────────

func (r *Repository) ListStokAll(ctx context.Context) ([]PerlengkapanStokAllRow, error) {
	// Pastikan semua kombinasi brand x item ada
	querySync := `
		INSERT INTO perlengkapan_stok (brand_id, perlengkapan_item_id, stok_tersedia)
		SELECT b.id, i.id, 0
		FROM brands b CROSS JOIN perlengkapan_items i
		WHERE NOT EXISTS (
			SELECT 1 FROM perlengkapan_stok s WHERE s.brand_id = b.id AND s.perlengkapan_item_id = i.id
		)
	`
	_, _ = r.db.ExecContext(ctx, querySync)

	query := `
		SELECT 
			s.brand_id, 
			b.name AS brand_name, 
			b.logo_url AS brand_logo_url, 
			b.icon_url AS brand_icon_url,
			s.perlengkapan_item_id, 
			i.nama AS nama_item, 
			COALESCE(t.qty, 0) AS qty_per_set,
			s.stok_tersedia
		FROM perlengkapan_stok s
		JOIN brands b ON s.brand_id = b.id
		JOIN perlengkapan_items i ON s.perlengkapan_item_id = i.id
		LEFT JOIN perlengkapan_set_template t ON t.perlengkapan_item_id = i.id
		ORDER BY b.name ASC, i.nama ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.ListStokAll: %w", err)
	}
	defer rows.Close()

	items := make([]PerlengkapanStokAllRow, 0)
	for rows.Next() {
		var it PerlengkapanStokAllRow
		var logoURL, iconURL sql.NullString
		if err := rows.Scan(
			&it.BrandID,
			&it.BrandName,
			&logoURL,
			&iconURL,
			&it.PerlengkapanItemID,
			&it.NamaItem,
			&it.QtyPerSet,
			&it.StokTersedia,
		); err != nil {
			return nil, fmt.Errorf("perlengkapan.ListStokAll scan: %w", err)
		}
		if logoURL.Valid {
			it.BrandLogoURL = &logoURL.String
		}
		if iconURL.Valid {
			it.BrandIconURL = &iconURL.String
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *Repository) ListStokByBrand(ctx context.Context, brandID uint64) ([]PerlengkapanStokItem, error) {
	// Pastikan semua item global terdaftar di perlengkapan_stok untuk brand ini
	querySync := `
		INSERT INTO perlengkapan_stok (brand_id, perlengkapan_item_id, stok_tersedia)
		SELECT ?, id, 0
		FROM perlengkapan_items
		WHERE id NOT IN (SELECT perlengkapan_item_id FROM perlengkapan_stok WHERE brand_id = ?)
	`
	_, _ = r.db.ExecContext(ctx, querySync, brandID, brandID)

	query := `
		SELECT s.perlengkapan_item_id, i.nama, COALESCE(t.qty, 0) AS qty_per_set, s.stok_tersedia
		FROM perlengkapan_stok s
		JOIN perlengkapan_items i ON s.perlengkapan_item_id = i.id
		LEFT JOIN perlengkapan_set_template t ON t.perlengkapan_item_id = i.id
		WHERE s.brand_id = ?
		ORDER BY i.nama ASC
	`
	rows, err := r.db.QueryContext(ctx, query, brandID)
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.ListStokByBrand: %w", err)
	}
	defer rows.Close()

	items := make([]PerlengkapanStokItem, 0)
	for rows.Next() {
		var it PerlengkapanStokItem
		if err := rows.Scan(&it.PerlengkapanItemID, &it.Nama, &it.QtyPerSet, &it.StokTersedia); err != nil {
			return nil, fmt.Errorf("perlengkapan.ListStokByBrand scan: %w", err)
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *Repository) UpdateStok(ctx context.Context, brandID, itemID uint64, stok int) (*PerlengkapanStokItem, error) {
	// Pastikan item ada
	var nama string
	err := r.db.QueryRowContext(ctx, "SELECT nama FROM perlengkapan_items WHERE id = ?", itemID).Scan(&nama)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("perlengkapan.UpdateStok check item: %w", err)
	}

	query := `
		INSERT INTO perlengkapan_stok (brand_id, perlengkapan_item_id, stok_tersedia)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE stok_tersedia = VALUES(stok_tersedia)
	`
	if _, err := r.db.ExecContext(ctx, query, brandID, itemID, stok); err != nil {
		return nil, fmt.Errorf("perlengkapan.UpdateStok upsert: %w", err)
	}

	return &PerlengkapanStokItem{
		PerlengkapanItemID: itemID,
		Nama:               nama,
		StokTersedia:       stok,
	}, nil
}

// ─── Set Template Global ─────────────────────────────────────────────────────

func (r *Repository) GetSetTemplate(ctx context.Context) ([]PerlengkapanSetTemplateItem, error) {
	query := `
		SELECT t.perlengkapan_item_id, i.nama, t.qty
		FROM perlengkapan_set_template t
		JOIN perlengkapan_items i ON t.perlengkapan_item_id = i.id
		ORDER BY i.nama ASC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.GetSetTemplate: %w", err)
	}
	defer rows.Close()

	items := make([]PerlengkapanSetTemplateItem, 0)
	for rows.Next() {
		var it PerlengkapanSetTemplateItem
		if err := rows.Scan(&it.PerlengkapanItemID, &it.Nama, &it.Qty); err != nil {
			return nil, fmt.Errorf("perlengkapan.GetSetTemplate scan: %w", err)
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *Repository) UpdateSetTemplate(ctx context.Context, inputs []SetTemplateItemInput) ([]PerlengkapanSetTemplateItem, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("perlengkapan.UpdateSetTemplate begin tx: %w", err)
	}
	defer tx.Rollback()

	// 1. Validasi semua perlengkapan_item_id valid
	if len(inputs) > 0 {
		ids := make([]string, len(inputs))
		args := make([]interface{}, len(inputs))
		for i, inp := range inputs {
			if inp.Qty < 1 {
				return nil, fmt.Errorf("%w: qty untuk item_id %d minimal 1", ErrInvalidInput, inp.PerlengkapanItemID)
			}
			ids[i] = "?"
			args[i] = inp.PerlengkapanItemID
		}

		queryValidate := fmt.Sprintf(
			"SELECT COUNT(*) FROM perlengkapan_items WHERE id IN (%s)",
			strings.Join(ids, ","),
		)
		var validCount int
		if err := tx.QueryRowContext(ctx, queryValidate, args...).Scan(&validCount); err != nil {
			return nil, fmt.Errorf("perlengkapan.UpdateSetTemplate validate: %w", err)
		}
		if validCount != len(inputs) {
			return nil, fmt.Errorf("%w: salah satu item tidak ditemukan di master items", ErrInvalidInput)
		}
	}

	// 2. Replace-all: DELETE semua lalu INSERT ulang
	if _, err := tx.ExecContext(ctx, "DELETE FROM perlengkapan_set_template"); err != nil {
		return nil, fmt.Errorf("perlengkapan.UpdateSetTemplate delete old: %w", err)
	}

	if len(inputs) > 0 {
		queryInsert := "INSERT INTO perlengkapan_set_template (perlengkapan_item_id, qty) VALUES (?, ?)"
		stmt, err := tx.PrepareContext(ctx, queryInsert)
		if err != nil {
			return nil, fmt.Errorf("perlengkapan.UpdateSetTemplate prepare insert: %w", err)
		}
		defer stmt.Close()

		for _, inp := range inputs {
			if _, err := stmt.ExecContext(ctx, inp.PerlengkapanItemID, inp.Qty); err != nil {
				return nil, fmt.Errorf("perlengkapan.UpdateSetTemplate insert item %d: %w", inp.PerlengkapanItemID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("perlengkapan.UpdateSetTemplate commit: %w", err)
	}

	return r.GetSetTemplate(ctx)
}
