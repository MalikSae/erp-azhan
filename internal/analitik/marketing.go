// Package analitik menyusun ringkasan KPI marketing untuk Dashboard Utama ERP.
//
// Keputusan 12 November 2026 (lihat GLOSSARY "Pax DP+"):
//   - Pax dihitung dari booking berstatus dp atau lunas (pax DP+), dikelompokkan menurut tanggal pax dibuat.
//   - "Dari agen" = jamaah berstatus terikat_agen. "Tanpa agen" = tanpa_agen dan belum_ditentukan,
//     karena jamaah web tanpa agen memang tetap belum_ditentukan. Kelompok mengikuti kaitan jamaah saat ini.
//   - Seat terisi hanya menghitung pax reguler (infant tidak memakai seat).
//   - 10 agen terbaik memakai PaxClosing yang sama dengan menu Agen (ledger komisi, saat lunas).
package analitik

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"erp-azhan/api/internal/agen"
)

// Periode ringkasan dan ambang tanda seat terisi. Ubah di sini bila target bisnis berubah.
const (
	periodeHari = 30

	// Seat terisi hanya untuk jadwal yang masih bisa dijual: booking online (web dan portal agen) ditutup
	// H-30 (selfbooking ErrCutoffBooking), jadi daftar mulai H-30 sampai H-90.
	batasJualHari   = 30
	jendelaSeatHari = 90

	// Merah: H-30 sampai H-45 dan terisi di bawah ambangMerahPersen.
	batasMerahHari    = 45
	ambangMerahPersen = 70
	// Kuning: H-46 sampai H-90 dan terisi di bawah ambangKuningPersen.
	ambangKuningPersen = 40
	// Target isian: "kurang N seat" dihitung terhadap ambang merah.
	targetPersen = ambangMerahPersen

	jumlahAgenTerbaik = 10
)

const (
	TandaMerah  = "merah"
	TandaKuning = "kuning"
)

type Periode struct {
	Dari   string `json:"dari"`
	Sampai string `json:"sampai"`
}

type PaxBrand struct {
	BrandID   int64  `json:"brand_id"`
	BrandName string `json:"brand_name"`
	Agen      int    `json:"agen"`
	TanpaAgen int    `json:"tanpa_agen"`
}

// PaxHarian: pax DP+ per tanggal pax dibuat, dipecah per channel (untuk grafik).
type PaxHarian struct {
	Tanggal   string `json:"tanggal"`
	Agen      int    `json:"agen"`
	TanpaAgen int    `json:"tanpa_agen"`
}

type PaxChannel struct {
	Agen                int         `json:"agen"`
	TanpaAgen           int         `json:"tanpa_agen"`
	AgenSebelumnya      int         `json:"agen_sebelumnya"`
	TanpaAgenSebelumnya int         `json:"tanpa_agen_sebelumnya"`
	PerBrand            []PaxBrand  `json:"per_brand"`
	Harian              []PaxHarian `json:"harian"` // hanya tanggal yang punya pax, urut tanggal
}

type SeatJadwal struct {
	ScheduleID       int64  `json:"schedule_id"`
	Nama             string `json:"nama"`
	BrandID          int64  `json:"brand_id"`
	BrandName        string `json:"brand_name"`
	BerangkatTanggal string `json:"berangkat_tanggal"`
	SisaHari         int    `json:"sisa_hari"`
	SeatTotal        int    `json:"seat_total"`
	Terisi           int    `json:"terisi"`      // pax reguler di booking dp/lunas
	Hold             int    `json:"hold"`        // seat terpakai selain Terisi (booking baru/draft, penyesuaian)
	Persen           int    `json:"persen"`      // Terisi / SeatTotal, dibulatkan ke bawah
	KurangSeat       int    `json:"kurang_seat"` // seat DP+ yang masih dibutuhkan untuk mencapai targetPersen
	Tanda            string `json:"tanda"`       // merah, kuning, atau kosong
}

// AktivasiAgen: berapa agen aktif yang punya pax closing dalam periode.
type AktivasiAgen struct {
	AgenAktif   int `json:"agen_aktif"`
	AgenClosing int `json:"agen_closing"`
}

