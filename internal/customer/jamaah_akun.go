package customer

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"erp-azhan/api/internal/mailer"
	"github.com/go-chi/chi/v5"
)

// Halaman Data Jamaah di akun pelanggan: daftar orang (jamaah dari semua booking akun), detail, dan simpan.
// Persetujuan admin untuk perubahan ada di perubahan_jamaah_admin.go.

type JamaahAkun struct {
	db   *sql.DB
	repo *Repository
	mail mailer.Sender
}

func NewJamaahAkun(db *sql.DB, repo *Repository, m mailer.Sender) *JamaahAkun {
	return &JamaahAkun{db: db, repo: repo, mail: m}
}

func (h *JamaahAkun) sendAsync(msg mailer.Message) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := h.mail.Send(ctx, msg); err != nil {
			log.Printf("[ERROR] mailer: %v", err)
		}
	}()
}

type bookingRingkas struct {
	ID        int64  `json:"id"`
	Kode      string `json:"kode"`
	Paket     string `json:"paket"`
	Berangkat string `json:"berangkat"`
	Status    string `json:"status"`
}

func ringkasBooking(a anggotaJamaah) bookingRingkas {
	return bookingRingkas{ID: a.BookingID, Kode: a.BookingKode, Paket: a.Paket, Berangkat: a.Berangkat, Status: a.StatusBooking}
}

type jamaahBaris struct {
	ID            int64          `json:"id"`
	NamaLengkap   string         `json:"nama_lengkap"`
	JenisKelamin  string         `json:"jenis_kelamin"`
	Bayi          bool           `json:"bayi"`
	Terisi        int            `json:"terisi"`
	Total         int            `json:"total"`
	Menunggu      bool           `json:"menunggu_persetujuan"`
	Booking       bookingRingkas `json:"booking"`
	JumlahBooking int            `json:"jumlah_booking"`
}

func adaBayi(grup []anggotaJamaah) bool { return grup[0].PaxType == "infant" }

