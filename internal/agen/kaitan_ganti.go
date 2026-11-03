package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Override admin (screen C4, spek 4.4): hanya Admin Master. Memindahkan kunci jamaah ke agen lain (atau tanpa
// agen) untuk semua kaitan_sumber, dan memasangkan atau mengganti agen sebuah akun (kaitan_akun.go). Bila
// komisi atas jamaah sudah tercatat, admin harus memilih pindahkan_komisi: baris ledger dipindah ke agen baru
// (langsung) dan upline-nya (pembinaan), sebagai ganti penolakan KA-01. Tidak cascading ke jamaah lain.

var (
	ErrKaitanSudahKomisi        = errors.New("jamaah ini sudah punya komisi tercatat; pilih pindahkan komisi agar komisi ikut ke agen baru")
	ErrKaitanTanpaAgenAdaKomisi = errors.New("jamaah yang sudah punya komisi tidak bisa dijadikan tanpa agen; pilih agen tujuan")
	ErrKomisiSudahDicairkan     = errors.New("pemindahan komisi membuat saldo agen lama negatif karena sudah dicairkan, selesaikan pencairannya dulu")
	ErrKaitanTidakBerubah       = errors.New("kaitan baru sama dengan kaitan saat ini")
	ErrKaitanModeTidakSah       = errors.New("pilih kaitkan ke agen atau tanpa agen")
)

type JamaahKaitan struct {
	JamaahID     int64   `json:"jamaah_id"`
	IDJamaah     string  `json:"id_jamaah"`
	NamaLengkap  string  `json:"nama_lengkap"`
	NoHP         *string `json:"no_hp"`
	BrandID      int64   `json:"brand_id"`
	BrandName    string  `json:"brand_name"`
	KaitanStatus string  `json:"kaitan_status"`
	KaitanSumber *string `json:"kaitan_sumber"`
	AgenID       *int64  `json:"agen_id"`
	AgenNama     *string `json:"agen_nama"`
	PunyaKomisi  bool    `json:"punya_komisi"`
	// BookingBelumLunas: booking aktif (draft/baru/dp) jamaah ini. Komisinya
	// dicatat saat lunas ke agen yang terkait saat itu, jadi ikut agen baru.
	BookingBelumLunas int `json:"booking_belum_lunas"`
}

type GantiKaitanRequest struct {
	Mode       string `json:"mode"` // 'agen' | 'tanpa_agen'
	AgenAkunID *int64 `json:"agen_akun_id"`
	Alasan     string `json:"alasan"`
	// PindahkanKomisi wajib true bila komisi atas jamaah ini sudah tercatat (mode agen saja).
	PindahkanKomisi bool `json:"pindahkan_komisi"`
}

type KaitanLog struct {
	ID             int64     `json:"id"`
	AgenLamaNama   *string   `json:"agen_lama_nama"`
	AgenBaruNama   *string   `json:"agen_baru_nama"`
	StatusLama     string    `json:"kaitan_status_lama"`
	StatusBaru     string    `json:"kaitan_status_baru"`
	DigantiOleh    string    `json:"diganti_oleh"`
	DigantiAt      time.Time `json:"diganti_at"`
	Alasan         string    `json:"alasan"`
	KomisiDipindah int       `json:"komisi_dipindah"`
	KomisiDihapus  int       `json:"komisi_dihapus"`
}

const qJamaahKaitan = `
	SELECT j.id, COALESCE(j.id_jamaah,''), j.nama_lengkap, j.no_hp, j.brand_id, b.name,
	       j.kaitan_status, j.kaitan_sumber, j.direkrut_oleh_akun_id, ag.nama_lengkap,
	       EXISTS (SELECT 1 FROM transaksi_komisi tk WHERE tk.jamaah_sumber_id = j.id),
	       (SELECT COUNT(DISTINCT bp.booking_id) FROM booking_pax bp JOIN bookings bk ON bk.id = bp.booking_id
	         WHERE bp.jamaah_id = j.id AND bp.pax_status = 'aktif' AND bk.status IN ('draft','baru','dp'))
	FROM jamaah j
	JOIN brands b ON b.id = j.brand_id
	LEFT JOIN customer_accounts ag ON ag.id = j.direkrut_oleh_akun_id`

func scanJamaahKaitan(sc interface{ Scan(...any) error }) (JamaahKaitan, error) {
	var j JamaahKaitan
	var agen sql.NullInt64
	err := sc.Scan(&j.JamaahID, &j.IDJamaah, &j.NamaLengkap, &j.NoHP, &j.BrandID, &j.BrandName,
		&j.KaitanStatus, &j.KaitanSumber, &agen, &j.AgenNama, &j.PunyaKomisi, &j.BookingBelumLunas)
	if err != nil {
		return j, err
	}
	if agen.Valid {
		j.AgenID = &agen.Int64
	}
	return j, nil
}

func cariJamaahKaitan(ctx context.Context, q querier, brandID *int64, cari string) ([]JamaahKaitan, error) {
	cari = strings.TrimSpace(cari)
	if len(cari) < 3 {
		return []JamaahKaitan{}, nil
	}
	query := qJamaahKaitan + ` WHERE (j.nama_lengkap LIKE ? OR j.no_hp LIKE ? OR j.id_jamaah LIKE ?)`
	like := "%" + cari + "%"
	args := []any{like, like, like}
	if brandID != nil {
		query += ` AND j.brand_id = ?`
		args = append(args, *brandID)
	}
	query += ` ORDER BY j.nama_lengkap LIMIT 30`
	rows, err := q.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("agen: cari jamaah kaitan: %w", err)
	}
	defer rows.Close()
	items := []JamaahKaitan{}
	for rows.Next() {
		j, err := scanJamaahKaitan(rows)
		if err != nil {
			return nil, fmt.Errorf("agen: scan jamaah kaitan: %w", err)
		}
		items = append(items, j)
	}
	return items, rows.Err()
}

