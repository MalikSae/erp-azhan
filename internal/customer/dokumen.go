package customer

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"erp-azhan/api/internal/dokumen"
	"erp-azhan/api/internal/media"
)

// Dokumen jamaah di portal pelanggan: pemesan melihat kelengkapan dokumen tiap jamaah di bookingnya
// dan mengirim berkas (status submitted). Persetujuan tetap oleh admin lewat /api/admin/dokumen.

var labelDokumen = map[string]string{
	"paspor":            "Paspor",
	"ktp":               "KTP",
	"pas_foto":          "Pas foto",
	"kk":                "Kartu Keluarga",
	"buku_nikah":        "Buku nikah",
	"akte_lahir":        "Akta lahir",
	"vaksin_meningitis": "Vaksin meningitis",
}

// dokumenWajib: reguler paspor, KTP, pas foto; infant paspor, akta lahir, pas foto. Selebihnya tambahan (opsional).
func dokumenWajib(paxType string) []string {
	if paxType == "infant" {
		return []string{"paspor", "akte_lahir", "pas_foto"}
	}
	return []string{"paspor", "ktp", "pas_foto"}
}

func dokumenTambahan(paxType string) []string {
	if paxType == "infant" {
		return []string{"kk"}
	}
	return []string{"kk", "buku_nikah", "vaksin_meningitis"}
}

type dokumenItem struct {
	Jenis           string  `json:"jenis"`
	Label           string  `json:"label"`
	Wajib           bool    `json:"wajib"`
	Status          string  `json:"status"` // belum_upload | submitted | approved | rejected
	RejectionReason *string `json:"rejection_reason"`
}

type paxDokumen struct {
	JamaahID int64         `json:"jamaah_id"`
	Nama     string        `json:"nama"`
	PaxType  string        `json:"pax_type"`
	Dokumen  []dokumenItem `json:"dokumen"`
}

// susunDokumen menggabungkan daftar jenis (wajib lalu tambahan) dengan status yang sudah tersimpan.
func susunDokumen(paxType string, ada map[string]dokumenItem) []dokumenItem {
	out := []dokumenItem{}
	tambah := func(jenis string, wajib bool) {
		it, ok := ada[jenis]
		if !ok {
			it = dokumenItem{Status: "belum_upload"}
		}
		it.Jenis, it.Label, it.Wajib = jenis, labelDokumen[jenis], wajib
		out = append(out, it)
	}
	for _, j := range dokumenWajib(paxType) {
		tambah(j, true)
	}
	for _, j := range dokumenTambahan(paxType) {
		tambah(j, false)
	}
	return out
}

// Dokumen GET /api/account/bookings/{id}/dokumen
func (o *Orders) Dokumen(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	if _, _, err := o.owned(r.Context(), acc.ID, id); errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	} else if err != nil {
		log.Printf("[ERROR] account dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil dokumen")
		return
	}
	rows, err := o.db.QueryContext(r.Context(), `
		SELECT j.id, j.nama_lengkap, bp.pax_type, COALESCE(d.jenis,''), COALESCE(d.status,''), d.rejection_reason
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN dokumen_jamaah d ON d.jamaah_id = j.id
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'
		ORDER BY bp.id`, id)
	if err != nil {
		log.Printf("[ERROR] account dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil dokumen")
		return
	}
	defer rows.Close()

	urut := []int64{}
	pax := map[int64]*paxDokumen{}
	ada := map[int64]map[string]dokumenItem{}
	for rows.Next() {
		var jid int64
		var nama, tipe, jenis, status string
		var alasan sql.NullString
		if err := rows.Scan(&jid, &nama, &tipe, &jenis, &status, &alasan); err != nil {
			log.Printf("[ERROR] account dokumen scan: %v", err)
			writeError(w, http.StatusInternalServerError, "gagal mengambil dokumen")
			return
		}
		if _, seen := pax[jid]; !seen {
			pax[jid] = &paxDokumen{JamaahID: jid, Nama: nama, PaxType: tipe}
			ada[jid] = map[string]dokumenItem{}
			urut = append(urut, jid)
		}
		if jenis != "" {
			it := dokumenItem{Status: status}
			if alasan.Valid && strings.TrimSpace(alasan.String) != "" && status == "rejected" {
				s := alasan.String
				it.RejectionReason = &s
			}
			ada[jid][jenis] = it
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] account dokumen rows: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil dokumen")
		return
	}
	out := make([]paxDokumen, 0, len(urut))
	for _, jid := range urut {
		p := pax[jid]
		p.Dokumen = susunDokumen(p.PaxType, ada[jid])
		out = append(out, *p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"pax": out})
}

type kirimDokumenRequest struct {
	JamaahID int64  `json:"jamaah_id"`
	Jenis    string `json:"jenis"`
	FileURL  string `json:"file_url"`
}

// KirimDokumen POST /api/account/bookings/{id}/dokumen
// Berkas harus unggahan akun ini (POST /api/account/media/upload). Dokumen yang sudah disetujui tidak bisa diganti.
func (o *Orders) KirimDokumen(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	var req kirimDokumenRequest
	if !decode(w, r, &req) {
		return
	}
	req.Jenis, req.FileURL = strings.TrimSpace(req.Jenis), strings.TrimSpace(req.FileURL)
	if _, valid := labelDokumen[req.Jenis]; !valid || req.JamaahID <= 0 || req.FileURL == "" {
		writeError(w, http.StatusBadRequest, "data dokumen tidak valid")
		return
	}
	order, _, err := o.owned(r.Context(), acc.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	}
	if err != nil {
		log.Printf("[ERROR] account kirim dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan dokumen")
		return
	}
	if order.Tab == "batal" {
		writeError(w, http.StatusConflict, "booking ini sudah dibatalkan atau kedaluwarsa")
		return
	}
	var jamaahOK bool
	if err := o.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM booking_pax WHERE booking_id=? AND jamaah_id=? AND pax_status='aktif')`, id, req.JamaahID).Scan(&jamaahOK); err != nil {
		log.Printf("[ERROR] account kirim dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan dokumen")
		return
	}
	if !jamaahOK {
		writeError(w, http.StatusNotFound, "jamaah tidak ditemukan di booking ini")
		return
	}
	var status sql.NullString
	if err := o.db.QueryRowContext(r.Context(), `SELECT status FROM dokumen_jamaah WHERE jamaah_id=? AND jenis=?`, req.JamaahID, req.Jenis).Scan(&status); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ERROR] account kirim dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan dokumen")
		return
	}
	if status.Valid && status.String == "approved" {
		writeError(w, http.StatusConflict, "dokumen ini sudah disetujui. Hubungi kami bila perlu menggantinya.")
		return
	}
	if err := media.ValidateCustomerUpload(r.Context(), o.db, req.FileURL, acc.ID); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	d, err := o.dokumen.Upsert(r.Context(), req.JamaahID, &dokumen.CreateDokumenRequest{Jenis: req.Jenis, FileURL: req.FileURL})
	if err != nil {
		log.Printf("[ERROR] account kirim dokumen upsert: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan dokumen")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jenis": d.Jenis, "status": d.Status})
}
