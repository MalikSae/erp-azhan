package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/komisi"
)

// Portal agen berbasis akun email (spek 10 butir 10): memakai sesi akun pelanggan (customer.RequireCustomer).
// Status portal dan pengajuan terbuka untuk akun mana pun; selebihnya hanya untuk agen aktif.

// RequireAgenAktif menolak akun yang bukan agen aktif dengan 403. Agen nonaktif ditolak juga (layar A4:
// halaman tertutup, hubungi admin). Dipasang setelah customer.RequireCustomer.
func (h *Handler) RequireAgenAktif(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := requireAgenAktif(r.Context(), h.repo.db, identity.GetCustomerAccountID(r.Context())); err != nil {
			handleKinerjaError(w, err)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// mediaMilikAkun memastikan url berkas adalah hasil unggahan akun ini (media_uploads), supaya foto dan bukti
// transfer tidak bisa diisi URL karangan atau berkas milik akun lain.
// MediaMilikAkun: versi publik untuk paket lain (foto profil akun).
func MediaMilikAkun(ctx context.Context, q querier, akunID int64, url string) (bool, error) {
	return mediaMilikAkun(ctx, q, akunID, url)
}

func mediaMilikAkun(ctx context.Context, q querier, akunID int64, url string) (bool, error) {
	var ada bool
	err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM media_uploads WHERE path = ? AND customer_account_id = ?)`, url, akunID).Scan(&ada)
	if err != nil {
		return false, fmt.Errorf("agen: cek kepemilikan berkas: %w", err)
	}
	return ada, nil
}

// ─── Rentang ujroh untuk landing /jadi-agen (K23) ────────────────────────────

type Rentang struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// UjrohPublik: rentang komisi dari jadwal tayang (published, belum berangkat) satu brand, memakai nominal
// efektif (nilai jadwal, selain itu bawaan brand). Rentang null bila tidak ada jadwal yang ikut program
// (nominal efektif > 0).
type UjrohPublik struct {
	BrandID              int64    `json:"brand_id"`
	JumlahJadwal         int      `json:"jumlah_jadwal"`
	KomisiLangsung       *Rentang `json:"komisi_langsung"`
	BonusPembinaan       *Rentang `json:"bonus_pembinaan"`
	BiayaPendaftaranAgen float64  `json:"biaya_pendaftaran_agen"`
	SyaratKetentuanAgen  *string  `json:"syarat_ketentuan_agen"`
	// Konten halaman Jadi Agen di app, diatur admin di form Brand: gambar rekrutmen (URL relatif ke API), benefit
	// (satu per butir), dan draft MoU (HTML terbatas). S&K juga HTML terbatas; teks lama tanpa tag tetap terbaca.
	GambarRekrutmenAgen *string  `json:"gambar_rekrutmen_agen"`
	BenefitAgen         []string `json:"benefit_agen"`
	MouAgen             *string  `json:"mou_agen"`
}

func rentangUjroh(ctx context.Context, q querier, brandID int64) (*UjrohPublik, error) {
	u := UjrohPublik{BrandID: brandID}
	var benefit sql.NullString
	err := q.QueryRowContext(ctx, `SELECT biaya_pendaftaran_agen, syarat_ketentuan_agen, gambar_rekrutmen_agen, benefit_agen, mou_agen FROM brands WHERE id = ?`, brandID).
		Scan(&u.BiayaPendaftaranAgen, &u.SyaratKetentuanAgen, &u.GambarRekrutmenAgen, &benefit, &u.MouAgen)
	u.BenefitAgen = []string{}
	for _, b := range strings.Split(benefit.String, "\n") {
		if t := strings.TrimSpace(b); t != "" {
			u.BenefitAgen = append(u.BenefitAgen, t)
		}
	}
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("agen: baca brand ujroh: %w", err)
	}
	var minL, maxL, minB, maxB sql.NullFloat64
	err = q.QueryRowContext(ctx, `
		SELECT COUNT(*),
		       MIN(CASE WHEN langsung > 0 THEN langsung END),
		       MAX(CASE WHEN langsung > 0 THEN langsung END),
		       MIN(CASE WHEN pembinaan > 0 THEN pembinaan END),
		       MAX(CASE WHEN pembinaan > 0 THEN pembinaan END)
		FROM (
			SELECT `+komisi.SQLKomisiLangsungEfektif+` AS langsung,
			       `+komisi.SQLBonusPembinaanEfektif+` AS pembinaan
			FROM schedules s JOIN brands br ON br.id = s.brand_id
			WHERE s.brand_id = ? AND s.status = 'published' AND s.berangkat_tanggal >= CURRENT_DATE
		) efektif
		WHERE langsung > 0 OR pembinaan > 0`, brandID).
		Scan(&u.JumlahJadwal, &minL, &maxL, &minB, &maxB)
	if err != nil {
		return nil, fmt.Errorf("agen: rentang ujroh: %w", err)
	}
	if minL.Valid {
		u.KomisiLangsung = &Rentang{Min: minL.Float64, Max: maxL.Float64}
	}
	if minB.Valid {
		u.BonusPembinaan = &Rentang{Min: minB.Float64, Max: maxB.Float64}
	}
	return &u, nil
}

// UjrohPublik GET /api/public/v1/agen/ujroh?brand_id= (publik, read-only)
func (h *Handler) UjrohPublik(w http.ResponseWriter, r *http.Request) {
	brandID, err := strconv.ParseInt(r.URL.Query().Get("brand_id"), 10, 64)
	if err != nil || brandID <= 0 {
		writeError(w, http.StatusBadRequest, "brand_id wajib diisi")
		return
	}
	u, err := rentangUjroh(r.Context(), h.repo.db, brandID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}
