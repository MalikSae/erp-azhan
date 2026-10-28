package customer

import (
	"errors"
	"log"
	"net/http"
)

// Progres persiapan keberangkatan di portal pelanggan. Semua dihitung dari data yang sudah ada (dokumen jamaah dan penanda
// progres di booking), tanpa mengubah status booking: status itu dihitung ulang dari pembayaran dan tidak boleh ditimpa.

type progressPax struct {
	JamaahID         int64  `json:"jamaah_id"`
	Nama             string `json:"nama"`
	PaxType          string `json:"pax_type"`
	DokumenWajib     int    `json:"dokumen_wajib"`
	DokumenSetuju    int    `json:"dokumen_disetujui"`
	DokumenLengkap   bool   `json:"dokumen_lengkap"`
	Visa             bool   `json:"visa"`
	Siskopatuh       bool   `json:"siskopatuh"`
	Manasik          bool   `json:"manasik"`
	VaksinMeningitis bool   `json:"vaksin_meningitis"`
}

type progressHeader struct {
	Tiket bool `json:"tiket"`
	Hotel bool `json:"hotel"`
	Land  bool `json:"land_arrangement"`
}

// hitungDokumenLengkap mengisi jumlah dokumen wajib yang disetujui dan apakah semuanya sudah disetujui.
func (p *progressPax) hitungDokumenLengkap(disetujui map[string]bool) {
	wajib := dokumenWajib(p.PaxType)
	p.DokumenWajib, p.DokumenSetuju = len(wajib), 0
	for _, j := range wajib {
		if disetujui[j] {
			p.DokumenSetuju++
		}
	}
	p.DokumenLengkap = p.DokumenWajib > 0 && p.DokumenSetuju == p.DokumenWajib
}

// Progress GET /api/account/bookings/{id}/progress
func (o *Orders) Progress(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	if _, _, err := o.owned(r.Context(), acc.ID, id); errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	} else if err != nil {
		log.Printf("[ERROR] account progress: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil progres")
		return
	}
	b, err := o.bookingRepo.GetByID(r.Context(), id, nil)
	if err != nil {
		log.Printf("[ERROR] account progress booking: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil progres")
		return
	}

	rows, err := o.db.QueryContext(r.Context(), `
		SELECT bp.jamaah_id, d.jenis FROM booking_pax bp
		JOIN dokumen_jamaah d ON d.jamaah_id = bp.jamaah_id AND d.status = 'approved'
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'`, id)
	if err != nil {
		log.Printf("[ERROR] account progress dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil progres")
		return
	}
	defer rows.Close()
	disetujui := map[int64]map[string]bool{}
	for rows.Next() {
		var jid int64
		var jenis string
		if err := rows.Scan(&jid, &jenis); err != nil {
			log.Printf("[ERROR] account progress scan: %v", err)
			writeError(w, http.StatusInternalServerError, "gagal mengambil progres")
			return
		}
		if disetujui[jid] == nil {
			disetujui[jid] = map[string]bool{}
		}
		disetujui[jid][jenis] = true
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] account progress rows: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil progres")
		return
	}

	pax := []progressPax{}
	semuaLengkap := true
	for _, p := range b.Pax {
		if p.PaxStatus != "aktif" {
			continue
		}
		item := progressPax{
			JamaahID: p.JamaahID, Nama: p.NamaJamaah, PaxType: p.PaxType,
			Visa: p.ProgressVisa, Siskopatuh: p.ProgressSiskopatuh, Manasik: p.ProgressManasik, VaksinMeningitis: p.ProgressVaksinMeningitis,
		}
		item.hitungDokumenLengkap(disetujui[p.JamaahID])
		if !item.DokumenLengkap {
			semuaLengkap = false
		}
		pax = append(pax, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"dokumen_lengkap": len(pax) > 0 && semuaLengkap,
		"siap_berangkat":  b.SiapBerangkat,
		"header":          progressHeader{Tiket: b.ProgressTiket, Hotel: b.ProgressHotel, Land: b.ProgressLandArrangement},
		"pax":             pax,
	})
}
