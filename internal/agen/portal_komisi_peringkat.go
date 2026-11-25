package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/komisi"
	"erp-azhan/api/internal/shared"
)

// Komisi per paket dan peringkat agen untuk portal agen (docs/spek/SPEK-KOMISI-PAKET-PERINGKAT-AGEN-2026-11-25.md).

// KomisiPaket: komisi langsung efektif satu jadwal (nilai jadwal, selain itu bawaan brand), per jamaah.
type KomisiPaket struct {
	ScheduleID     int64   `json:"schedule_id"`
	KomisiLangsung float64 `json:"komisi_langsung"`
}

// brandAkunAgen: brand akun agen aktif. Agen tidak aktif -> ErrBukanAgenAktif (rute sudah dijaga RequireAgenAktif).
func brandAkunAgen(ctx context.Context, q querier, akunID int64) (int64, error) {
	var brandID int64
	var status string
	err := q.QueryRowContext(ctx, `SELECT brand_id, status_agen FROM customer_accounts WHERE id = ? AND dihapus_at IS NULL`, akunID).
		Scan(&brandID, &status)
	if errors.Is(err, sql.ErrNoRows) || (err == nil && status != "aktif") {
		return 0, ErrBukanAgenAktif
	}
	if err != nil {
		return 0, fmt.Errorf("agen: brand akun: %w", err)
	}
	return brandID, nil
}

