package customer

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"erp-azhan/api/internal/jamaah"
	"github.com/go-chi/chi/v5"
	"github.com/go-sql-driver/mysql"
)

// Data lengkap jamaah di portal pelanggan: pemesan melengkapi data tiap jamaah di bookingnya.
// Nama lengkap, NIK, dan email sengaja tidak bisa diubah di sini (identitas; perubahan lewat tim kami).

type dataJamaah struct {
	JamaahID            int64    `json:"jamaah_id"`
	NamaLengkap         string   `json:"nama_lengkap"`
	NIKTersamar         string   `json:"nik_tersamar"`
	PaxType             string   `json:"pax_type"`
	JenisKelamin        string   `json:"jenis_kelamin"`
	NamaAyahKandung     string   `json:"nama_ayah_kandung"`
	TempatLahir         string   `json:"tempat_lahir"`
	TanggalLahir        string   `json:"tanggal_lahir"`
	NoPaspor            string   `json:"no_paspor"`
	TempatPasporKeluar  string   `json:"tempat_paspor_keluar"`
	TanggalPasporKeluar string   `json:"tanggal_paspor_keluar"`
	PasporBerlakuSampai string   `json:"paspor_berlaku_sampai"`
	NoHP                string   `json:"no_hp"`
	Pekerjaan           string   `json:"pekerjaan"`
	PendidikanTerakhir  string   `json:"pendidikan_terakhir"`
	PenjaminKesehatan   string   `json:"penjamin_kesehatan"`
	NoAsuransiBPJS      string   `json:"no_asuransi_bpjs"`
	Alamat              string   `json:"alamat"`
	Kota                string   `json:"kota"`
	EmergencyNama       string   `json:"emergency_nama"`
	EmergencyHP         string   `json:"emergency_hp"`
	EmergencyHubungan   string   `json:"emergency_hubungan"`
	EmergencyAlamat     string   `json:"emergency_alamat"`
	Terisi              int      `json:"terisi"`
	Total               int      `json:"total"`
	Kurang              []string `json:"kurang"`
}

// Data wajib untuk kelengkapan (kunci JSON -> label tampil). Infant tidak wajib kontak darurat.
var dataWajibJamaah = []struct{ key, label string }{
	{"tempat_lahir", "Tempat lahir"},
	{"tanggal_lahir", "Tanggal lahir"},
	{"nama_ayah_kandung", "Nama ayah kandung"},
	{"no_paspor", "Nomor paspor"},
	{"paspor_berlaku_sampai", "Masa berlaku paspor"},
	{"alamat", "Alamat"},
	{"kota", "Kota"},
	{"no_hp", "Nomor WhatsApp"},
	{"emergency_nama", "Nama kontak darurat"},
	{"emergency_hp", "Nomor kontak darurat"},
}

func (d *dataJamaah) nilai(key string) string {
	switch key {
	case "tempat_lahir":
		return d.TempatLahir
	case "tanggal_lahir":
		return d.TanggalLahir
	case "nama_ayah_kandung":
		return d.NamaAyahKandung
	case "no_paspor":
		return d.NoPaspor
	case "paspor_berlaku_sampai":
		return d.PasporBerlakuSampai
	case "alamat":
		return d.Alamat
	case "kota":
		return d.Kota
	case "no_hp":
		return d.NoHP
	case "emergency_nama":
		return d.EmergencyNama
	case "emergency_hp":
		return d.EmergencyHP
	}
	return ""
}

// hitungKelengkapan mengisi Terisi, Total, dan Kurang dari data wajib.
func (d *dataJamaah) hitungKelengkapan() {
	d.Kurang = []string{}
	d.Total, d.Terisi = 0, 0
	for _, f := range dataWajibJamaah {
		if d.PaxType == "infant" && strings.HasPrefix(f.key, "emergency_") {
			continue
		}
		d.Total++
		if strings.TrimSpace(d.nilai(f.key)) != "" {
			d.Terisi++
		} else {
			d.Kurang = append(d.Kurang, f.label)
		}
	}
}

