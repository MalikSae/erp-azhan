package komisi

import (
	"context"
	"database/sql"
	"fmt"
)

// Ringkasan agen dan komisi satu booking untuk Detail Booking admin (keputusan 2026-11-24).
//   - Agen per pax: sebelum komisi tercatat, penerima komisi langsung menurut kaitan saat ini (aturan loadPax);
//     sesudahnya, penerima yang tercatat. UplineBerubah bila keduanya berbeda.
//   - Komisi: baris transaksi_komisi booking ini (Tercatat / Batal). Belum ada baris dan booking belum lunas maupun
//     batal: estimasi dari snapshot nominal booking.
//   - Nominal hanya diisi bila lihatNominal (izin komisi.view).

const (
	StatusKomisiEstimasi = "estimasi"
	StatusKomisiTercatat = "tercatat"
	StatusKomisiBatal    = "batal"
)

type AgenRingkas struct {
	ID           int64   `json:"id"`
	NamaLengkap  string  `json:"nama_lengkap"`
	KodeReferral *string `json:"kode_referral"`
}

type PaxAgen struct {
	BookingPaxID  int64        `json:"booking_pax_id"`
	JamaahID      int64        `json:"jamaah_id"`
	KaitanStatus  string       `json:"kaitan_status"`
	Agen          *AgenRingkas `json:"agen"`
	UplineBerubah bool         `json:"upline_berubah"`
	// DikaitkanAdminAt: waktu terakhir admin mengubah kaitan jamaah ini ("YYYY-MM-DD HH:MM").
	DikaitkanAdminAt *string `json:"dikaitkan_admin_at"`
}

type RincianKomisi struct {
	Penerima   AgenRingkas `json:"penerima"`
	NamaJamaah string      `json:"nama_jamaah"`
	Jenis      string      `json:"jenis"`
	Status     string      `json:"status"`
	Nominal    *float64    `json:"nominal"`
}

type BookingAgenKomisi struct {
	DibuatOleh   *AgenRingkas    `json:"dibuat_oleh"`
	Klaim        *AgenRingkas    `json:"klaim"`
	Pax          []PaxAgen       `json:"pax"`
	LihatNominal bool            `json:"lihat_nominal"`
	Total        *float64        `json:"total"`
	Komisi       []RincianKomisi `json:"komisi"`
}

func agenDari(id sql.NullInt64, nama sql.NullString, kode sql.NullString) *AgenRingkas {
	if !id.Valid {
		return nil
	}
	a := &AgenRingkas{ID: id.Int64, NamaLengkap: nama.String}
	if kode.Valid {
		a.KodeReferral = &kode.String
	}
	return a
}

// penerimaLangsungSQL: penerima komisi langsung menurut kaitan saat ini, sama dengan loadPax.
const penerimaLangsungSQL = `CASE
	WHEN diri.id IS NULL THEN IF(j.kaitan_status = 'tanpa_agen', NULL, j.direkrut_oleh_akun_id)
	WHEN j.kaitan_status = 'terikat_agen' AND j.direkrut_oleh_akun_id <> diri.id THEN j.direkrut_oleh_akun_id
	WHEN diri.status_agen IN ('aktif','nonaktif') THEN diri.upline_akun_id
	ELSE NULL END`

