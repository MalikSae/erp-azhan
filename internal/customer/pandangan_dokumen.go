package customer

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"erp-azhan/api/internal/surat"
)

// Dokumen dan surat untuk peran jamaah (ADR 0006): hanya baca status, tanpa isi berkas. Jalur terpisah dari
// Dokumen/KirimDokumen/SuratDaftar/SuratUnduh yang tetap pemilik-saja.

type dokumenPeran struct {
	JamaahID int64         `json:"jamaah_id"`
	Nama     string        `json:"nama"`
	PaxType  string        `json:"pax_type"`
	Dokumen  []dokumenItem `json:"dokumen"`
}

type suratPeran struct {
	Slug   string `json:"slug"`
	Label  string `json:"label"`
	Nomor  string `json:"nomor"`
	Terbit string `json:"terbit"`
}

// bookingPeran memeriksa booking termasuk kartu peran jamaah akun ini (jamaah tertaut hidup).
func bookingPeran(ctx context.Context, q dbq, akunID, bookingID int64) (jid int64, nama, paxType string, ok bool, err error) {
	jid, tertaut, err := jamaahTertaut(ctx, q, akunID)
	if err != nil || !tertaut {
		return 0, "", "", false, err
	}
	err = q.QueryRowContext(ctx, `SELECT j.nama_lengkap, bp.pax_type `+sqlBookingPeran+` AND b.id = ? LIMIT 1`, jid, akunID, bookingID).Scan(&nama, &paxType)
	if err == sql.ErrNoRows {
		return 0, "", "", false, nil
	}
	if err != nil {
		return 0, "", "", false, err
	}
	return jid, nama, paxType, true, nil
}

// muatDokumenPeran: jenis dan status dokumen jamaah tertaut. Tanpa file_url dan tanpa alasan penolakan.
func muatDokumenPeran(ctx context.Context, q dbq, jid int64, paxType string) ([]dokumenItem, error) {
	rows, err := q.QueryContext(ctx, `SELECT jenis, status FROM dokumen_jamaah WHERE jamaah_id = ?`, jid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ada := map[string]dokumenItem{}
	for rows.Next() {
		var jenis, status string
		if err := rows.Scan(&jenis, &status); err != nil {
			return nil, err
		}
		ada[jenis] = dokumenItem{Status: status}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return susunDokumen(paxType, ada), nil
}

// muatSuratPeran: surat yang sudah diterbitkan untuk jamaah tertaut di booking ini. Tanpa unduhan.
func muatSuratPeran(ctx context.Context, q dbq, jid, bookingID int64) ([]suratPeran, error) {
	rows, err := q.QueryContext(ctx, `SELECT jenis, nomor, DATE_FORMAT(terbit_at,'%Y-%m-%d') FROM surat_jamaah WHERE booking_id = ? AND jamaah_id = ? ORDER BY id`, bookingID, jid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []suratPeran{}
	for rows.Next() {
		var jenis string
		var s suratPeran
		if err := rows.Scan(&jenis, &s.Nomor, &s.Terbit); err != nil {
			return nil, err
		}
		s.Slug = slugJenis(jenis)
		if s.Slug == "" {
			continue
		}
		s.Label = surat.LabelJenis(jenis)
		out = append(out, s)
	}
	return out, rows.Err()
}

// DokumenPeran GET /api/account/perjalanan-jamaah/{id}/dokumen
func (o *Orders) DokumenPeran(w http.ResponseWriter, r *http.Request) {
	dokumenPeranHandler(o.db, w, r)
}

func dokumenPeranHandler(q dbq, w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	jid, nama, tipe, ada, err := bookingPeran(r.Context(), q, acc.ID, id)
	if err == nil && ada {
		var items []dokumenItem
		if items, err = muatDokumenPeran(r.Context(), q, jid, tipe); err == nil {
			writeJSON(w, http.StatusOK, map[string]any{"jamaah": dokumenPeran{JamaahID: jid, Nama: nama, PaxType: tipe, Dokumen: items}})
			return
		}
	}
	if err != nil {
		log.Printf("[ERROR] account dokumen peran: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil dokumen.")
		return
	}
	writeError(w, http.StatusNotFound, "Perjalanan tidak ditemukan.")
}

// SuratPeran GET /api/account/perjalanan-jamaah/{id}/surat
func (o *Orders) SuratPeran(w http.ResponseWriter, r *http.Request) {
	suratPeranHandler(o.db, w, r)
}

func suratPeranHandler(q dbq, w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	jid, _, _, ada, err := bookingPeran(r.Context(), q, acc.ID, id)
	if err == nil && ada {
		var items []suratPeran
		if items, err = muatSuratPeran(r.Context(), q, jid, id); err == nil {
			writeJSON(w, http.StatusOK, map[string]any{"surat": items})
			return
		}
	}
	if err != nil {
		log.Printf("[ERROR] account surat peran: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil surat.")
		return
	}
	writeError(w, http.StatusNotFound, "Perjalanan tidak ditemukan.")
}
