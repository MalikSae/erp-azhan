package customer

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"erp-azhan/api/internal/perlengkapan"
)

// Perlengkapan di portal pelanggan (migrasi 141, keputusan 14 November 2026). Per jamaah: belum diajukan -> diajukan
// (akun, dari app) -> diserahkan (petugas menekan Serahkan di dashboard) -> diterima (akun pemilik booking mengonfirmasi).
// Bayi tidak mendapat jatah perlengkapan (sama dengan aturan penyerahan di booking). Saat ini hanya metode ambil di
// kantor; kirim ke rumah menunggu integrasi ekspedisi.

const metodeAmbilKantor = "ambil_kantor"

type isiPerlengkapan struct {
	Nama     string  `json:"nama"`
	PhotoURL *string `json:"photo_url"`
	Qty      int     `json:"qty"`
}

type perlengkapanJamaah struct {
	JamaahID     int64  `json:"jamaah_id"`
	Nama         string `json:"nama"`
	PaxType      string `json:"pax_type"`
	JenisKelamin string `json:"jenis_kelamin"`
	// Status: belum | diajukan | diserahkan | diterima.
	Status         string            `json:"status"`
	Metode         *string           `json:"metode"`
	DiajukanAt     *string           `json:"diajukan_at"`
	DiserahkanAt   *string           `json:"diserahkan_at"`
	DiserahkanNama *string           `json:"diserahkan_nama"`
	DiterimaAt     *string           `json:"diterima_at"`
	Isi            []isiPerlengkapan `json:"isi"`
}

type perlengkapanView struct {
	// BookingCode ditunjukkan ke petugas saat mengambil perlengkapan di kantor.
	BookingCode string               `json:"booking_code"`
	Jamaah      []perlengkapanJamaah `json:"jamaah"`
	Kantor      kantorView           `json:"kantor"`
	// BisaAjukan: booking sudah DP dan belum batal (pengajuan dari app dibuka setelah DP).
	BisaAjukan bool `json:"bisa_ajukan"`
}

// statusPerlengkapan menurunkan status tampilan dari kolom booking_pax.
func statusPerlengkapan(diberikan bool, diajukan, diterima bool) string {
	switch {
	case diterima:
		return "diterima"
	case diberikan:
		return "diserahkan"
	case diajukan:
		return "diajukan"
	}
	return "belum"
}

// isiUntuk memilih barang set yang berlaku untuk jenis kelamin jamaah ("semua" ditambah "pria" atau "wanita").
func isiUntuk(set []perlengkapan.SetItem, jenisKelamin string) []isiPerlengkapan {
	out := []isiPerlengkapan{}
	for _, it := range set {
		if (it.Untuk == "pria" && jenisKelamin != "L") || (it.Untuk == "wanita" && jenisKelamin != "P") {
			continue
		}
		out = append(out, isiPerlengkapan{Nama: it.Nama, PhotoURL: it.PhotoURL, Qty: it.Qty})
	}
	return out
}

func bisaAjukanPerlengkapan(o *orderSummary) bool {
	return o.Tab != "batal" && o.Status != "baru"
}

func (o *Orders) perlengkapanBooking(ctx context.Context, acc *Account, order *orderSummary) (*perlengkapanView, error) {
	view := &perlengkapanView{BookingCode: order.BookingCode, Jamaah: []perlengkapanJamaah{}, BisaAjukan: bisaAjukanPerlengkapan(order)}
	set, err := perlengkapan.NewRepository(o.db).GetScheduleSet(ctx, order.ScheduleID, nil)
	if err != nil && !errors.Is(err, perlengkapan.ErrNotFound) {
		return nil, err
	}
	var items []perlengkapan.SetItem
	if set != nil {
		items = set.Items
	}
	if view.Jamaah, err = daftarPerlengkapan(ctx, o.db, order.ID, items); err != nil {
		return nil, err
	}
	if err := o.db.QueryRowContext(ctx, `SELECT NULLIF(TRIM(address),''), NULLIF(TRIM(gmaps_url),''), NULLIF(TRIM(jam_kantor),'') FROM brands WHERE id=?`,
		acc.BrandID).Scan(&view.Kantor.Alamat, &view.Kantor.GmapsURL, &view.Kantor.JamKantor); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return view, nil
}

