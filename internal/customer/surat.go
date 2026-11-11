package customer

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/surat"
)

// Surat jamaah (Surat Keterangan Umroh, Surat Rekomendasi Paspor) di halaman booking pelanggan. Aturan siapa mendapat surat apa ada di
// evaluasiSurat; nomor dan isi PDF ada di paket surat.

var zonaWIB = time.FixedZone("WIB", 7*3600)

var slugSurat = map[string]string{
	"keterangan-umroh": surat.JenisKeteranganUmroh,
	"rekom-paspor":     surat.JenisRekomPaspor,
}

type suratItem struct {
	Slug   string `json:"slug"`
	Jenis  string `json:"jenis"`
	Label  string `json:"label"`
	Bisa   bool   `json:"bisa"`
	Alasan string `json:"alasan"`
	Nomor  string `json:"nomor"`
	// Kurang: isian jamaah yang boleh dilengkapi pelanggan agar surat bisa diunduh (kunci isian: tempat_lahir, tanggal_lahir). Terisi hanya bila
	// data itulah satu-satunya penghalang; web menampilkan formulir singkat lalu mengunduh.
	Kurang []string `json:"kurang,omitempty"`
}

type suratJamaah struct {
	JamaahID int64       `json:"jamaah_id"`
	Nama     string      `json:"nama"`
	Surat    []suratItem `json:"surat"`
}

func slugJenis(jenis string) string {
	for slug, j := range slugSurat {
		if j == jenis {
			return slug
		}
	}
	return ""
}

// dpTerpenuhi: pembayaran terkonfirmasi sudah mencapai DP minimal (sama dengan aturan "DP terkonfirmasi" di halaman booking).
func dpTerpenuhi(o *orderSummary) bool {
	if o.Tab == "lunas" {
		return true
	}
	if o.MinimalDP > 0 {
		return o.Terbayar >= o.MinimalDP
	}
	return o.Terbayar > 0
}

func kurangData(d *dataJamaah) []string {
	var kurang []string
	if strings.TrimSpace(d.NamaLengkap) == "" {
		kurang = append(kurang, "nama")
	}
	if strings.TrimSpace(d.TempatLahir) == "" {
		kurang = append(kurang, "tempat lahir")
	}
	if strings.TrimSpace(d.TanggalLahir) == "" {
		kurang = append(kurang, "tanggal lahir")
	}
	return kurang
}

// kunciKurang: isian yang bisa dilengkapi pelanggan (nama tidak: identitas terkunci).
func kunciKurang(d *dataJamaah) []string {
	var k []string
	if strings.TrimSpace(d.TempatLahir) == "" {
		k = append(k, "tempat_lahir")
	}
	if strings.TrimSpace(d.TanggalLahir) == "" {
		k = append(k, "tanggal_lahir")
	}
	return k
}

func gabungDan(s []string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return s[0]
	}
	return strings.Join(s[:len(s)-1], ", ") + " dan " + s[len(s)-1]
}

