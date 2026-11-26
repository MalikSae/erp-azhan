package customer

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strings"

	"erp-azhan/api/internal/dokumen"
	"erp-azhan/api/internal/media"
	"erp-azhan/api/internal/shared"
)

// Dokumen jamaah di portal pelanggan: pemesan melihat kelengkapan dokumen tiap jamaah di bookingnya
// dan mengirim berkas (status submitted). Persetujuan tetap oleh admin lewat PUT /api/admin/dokumen/{id}/status.

var labelDokumen = shared.LabelDokumen

// dokumenWajib: reguler paspor, KTP, pas foto; infant paspor, KK, pas foto. Selebihnya tambahan (opsional).
func dokumenWajib(paxType string) []string { return shared.DokumenWajibJamaah(paxType) }

// dokumenTambahan: infant tanpa tambahan; akta lahir bayi tidak tampil di web dan app (keputusan 2026-11-26),
// hanya tersedia di admin.
func dokumenTambahan(paxType string) []string {
	if paxType == "infant" {
		return []string{}
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
	// AksesTerbatas: akun bukan pembuat jamaah; hanya boleh mengunggah jenis yang belum ada berkasnya
	// atau yang ditolak (keputusan 2026-11-25).
	AksesTerbatas bool     `json:"akses_terbatas"`
	// WajibDisetujui dari WajibTotal: dokumen wajib yang sudah disetujui admin (keputusan 2026-11-25:
	// hanya approved yang dihitung lengkap).
	WajibDisetujui int           `json:"wajib_disetujui"`
	WajibTotal     int           `json:"wajib_total"`
	Dokumen        []dokumenItem `json:"dokumen"`
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

// hitungWajibDisetujui: jumlah dokumen wajib berstatus approved dan jumlah dokumen wajib.
func hitungWajibDisetujui(items []dokumenItem) (disetujui, total int) {
	for _, it := range items {
		if !it.Wajib {
			continue
		}
		total++
		if it.Status == "approved" {
			disetujui++
		}
	}
	return disetujui, total
}

// Dokumen GET /api/account/bookings/{id}/dokumen
func (o *Orders) Dokumen(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	if _, _, err := o.owned(r.Context(), acc.ID, id); errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Booking tidak ditemukan.")
		return
	} else if err != nil {
		log.Printf("[ERROR] account dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil dokumen.")
		return
	}
	rows, err := o.db.QueryContext(r.Context(), `
		SELECT j.id, j.nama_lengkap, bp.pax_type, COALESCE(d.jenis,''), COALESCE(d.status,''), d.rejection_reason, ` + sqlAksesPenuh + `
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN dokumen_jamaah d ON d.jamaah_id = j.id
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'
		ORDER BY bp.id`, acc.ID, id)
	if err != nil {
		log.Printf("[ERROR] account dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil dokumen.")
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
		var penuh bool
		if err := rows.Scan(&jid, &nama, &tipe, &jenis, &status, &alasan, &penuh); err != nil {
			log.Printf("[ERROR] account dokumen scan: %v", err)
			writeError(w, http.StatusInternalServerError, "Gagal mengambil dokumen.")
			return
		}
		if _, seen := pax[jid]; !seen {
			pax[jid] = &paxDokumen{JamaahID: jid, Nama: nama, PaxType: tipe, AksesTerbatas: !penuh}
			ada[jid] = map[string]dokumenItem{}
			urut = append(urut, jid)
		}
		if jenis != "" {
			it := dokumenItem{Status: status}
			// Alasan penolakan tampil juga untuk akses terbatas: akun itu boleh mengganti dokumen yang ditolak.
			if alasan.Valid && strings.TrimSpace(alasan.String) != "" && status == "rejected" {
				s := alasan.String
				it.RejectionReason = &s
			}
			ada[jid][jenis] = it
		}
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] account dokumen rows: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal mengambil dokumen.")
		return
	}
	out := make([]paxDokumen, 0, len(urut))
	for _, jid := range urut {
		p := pax[jid]
		p.Dokumen = susunDokumen(p.PaxType, ada[jid])
		p.WajibDisetujui, p.WajibTotal = hitungWajibDisetujui(p.Dokumen)
		out = append(out, *p)
	}
	writeJSON(w, http.StatusOK, map[string]any{"pax": out})
}

type kirimDokumenRequest struct {
	JamaahID int64  `json:"jamaah_id"`
	Jenis    string `json:"jenis"`
	FileURL  string `json:"file_url"`
}

// bolehUnggahDokumen: unggah dokumen oleh pelanggan dibuka setelah booking minimal DP (keputusan 2026-11-25).
// dokumen_lengkap dan siap_berangkat adalah status lama sesudah lunas.
func bolehUnggahDokumen(statusBooking string) bool {
	switch statusBooking {
	case "dp", "lunas", "dokumen_lengkap", "siap_berangkat":
		return true
	}
	return false
}

type keputusanGanti int

const (
	gantiBoleh keputusanGanti = iota
	gantiSudahDisetujui
	gantiMenungguVerifikasi
)

// putuskanGantiDokumen: boleh tidaknya akun mengirim berkas untuk satu jenis dokumen (keputusan 2026-11-25).
// statusAda kosong bila jenis itu belum punya berkas. Dokumen disetujui tidak bisa diganti siapa pun;
// dokumen menunggu verifikasi hanya bisa diganti akun berakses penuh; dokumen ditolak bisa diganti semua akun.
func putuskanGantiDokumen(statusAda string, aksesPenuh bool) keputusanGanti {
	switch statusAda {
	case "approved":
		return gantiSudahDisetujui
	case "submitted":
		if !aksesPenuh {
			return gantiMenungguVerifikasi
		}
	}
	return gantiBoleh
}