type pembacaDB interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// daftarPerlengkapan: jamaah dewasa aktif sebuah booking beserta status, log, dan isi paket sesuai jenis kelaminnya.
func daftarPerlengkapan(ctx context.Context, q pembacaDB, bookingID int64, items []perlengkapan.SetItem) ([]perlengkapanJamaah, error) {
	out := []perlengkapanJamaah{}
	rows, err := q.QueryContext(ctx, `
		SELECT j.id, j.nama_lengkap, bp.pax_type, COALESCE(j.jenis_kelamin,''),
		       bp.perlengkapan_status = 'sudah_diberikan', bp.perlengkapan_metode,
		       DATE_FORMAT(bp.perlengkapan_diajukan_at, '%Y-%m-%d %H:%i'),
		       COALESCE(DATE_FORMAT(bp.perlengkapan_diserahkan_at, '%Y-%m-%d %H:%i'), DATE_FORMAT(bp.perlengkapan_tanggal, '%Y-%m-%d')),
		       bp.perlengkapan_diserahkan_nama,
		       DATE_FORMAT(bp.perlengkapan_diterima_at, '%Y-%m-%d %H:%i')
		FROM booking_pax bp JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE bp.booking_id = ? AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant'
		ORDER BY bp.id`, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p perlengkapanJamaah
		var diberikan bool
		if err := rows.Scan(&p.JamaahID, &p.Nama, &p.PaxType, &p.JenisKelamin, &diberikan, &p.Metode,
			&p.DiajukanAt, &p.DiserahkanAt, &p.DiserahkanNama, &p.DiterimaAt); err != nil {
			return nil, err
		}
		if !diberikan {
			p.DiserahkanAt, p.DiserahkanNama = nil, nil
		}
		p.Status = statusPerlengkapan(diberikan, p.DiajukanAt != nil, p.DiterimaAt != nil)
		p.Isi = isiUntuk(items, p.JenisKelamin)
		out = append(out, p)
	}
	return out, rows.Err()
}

// pesananMilik memuat pesanan milik akun dan menulis galat 404/500 bila gagal.
func (o *Orders) pesananMilik(w http.ResponseWriter, r *http.Request) (*Account, *orderSummary, bool) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return nil, nil, false
	}
	order, _, err := o.owned(r.Context(), acc.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return nil, nil, false
	}
	if err != nil {
		log.Printf("[ERROR] account perlengkapan booking: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil booking")
		return nil, nil, false
	}
	return acc, order, true
}

func (o *Orders) tulisPerlengkapan(w http.ResponseWriter, r *http.Request, acc *Account, order *orderSummary) {
	view, err := o.perlengkapanBooking(r.Context(), acc, order)
	if err != nil {
		log.Printf("[ERROR] account perlengkapan: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil perlengkapan")
		return
	}
	writeJSON(w, http.StatusOK, view)
}

// Perlengkapan GET /api/account/bookings/{id}/perlengkapan
func (o *Orders) Perlengkapan(w http.ResponseWriter, r *http.Request) {
	acc, order, ok := o.pesananMilik(w, r)
	if !ok {
		return
	}
	o.tulisPerlengkapan(w, r, acc, order)
}

type ajukanPerlengkapanRequest struct {
	JamaahIDs []int64 `json:"jamaah_ids"`
	Metode    string  `json:"metode"`
}

// AjukanPerlengkapan POST /api/account/bookings/{id}/perlengkapan/ajukan: {jamaah_ids, metode}. Jamaah yang sudah
// diajukan atau sudah diserahkan ditolak agar pengajuan tidak tercatat ganda.
func (o *Orders) AjukanPerlengkapan(w http.ResponseWriter, r *http.Request) {
	acc, order, ok := o.pesananMilik(w, r)
	if !ok {
		return
	}
	var req ajukanPerlengkapanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.JamaahIDs) == 0 || len(req.JamaahIDs) > 50 {
		writeError(w, http.StatusBadRequest, "pilih minimal satu jamaah")
		return
	}
	if strings.TrimSpace(req.Metode) != metodeAmbilKantor {
		writeError(w, http.StatusBadRequest, "metode pengambilan tidak tersedia")
		return
	}
	if !bisaAjukanPerlengkapan(order) {
		writeError(w, http.StatusConflict, "perlengkapan bisa diajukan setelah DP terverifikasi")
		return
	}
	err := dalamTx(r.Context(), o.db, func(tx *sql.Tx) error {
		return ajukanPerlengkapanTx(r.Context(), tx, acc.ID, order.ID, req.JamaahIDs)
	})
	switch {
	case errors.Is(err, errJamaahBukanBooking):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, errSudahDiajukan):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		log.Printf("[ERROR] account ajukan perlengkapan: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengajukan perlengkapan")
		return
	}
	o.tulisPerlengkapan(w, r, acc, order)
}

