package category

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

var (
	ErrNotFound      = errors.New("category tidak ditemukan")
	ErrSlugDuplicate = errors.New("slug kategori sudah digunakan")
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// List mengambil daftar kategori, opsional filter brandID dan activeOnly.
func (r *Repository) List(ctx context.Context, brandID *int64, activeOnly bool) ([]CategoryListItem, error) {
	q := `
		SELECT 
			c.id, c.name, c.slug, c.description, c.is_active,
			c.created_at, c.updated_at,
			(
				SELECT COALESCE(JSON_ARRAYAGG(JSON_OBJECT('id', b.id, 'name', b.name, 'primary_color', b.primary_color)), '[]')
				FROM category_brands cb
				JOIN brands b ON b.id = cb.brand_id
				WHERE cb.category_id = c.id
			) AS brands,
			(
				SELECT COUNT(*) 
				FROM schedules s 
				WHERE s.category_id = c.id
			) AS package_count
		FROM package_categories c
		WHERE 1=1`

	var args []interface{}
	if activeOnly {
		q += " AND c.is_active = TRUE"
	}
	if brandID != nil && *brandID > 0 {
		q += " AND EXISTS (SELECT 1 FROM category_brands cb WHERE cb.category_id = c.id AND cb.brand_id = ?)"
		args = append(args, *brandID)
	}

	q += " ORDER BY c.name ASC"

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("category.List: %w", err)
	}
	defer rows.Close()

	items := make([]CategoryListItem, 0)
	for rows.Next() {
		var item CategoryListItem
		var brandsJSON []byte
		if err := rows.Scan(
			&item.ID, &item.Name, &item.Slug, &item.Description, &item.IsActive,
			&item.CreatedAt, &item.UpdatedAt,
			&brandsJSON, &item.PackageCount,
		); err != nil {
			return nil, fmt.Errorf("category.List scan: %w", err)
		}

		item.Brands = make([]BrandRef, 0)
		if len(brandsJSON) > 0 {
			_ = json.Unmarshal(brandsJSON, &item.Brands)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// GetByID mengambil detail kategori berdasarkan ID.
func (r *Repository) GetByID(ctx context.Context, id int64) (*Category, error) {
	q := `
		SELECT 
			c.id, c.name, c.slug, c.description, c.is_active,
			c.created_at, c.updated_at,
			(
				SELECT COALESCE(JSON_ARRAYAGG(JSON_OBJECT('id', b.id, 'name', b.name, 'primary_color', b.primary_color)), '[]')
				FROM category_brands cb
				JOIN brands b ON b.id = cb.brand_id
				WHERE cb.category_id = c.id
			) AS brands
		FROM package_categories c
		WHERE c.id = ?`

	var cat Category
	var brandsJSON []byte
	err := r.db.QueryRowContext(ctx, q, id).Scan(
		&cat.ID, &cat.Name, &cat.Slug, &cat.Description, &cat.IsActive,
		&cat.CreatedAt, &cat.UpdatedAt,
		&brandsJSON,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("category.GetByID: %w", err)
	}

	cat.Brands = make([]BrandRef, 0)
	if len(brandsJSON) > 0 {
		_ = json.Unmarshal(brandsJSON, &cat.Brands)
	}

	return &cat, nil
}

// CheckSlugExists memeriksa apakah slug sudah dipakai.
func (r *Repository) CheckSlugExists(ctx context.Context, slug string, excludeID *int64) (bool, error) {
	q := "SELECT COUNT(*) FROM package_categories WHERE slug = ?"
	args := []interface{}{slug}
	if excludeID != nil {
		q += " AND id != ?"
		args = append(args, *excludeID)
	}

	var count int
	if err := r.db.QueryRowContext(ctx, q, args...).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

// ValidateBrandIDs memastikan semua ID brand yang dikirim valid dan aktif di DB.
func (r *Repository) ValidateBrandIDs(ctx context.Context, brandIDs []int64) (bool, error) {
	if len(brandIDs) == 0 {
		return true, nil
	}

	for _, id := range brandIDs {
		var exists bool
		err := r.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM brands WHERE id = ?)", id).Scan(&exists)
		if err != nil {
			return false, err
		}
		if !exists {
			return false, nil
		}
	}
	return true, nil
}

// Create menyisipkan kategori baru beserta relasi brand-nya.
func (r *Repository) Create(ctx context.Context, req CreateCategoryRequest) (*Category, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("category.Create tx: %w", err)
	}
	defer tx.Rollback()

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	res, err := tx.ExecContext(ctx,
		`INSERT INTO package_categories (name, slug, description, is_active) VALUES (?, ?, ?, ?)`,
		req.Name, req.Slug, req.Description, isActive,
	)
	if err != nil {
		return nil, fmt.Errorf("category.Create insert: %w", err)
	}

	catID, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("category.Create lastInsertId: %w", err)
	}

	for _, brandID := range req.BrandIDs {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO category_brands (category_id, brand_id) VALUES (?, ?)`,
			catID, brandID,
		)
		if err != nil {
			return nil, fmt.Errorf("category.Create brand link: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("category.Create commit: %w", err)
	}

	return r.GetByID(ctx, catID)
}

// Update memperbarui data kategori dan relasi brand-nya (replace-all).
func (r *Repository) Update(ctx context.Context, id int64, req UpdateCategoryRequest) (*Category, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("category.Update tx: %w", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM package_categories WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrNotFound
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	_, err = tx.ExecContext(ctx,
		`UPDATE package_categories SET name = ?, slug = ?, description = ?, is_active = ? WHERE id = ?`,
		req.Name, req.Slug, req.Description, isActive, id,
	)
	if err != nil {
		return nil, fmt.Errorf("category.Update update: %w", err)
	}

	// Replace-all relasi brand
	if _, err := tx.ExecContext(ctx, "DELETE FROM category_brands WHERE category_id = ?", id); err != nil {
		return nil, fmt.Errorf("category.Update delete brands: %w", err)
	}

	for _, brandID := range req.BrandIDs {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO category_brands (category_id, brand_id) VALUES (?, ?)`,
			id, brandID,
		)
		if err != nil {
			return nil, fmt.Errorf("category.Update brand link: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("category.Update commit: %w", err)
	}

	return r.GetByID(ctx, id)
}

// Delete menghapus kategori paket.
func (r *Repository) Delete(ctx context.Context, id int64) error {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM schedules WHERE category_id = ?", id).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("tidak bisa dihapus, masih dipakai oleh %d paket umroh", count)
	}

	res, err := r.db.ExecContext(ctx, "DELETE FROM package_categories WHERE id = ?", id)
	if err != nil {
		return err
	}
	rowsAff, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAff == 0 {
		return ErrNotFound
	}
	return nil
}