type AgenTerbaik struct {
	Peringkat    int     `json:"peringkat"`
	AkunID       int64   `json:"akun_id"`
	NamaLengkap  string  `json:"nama_lengkap"`
	KodeReferral *string `json:"kode_referral"`
	BrandID      int64   `json:"brand_id"`
	BrandName    string  `json:"brand_name"`
	StatusAgen   string  `json:"status_agen"`
	PaxClosing   int     `json:"pax_closing"`
}

type MarketingRingkas struct {
	Periode     Periode       `json:"periode"`
	PaxChannel  PaxChannel    `json:"pax_channel"`
	SeatTerisi  []SeatJadwal  `json:"seat_terisi"`
	AgenTerbaik []AgenTerbaik `json:"agen_terbaik"`
	Aktivasi    AktivasiAgen  `json:"aktivasi_agen"`
}

type Repository struct {
	db   *sql.DB
	agen *agen.Repository
}

func NewRepository(db *sql.DB, agenRepo *agen.Repository) *Repository {
	return &Repository{db: db, agen: agenRepo}
}

type querier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// MarketingRingkas menyusun ketiga KPI. brandID nil = semua brand (Owner/holding).
func (r *Repository) MarketingRingkas(ctx context.Context, brandID *int64) (*MarketingRingkas, error) {
	hasil, err := ringkasTanpaAgen(ctx, r.db, brandID)
	if err != nil {
		return nil, err
	}
	peringkat, err := r.agen.PeringkatAgen(ctx, brandID, hasil.Periode.Dari, hasil.Periode.Sampai)
	if err != nil {
		return nil, fmt.Errorf("analitik: peringkat agen: %w", err)
	}
	hasil.AgenTerbaik = pilihAgenTerbaik(peringkat)
	hasil.Aktivasi, err = aktivasiAgen(ctx, r.db, brandID, peringkat)
	if err != nil {
		return nil, err
	}
	return hasil, nil
}

// aktivasiAgen: agen aktif sekarang (scope brand) dan berapa di antaranya punya pax closing dalam periode.
// Agen yang sudah nonaktif tetap tampil di peringkat, tapi tidak dihitung di sini.
func aktivasiAgen(ctx context.Context, q querier, brandID *int64, peringkat []agen.PeringkatItem) (AktivasiAgen, error) {
	query := `SELECT COUNT(*) FROM customer_accounts WHERE status_agen = 'aktif'`
	var args []any
	if brandID != nil {
		query += ` AND brand_id = ?`
		args = append(args, *brandID)
	}
	var a AktivasiAgen
	if err := q.QueryRowContext(ctx, query, args...).Scan(&a.AgenAktif); err != nil {
		return a, fmt.Errorf("analitik: jumlah agen aktif: %w", err)
	}
	for _, p := range peringkat {
		if p.PaxClosing > 0 && p.StatusAgen == "aktif" {
			a.AgenClosing++
		}
	}
	return a, nil
}

// ringkasTanpaAgen menghitung periode, pax per channel, dan seat terisi. Dipisah supaya bisa diuji di
// dalam transaksi test.
func ringkasTanpaAgen(ctx context.Context, q querier, brandID *int64) (*MarketingRingkas, error) {
	var hariIni time.Time
	if err := q.QueryRowContext(ctx, `SELECT CURDATE()`).Scan(&hariIni); err != nil {
		return nil, fmt.Errorf("analitik: tanggal hari ini: %w", err)
	}
	hasil := &MarketingRingkas{Periode: Periode{
		Dari:   hariIni.AddDate(0, 0, -(periodeHari - 1)).Format("2006-01-02"),
		Sampai: hariIni.Format("2006-01-02"),
	}}
	pc, err := paxChannel(ctx, q, brandID)
	if err != nil {
		return nil, err
	}
	hasil.PaxChannel = pc
	seat, err := seatTerisi(ctx, q, brandID)
	if err != nil {
		return nil, err
	}
	hasil.SeatTerisi = seat
	return hasil, nil
}

