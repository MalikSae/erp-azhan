package jamaah

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"erp-azhan/api/internal/identity"
)

// AnggotaDuplikat adalah satu baris jamaah dalam kelompok kemungkinan duplikat lintas brand.
type AnggotaDuplikat struct {
	ID           int64   `json:"id"`
	BrandID      int64   `json:"brand_id"`
	BrandName    string  `json:"brand_name"`
	IDJamaah     string  `json:"id_jamaah"`
	NamaLengkap  string  `json:"nama_lengkap"`
	NIK          *string `json:"nik"`
	NoHP         *string `json:"no_hp"`
	Status       string  `json:"status"`
	TotalBooking int     `json:"total_booking"`
}

// KelompokDuplikat mengelompokkan jamaah di brand berbeda yang memakai identitas yang sama.
// Jenis "nik" = NIK sama persis (kemungkinan besar orang yang sama); "no_hp" = nomor HP
// sama (bisa orang yang sama, bisa juga keluarga yang berbagi nomor).
type KelompokDuplikat struct {
	Jenis   string            `json:"jenis"`
	Kunci   string            `json:"kunci"`
	Anggota []AnggotaDuplikat `json:"anggota"`
}

// ListDuplikatLintasBrand mengembalikan kelompok jamaah dengan NIK atau nomor HP sama
// yang tersebar di lebih dari satu brand.
func (r *Repository) ListDuplikatLintasBrand(ctx context.Context) ([]KelompokDuplikat, error) {
	out := make([]KelompokDuplikat, 0)
	for _, spec := range []struct{ jenis, kolom string }{
		{"nik", "j.nik"},
		{"no_hp", "j.no_hp_normal"},
	} {
		q := fmt.Sprintf(`
			SELECT %[1]s AS kunci, j.id, j.brand_id, b.name, COALESCE(j.id_jamaah, ''), j.nama_lengkap,
				j.nik, j.no_hp, j.status,
				(SELECT COUNT(*) FROM (
					SELECT id AS booking_id FROM bookings WHERE pic_jamaah_id = j.id
					UNION
					SELECT booking_id FROM booking_pax WHERE jamaah_id = j.id
				) bk) AS total_booking
			FROM jamaah j
			JOIN brands b ON b.id = j.brand_id
			WHERE %[1]s IN (
				SELECT %[2]s FROM jamaah
				WHERE %[2]s IS NOT NULL AND %[2]s <> ''
				GROUP BY %[2]s HAVING COUNT(DISTINCT brand_id) > 1
			)
			ORDER BY %[1]s, b.name, j.nama_lengkap`, spec.kolom, spec.kolom[2:])
		rows, err := r.db.QueryContext(ctx, q)
		if err != nil {
			return nil, fmt.Errorf("jamaah.ListDuplikatLintasBrand %s: %w", spec.jenis, err)
		}
		var cur *KelompokDuplikat
		for rows.Next() {
			var kunci string
			var a AnggotaDuplikat
			if err := rows.Scan(&kunci, &a.ID, &a.BrandID, &a.BrandName, &a.IDJamaah, &a.NamaLengkap,
				&a.NIK, &a.NoHP, &a.Status, &a.TotalBooking); err != nil {
				rows.Close()
				return nil, fmt.Errorf("jamaah.ListDuplikatLintasBrand scan: %w", err)
			}
			if cur == nil || cur.Kunci != kunci {
				out = append(out, KelompokDuplikat{Jenis: spec.jenis, Kunci: kunci})
				cur = &out[len(out)-1]
			}
			cur.Anggota = append(cur.Anggota, a)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// ListDuplikatLintasBrand godoc
// GET /api/admin/jamaah/duplikat-lintas-brand  (khusus holding)
func (h *Handler) ListDuplikatLintasBrand(w http.ResponseWriter, r *http.Request) {
	if identity.GetBrandID(r.Context()) != nil {
		writeError(w, http.StatusForbidden, "hanya tersedia untuk akun holding")
		return
	}
	items, err := h.repo.ListDuplikatLintasBrand(r.Context())
	if err != nil {
		log.Printf("[ERROR] jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar duplikat")
		return
	}
	writeJSON(w, http.StatusOK, items)
}