// komisiPaketAgen: jadwal tayang (published, belum berangkat) brand agen dengan komisi langsung efektif > 0.
// Sama dengan nilai yang dibekukan ke booking saat dibuat (komisi.SQLKomisiLangsungEfektif).
func komisiPaketAgen(ctx context.Context, q querier, akunID int64) ([]KomisiPaket, error) {
	brandID, err := brandAkunAgen(ctx, q, akunID)
	if err != nil {
		return nil, err
	}
	rows, err := q.QueryContext(ctx, `
		SELECT id, langsung FROM (
			SELECT s.id, `+komisi.SQLKomisiLangsungEfektif+` AS langsung
			FROM schedules s JOIN brands br ON br.id = s.brand_id
			WHERE s.brand_id = ? AND s.status = 'published' AND s.berangkat_tanggal >= CURRENT_DATE
		) efektif
		WHERE langsung > 0
		ORDER BY id`, brandID)
	if err != nil {
		return nil, fmt.Errorf("agen: komisi paket: %w", err)
	}
	defer rows.Close()
	out := []KomisiPaket{}
	for rows.Next() {
		var k KomisiPaket
		if err := rows.Scan(&k.ScheduleID, &k.KomisiLangsung); err != nil {
			return nil, fmt.Errorf("agen: scan komisi paket: %w", err)
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

// MaksPeringkatPortal: jumlah agen teratas yang tampil di portal agen.
const MaksPeringkatPortal = 10

type PeringkatSaya struct {
	// Peringkat null bila agen belum punya pax closing di periode itu ("Butuh 1 pax untuk masuk peringkat").
	Peringkat  *int `json:"peringkat"`
	PaxClosing int  `json:"pax_closing"`
	// Target: kelompok tepat di atas agen; null bila sudah #1 atau belum masuk peringkat.
	Target *TargetPeringkat `json:"target"`
}

// TargetPeringkat: menyamai pax kelompok tepat di atas berarti ikut memegang nomor itu (peringkat bersama).
type TargetPeringkat struct {
	Peringkat int `json:"peringkat"`
	KurangPax int `json:"kurang_pax"`
}

type PeringkatBaris struct {
	Peringkat int     `json:"peringkat"`
	AkunID    int64   `json:"akun_id"`
	Nama      string  `json:"nama"`
	Kota      *string `json:"kota"`
	// AdaFoto: foto profil tersedia di GET /api/portal/agen/peringkat/{akun_id}/foto.
	AdaFoto    bool `json:"ada_foto"`
	PaxClosing int  `json:"pax_closing"`
	Saya       bool `json:"saya"`
}

type PeringkatPortal struct {
	Periode string `json:"periode"`
	Dari    string `json:"dari"`
	Sampai  string `json:"sampai"`
	// JumlahAgen: agen aktif yang masuk peringkat (minimal satu pax closing di periode itu).
	JumlahAgen int              `json:"jumlah_agen"`
	Saya       PeringkatSaya    `json:"saya"`
	Teratas    []PeringkatBaris `json:"teratas"`
}

var ErrPeriodePeringkat = errors.New("periode harus bulan atau tahun")

// rentangPeriode: awal bulan atau awal tahun (WIB) sampai hari ini.
func rentangPeriode(periode string, now time.Time) (string, string, error) {
	t := now.In(zonaWIB)
	switch periode {
	case "", "bulan":
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, zonaWIB).Format("2006-01-02"), t.Format("2006-01-02"), nil
	case "tahun":
		return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, zonaWIB).Format("2006-01-02"), t.Format("2006-01-02"), nil
	}
	return "", "", ErrPeriodePeringkat
}

// peringkatPortalAgen: peringkat pax closing agen aktif satu brand. Ukuran sama dengan admin (pax closing dari
// komisi langsung + repeat order), nomor memakai peringkat bersama: pax sama = nomor sama, nomor berikutnya melompat
// (1, 2, 2, 4). Komisi lebih besar lalu nama hanya mengatur urutan tampil. Agen nonaktif dan tanpa pax tidak diberi
// nomor. Tampil maksimal 10 baris pertama ditambah baris agen sendiri; nama tanpa kode referral.
func peringkatPortalAgen(ctx context.Context, q querier, akunID int64, periode string, now time.Time) (*PeringkatPortal, error) {
	if periode == "" {
		periode = "bulan"
	}
	dari, sampai, err := rentangPeriode(periode, now)
	if err != nil {
		return nil, err
	}
	brandID, err := brandAkunAgen(ctx, q, akunID)
	if err != nil {
		return nil, err
	}
	semua, err := peringkatAgen(ctx, q, &brandID, dari, sampai)
	if err != nil {
		return nil, err
	}
	aktif := make([]PeringkatItem, 0, len(semua))
	for _, it := range semua {
		if it.StatusAgen == "aktif" && it.PaxClosing > 0 {
			aktif = append(aktif, it)
		}
	}
	sort.SliceStable(aktif, func(i, j int) bool {
		if aktif[i].PaxClosing != aktif[j].PaxClosing {
			return aktif[i].PaxClosing > aktif[j].PaxClosing
		}
		if aktif[i].TotalKomisi != aktif[j].TotalKomisi {
			return aktif[i].TotalKomisi > aktif[j].TotalKomisi
		}
		return aktif[i].NamaLengkap < aktif[j].NamaLengkap
	})

	p := &PeringkatPortal{Periode: periode, Dari: dari, Sampai: sampai, JumlahAgen: len(aktif), Teratas: []PeringkatBaris{}}
	baris := make([]PeringkatBaris, len(aktif))
	for i, it := range aktif {
		nomor := i + 1
		if i > 0 && it.PaxClosing == aktif[i-1].PaxClosing {
			nomor = baris[i-1].Peringkat
		}
		baris[i] = PeringkatBaris{Peringkat: nomor, AkunID: it.AkunID, Nama: it.NamaLengkap, PaxClosing: it.PaxClosing, Saya: it.AkunID == akunID}
		if baris[i].Saya {
			p.Saya = PeringkatSaya{Peringkat: &nomor, PaxClosing: it.PaxClosing}
			// Kelompok tepat di atas: baris terakhir sebelum kelompok sendiri.
			for k := i - 1; k >= 0; k-- {
				if baris[k].PaxClosing > it.PaxClosing {
					p.Saya.Target = &TargetPeringkat{Peringkat: baris[k].Peringkat, KurangPax: baris[k].PaxClosing - it.PaxClosing}
					break
				}
			}
		}
	}
	// Kota dan foto hanya untuk baris yang tampil: 10 pertama dan baris agen sendiri.
	for i := range baris {
		if i < MaksPeringkatPortal || baris[i].Saya {
			var foto sql.NullString
			if err := q.QueryRowContext(ctx, `SELECT NULLIF(TRIM(domisili), ''), foto_url FROM customer_accounts WHERE id = ?`, baris[i].AkunID).Scan(&baris[i].Kota, &foto); err != nil {
				return nil, fmt.Errorf("agen: kota peringkat: %w", err)
			}
			baris[i].AdaFoto = FotoPortalURL.MatchString(foto.String)
			p.Teratas = append(p.Teratas, baris[i])
		}
	}
	return p, nil
}

func (r *Repository) KomisiPaketAgen(ctx context.Context, akunID int64) ([]KomisiPaket, error) {
	return komisiPaketAgen(ctx, r.db, akunID)
}

func (r *Repository) PeringkatPortalAgen(ctx context.Context, akunID int64, periode string, now time.Time) (*PeringkatPortal, error) {
	return peringkatPortalAgen(ctx, r.db, akunID, periode, now)
}

// fotoAgenSebrand: berkas foto profil agen aktif yang sebrand dengan pemanggil, untuk avatar peringkat.
// "" bila bukan agen aktif sebrand atau tanpa foto.
func fotoAgenSebrand(ctx context.Context, q querier, pemanggil, akunID int64) (string, error) {
	brandID, err := brandAkunAgen(ctx, q, pemanggil)
	if err != nil {
		return "", err
	}
	var foto sql.NullString
	err = q.QueryRowContext(ctx, `SELECT foto_url FROM customer_accounts
		WHERE id = ? AND brand_id = ? AND status_agen = 'aktif' AND dihapus_at IS NULL`, akunID, brandID).Scan(&foto)
	if errors.Is(err, sql.ErrNoRows) || !FotoPortalURL.MatchString(foto.String) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("agen: foto peringkat: %w", err)
	}
	return "uploads/dokumen-jamaah/" + path.Base(foto.String), nil
}