// evaluasiSurat menyusun daftar surat satu jamaah beserta bisa-tidaknya diunduh dan alasannya.
//   - bayi tidak mendapat surat;
//   - Surat Keterangan Umroh: semua jamaah dewasa;
//   - Surat Rekomendasi Paspor: hanya jamaah yang nomor paspornya masih kosong.
//
// Syarat unduh: DP terkonfirmasi, pengaturan surat brand lengkap, nama dan tempat/tanggal lahir jamaah terisi.
func evaluasiSurat(o *orderSummary, d *dataJamaah, pengaturanLengkap bool, nomor map[string]string) []suratItem {
	if d.PaxType == "infant" {
		return nil
	}
	jenis := []string{surat.JenisKeteranganUmroh}
	// Akses terbatas: nomor paspor tidak terbaca, jadi surat rekomendasi paspor tidak ditawarkan.
	if !d.AksesTerbatas && strings.TrimSpace(d.NoPaspor) == "" {
		jenis = append(jenis, surat.JenisRekomPaspor)
	}
	out := make([]suratItem, 0, len(jenis))
	for _, j := range jenis {
		it := suratItem{Slug: slugJenis(j), Jenis: j, Label: surat.LabelJenis(j), Nomor: nomor[fmt.Sprintf("%d|%s", d.JamaahID, j)]}
		switch kurang := kurangData(d); {
		case d.AksesTerbatas:
			// Kedua jenis surat memuat tempat dan tanggal lahir serta kota jamaah.
			it.Alasan = "Data jamaah ini sudah terdaftar. Hubungi kami untuk surat ini."
		case o.Tab == "batal":
			it.Alasan = "Booking tidak aktif."
		case !dpTerpenuhi(o):
			it.Alasan = "Tersedia setelah DP terkonfirmasi."
		case !pengaturanLengkap:
			it.Alasan = "Surat belum tersedia. Hubungi kami."
		case len(kurang) > 0:
			it.Alasan = "Lengkapi " + gabungDan(kurang) + " terlebih dahulu."
			it.Kurang = kunciKurang(d)
		default:
			it.Bisa = true
		}
		out = append(out, it)
	}
	return out
}

