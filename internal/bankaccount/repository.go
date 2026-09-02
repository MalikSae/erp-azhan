package bankaccount

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("rekening tidak ditemukan")

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) List(ctx context.Context, brandID *int64, activeOnly bool) ([]BankAccount, error) {
	q := `SELECT a.id,a.brand_id,b.name,a.bank_name,a.logo_url,a.account_number,a.account_holder,a.instructions,a.is_active,a.sort_order,a.created_at FROM bank_accounts a JOIN brands b ON b.id=a.brand_id WHERE 1=1`
	args := []any{}
	if brandID != nil {
		q += " AND a.brand_id=?"
		args = append(args, *brandID)
	}
	if activeOnly {
		q += " AND a.is_active=TRUE"
	}
	q += " ORDER BY b.name,a.sort_order,a.id"
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BankAccount{}
	for rows.Next() {
		var a BankAccount
		if err := rows.Scan(&a.ID, &a.BrandID, &a.BrandName, &a.BankName, &a.LogoURL, &a.AccountNumber, &a.AccountHolder, &a.Instructions, &a.IsActive, &a.SortOrder, &a.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, a)
	}
	return items, rows.Err()
}

func (r *Repository) Get(ctx context.Context, id int64) (*BankAccount, error) {
	var a BankAccount
	err := r.db.QueryRowContext(ctx, `SELECT a.id,a.brand_id,b.name,a.bank_name,a.logo_url,a.account_number,a.account_holder,a.instructions,a.is_active,a.sort_order,a.created_at FROM bank_accounts a JOIN brands b ON b.id=a.brand_id WHERE a.id=?`, id).Scan(&a.ID, &a.BrandID, &a.BrandName, &a.BankName, &a.LogoURL, &a.AccountNumber, &a.AccountHolder, &a.Instructions, &a.IsActive, &a.SortOrder, &a.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &a, err
}

func (r *Repository) Create(ctx context.Context, req UpsertRequest) (*BankAccount, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO bank_accounts(brand_id,bank_name,logo_url,account_number,account_holder,instructions,is_active,sort_order) VALUES(?,?,?,?,?,?,?,?)`, req.BrandID, req.BankName, req.LogoURL, req.AccountNumber, req.AccountHolder, req.Instructions, req.IsActive, req.SortOrder)
	if err != nil {
		return nil, fmt.Errorf("bankaccount.Create: %w", err)
	}
	id, _ := res.LastInsertId()
	return r.Get(ctx, id)
}
func (r *Repository) Update(ctx context.Context, id int64, req UpsertRequest) (*BankAccount, error) {
	res, err := r.db.ExecContext(ctx, `UPDATE bank_accounts SET brand_id=?,bank_name=?,logo_url=?,account_number=?,account_holder=?,instructions=?,is_active=?,sort_order=? WHERE id=?`, req.BrandID, req.BankName, req.LogoURL, req.AccountNumber, req.AccountHolder, req.Instructions, req.IsActive, req.SortOrder, id)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, ErrNotFound
	}
	return r.Get(ctx, id)
}
func (r *Repository) Delete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM bank_accounts WHERE id=?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
