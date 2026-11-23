package agen

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/shared"

	"github.com/go-chi/chi/v5"
)

// errFilterKomisi: parameter angka pada filter komisi bukan bilangan bulat tak negatif.
var errFilterKomisi = errors.New("parameter limit, offset, dan agen_id harus bilangan bulat tak negatif")

// angkaQuery membaca parameter angka opsional. Kosong berarti 0; isi yang bukan bilangan bulat tak negatif
// ditolak, supaya salah ketik tidak diam-diam menjadi "tanpa batas" atau "semua agen".
func angkaQuery(q url.Values, nama string) (int64, error) {
	v := strings.TrimSpace(q.Get(nama))
	if v == "" {
		return 0, nil
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		return 0, errFilterKomisi
	}
	return n, nil
}

func komisiFilterFromQuery(r *http.Request) (KomisiFilter, error) {
	q := r.URL.Query()
	limit, err := angkaQuery(q, "limit")
	if err != nil {
		return KomisiFilter{}, err
	}
	offset, err := angkaQuery(q, "offset")
	if err != nil {
		return KomisiFilter{}, err
	}
	penerima, err := angkaQuery(q, "agen_id")
	if err != nil {
		return KomisiFilter{}, err
	}
	return KomisiFilter{
		PenerimaID: penerima,
		Jenis:      q.Get("jenis"),
		Dari:       q.Get("dari"),
		Sampai:     q.Get("sampai"),
		Status:     q.Get("status"),
		Cari:       q.Get("q"),
		Limit:      int(limit),
		Offset:     int(offset),
	}, nil
}

// adminBrandScope: Admin Travel selalu brand-nya sendiri; Super Admin boleh
// memfilter brand lewat ?brand_id= (kosong = semua brand).
func adminBrandScope(r *http.Request) *int64 {
	if b := identity.GetBrandID(r.Context()); b != nil {
		return b
	}
	if id, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64); err == nil && id > 0 {
		return &id
	}
	return nil
}

func handleKinerjaError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrBukanAgenAktif) {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	handleError(w, err)
}