var namaIsianData = strings.NewReplacer(
	"tanggal_paspor_keluar", "Tanggal paspor dikeluarkan",
	"paspor_berlaku_sampai", "Masa berlaku paspor",
	"tanggal_lahir", "Tanggal lahir",
	"no HP kontak darurat", "Nomor telepon kontak darurat",
	"no_hp", "Nomor WhatsApp",
)

// ramahPesanData mengganti nama kolom pada pesan validasi admin dengan nama isian yang dilihat pelanggan.
func ramahPesanData(msg string) string {
	return namaIsianData.Replace(msg)
}

var (
	polaNamaOrang = regexp.MustCompile(`^[\p{L}][\p{L} .,'’-]{1,99}$`)
	polaPaspor    = regexp.MustCompile(`^[A-Za-z0-9]{5,12}$`)
	polaNomorAsur = regexp.MustCompile(`^[A-Za-z0-9]{3,25}$`)
)

// periksaFormatData memeriksa isian yang punya format tetap; isian kosong dilewati. Mengembalikan pesan untuk pelanggan.
func periksaFormatData(req *simpanDataJamaahRequest) string {
	nama := []struct{ label, nilai string }{
		{"Nama ayah kandung", req.NamaAyahKandung}, {"Nama kontak darurat", req.EmergencyNama}, {"Hubungan kontak darurat", req.EmergencyHubungan},
	}
	for _, n := range nama {
		if n.nilai != "" && !polaNamaOrang.MatchString(n.nilai) {
			return n.label + " hanya boleh berisi huruf, spasi, titik, koma, apostrof, dan tanda hubung"
		}
	}
	if req.NoPaspor != "" && !polaPaspor.MatchString(req.NoPaspor) {
		return "Nomor paspor hanya boleh berisi huruf dan angka (5 sampai 12 karakter)"
	}
	if req.NoAsuransiBPJS != "" && !polaNomorAsur.MatchString(req.NoAsuransiBPJS) {
		return "Nomor BPJS atau asuransi hanya boleh berisi huruf dan angka"
	}
	return ""
}

func nikTersamar(nik string) string {
	if len(nik) < 4 {
		return ""
	}
	return strings.Repeat("•", len(nik)-4) + nik[len(nik)-4:]
}

const dataJamaahSelect = `
	SELECT j.id, j.nama_lengkap, COALESCE(j.nik,''), bp.pax_type, COALESCE(j.jenis_kelamin,''), COALESCE(j.nama_ayah_kandung,''),
	       COALESCE(j.tempat_lahir,''), COALESCE(DATE_FORMAT(j.tanggal_lahir,'%Y-%m-%d'),''),
	       COALESCE(j.no_paspor,''), COALESCE(j.tempat_paspor_keluar,''),
	       COALESCE(DATE_FORMAT(j.tanggal_paspor_keluar,'%Y-%m-%d'),''), COALESCE(DATE_FORMAT(j.paspor_berlaku_sampai,'%Y-%m-%d'),''),
	       COALESCE(j.no_hp,''), COALESCE(j.pekerjaan,''), COALESCE(j.pendidikan_terakhir,''),
	       COALESCE(j.penjamin_kesehatan,''), COALESCE(j.no_asuransi_bpjs,''), COALESCE(j.alamat,''), COALESCE(j.kota,''),
	       COALESCE(j.emergency_nama,''), COALESCE(j.emergency_hp,''), COALESCE(j.emergency_hubungan,''), COALESCE(j.emergency_alamat,'')
	FROM booking_pax bp JOIN jamaah j ON j.id = bp.jamaah_id
	WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'`