// Daftar GET /api/account/jamaah: satu baris per orang, yang datanya belum lengkap di atas.
func (h *JamaahAkun) Daftar(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	semua, err := muatAnggota(r.Context(), h.db, acc.ID)
	if err != nil {
		log.Printf("[ERROR] account daftar jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	menunggu, err := jamaahMenunggu(r.Context(), h.db, acc.ID)
	if err != nil {
		log.Printf("[ERROR] account daftar jamaah menunggu: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	out := []jamaahBaris{}
	for _, grup := range kelompokOrang(semua) {
		d, err := muatDataOrang(r.Context(), h.db, idGrup(grup))
		if err != nil {
			log.Printf("[ERROR] account daftar jamaah data: %v", err)
			writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
			return
		}
		terisi, total, _ := kelengkapanOrang(d.Nilai, adaBayi(grup))
		b := jamaahBaris{ID: grup[0].ID, NamaLengkap: d.Nama, JenisKelamin: d.Nilai["jenis_kelamin"], Bayi: adaBayi(grup),
			Terisi: terisi, Total: total, Booking: ringkasBooking(grup[0]), JumlahBooking: len(grup)}
		for _, a := range grup {
			if menunggu[a.ID] {
				b.Menunggu = true
			}
		}
		out = append(out, b)
	}
	sort.SliceStable(out, func(i, j int) bool {
		return (out[i].Terisi < out[i].Total) && !(out[j].Terisi < out[j].Total)
	})
	writeJSON(w, http.StatusOK, map[string]any{"jamaah": out})
}

func jamaahMenunggu(ctx context.Context, q dbq, akunID int64) (map[int64]bool, error) {
	rows, err := q.QueryContext(ctx, `SELECT DISTINCT jamaah_id FROM jamaah_perubahan WHERE akun_id=? AND sumber='pelanggan' AND status='menunggu'`, akunID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out[id] = true
	}
	return out, rows.Err()
}

type jamaahDetail struct {
	ID          int64             `json:"id"`
	NamaLengkap string            `json:"nama_lengkap"`
	NIKTersamar string            `json:"nik_tersamar"`
	Email       string            `json:"email"`
	Bayi        bool              `json:"bayi"`
	Data        map[string]string `json:"data"`
	Terisi      int               `json:"terisi"`
	Total       int               `json:"total"`
	Kurang      []string          `json:"kurang"`
	Pengajuan   *pengajuanView    `json:"pengajuan"`
	Booking     []bookingRingkas  `json:"booking"`
	Hasil       *hasilJSON        `json:"hasil,omitempty"`
}

type hasilJSON struct {
	Langsung  int `json:"langsung"`
	Pengajuan int `json:"pengajuan"`
}

func (h *JamaahAkun) bangunDetail(ctx context.Context, q dbq, akunID int64, grup []anggotaJamaah) (*jamaahDetail, error) {
	ids := idGrup(grup)
	d, err := muatDataOrang(ctx, q, ids)
	if err != nil {
		return nil, err
	}
	terisi, total, kurang := kelengkapanOrang(d.Nilai, adaBayi(grup))
	pj, err := pengajuanUntukPelanggan(ctx, q, akunID, ids)
	if err != nil {
		return nil, err
	}
	out := &jamaahDetail{ID: grup[0].ID, NamaLengkap: d.Nama, NIKTersamar: nikTersamar(d.NIK), Email: d.Email, Bayi: adaBayi(grup),
		Data: d.Nilai, Terisi: terisi, Total: total, Kurang: kurang, Pengajuan: pj}
	for _, a := range grup {
		out.Booking = append(out.Booking, ringkasBooking(a))
	}
	return out, nil
}

func idJamaahParam(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "id jamaah tidak valid")
		return 0, false
	}
	return id, true
}

// Detail GET /api/account/jamaah/{id}
func (h *JamaahAkun) Detail(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := idJamaahParam(w, r)
	if !ok {
		return
	}
	grup, err := kelompokDari(r.Context(), h.db, acc.ID, id)
	if errors.Is(err, errJamaahTidakAda) {
		writeError(w, http.StatusNotFound, "jamaah tidak ditemukan")
		return
	}
	if err != nil {
		log.Printf("[ERROR] account detail jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	d, err := h.bangunDetail(r.Context(), h.db, acc.ID, grup)
	if err != nil {
		log.Printf("[ERROR] account detail jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func tulisGalatSimpan(w http.ResponseWriter, err error) {
	var val *ErrValidasi
	var hp errBentrokHP
	switch {
	case errors.As(err, &val):
		writeError(w, http.StatusBadRequest, val.Pesan)
	case errors.Is(err, ErrTidakBerubah):
		writeError(w, http.StatusBadRequest, "tidak ada perubahan untuk disimpan")
	case errors.As(err, &hp):
		writeError(w, http.StatusConflict, hp.Error())
	case errors.Is(err, errJamaahTidakAda):
		writeError(w, http.StatusNotFound, "jamaah tidak ditemukan")
	default:
		log.Printf("[ERROR] account simpan jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan data jamaah")
	}
}

// Simpan PUT /api/account/jamaah/{id}: badan JSON berisi isian yang diubah. Isian kosong yang diisi langsung tersimpan; mengubah
// atau mengosongkan isian yang sudah terisi menjadi satu pengajuan yang menunggu persetujuan admin.
func (h *JamaahAkun) Simpan(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := idJamaahParam(w, r)
	if !ok {
		return
	}
	var req map[string]string
	if !decode(w, r, &req) {
		return
	}
	var hasil *hasilSimpan
	err := dalamTx(r.Context(), h.db, func(tx *sql.Tx) error {
		grup, err := kelompokDari(r.Context(), tx, acc.ID, id)
		if err != nil {
			return err
		}
		hasil, err = simpanPerubahanPelanggan(r.Context(), tx, acc.ID, grup, req)
		return err
	})
	if err != nil {
		tulisGalatSimpan(w, err)
		return
	}
	grup, err := kelompokDari(r.Context(), h.db, acc.ID, id)
	if err != nil {
		tulisGalatSimpan(w, err)
		return
	}
	d, err := h.bangunDetail(r.Context(), h.db, acc.ID, grup)
	if err != nil {
		tulisGalatSimpan(w, err)
		return
	}
	d.Hasil = &hasilJSON{Langsung: hasil.Langsung, Pengajuan: hasil.Pengajuan}
	if hasil.Pengajuan > 0 {
		log.Printf("[AUDIT] perubahan_jamaah_diajukan akun_id=%d jamaah_id=%d pengajuan_id=%d isian=%d", acc.ID, id, hasil.PengajuanID, hasil.Pengajuan)
	}
	writeJSON(w, http.StatusOK, d)
}

func dalamTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit()
}