// gerbangKirimDokumen memeriksa booking (tidak batal, minimal DP), hak akun atas jamaah, dan pax aktif di dalam
// transaksi unggah. Baris booking dan pax dikunci FOR SHARE, jadi pembatalan atau perubahan status di tengah jalan
// menunggu unggah ini selesai, dan unggah sesudahnya membaca status terbaru. kode 0 = lolos.
func gerbangKirimDokumen(ctx context.Context, tx dbq, akunID, bookingID, jamaahID int64) (kode int, pesan string, err error) {
	var bid int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM bookings WHERE id=? FOR SHARE`, bookingID).Scan(&bid); errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, "Booking tidak ditemukan.", nil
	} else if err != nil {
		return 0, "", err
	}
	order, _, err := ownedQ(ctx, tx, akunID, bookingID)
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound, "Booking tidak ditemukan.", nil
	}
	if err != nil {
		return 0, "", err
	}
	if order.Tab == "batal" {
		return http.StatusConflict, "Booking ini sudah dibatalkan atau kedaluwarsa.", nil
	}
	if !bolehUnggahDokumen(order.Status) {
		return http.StatusConflict, "Dokumen bisa diunggah setelah DP terverifikasi.", nil
	}
	if !order.bolehJamaah(jamaahID) {
		return http.StatusForbidden, pesanBukanDiriSendiri, nil
	}
	var paxID int64
	if err := tx.QueryRowContext(ctx, `SELECT id FROM booking_pax WHERE booking_id=? AND jamaah_id=? AND pax_status='aktif' LIMIT 1 FOR SHARE`, bookingID, jamaahID).Scan(&paxID); errors.Is(err, sql.ErrNoRows) {
		return http.StatusNotFound, "Jamaah tidak ditemukan di booking ini.", nil
	} else if err != nil {
		return 0, "", err
	}
	return 0, "", nil
}

// KirimDokumen POST /api/account/bookings/{id}/dokumen
// Berkas harus unggahan akun ini (POST /api/account/media/upload). Status dokumen diperiksa dan berkas disimpan
// dalam satu transaksi (baris dokumen dikunci), jadi persetujuan admin di tengah jalan tidak tertimpa.
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
		writeError(w, http.StatusBadRequest, "Data dokumen tidak valid.")
		return
	}
	tx, err := o.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("[ERROR] account kirim dokumen begin: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan dokumen.")
		return
	}
	defer tx.Rollback()
	if kode, pesan, err := gerbangKirimDokumen(r.Context(), tx, acc.ID, id, req.JamaahID); err != nil {
		log.Printf("[ERROR] account kirim dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan dokumen.")
		return
	} else if kode != 0 {
		writeError(w, kode, pesan)
		return
	}
	// Kunci jamaah lalu baris dokumen, urutan sama dengan dokumen.UpsertTx. UPDATE status oleh admin menunggu
	// kunci ini dan gagal cek versi setelah berkas diganti, jadi keputusan di bawah tidak basi.
	var jid int64
	if err := tx.QueryRowContext(r.Context(), `SELECT id FROM jamaah WHERE id=? FOR UPDATE`, req.JamaahID).Scan(&jid); err != nil {
		log.Printf("[ERROR] account kirim dokumen kunci jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan dokumen.")
		return
	}
	var status sql.NullString
	if err := tx.QueryRowContext(r.Context(), `SELECT status FROM dokumen_jamaah WHERE jamaah_id=? AND jenis=? FOR UPDATE`, req.JamaahID, req.Jenis).Scan(&status); err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[ERROR] account kirim dokumen: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan dokumen.")
		return
	}
	penuh := true
	if status.String == "submitted" {
		if penuh, err = aksesPenuhJamaah(r.Context(), tx, acc.ID, req.JamaahID); err != nil {
			log.Printf("[ERROR] account kirim dokumen akses: %v", err)
			writeError(w, http.StatusInternalServerError, "Gagal menyimpan dokumen.")
			return
		}
	}
	switch putuskanGantiDokumen(status.String, penuh) {
	case gantiSudahDisetujui:
		writeError(w, http.StatusConflict, "Dokumen ini sudah disetujui.")
		return
	case gantiMenungguVerifikasi:
		writeError(w, http.StatusForbidden, "Dokumen ini sedang menunggu verifikasi.")
		return
	}
	if err := media.ValidateCustomerUpload(r.Context(), o.db, req.FileURL, acc.ID); err != nil {
		var uploadErr *media.UploadError
		if errors.As(err, &uploadErr) {
			writeError(w, http.StatusBadRequest, uploadErr.Message)
			return
		}
		log.Printf("[ERROR] account kirim dokumen validasi berkas: %v", err)
		writeError(w, http.StatusInternalServerError, "Permintaan gagal diproses. Coba lagi.")
		return
	}
	docID, err := dokumen.UpsertTx(r.Context(), tx, req.JamaahID, &dokumen.CreateDokumenRequest{Jenis: req.Jenis, FileURL: req.FileURL})
	if err != nil {
		log.Printf("[ERROR] account kirim dokumen upsert: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan dokumen.")
		return
	}
	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR] account kirim dokumen commit: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan dokumen.")
		return
	}
	d, err := o.dokumen.GetByID(r.Context(), docID, nil)
	if err != nil {
		log.Printf("[ERROR] account kirim dokumen baca ulang: %v", err)
		writeError(w, http.StatusInternalServerError, "Gagal menyimpan dokumen.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jenis": d.Jenis, "status": d.Status})
}