func paxChannel(ctx context.Context, q querier, brandID *int64) (PaxChannel, error) {
	query := `
		SELECT s.brand_id, br.name, DATE_FORMAT(bp.created_at, '%Y-%m-%d') AS tanggal,
		       DATE(bp.created_at) >= CURDATE() - INTERVAL ? DAY AS periode_ini,
		       COALESCE(j.kaitan_status, '') = 'terikat_agen' AS dari_agen,
		       COUNT(*)
		FROM booking_pax bp
		JOIN bookings b ON b.id = bp.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		JOIN brands br ON br.id = s.brand_id
		LEFT JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE bp.pax_status = 'aktif'
		  AND b.status IN ('dp', 'lunas')
		  AND DATE(bp.created_at) BETWEEN CURDATE() - INTERVAL ? DAY AND CURDATE()`
	args := []any{periodeHari - 1, 2*periodeHari - 1}
	if brandID != nil {
		query += ` AND s.brand_id = ?`
		args = append(args, *brandID)
	}
	query += `
		GROUP BY s.brand_id, br.name, tanggal, periode_ini, dari_agen
		ORDER BY br.name, tanggal`
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return PaxChannel{}, fmt.Errorf("analitik: pax per channel: %w", err)
	}
	defer rows.Close()

	pc := PaxChannel{PerBrand: []PaxBrand{}, Harian: []PaxHarian{}}
	idx := map[int64]int{}
	harian := map[string]*PaxHarian{}
	for rows.Next() {
		var (
			b                    PaxBrand
			tanggal              string
			periodeIni, dariAgen bool
			n                    int
		)
		if err := rows.Scan(&b.BrandID, &b.BrandName, &tanggal, &periodeIni, &dariAgen, &n); err != nil {
			return PaxChannel{}, fmt.Errorf("analitik: scan pax per channel: %w", err)
		}
		if !periodeIni {
			if dariAgen {
				pc.AgenSebelumnya += n
			} else {
				pc.TanpaAgenSebelumnya += n
			}
			continue
		}
		i, ada := idx[b.BrandID]
		if !ada {
			i = len(pc.PerBrand)
			idx[b.BrandID] = i
			pc.PerBrand = append(pc.PerBrand, PaxBrand{BrandID: b.BrandID, BrandName: b.BrandName})
		}
		h := harian[tanggal]
		if h == nil {
			h = &PaxHarian{Tanggal: tanggal}
			harian[tanggal] = h
		}
		if dariAgen {
			pc.Agen += n
			pc.PerBrand[i].Agen += n
			h.Agen += n
		} else {
			pc.TanpaAgen += n
			pc.PerBrand[i].TanpaAgen += n
			h.TanpaAgen += n
		}
	}
	if err := rows.Err(); err != nil {
		return PaxChannel{}, err
	}
	for _, h := range harian {
		pc.Harian = append(pc.Harian, *h)
	}
	sort.Slice(pc.Harian, func(i, j int) bool { return pc.Harian[i].Tanggal < pc.Harian[j].Tanggal })
	return pc, nil
}