func scanDataJamaah(rows interface{ Scan(...any) error }) (*dataJamaah, error) {
	var d dataJamaah
	var nik string
	if err := rows.Scan(&d.JamaahID, &d.NamaLengkap, &nik, &d.PaxType, &d.JenisKelamin, &d.NamaAyahKandung,
		&d.TempatLahir, &d.TanggalLahir, &d.NoPaspor, &d.TempatPasporKeluar, &d.TanggalPasporKeluar, &d.PasporBerlakuSampai,
		&d.NoHP, &d.Pekerjaan, &d.PendidikanTerakhir, &d.PenjaminKesehatan, &d.NoAsuransiBPJS, &d.Alamat, &d.Kota,
		&d.EmergencyNama, &d.EmergencyHP, &d.EmergencyHubungan, &d.EmergencyAlamat); err != nil {
		return nil, err
	}
	d.NIKTersamar = nikTersamar(nik)
	d.hitungKelengkapan()
	return &d, nil
}

// DataJamaah GET /api/account/bookings/{id}/jamaah
func (o *Orders) DataJamaah(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	if _, _, err := o.owned(r.Context(), acc.ID, id); errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	} else if err != nil {
		log.Printf("[ERROR] account data jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	rows, err := o.db.QueryContext(r.Context(), dataJamaahSelect+` ORDER BY bp.id`, id)
	if err != nil {
		log.Printf("[ERROR] account data jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	defer rows.Close()
	out := []dataJamaah{}
	for rows.Next() {
		d, err := scanDataJamaah(rows)
		if err != nil {
			log.Printf("[ERROR] account data jamaah scan: %v", err)
			writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
			return
		}
		out = append(out, *d)
	}
	if err := rows.Err(); err != nil {
		log.Printf("[ERROR] account data jamaah rows: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"jamaah": out})
}

type simpanDataJamaahRequest struct {
	JenisKelamin        string `json:"jenis_kelamin"`
	NamaAyahKandung     string `json:"nama_ayah_kandung"`
	TempatLahir         string `json:"tempat_lahir"`
	TanggalLahir        string `json:"tanggal_lahir"`
	NoPaspor            string `json:"no_paspor"`
	TempatPasporKeluar  string `json:"tempat_paspor_keluar"`
	TanggalPasporKeluar string `json:"tanggal_paspor_keluar"`
	PasporBerlakuSampai string `json:"paspor_berlaku_sampai"`
	NoHP                string `json:"no_hp"`
	Pekerjaan           string `json:"pekerjaan"`
	PendidikanTerakhir  string `json:"pendidikan_terakhir"`
	PenjaminKesehatan   string `json:"penjamin_kesehatan"`
	NoAsuransiBPJS      string `json:"no_asuransi_bpjs"`
	Alamat              string `json:"alamat"`
	Kota                string `json:"kota"`
	EmergencyNama       string `json:"emergency_nama"`
	EmergencyHP         string `json:"emergency_hp"`
	EmergencyHubungan   string `json:"emergency_hubungan"`
	EmergencyAlamat     string `json:"emergency_alamat"`
}

// panjangMaksData: batas panjang tiap isian teks (sama dengan kolom database).
const panjangMaksData = 255

// SimpanDataJamaah PUT /api/account/bookings/{id}/jamaah/{jamaah_id}
// Mengganti seluruh isian yang boleh diubah pelanggan; isian kosong dikosongkan.
func (o *Orders) SimpanDataJamaah(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	jamaahID, err := strconv.ParseInt(chi.URLParam(r, "jamaah_id"), 10, 64)
	if err != nil || jamaahID <= 0 {
		writeError(w, http.StatusBadRequest, "id jamaah tidak valid")
		return
	}
	var req simpanDataJamaahRequest
	if !decode(w, r, &req) {
		return
	}
	order, _, err := o.owned(r.Context(), acc.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	}
	if err != nil {
		log.Printf("[ERROR] account simpan data jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan data jamaah")
		return
	}
	if order.Tab == "batal" {
		writeError(w, http.StatusConflict, "booking ini sudah dibatalkan atau kedaluwarsa")
		return
	}
	var punya bool
	if err := o.db.QueryRowContext(r.Context(), `SELECT EXISTS(SELECT 1 FROM booking_pax WHERE booking_id=? AND jamaah_id=? AND pax_status='aktif')`, id, jamaahID).Scan(&punya); err != nil {
		log.Printf("[ERROR] account simpan data jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan data jamaah")
		return
	}
	if !punya {
		writeError(w, http.StatusNotFound, "jamaah tidak ditemukan di booking ini")
		return
	}

	teks := []*string{&req.NamaAyahKandung, &req.TempatLahir, &req.NoPaspor, &req.TempatPasporKeluar, &req.NoHP, &req.Pekerjaan,
		&req.PendidikanTerakhir, &req.PenjaminKesehatan, &req.NoAsuransiBPJS, &req.Alamat, &req.Kota,
		&req.EmergencyNama, &req.EmergencyHP, &req.EmergencyHubungan, &req.EmergencyAlamat}
	for _, p := range teks {
		*p = strings.TrimSpace(*p)
		if len([]rune(*p)) > panjangMaksData {
			writeError(w, http.StatusBadRequest, "isian terlalu panjang (maksimal 255 karakter)")
			return
		}
	}
	req.JenisKelamin = strings.TrimSpace(req.JenisKelamin)
	if req.JenisKelamin != "" && req.JenisKelamin != "L" && req.JenisKelamin != "P" {
		writeError(w, http.StatusBadRequest, "jenis kelamin tidak valid")
		return
	}
	if msg := periksaFormatData(&req); msg != "" {
		writeError(w, http.StatusBadRequest, msg)
		return
	}
	if msg := jamaah.ValidasiInput(&jamaah.CreateJamaahRequest{
		TanggalLahir: &req.TanggalLahir, TanggalPasporKeluar: &req.TanggalPasporKeluar, PasporBerlakuSampai: &req.PasporBerlakuSampai,
		NoHP: &req.NoHP, EmergencyHP: &req.EmergencyHP,
	}); msg != "" {
		writeError(w, http.StatusBadRequest, ramahPesanData(msg))
		return
	}

	_, err = o.db.ExecContext(r.Context(), `UPDATE jamaah SET
		jenis_kelamin=NULLIF(?,''), nama_ayah_kandung=NULLIF(?,''), tempat_lahir=NULLIF(?,''), tanggal_lahir=NULLIF(?,''),
		no_paspor=NULLIF(?,''), tempat_paspor_keluar=NULLIF(?,''), tanggal_paspor_keluar=NULLIF(?,''), paspor_berlaku_sampai=NULLIF(?,''),
		no_hp=NULLIF(?,''), pekerjaan=NULLIF(?,''), pendidikan_terakhir=NULLIF(?,''), penjamin_kesehatan=NULLIF(?,''),
		no_asuransi_bpjs=NULLIF(?,''), alamat=NULLIF(?,''), kota=NULLIF(?,''),
		emergency_nama=NULLIF(?,''), emergency_hp=NULLIF(?,''), emergency_hubungan=NULLIF(?,''), emergency_alamat=NULLIF(?,'')
		WHERE id=?`,
		req.JenisKelamin, req.NamaAyahKandung, req.TempatLahir, req.TanggalLahir,
		req.NoPaspor, req.TempatPasporKeluar, req.TanggalPasporKeluar, req.PasporBerlakuSampai,
		req.NoHP, req.Pekerjaan, req.PendidikanTerakhir, req.PenjaminKesehatan,
		req.NoAsuransiBPJS, req.Alamat, req.Kota,
		req.EmergencyNama, req.EmergencyHP, req.EmergencyHubungan, req.EmergencyAlamat,
		jamaahID)
	if err != nil {
		var myErr *mysql.MySQLError
		if errors.As(err, &myErr) && myErr.Number == 1062 {
			writeError(w, http.StatusConflict, "nomor WhatsApp ini sudah dipakai jamaah lain")
			return
		}
		log.Printf("[ERROR] account simpan data jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menyimpan data jamaah")
		return
	}

	d, err := scanDataJamaah(o.db.QueryRowContext(r.Context(), dataJamaahSelect+` AND j.id = ?`, id, jamaahID))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("[ERROR] account simpan data jamaah baca ulang: %v", err)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	writeJSON(w, http.StatusOK, d)
}