func gantiKaitanTx(ctx context.Context, tx *sql.Tx, jamaahID, adminID int64, req GantiKaitanRequest) error {
	// FOR UPDATE OF j: kunci hanya baris jamaah target, bukan baris brands
	// (kunci counter nomor jamaah) atau baris agen lama (audit KA-02).
	j, err := scanJamaahKaitan(tx.QueryRowContext(ctx, qJamaahKaitan+` WHERE j.id = ? FOR UPDATE OF j`, jamaahID))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("agen: kunci jamaah kaitan: %w", err)
	}
	// Komisi dibaca ulang di dalam transaksi (scanJamaahKaitan): bisa tercatat setelah halaman dibuka.

	var agenBaru any
	statusBaru := "tanpa_agen"
	switch req.Mode {
	case "agen":
		if req.AgenAkunID == nil {
			return ErrAgenTidakValid
		}
		if err := ValidasiAgenAktif(ctx, tx, j.BrandID, *req.AgenAkunID); err != nil {
			return err
		}
		agenBaru = *req.AgenAkunID
		statusBaru = "terikat_agen"
		if j.AgenID != nil && *j.AgenID == *req.AgenAkunID {
			return ErrKaitanTidakBerubah
		}
	case "tanpa_agen":
		if j.KaitanStatus == "tanpa_agen" {
			return ErrKaitanTidakBerubah
		}
		if j.PunyaKomisi {
			return ErrKaitanTanpaAgenAdaKomisi
		}
	default:
		return ErrKaitanModeTidakSah
	}

	var dipindah, dihapus int
	if j.PunyaKomisi {
		if !req.PindahkanKomisi {
			return ErrKaitanSudahKomisi
		}
		if dipindah, dihapus, err = pindahkanKomisiTx(ctx, tx, jamaahID, *req.AgenAkunID); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE jamaah SET direkrut_oleh_akun_id = ?, kaitan_status = ?, kaitan_sumber = 'jalur3' WHERE id = ?`, agenBaru, statusBaru, jamaahID); err != nil {
		return fmt.Errorf("agen: ganti kaitan: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO jamaah_kaitan_log (jamaah_id, direkrut_oleh_akun_id_lama, direkrut_oleh_akun_id_baru,
		                               kaitan_status_lama, kaitan_status_baru, diganti_oleh, alasan, komisi_dipindah, komisi_dihapus)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, jamaahID, j.AgenID, agenBaru, j.KaitanStatus, statusBaru, adminID, req.Alasan, dipindah, dihapus); err != nil {
		return fmt.Errorf("agen: log kaitan: %w", err)
	}
	return nil
}

func listKaitanLog(ctx context.Context, q querier, jamaahID int64) ([]KaitanLog, error) {
	var ada bool
	if err := q.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM jamaah WHERE id = ?)`, jamaahID).Scan(&ada); err != nil {
		return nil, fmt.Errorf("agen: cek jamaah log kaitan: %w", err)
	}
	if !ada {
		return nil, ErrNotFound
	}
	rows, err := q.QueryContext(ctx, `
		SELECT l.id, lama.nama_lengkap, baru.nama_lengkap, l.kaitan_status_lama, l.kaitan_status_baru,
		       COALESCE(au.display_name, au.email), l.diganti_at, l.alasan, l.komisi_dipindah, l.komisi_dihapus
		FROM jamaah_kaitan_log l
		LEFT JOIN customer_accounts lama ON lama.id = l.direkrut_oleh_akun_id_lama
		LEFT JOIN customer_accounts baru ON baru.id = l.direkrut_oleh_akun_id_baru
		JOIN admin_users au ON au.id = l.diganti_oleh
		WHERE l.jamaah_id = ? ORDER BY l.id DESC`, jamaahID)
	if err != nil {
		return nil, fmt.Errorf("agen: log kaitan: %w", err)
	}
	defer rows.Close()
	items := []KaitanLog{}
	for rows.Next() {
		var l KaitanLog
		if err := rows.Scan(&l.ID, &l.AgenLamaNama, &l.AgenBaruNama, &l.StatusLama, &l.StatusBaru, &l.DigantiOleh, &l.DigantiAt, &l.Alasan, &l.KomisiDipindah, &l.KomisiDihapus); err != nil {
			return nil, fmt.Errorf("agen: scan log kaitan: %w", err)
		}
		items = append(items, l)
	}
	return items, rows.Err()
}

func (r *Repository) CariJamaahKaitan(ctx context.Context, brandID *int64, cari string) ([]JamaahKaitan, error) {
	return cariJamaahKaitan(ctx, r.db, brandID, cari)
}

func (r *Repository) GantiKaitan(ctx context.Context, jamaahID, adminID int64, req GantiKaitanRequest) error {
	return r.withTx(ctx, func(tx *sql.Tx) error { return gantiKaitanTx(ctx, tx, jamaahID, adminID, req) })
}

func (r *Repository) ListKaitanLog(ctx context.Context, jamaahID int64) ([]KaitanLog, error) {
	return listKaitanLog(ctx, r.db, jamaahID)
}