func seatTerisi(ctx context.Context, q querier, brandID *int64) ([]SeatJadwal, error) {
	query := `
		SELECT s.id, COALESCE(NULLIF(s.jadwal_nama, ''), CONCAT('Jadwal #', s.id)), s.brand_id, br.name,
		       DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d'), DATEDIFF(s.berangkat_tanggal, CURDATE()),
		       s.seat_total, s.seat_sisa,
		       (SELECT COUNT(*) FROM booking_pax bp JOIN bookings b ON b.id = bp.booking_id
		         WHERE b.schedule_id = s.id AND b.status IN ('dp', 'lunas')
		           AND bp.pax_status = 'aktif' AND bp.pax_type = 'reguler')
		FROM schedules s
		JOIN brands br ON br.id = s.brand_id
		WHERE s.status = 'published'
		  AND s.berangkat_tanggal BETWEEN CURDATE() + INTERVAL ? DAY AND CURDATE() + INTERVAL ? DAY`
	args := []any{batasJualHari, jendelaSeatHari}
	if brandID != nil {
		query += ` AND s.brand_id = ?`
		args = append(args, *brandID)
	}
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("analitik: seat terisi: %w", err)
	}
	defer rows.Close()

	items := []SeatJadwal{}
	for rows.Next() {
		var (
			s        SeatJadwal
			seatSisa int
		)
		if err := rows.Scan(&s.ScheduleID, &s.Nama, &s.BrandID, &s.BrandName, &s.BerangkatTanggal, &s.SisaHari,
			&s.SeatTotal, &seatSisa, &s.Terisi); err != nil {
			return nil, fmt.Errorf("analitik: scan seat terisi: %w", err)
		}
		s.Hold = max(0, s.SeatTotal-seatSisa-s.Terisi)
		if s.SeatTotal > 0 {
			s.Persen = s.Terisi * 100 / s.SeatTotal
		}
		s.KurangSeat = kurangSeat(s.SeatTotal, s.Terisi)
		s.Tanda = tandaSeat(s.SisaHari, s.Persen)
		items = append(items, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	urutkanSeat(items)
	return items, nil
}

// kurangSeat: seat DP+ yang masih dibutuhkan supaya terisi mencapai targetPersen (pembulatan ke atas).
func kurangSeat(seatTotal, terisi int) int {
	target := (seatTotal*targetPersen + 99) / 100
	return max(0, target-terisi)
}

// tandaSeat memberi tanda jadwal yang butuh dorongan marketing.
func tandaSeat(sisaHari, persen int) string {
	switch {
	case sisaHari <= batasMerahHari && persen < ambangMerahPersen:
		return TandaMerah
	case sisaHari > batasMerahHari && sisaHari <= jendelaSeatHari && persen < ambangKuningPersen:
		return TandaKuning
	}
	return ""
}

// urutkanSeat: merah dulu, lalu kuning. Di dalam merah dan kuning, kekurangan seat terbanyak di atas;
// jadwal tanpa tanda urut tanggal berangkat terdekat.
func urutkanSeat(items []SeatJadwal) {
	bobot := map[string]int{TandaMerah: 0, TandaKuning: 1, "": 2}
	sort.SliceStable(items, func(i, j int) bool {
		if bobot[items[i].Tanda] != bobot[items[j].Tanda] {
			return bobot[items[i].Tanda] < bobot[items[j].Tanda]
		}
		if items[i].Tanda != "" && items[i].KurangSeat != items[j].KurangSeat {
			return items[i].KurangSeat > items[j].KurangSeat
		}
		if items[i].BerangkatTanggal != items[j].BerangkatTanggal {
			return items[i].BerangkatTanggal < items[j].BerangkatTanggal
		}
		return items[i].ScheduleID < items[j].ScheduleID
	})
}

// pilihAgenTerbaik: agen dengan pax closing > 0, urut pax closing, lalu total komisi, lalu nama.
func pilihAgenTerbaik(peringkat []agen.PeringkatItem) []AgenTerbaik {
	daftar := make([]agen.PeringkatItem, 0, len(peringkat))
	for _, p := range peringkat {
		if p.PaxClosing > 0 {
			daftar = append(daftar, p)
		}
	}
	sort.SliceStable(daftar, func(i, j int) bool {
		if daftar[i].PaxClosing != daftar[j].PaxClosing {
			return daftar[i].PaxClosing > daftar[j].PaxClosing
		}
		if daftar[i].TotalKomisi != daftar[j].TotalKomisi {
			return daftar[i].TotalKomisi > daftar[j].TotalKomisi
		}
		return daftar[i].NamaLengkap < daftar[j].NamaLengkap
	})
	if len(daftar) > jumlahAgenTerbaik {
		daftar = daftar[:jumlahAgenTerbaik]
	}
	hasil := make([]AgenTerbaik, 0, len(daftar))
	for i, p := range daftar {
		hasil = append(hasil, AgenTerbaik{
			Peringkat: i + 1, AkunID: p.AkunID, NamaLengkap: p.NamaLengkap, KodeReferral: p.KodeReferral,
			BrandID: p.BrandID, BrandName: p.BrandName, StatusAgen: p.StatusAgen, PaxClosing: p.PaxClosing,
		})
	}
	return hasil
}
