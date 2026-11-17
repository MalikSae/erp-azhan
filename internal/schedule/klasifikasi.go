package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Klasifikasi paket untuk web publik/CMS SEO (SPEK-WEB-PUBLIK-CMS-2026-10-20.md §4.1).
// ERP hanya menyimpan jenis perjalanan dan tier; program, destinasi, slug URL, dan konten SEO dimiliki CMS
// (keputusan 17 November 2026).

// Jenis perjalanan yang dikenal spesifikasi SEO.
const (
	JourneyUmroh       = "umroh"
	JourneyHaji        = "haji"
	JourneyWisataHalal = "wisata_halal"
)

// TierRef menyatakan tier paket yang diturunkan dari kategori. SEO kosong bila kategori
// belum dipetakan ke tier SEO (package_categories.seo_tier).
type TierRef struct {
	SEO          *string `json:"seo"`
	CategoryID   int64   `json:"category_id"`
	CategoryName string  `json:"category_name"`
	CategorySlug string  `json:"category_slug"`
}

// Klasifikasi adalah atribut klasifikasi satu keberangkatan.
type Klasifikasi struct {
	JourneyType string   `json:"journey_type"`
	Tier        *TierRef `json:"tier"`
}

// KlasifikasiInput adalah klasifikasi yang sudah divalidasi untuk disimpan.
type KlasifikasiInput struct {
	JourneyType string
}

func validJourneyType(s string) bool {
	return s == JourneyUmroh || s == JourneyHaji || s == JourneyWisataHalal
}

// normalizeKlasifikasi memvalidasi isian klasifikasi dari form. journey kosong berarti
// "klasifikasi tidak dikirim" dan mengembalikan (nil, nil) agar nilai tersimpan tidak berubah.
func normalizeKlasifikasi(journey string) (*KlasifikasiInput, error) {
	journey = strings.ToLower(strings.TrimSpace(journey))
	if journey == "" {
		return nil, nil
	}
	if !validJourneyType(journey) {
		return nil, fmt.Errorf("journey_type tidak valid, pilih umroh, haji, atau wisata_halal")
	}
	return &KlasifikasiInput{JourneyType: journey}, nil
}

// setKlasifikasiTx menyimpan klasifikasi dalam transaksi yang sama dengan penyimpanan paket.
func setKlasifikasiTx(ctx context.Context, tx *sql.Tx, id int64, k *KlasifikasiInput) error {
	if k == nil {
		return nil
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE schedules SET journey_type=? WHERE id=?`, k.JourneyType, id); err != nil {
		return fmt.Errorf("schedule.setKlasifikasi: %w", err)
	}
	return nil
}

// loadKlasifikasi mengambil klasifikasi (termasuk tier dari kategori) untuk banyak paket sekaligus.
func (r *Repository) loadKlasifikasi(ctx context.Context, ids []int64) (map[int64]Klasifikasi, error) {
	out := make(map[int64]Klasifikasi, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	marks := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		marks[i] = "?"
		args[i] = id
	}
	q := `
		SELECT s.id, s.journey_type,
			cat.id, cat.name, cat.slug, cat.seo_tier
		FROM schedules s
		LEFT JOIN package_categories cat ON cat.id = s.category_id
		WHERE s.id IN (` + strings.Join(marks, ",") + `)`
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("schedule.loadKlasifikasi: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id                     int64
			k                      Klasifikasi
			cID                    sql.NullInt64
			cName, cSlug, cSEOTier sql.NullString
		)
		if err := rows.Scan(&id, &k.JourneyType, &cID, &cName, &cSlug, &cSEOTier); err != nil {
			return nil, fmt.Errorf("schedule.loadKlasifikasi scan: %w", err)
		}
		if cID.Valid {
			t := &TierRef{CategoryID: cID.Int64, CategoryName: cName.String, CategorySlug: cSlug.String}
			if cSEOTier.Valid {
				t.SEO = &cSEOTier.String
			}
			k.Tier = t
		}
		out[id] = k
	}
	return out, rows.Err()
}

// applyKlasifikasi menempelkan klasifikasi ke Schedule admin.
func (s *Schedule) applyKlasifikasi(k Klasifikasi, ok bool) {
	if !ok {
		s.JourneyType = JourneyUmroh
		return
	}
	s.JourneyType = k.JourneyType
}