// ajukanPerlengkapanTx mencatat pengajuan (ambil di kantor) untuk jamaah booking; semua atau tidak sama sekali.
func ajukanPerlengkapanTx(ctx context.Context, tx *sql.Tx, accountID, bookingID int64, jamaahIDs []int64) error {
	// Periksa semua jamaah dulu, baru simpan: satu jamaah yang ditolak membatalkan seluruh pengajuan.
	paxIDs := make([]int64, 0, len(jamaahIDs))
	for _, jid := range jamaahIDs {
		var paxID int64
		var diberikan bool
		var diajukan sql.NullTime
		err := tx.QueryRowContext(ctx, `
				SELECT id, perlengkapan_status = 'sudah_diberikan', perlengkapan_diajukan_at
				FROM booking_pax WHERE booking_id = ? AND jamaah_id = ? AND pax_status = 'aktif' AND pax_type <> 'infant' FOR UPDATE`, bookingID, jid).
			Scan(&paxID, &diberikan, &diajukan)
		if errors.Is(err, sql.ErrNoRows) {
			return errJamaahBukanBooking
		}
		if err != nil {
			return err
		}
		if diberikan || diajukan.Valid {
			return errSudahDiajukan
		}
		paxIDs = append(paxIDs, paxID)
	}
	for _, paxID := range paxIDs {
		if _, err := tx.ExecContext(ctx, `
				UPDATE booking_pax SET perlengkapan_metode = ?, perlengkapan_diajukan_at = NOW(), perlengkapan_diajukan_oleh = ?
				WHERE id = ?`, metodeAmbilKantor, accountID, paxID); err != nil {
			return err
		}
	}
	return nil
}

var (
	errJamaahBukanBooking = errors.New("jamaah tidak terdaftar di booking ini atau tidak mendapat perlengkapan")
	errSudahDiajukan      = errors.New("perlengkapan salah satu jamaah sudah diajukan atau diserahkan, muat ulang halaman")
	errBelumDiserahkan    = errors.New("perlengkapan jamaah ini belum diserahkan petugas")
	errSudahDiterima      = errors.New("penerimaan perlengkapan jamaah ini sudah dikonfirmasi")
)

type terimaPerlengkapanRequest struct {
	JamaahID int64 `json:"jamaah_id"`
}

// TerimaPerlengkapan POST /api/account/bookings/{id}/perlengkapan/terima: {jamaah_id}. Akun pemilik booking
// mengonfirmasi perlengkapan yang sudah diserahkan petugas sudah diterima; setelah itu penyerahan terkunci.
func (o *Orders) TerimaPerlengkapan(w http.ResponseWriter, r *http.Request) {
	acc, order, ok := o.pesananMilik(w, r)
	if !ok {
		return
	}
	var req terimaPerlengkapanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.JamaahID <= 0 {
		writeError(w, http.StatusBadRequest, "jamaah_id wajib diisi")
		return
	}
	err := dalamTx(r.Context(), o.db, func(tx *sql.Tx) error {
		return terimaPerlengkapanTx(r.Context(), tx, acc.ID, order.ID, req.JamaahID)
	})
	switch {
	case errors.Is(err, errJamaahBukanBooking):
		writeError(w, http.StatusBadRequest, err.Error())
		return
	case errors.Is(err, errBelumDiserahkan), errors.Is(err, errSudahDiterima):
		writeError(w, http.StatusConflict, err.Error())
		return
	case err != nil:
		log.Printf("[ERROR] account terima perlengkapan: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengonfirmasi penerimaan")
		return
	}
	o.tulisPerlengkapan(w, r, acc, order)
}

// terimaPerlengkapanTx mencatat konfirmasi terima dari akun untuk perlengkapan yang sudah diserahkan.
func terimaPerlengkapanTx(ctx context.Context, tx *sql.Tx, accountID, bookingID, jamaahID int64) error {
	var paxID int64
	var diberikan bool
	var diterima sql.NullTime
	err := tx.QueryRowContext(ctx, `
			SELECT id, perlengkapan_status = 'sudah_diberikan', perlengkapan_diterima_at
			FROM booking_pax WHERE booking_id = ? AND jamaah_id = ? AND pax_status = 'aktif' AND pax_type <> 'infant' FOR UPDATE`, bookingID, jamaahID).
		Scan(&paxID, &diberikan, &diterima)
	if errors.Is(err, sql.ErrNoRows) {
		return errJamaahBukanBooking
	}
	if err != nil {
		return err
	}
	if !diberikan {
		return errBelumDiserahkan
	}
	if diterima.Valid {
		return errSudahDiterima
	}
	_, err = tx.ExecContext(ctx, `UPDATE booking_pax SET perlengkapan_diterima_at = NOW(), perlengkapan_diterima_oleh = ? WHERE id = ?`, accountID, paxID)
	return err
}
