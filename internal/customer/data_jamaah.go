package customer

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
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
	// AksesTerbatas: akun bukan pembuat jamaah ini. Isian dikosongkan dan tidak bisa diubah (akses_jamaah.go).
	AksesTerbatas       bool     `json:"akses_terbatas"`
	// Hasil simpan terakhir: isian yang langsung terisi dan isian yang menunggu persetujuan admin.
	PerubahanLangsung int `json:"perubahan_langsung,omitempty"`
	PerubahanMenunggu int `json:"perubahan_menunggu,omitempty"`
}

// Data wajib untuk kelengkapan (kunci JSON -> label tampil), sengaja seminim mungkin agar tidak membebani jamaah. Infant tidak wajib kontak darurat
// dan nomor WhatsApp (bayi tidak punya nomor sendiri; kontak ada di pemesan). Tanda * di formulir jamaah mengikuti daftar ini.
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
		if d.PaxType == "infant" && (strings.HasPrefix(f.key, "emergency_") || f.key == "no_hp") {
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

// dataJamaahSelect: parameter pertama id akun (penentu akses), lalu id booking.
const dataJamaahSelect = `
	SELECT j.id, j.nama_lengkap, COALESCE(j.nik,''), bp.pax_type, COALESCE(j.jenis_kelamin,''), COALESCE(j.nama_ayah_kandung,''),
	       COALESCE(j.tempat_lahir,''), COALESCE(DATE_FORMAT(j.tanggal_lahir,'%Y-%m-%d'),''),
	       COALESCE(j.no_paspor,''), COALESCE(j.tempat_paspor_keluar,''),
	       COALESCE(DATE_FORMAT(j.tanggal_paspor_keluar,'%Y-%m-%d'),''), COALESCE(DATE_FORMAT(j.paspor_berlaku_sampai,'%Y-%m-%d'),''),
	       COALESCE(j.no_hp,''), COALESCE(j.pekerjaan,''), COALESCE(j.pendidikan_terakhir,''),
	       COALESCE(j.penjamin_kesehatan,''), COALESCE(j.no_asuransi_bpjs,''), COALESCE(j.alamat,''), COALESCE(j.kota,''),
	       COALESCE(j.emergency_nama,''), COALESCE(j.emergency_hp,''), COALESCE(j.emergency_hubungan,''), COALESCE(j.emergency_alamat,''),
	       ` + sqlAksesPenuh + `
	FROM booking_pax bp JOIN jamaah j ON j.id = bp.jamaah_id
	WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'`

func scanDataJamaah(rows interface{ Scan(...any) error }) (*dataJamaah, error) {
	var d dataJamaah
	var nik string
	var penuh bool
	if err := rows.Scan(&d.JamaahID, &d.NamaLengkap, &nik, &d.PaxType, &d.JenisKelamin, &d.NamaAyahKandung,
		&d.TempatLahir, &d.TanggalLahir, &d.NoPaspor, &d.TempatPasporKeluar, &d.TanggalPasporKeluar, &d.PasporBerlakuSampai,
		&d.NoHP, &d.Pekerjaan, &d.PendidikanTerakhir, &d.PenjaminKesehatan, &d.NoAsuransiBPJS, &d.Alamat, &d.Kota,
		&d.EmergencyNama, &d.EmergencyHP, &d.EmergencyHubungan, &d.EmergencyAlamat, &penuh); err != nil {
		return nil, err
	}
	d.NIKTersamar = nikTersamar(nik)
	d.hitungKelengkapan()
	if !penuh {
		d.batasi()
	}
	return &d, nil
}

// batasi menyisakan yang boleh dilihat akun dengan akses terbatas: nama, NIK tersamar, jenis pax, dan
// angka kelengkapan. Nilai isian dan daftar isian yang kurang tidak ikut.
func (d *dataJamaah) batasi() {
	*d = dataJamaah{JamaahID: d.JamaahID, NamaLengkap: d.NamaLengkap, NIKTersamar: d.NIKTersamar, PaxType: d.PaxType,
		Terisi: d.Terisi, Total: d.Total, Kurang: []string{}, AksesTerbatas: true}
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
	rows, err := o.db.QueryContext(r.Context(), dataJamaahSelect+` ORDER BY bp.id`, acc.ID, id)
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

// peta mengubah isi formulir menjadi peta isian (kunci JSON -> nilai) untuk simpanPerubahanPelanggan.
func (r *simpanDataJamaahRequest) peta() map[string]string {
	return map[string]string{
		"jenis_kelamin": r.JenisKelamin, "nama_ayah_kandung": r.NamaAyahKandung, "tempat_lahir": r.TempatLahir, "tanggal_lahir": r.TanggalLahir,
		"no_paspor": r.NoPaspor, "tempat_paspor_keluar": r.TempatPasporKeluar, "tanggal_paspor_keluar": r.TanggalPasporKeluar,
		"paspor_berlaku_sampai": r.PasporBerlakuSampai, "no_hp": r.NoHP, "pekerjaan": r.Pekerjaan, "pendidikan_terakhir": r.PendidikanTerakhir,
		"penjamin_kesehatan": r.PenjaminKesehatan, "no_asuransi_bpjs": r.NoAsuransiBPJS, "alamat": r.Alamat, "kota": r.Kota,
		"emergency_nama": r.EmergencyNama, "emergency_hp": r.EmergencyHP, "emergency_hubungan": r.EmergencyHubungan, "emergency_alamat": r.EmergencyAlamat,
	}
}

// panjangMaksData: batas panjang tiap isian teks (sama dengan kolom database).
const panjangMaksData = 255

// SimpanDataJamaah PUT /api/account/bookings/{id}/jamaah/{jamaah_id}
// Isian kosong yang diisi langsung tersimpan; mengubah atau mengosongkan isian yang sudah terisi menjadi pengajuan yang menunggu persetujuan admin.
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

	grup, err := kelompokDari(r.Context(), o.db, acc.ID, jamaahID)
	if err != nil {
		tulisGalatSimpan(w, err)
		return
	}
	// Isian kosong yang diisi langsung tersimpan; mengubah isian yang sudah terisi menunggu persetujuan admin (lihat perubahan_jamaah.go).
	var hasil *hasilSimpan
	err = dalamTx(r.Context(), o.db, func(tx *sql.Tx) error {
		var e error
		hasil, e = simpanPerubahanPelanggan(r.Context(), tx, acc.ID, grup, req.peta())
		return e
	})
	if err != nil && !errors.Is(err, ErrTidakBerubah) {
		tulisGalatSimpan(w, err)
		return
	}
	if hasil == nil {
		hasil = &hasilSimpan{}
	}

	d, err := scanDataJamaah(o.db.QueryRowContext(r.Context(), dataJamaahSelect+` AND j.id = ?`, acc.ID, id, jamaahID))
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			log.Printf("[ERROR] account simpan data jamaah baca ulang: %v", err)
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	d.PerubahanLangsung, d.PerubahanMenunggu = hasil.Langsung, hasil.Pengajuan
	writeJSON(w, http.StatusOK, d)
}