// FotoPeringkatPortal GET /api/portal/agen/peringkat/{akunID}/foto: foto profil agen sebrand (404 bila tidak ada).
func (h *Handler) FotoPeringkatPortal(w http.ResponseWriter, r *http.Request) {
	akunID, err := strconv.ParseInt(chi.URLParam(r, "akunID"), 10, 64)
	if err != nil || akunID <= 0 {
		http.NotFound(w, r)
		return
	}
	berkas, err := fotoAgenSebrand(r.Context(), h.repo.db, identity.GetCustomerAccountID(r.Context()), akunID)
	if err != nil && !errors.Is(err, ErrBukanAgenAktif) {
		log.Printf("[ERROR] foto peringkat: %v", err)
	}
	if err != nil || berkas == "" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=300")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	http.ServeFile(w, r, filepath.FromSlash(berkas))
}

// KomisiPaketPortal GET /api/portal/agen/komisi-paket: komisi langsung per jadwal tayang brand agen.
func (h *Handler) KomisiPaketPortal(w http.ResponseWriter, r *http.Request) {
	items, err := h.repo.KomisiPaketAgen(r.Context(), identity.GetCustomerAccountID(r.Context()))
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, items)
}

// PeringkatPortal GET /api/portal/agen/peringkat?periode=bulan|tahun: 10 teratas dan posisi agen sendiri.
func (h *Handler) PeringkatPortal(w http.ResponseWriter, r *http.Request) {
	p, err := h.repo.PeringkatPortalAgen(r.Context(), identity.GetCustomerAccountID(r.Context()), r.URL.Query().Get("periode"), time.Now())
	if errors.Is(err, ErrPeriodePeringkat) {
		shared.WriteError(w, http.StatusBadRequest, "Periode tidak valid.")
		return
	}
	if err != nil {
		handleKinerjaError(w, err)
		return
	}
	shared.WriteJSON(w, http.StatusOK, p)
}