// RingkasanBooking membaca ringkasan agen dan komisi. sql.ErrNoRows bila booking tidak ada.
func RingkasanBooking(ctx context.Context, q querier, bookingID int64, lihatNominal bool) (*BookingAgenKomisi, error) {
	out := &BookingAgenKomisi{Pax: []PaxAgen{}, Komisi: []RincianKomisi{}, LihatNominal: lihatNominal}

	var status string
	var jadwalID int64
	var pertamaLunas sql.NullTime
	var langsung, pembinaan sql.NullFloat64
	var dibuatID, klaimID sql.NullInt64
	var dibuatNama, dibuatKode, klaimNama, klaimKode sql.NullString
	err := q.QueryRowContext(ctx, `
		SELECT b.status, b.schedule_id, b.pertama_lunas_at, b.snapshot_komisi_langsung, b.snapshot_bonus_pembinaan,
		       da.id, da.nama_lengkap, da.kode_referral, ka.id, ka.nama_lengkap, ka.kode_referral
		FROM bookings b
		LEFT JOIN customer_accounts da ON da.id = b.dibuat_oleh_agen_akun_id
		LEFT JOIN customer_accounts ka ON ka.id = b.agen_klaim_akun_id
		WHERE b.id = ?`, bookingID).Scan(&status, &jadwalID, &pertamaLunas, &langsung, &pembinaan,
		&dibuatID, &dibuatNama, &dibuatKode, &klaimID, &klaimNama, &klaimKode)
	if err != nil {
		return nil, err
	}
	out.DibuatOleh = agenDari(dibuatID, dibuatNama, dibuatKode)

	// Komisi tercatat (aktif maupun batal).
	type tercatat struct {
		paxID   int64
		jenis   string
		rincian RincianKomisi
	}
	var rows []tercatat
	rs, err := q.QueryContext(ctx, `
		SELECT tk.booking_pax_id, tk.jenis, tk.nominal, tk.dibatalkan_at IS NOT NULL,
		       a.id, a.nama_lengkap, a.kode_referral, j.nama_lengkap
		FROM transaksi_komisi tk
		JOIN customer_accounts a ON a.id = tk.akun_penerima_id
		JOIN jamaah j ON j.id = tk.jamaah_sumber_id
		WHERE tk.booking_id = ?
		ORDER BY tk.booking_pax_id, FIELD(tk.jenis, 'langsung', 'pembinaan', 'repeat_order', 'cashback'), tk.id`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("komisi: ringkasan tercatat: %w", err)
	}
	for rs.Next() {
		var t tercatat
		var nominal float64
		var batal bool
		var id sql.NullInt64
		var nama, kode sql.NullString
		if err := rs.Scan(&t.paxID, &t.jenis, &nominal, &batal, &id, &nama, &kode, &t.rincian.NamaJamaah); err != nil {
			rs.Close()
			return nil, fmt.Errorf("komisi: scan ringkasan tercatat: %w", err)
		}
		t.rincian.Penerima = *agenDari(id, nama, kode)
		t.rincian.Jenis = t.jenis
		t.rincian.Status = StatusKomisiTercatat
		if batal {
			t.rincian.Status = StatusKomisiBatal
		}
		n := nominal
		t.rincian.Nominal = &n
		rows = append(rows, t)
	}
	rs.Close()
	if err := rs.Err(); err != nil {
		return nil, err
	}

	// Penerima tercatat per pax: komisi langsung yang masih aktif, atau yang terakhir bila semuanya batal.
	tercatatPax := map[int64]AgenRingkas{}
	for _, t := range rows {
		if t.jenis != JenisLangsung && t.jenis != JenisRepeatOrder {
			continue
		}
		if _, ada := tercatatPax[t.paxID]; !ada || t.rincian.Status == StatusKomisiTercatat {
			tercatatPax[t.paxID] = t.rincian.Penerima
		}
	}

	// Pax beserta penerima menurut kaitan saat ini dan upline-nya (untuk estimasi bonus pembinaan).
	type paxSaatIni struct {
		PaxAgen
		nama        string
		infant      bool
		aktif       bool
		agenAktif   bool
		upline      *AgenRingkas
		uplineAktif bool
	}
	var paxes []paxSaatIni
	rs, err = q.QueryContext(ctx, `
		SELECT bp.id, bp.jamaah_id, j.nama_lengkap, j.kaitan_status, bp.pax_type = 'infant', bp.pax_status = 'aktif',
		       a.id, a.nama_lengkap, a.kode_referral, COALESCE(a.status_agen = 'aktif', FALSE),
		       u.id, u.nama_lengkap, u.kode_referral, COALESCE(u.status_agen = 'aktif', FALSE),
		       (SELECT MAX(l.diganti_at) FROM jamaah_kaitan_log l WHERE l.jamaah_id = j.id
		          AND l.kaitan_status_lama = 'tanpa_agen' AND l.kaitan_status_baru = 'terikat_agen')
		FROM booking_pax bp
		JOIN jamaah j ON j.id = bp.jamaah_id
		LEFT JOIN customer_accounts diri ON diri.id = COALESCE(bp.saya_sendiri_akun_id,
		     (SELECT ca.id FROM customer_accounts ca WHERE ca.jamaah_id = j.id LIMIT 1))
		LEFT JOIN customer_accounts a ON a.id = `+penerimaLangsungSQL+`
		LEFT JOIN customer_accounts u ON u.id = a.upline_akun_id
		WHERE bp.booking_id = ?
		ORDER BY bp.id`, bookingID)
	if err != nil {
		return nil, fmt.Errorf("komisi: ringkasan pax: %w", err)
	}
	for rs.Next() {
		var p paxSaatIni
		var aID, uID sql.NullInt64
		var aNama, aKode, uNama, uKode sql.NullString
		var dikaitkan sql.NullTime
		if err := rs.Scan(&p.BookingPaxID, &p.JamaahID, &p.nama, &p.KaitanStatus, &p.infant, &p.aktif,
			&aID, &aNama, &aKode, &p.agenAktif, &uID, &uNama, &uKode, &p.uplineAktif, &dikaitkan); err != nil {
			rs.Close()
			return nil, fmt.Errorf("komisi: scan ringkasan pax: %w", err)
		}
		p.Agen = agenDari(aID, aNama, aKode)
		p.upline = agenDari(uID, uNama, uKode)
		if dikaitkan.Valid && p.KaitanStatus == "terikat_agen" {
			s := dikaitkan.Time.Format("2006-01-02 15:04")
			p.DikaitkanAdminAt = &s
		}
		paxes = append(paxes, p)
	}
	rs.Close()
	if err := rs.Err(); err != nil {
		return nil, err
	}

	adaTanpaAgen := false
	for _, p := range paxes {
		pa := p.PaxAgen
		if rec, ada := tercatatPax[p.BookingPaxID]; ada {
			pa.UplineBerubah = p.Agen == nil || p.Agen.ID != rec.ID
			r := rec
			pa.Agen = &r
		}
		// Pax diri agen tanpa_agen sudah punya penerima (upline agen), bukan bahan klaim.
		if p.aktif && p.KaitanStatus == "tanpa_agen" && pa.Agen == nil {
			adaTanpaAgen = true
		}
		out.Pax = append(out.Pax, pa)
	}
	if adaTanpaAgen {
		out.Klaim = agenDari(klaimID, klaimNama, klaimKode)
	}

	for _, t := range rows {
		out.Komisi = append(out.Komisi, t.rincian)
	}

	// Estimasi untuk booking yang masih berjalan, per pax dan jenis, mengikuti aturan ProcessBookingLunas: pax yang
	// komisinya sudah aktif tercatat dilewati, pengaitan admin setelah lunas pertama tidak berkomisi, dan satu komisi
	// per jamaah per keberangkatan.
	if status == "baru" || status == "dp" {
		aktif := map[[2]any]bool{}
		for _, t := range rows {
			if t.rincian.Status == StatusKomisiTercatat {
				aktif[[2]any{t.paxID, t.jenis}] = true
			}
		}
		tambah := func(p paxSaatIni, penerima *AgenRingkas, jenis string, nominal float64) error {
			if aktif[[2]any{p.BookingPaxID, jenis}] {
				return nil
			}
			ada, err := sudahDibayarKeberangkatan(ctx, q, p.JamaahID, jadwalID, bookingID, jenis)
			if err != nil || ada {
				return err
			}
			n := nominal
			out.Komisi = append(out.Komisi, RincianKomisi{Penerima: *penerima, NamaJamaah: p.nama,
				Jenis: jenis, Status: StatusKomisiEstimasi, Nominal: &n})
			return nil
		}
		for _, p := range paxes {
			if !p.aktif || p.infant || p.Agen == nil {
				continue
			}
			if pertamaLunas.Valid {
				setelah, err := dikaitkanSetelah(ctx, q, p.JamaahID, pertamaLunas.Time)
				if err != nil {
					return nil, err
				}
				if setelah {
					continue
				}
			}
			if p.agenAktif && positive(langsung) {
				if err := tambah(p, p.Agen, JenisLangsung, langsung.Float64); err != nil {
					return nil, err
				}
			}
			if p.upline != nil && p.uplineAktif && positive(pembinaan) {
				if err := tambah(p, p.upline, JenisPembinaan, pembinaan.Float64); err != nil {
					return nil, err
				}
			}
		}
	}

	var total float64
	for i := range out.Komisi {
		if out.Komisi[i].Status != StatusKomisiBatal && out.Komisi[i].Nominal != nil {
			total += *out.Komisi[i].Nominal
		}
		if !lihatNominal {
			out.Komisi[i].Nominal = nil
		}
	}
	if lihatNominal {
		out.Total = &total
	}
	return out, nil
}