func (o *Orders) dataJamaahBooking(r *http.Request, bookingID int64) ([]dataJamaah, error) {
	rows, err := o.db.QueryContext(r.Context(), dataJamaahSelect+` ORDER BY bp.id`, FromContext(r.Context()).ID, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []dataJamaah{}
	for rows.Next() {
		d, err := scanDataJamaah(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// SuratDaftar GET /api/account/bookings/{id}/surat
func (o *Orders) SuratDaftar(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	order, _, err := o.owned(r.Context(), acc.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	}
	if err != nil {
		log.Printf("[ERROR] account surat daftar: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar surat")
		return
	}
	data, err := o.dataJamaahBooking(r, id)
	if err != nil {
		log.Printf("[ERROR] account surat jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	pengaturan, err := o.surat.Pengaturan(r.Context(), uint64(acc.BrandID))
	if err != nil {
		log.Printf("[ERROR] account surat pengaturan: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil pengaturan surat")
		return
	}
	nomor, err := o.surat.NomorBooking(r.Context(), id)
	if err != nil {
		log.Printf("[ERROR] account surat nomor: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil nomor surat")
		return
	}
	jamaah := []suratJamaah{}
	for i := range data {
		items := evaluasiSurat(order, &data[i], pengaturan.Lengkap(), nomor)
		if len(items) == 0 {
			continue
		}
		jamaah = append(jamaah, suratJamaah{JamaahID: data[i].JamaahID, Nama: data[i].NamaLengkap, Surat: items})
	}
	writeJSON(w, http.StatusOK, map[string]any{"dp_terpenuhi": dpTerpenuhi(order), "jamaah": jamaah})
}

func parseTanggal(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(s), zonaWIB)
	if err != nil {
		return time.Time{}
	}
	return t
}

// SuratUnduh GET /api/account/bookings/{id}/surat/{jenis}?jamaah={jamaah_id}: mengunduh PDF surat satu jamaah.
func (o *Orders) SuratUnduh(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	id, ok := bookingIDParam(w, r)
	if !ok {
		return
	}
	jenis, ok := slugSurat[chi.URLParam(r, "jenis")]
	if !ok {
		writeError(w, http.StatusNotFound, "jenis surat tidak dikenal")
		return
	}
	jamaahID, err := strconv.ParseInt(r.URL.Query().Get("jamaah"), 10, 64)
	if err != nil || jamaahID <= 0 {
		writeError(w, http.StatusBadRequest, "jamaah tidak valid")
		return
	}
	order, _, err := o.owned(r.Context(), acc.ID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "booking tidak ditemukan")
		return
	}
	if err != nil {
		log.Printf("[ERROR] account surat unduh: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil booking")
		return
	}
	data, err := o.dataJamaahBooking(r, id)
	if err != nil {
		log.Printf("[ERROR] account surat jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data jamaah")
		return
	}
	var jm *dataJamaah
	for i := range data {
		if data[i].JamaahID == jamaahID {
			jm = &data[i]
		}
	}
	if jm == nil {
		writeError(w, http.StatusNotFound, "jamaah tidak ada di booking ini")
		return
	}
	pengaturan, err := o.surat.Pengaturan(r.Context(), uint64(acc.BrandID))
	if err != nil {
		log.Printf("[ERROR] account surat pengaturan: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil pengaturan surat")
		return
	}
	var item *suratItem
	items := evaluasiSurat(order, jm, pengaturan.Lengkap(), nil)
	for i := range items {
		if items[i].Jenis == jenis {
			item = &items[i]
		}
	}
	if item == nil {
		writeError(w, http.StatusNotFound, "surat ini tidak tersedia untuk jamaah tersebut")
		return
	}
	if !item.Bisa {
		writeError(w, http.StatusForbidden, item.Alasan)
		return
	}

	var berangkat, pulang sql.NullString
	if err := o.db.QueryRowContext(r.Context(), `SELECT DATE_FORMAT(berangkat_tanggal,'%Y-%m-%d'), DATE_FORMAT(pulang_tanggal,'%Y-%m-%d') FROM schedules WHERE id=?`, order.ScheduleID).Scan(&berangkat, &pulang); err != nil {
		log.Printf("[ERROR] account surat jadwal: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil jadwal keberangkatan")
		return
	}
	tglBerangkat, tglPulang := parseTanggal(berangkat.String), parseTanggal(pulang.String)
	if tglBerangkat.IsZero() || tglPulang.IsZero() {
		writeError(w, http.StatusConflict, "tanggal keberangkatan atau kepulangan belum tersedia, hubungi kami")
		return
	}

	var m surat.Merek
	var nama, alamat, kota, telepon, email, wa, logo, ppiu, pihk sql.NullString
	if err := o.db.QueryRowContext(r.Context(), `SELECT name, address, city, phone, email, whatsapp_number, logo_url, ppiu_number, pihk_number FROM brands WHERE id=?`, acc.BrandID).
		Scan(&nama, &alamat, &kota, &telepon, &email, &wa, &logo, &ppiu, &pihk); err != nil {
		log.Printf("[ERROR] account surat brand: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil data brand")
		return
	}
	m = surat.Merek{Nama: nama.String, Alamat: alamat.String, Kota: kota.String, Telepon: telepon.String, Email: email.String, WhatsApp: wa.String, LogoURL: logo.String, PPIU: ppiu.String, PIHK: pihk.String}

	terbitan, err := o.surat.Terbit(r.Context(), uint64(acc.BrandID), id, jamaahID, jenis, time.Now().In(zonaWIB))
	if errors.Is(err, surat.ErrPengaturanBelumLengkap) {
		writeError(w, http.StatusForbidden, "Surat belum tersedia. Hubungi kami.")
		return
	}
	if err != nil {
		log.Printf("[ERROR] account surat terbit: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal menerbitkan nomor surat")
		return
	}
	pdf, err := surat.Render(surat.Data{
		Jenis:      jenis,
		Nomor:      terbitan.Nomor,
		TerbitAt:   terbitan.TerbitAt.In(zonaWIB),
		Pengaturan: pengaturan,
		Merek:      m,
		Jamaah:     surat.Jamaah{Nama: jm.NamaLengkap, TempatLahir: jm.TempatLahir, TanggalLahir: parseTanggal(jm.TanggalLahir), Kota: jm.Kota},
		Berangkat:  tglBerangkat,
		Pulang:     tglPulang,
	})
	if err != nil {
		log.Printf("[ERROR] account surat render: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal membuat surat")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, surat.NamaBerkas(jenis, jm.NamaLengkap)))
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(pdf)
}
