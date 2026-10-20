package schedule

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
)

// Klasifikasi paket untuk web publik/CMS SEO (SPEK-WEB-PUBLIK-CMS-2026-10-20.md §4.1).
// ERP hanya menyimpan fakta klasifikasi; slug URL dan konten SEO dimiliki CMS.

// Jenis perjalanan yang dikenal spesifikasi SEO.
const (
	JourneyUmroh       = "umroh"
	JourneyHaji        = "haji"
	JourneyWisataHalal = "wisata_halal"
)

// ProgramPlus adalah program yang mewajibkan destinasi (hub /paket/umroh/plus/{destinasi}/).
const ProgramPlus = "plus"

const maxProgramSlugLen = 50

var programSlugRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// DestinationRef adalah ringkasan destinasi yang ditempel pada paket.
type DestinationRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

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
	JourneyType string          `json:"journey_type"`
	ProgramSlug *string         `json:"program"`
	Destination *DestinationRef `json:"destination"`
	Tier        *TierRef        `json:"tier"`
}

// KlasifikasiInput adalah klasifikasi yang sudah divalidasi untuk disimpan.
type KlasifikasiInput struct {
	JourneyType   string
	ProgramSlug   *string
	DestinationID *int64
}

func validJourneyType(s string) bool {
	return s == JourneyUmroh || s == JourneyHaji || s == JourneyWisataHalal
}

// normalizeKlasifikasi memvalidasi isian klasifikasi dari form. journey kosong berarti
// "klasifikasi tidak dikirim" dan mengembalikan (nil, nil) agar nilai tersimpan tidak berubah.
func normalizeKlasifikasi(journey string, program *string, destinationID *int64) (*KlasifikasiInput, error) {
	journey = strings.ToLower(strings.TrimSpace(journey))
	if journey == "" {
		return nil, nil
	}
	if !validJourneyType(journey) {
		return nil, fmt.Errorf("journey_type tidak valid, pilih umroh, haji, atau wisata_halal")
	}

	var prog *string
	if program != nil {
		p := strings.ToLower(strings.TrimSpace(*program))
		if p != "" {
			if len(p) > maxProgramSlugLen || !programSlugRe.MatchString(p) {
				return nil, fmt.Errorf("program_slug hanya huruf kecil, angka, dan tanda hubung, maksimal %d karakter", maxProgramSlugLen)
			}
			prog = &p
		}
	}

	var dest *int64
	if destinationID != nil && *destinationID > 0 {
		d := *destinationID
		dest = &d
	}

	if prog != nil && *prog == ProgramPlus && dest == nil {
		return nil, fmt.Errorf("destination_id wajib diisi untuk program plus")
	}
	return &KlasifikasiInput{JourneyType: journey, ProgramSlug: prog, DestinationID: dest}, nil
}

// DestinationActive memeriksa destinasi ada dan aktif.
func (r *Repository) DestinationActive(ctx context.Context, id int64) (bool, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM destinations WHERE id=? AND is_active=TRUE`, id).Scan(&n); err != nil {
		return false, fmt.Errorf("schedule.DestinationActive: %w", err)
	}
	return n > 0, nil
}

// setKlasifikasiTx menyimpan klasifikasi dalam transaksi yang sama dengan penyimpanan paket.
func setKlasifikasiTx(ctx context.Context, tx *sql.Tx, id int64, k *KlasifikasiInput) error {
	if k == nil {
		return nil
	}
	var prog sql.NullString
	if k.ProgramSlug != nil {
		prog = sql.NullString{String: *k.ProgramSlug, Valid: true}
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE schedules SET journey_type=?, program_slug=?, destination_id=? WHERE id=?`,
		k.JourneyType, prog, nullInt64Ptr(k.DestinationID), id); err != nil {
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
		SELECT s.id, s.journey_type, s.program_slug,
			d.id, d.name, d.slug,
			cat.id, cat.name, cat.slug, cat.seo_tier
		FROM schedules s
		LEFT JOIN destinations d ON d.id = s.destination_id
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
			prog                   sql.NullString
			dID                    sql.NullInt64
			dName, dSlug           sql.NullString
			cID                    sql.NullInt64
			cName, cSlug, cSEOTier sql.NullString
		)
		if err := rows.Scan(&id, &k.JourneyType, &prog, &dID, &dName, &dSlug, &cID, &cName, &cSlug, &cSEOTier); err != nil {
			return nil, fmt.Errorf("schedule.loadKlasifikasi scan: %w", err)
		}
		if prog.Valid {
			k.ProgramSlug = &prog.String
		}
		if dID.Valid {
			k.Destination = &DestinationRef{ID: dID.Int64, Name: dName.String, Slug: dSlug.String}
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
	s.ProgramSlug = k.ProgramSlug
	s.Destination = k.Destination
	if k.Destination != nil {
		id := k.Destination.ID
		s.DestinationID = &id
	}
}