// Dashboard GET /api/portal/agen/dashboard (A3)
func (h *Handler) Dashboard(w http.ResponseWriter, r *http.Request) {
	d, err := h.repo.GetDashboardAgen(r.Context(), identity.GetCustomerAccountID(r.Context()))
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// MutasiPortal GET /api/portal/agen/mutasi: komisi, koreksi, dan pencairan terbaru untuk app.
func (h *Handler) MutasiPortal(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListMutasiAgen(r.Context(), identity.GetCustomerAccountID(r.Context()))
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// DataJamaahPortal GET /api/portal/agen/data-jamaah: jamaah per booking minimal DP dari jamaah terikat.
func (h *Handler) DataJamaahPortal(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListDataJamaahAgen(r.Context(), identity.GetCustomerAccountID(r.Context()), time.Now())
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// AgenBinaanPortal GET /api/portal/agen/binaan: agen yang upline-nya agen ini.
func (h *Handler) AgenBinaanPortal(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.ListAgenBinaan(r.Context(), identity.GetCustomerAccountID(r.Context()), time.Now())
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// Batas klik referral per IP pengunjung. Penanda pengunjung dikirim situs dan bisa diisi bebas oleh pemanggil
// langsung, jadi tanpa batas ini satu IP bisa menambah klik tanpa akhir dengan penanda acak. Klik di atas batas
// tetap dijawab 204 tetapi tidak dicatat. Jatah hanya terpakai oleh klik yang benar-benar tercatat: buka ulang oleh
// pengunjung yang sama, kode tak dikenal, atau klik yang ditolak tidak menghabiskan jatah orang lain di balik satu
// NAT (kantor, operator seluler).
const (
	klikPerIPPerJam        = 60 // semua kode
	klikPerIPPerKodeSehari = 20 // satu kode
)

// pengunjungValid: penanda acak dari situs (UUID dari azhan-booking), 16-64 huruf, angka, atau tanda hubung.
func pengunjungValid(s string) bool {
	if len(s) < 16 || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

func kunciKlikKode(ip string, brandID int64, kode string) string {
	return ip + "|" + strconv.FormatInt(brandID, 10) + "|" + kode
}

// ambilJatahKlik mengambil satu jatah per jam untuk IP ini dan satu jatah harian untuk kode itu. Gagal di jatah
// kode mengembalikan jatah per jam, jadi klik yang ditolak tidak mengurangi jatah apa pun.
func (h *Handler) ambilJatahKlik(ip string, brandID int64, kode string) bool {
	if !h.klikIP.Take(ip) {
		return false
	}
	if !h.klikIPKode.Take(kunciKlikKode(ip, brandID, kode)) {
		h.klikIP.Refund(ip)
		return false
	}
	return true
}

// kembalikanJatahKlik dipanggil bila klik tidak menambah baris (pengunjung sama hari ini, kode tak dikenal, galat).
func (h *Handler) kembalikanJatahKlik(ip string, brandID int64, kode string) {
	h.klikIP.Refund(ip)
	h.klikIPKode.Refund(kunciKlikKode(ip, brandID, kode))
}

// KlikReferral POST /api/public/referral/klik: {brand_id, kode_referral, pengunjung}. Selalu 204; kode yang tidak
// dikenal diabaikan supaya endpoint tidak bisa dipakai menebak kode.
func (h *Handler) KlikReferral(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BrandID      int64  `json:"brand_id"`
		KodeReferral string `json:"kode_referral"`
		Pengunjung   string `json:"pengunjung"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 4096)).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Data tidak valid.")
		return
	}
	// IP dari shared.ClientIP: header proxy hanya dipercaya dari proxy tepercaya (TRUSTED_PROXIES), jadi pemanggil
	// langsung tidak bisa memalsukan IP lewat X-Forwarded-For.
	ip := shared.ClientIP(r)
	kode := strings.ToUpper(strings.TrimSpace(req.KodeReferral))
	if !h.ambilJatahKlik(ip, req.BrandID, kode) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	pengunjung := strings.TrimSpace(req.Pengunjung)
	if !pengunjungValid(pengunjung) {
		pengunjung = "" // penanda tidak sah: pakai IP dan user agent
	}
	hash := HashPengunjung(pengunjung, ip, r.UserAgent())
	baru, err := h.repo.CatatKlikReferral(r.Context(), req.BrandID, kode, hash, time.Now())
	if err != nil {
		log.Printf("[ERROR] klik referral: %v", err)
	}
	if !baru {
		h.kembalikanJatahKlik(ip, req.BrandID, kode)
	}
	w.WriteHeader(http.StatusNoContent)
}

// RiwayatKomisiPortal GET /api/portal/agen/komisi (A6)
func (h *Handler) RiwayatKomisiPortal(w http.ResponseWriter, r *http.Request) {
	f, err := komisiFilterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	items, err := h.repo.ListKomisiAgen(r.Context(), identity.GetCustomerAccountID(r.Context()), f)
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// ListAgen GET /api/admin/agen?q=&dari=&sampai= (B2)
// dari dan sampai (YYYY-MM-DD) membatasi pax_closing dan total_komisi ke komisi yang tercatat di rentang itu;
// keduanya kosong berarti sepanjang waktu.
func (h *Handler) ListAgen(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.repo.ListAgen(r.Context(), adminBrandScope(r), q.Get("q"), RentangAgen{Dari: q.Get("dari"), Sampai: q.Get("sampai")})
	if errors.Is(err, ErrPeriodeTidakValid) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// DetailAgen GET /api/admin/agen/{akunID} (B2a)
func (h *Handler) DetailAgen(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "akunID")
	if !ok {
		return
	}
	d, err := h.repo.GetDetailAgen(r.Context(), id, identity.GetBrandID(r.Context()))
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// PeringkatAgen GET /api/admin/agen/peringkat?dari=&sampai=&brand_id=
// Periode wajib; tanggal mengacu ke waktu komisi tercatat (booking lunas).
func (h *Handler) PeringkatAgen(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := h.repo.PeringkatAgen(r.Context(), adminBrandScope(r), q.Get("dari"), q.Get("sampai"))
	if errors.Is(err, ErrPeriodeTidakValid) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// RiwayatKomisiAdmin GET /api/admin/agen/komisi?agen_id=&jenis=&dari=&sampai=&status=&limit=&offset= (B3)
// Jawaban: {items, ringkasan}. Ringkasan dihitung server atas seluruh hasil filter, bukan halaman yang tampil.
// status: kosong = komisi aktif, "batal" = yang dibatalkan, "semua" = keduanya.
func (h *Handler) RiwayatKomisiAdmin(w http.ResponseWriter, r *http.Request) {
	f, err := komisiFilterFromQuery(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	f.BrandID = adminBrandScope(r)
	halaman, err := h.repo.HalamanKomisiBrand(r.Context(), f)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, halaman)
}

// DetailJamaahPortal GET /api/portal/agen/data-jamaah/{bookingID}/{jamaahID}: status satu jamaah untuk agen perekrutnya.
func (h *Handler) DetailJamaahPortal(w http.ResponseWriter, r *http.Request) {
	bookingID, err1 := strconv.ParseInt(chi.URLParam(r, "bookingID"), 10, 64)
	jamaahID, err2 := strconv.ParseInt(chi.URLParam(r, "jamaahID"), 10, 64)
	if err1 != nil || err2 != nil || bookingID <= 0 || jamaahID <= 0 {
		writeError(w, http.StatusNotFound, "Jamaah tidak ditemukan.")
		return
	}
	d, err := h.repo.DetailJamaahAgen(r.Context(), identity.GetCustomerAccountID(r.Context()), bookingID, jamaahID, time.Now())
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "Jamaah tidak ditemukan.")
		return
	}
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}
