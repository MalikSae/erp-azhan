package brand

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

var ErrNotFound = errors.New("brand tidak ditemukan")
var ErrInUse = errors.New("tidak bisa dihapus, masih dipakai oleh paket lain")

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const selectCols = `id, kode_brand, jamaah_counter, name, domain, whatsapp_number, email, phone, address, city, province, gmaps_url, legalitas, bank_name, bank_account_number, bank_account_holder, social_facebook, social_instagram, social_tiktok, social_youtube, logo_url, icon_url, primary_color, meta_title, meta_description, og_image_url, google_verification_code, minimal_dp, created_at`

func scanBrand(scanner interface{ Scan(dest ...any) error }, b *Brand) error {
	return scanner.Scan(
		&b.ID,
		&b.KodeBrand,
		&b.JamaahCounter,
		&b.Name,
		&b.Domain,
		&b.WhatsappNumber,
		&b.Email,
		&b.Phone,
		&b.Address,
		&b.City,
		&b.Province,
		&b.GmapsURL,
		&b.Legalitas,
		&b.BankName,
		&b.BankAccountNumber,
		&b.BankAccountHolder,
		&b.SocialFacebook,
		&b.SocialInstagram,
		&b.SocialTiktok,
		&b.SocialYoutube,
		&b.LogoURL,
		&b.IconURL,
		&b.PrimaryColor,
		&b.MetaTitle,
		&b.MetaDescription,
		&b.OgImageURL,
		&b.GoogleVerificationCode,
		&b.MinimalDP,
		&b.CreatedAt,
	)
}

func (r *Repository) List(ctx context.Context) ([]Brand, error) {
	q := fmt.Sprintf(`SELECT %s FROM brands ORDER BY name ASC`, selectCols)
	rows, err := r.db.QueryContext(ctx, q)
	if err != nil {
		return nil, fmt.Errorf("brand.List Query: %w", err)
	}
	defer rows.Close()

	var result []Brand
	for rows.Next() {
		var b Brand
		if err := scanBrand(rows, &b); err != nil {
			return nil, fmt.Errorf("brand.List Scan: %w", err)
		}
		result = append(result, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("brand.List Rows: %w", err)
	}

	if result == nil {
		result = []Brand{}
	}

	return result, nil
}

func (r *Repository) GetByID(ctx context.Context, id uint64) (*Brand, error) {
	q := fmt.Sprintf(`SELECT %s FROM brands WHERE id = ?`, selectCols)
	var b Brand
	err := scanBrand(r.db.QueryRowContext(ctx, q, id), &b)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("brand.GetByID: %w", err)
	}
	return &b, nil
}

func (r *Repository) GetByDomain(ctx context.Context, domain string) (*Brand, error) {
	q := fmt.Sprintf(`SELECT %s FROM brands WHERE domain = ?`, selectCols)
	var b Brand
	err := scanBrand(r.db.QueryRowContext(ctx, q, domain), &b)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // return nil (bukan error) kalau tidak ketemu
		}
		return nil, fmt.Errorf("brand.GetByDomain: %w", err)
	}
	return &b, nil
}

func (r *Repository) GetByKodeBrand(ctx context.Context, kode string) (*Brand, error) {
	q := fmt.Sprintf(`SELECT %s FROM brands WHERE UPPER(kode_brand) = UPPER(?)`, selectCols)
	var b Brand
	err := scanBrand(r.db.QueryRowContext(ctx, q, kode), &b)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // return nil if not found
		}
		return nil, fmt.Errorf("brand.GetByKodeBrand: %w", err)
	}
	return &b, nil
}

func (r *Repository) Create(ctx context.Context, req CreateBrandRequest) (*Brand, error) {
	const q = `
		INSERT INTO brands (
			kode_brand, name, domain, whatsapp_number, email, phone, address, city, province, gmaps_url, legalitas,
			bank_name, bank_account_number, bank_account_holder,
			social_facebook, social_instagram, social_tiktok, social_youtube,
			logo_url, icon_url, primary_color,
			meta_title, meta_description, og_image_url, google_verification_code, minimal_dp
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	res, err := r.db.ExecContext(ctx, q,
		req.KodeBrand, req.Name, req.Domain, req.WhatsappNumber, req.Email, req.Phone, req.Address, req.City, req.Province, req.GmapsURL, req.Legalitas,
		req.BankName, req.BankAccountNumber, req.BankAccountHolder,
		req.SocialFacebook, req.SocialInstagram, req.SocialTiktok, req.SocialYoutube,
		req.LogoURL, req.IconURL, req.PrimaryColor,
		req.MetaTitle, req.MetaDescription, req.OgImageURL, req.GoogleVerificationCode, req.MinimalDP,
	)
	if err != nil {
		return nil, fmt.Errorf("brand.Create Exec: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("brand.Create LastInsertId: %w", err)
	}

	return r.GetByID(ctx, uint64(id))
}

func (r *Repository) Update(ctx context.Context, id uint64, req UpdateBrandRequest) (*Brand, error) {
	if _, err := r.GetByID(ctx, id); err != nil {
		return nil, err
	}

	const q = `
		UPDATE brands 
		SET kode_brand = ?, name = ?, domain = ?, whatsapp_number = ?, email = ?, phone = ?, address = ?, city = ?, province = ?, gmaps_url = ?, legalitas = ?,
		    bank_name = ?, bank_account_number = ?, bank_account_holder = ?,
		    social_facebook = ?, social_instagram = ?, social_tiktok = ?, social_youtube = ?,
		    logo_url = ?, icon_url = ?, primary_color = ?,
		    meta_title = ?, meta_description = ?, og_image_url = ?, google_verification_code = ?, minimal_dp = ?
		WHERE id = ?
	`
	_, err := r.db.ExecContext(ctx, q,
		req.KodeBrand, req.Name, req.Domain, req.WhatsappNumber, req.Email, req.Phone, req.Address, req.City, req.Province, req.GmapsURL, req.Legalitas,
		req.BankName, req.BankAccountNumber, req.BankAccountHolder,
		req.SocialFacebook, req.SocialInstagram, req.SocialTiktok, req.SocialYoutube,
		req.LogoURL, req.IconURL, req.PrimaryColor,
		req.MetaTitle, req.MetaDescription, req.OgImageURL, req.GoogleVerificationCode, req.MinimalDP,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("brand.Update Exec: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id uint64) error {
	const q = `DELETE FROM brands WHERE id = ?`
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		if strings.Contains(err.Error(), "1451") {
			return ErrInUse
		}
		return fmt.Errorf("brand.Delete Exec: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
