package agen

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Ganti Kaitan Agen (screen C4, agen-azhan.md 3.5, 7.8, §5 poin 17): hanya
// Admin Master, hanya kaitan hasil Jalur 3, dan hanya selama jamaah belum
// pernah menghasilkan komisi. Tidak cascading ke jamaah lain dan tidak
// menghitung ulang komisi apa pun.

var (
	ErrKaitanBukanJalur3  = errors.New("kaitan jamaah ini bukan hasil input Admin (Jalur 3), tidak bisa diganti")
	ErrKaitanSudahKomisi  = errors.New("jamaah ini sudah pernah menghasilkan komisi (booking sudah pernah lunas), kaitan tidak bisa diganti")
	ErrKaitanJamaahAgen   = errors.New("jamaah ini sudah menjadi atau sedang mengajukan diri sebagai agen, kaitan tidak bisa diganti")
	ErrKaitanTidakBerubah = errors.New("kaitan baru sama dengan kaitan saat ini")
	ErrKaitanModeTidakSah = errors.New("pilih kaitkan ke agen atau tanpa agen")
)

type JamaahKaitan struct {
	JamaahID       int64   `json:"jamaah_id"`
	IDJamaah       string  `json:"id_jamaah"`
	NamaLengkap    string  `json:"nama_lengkap"`
	NoHP           *string `json:"no_hp"`
	BrandID        int64   `json:"brand_id"`
	BrandName      string  `json:"brand_name"`
	StatusAgen     string  `json:"status_agen"`
	KaitanStatus   string  `json:"kaitan_status"`
	KaitanSumber   *string `json:"kaitan_sumber"`
	AgenID         *int64  `json:"agen_id"`
	AgenNama       *string `json:"agen_nama"`
	PunyaKomisi    bool    `json:"punya_komisi"`
	BisaDiganti    bool    `json:"bisa_diganti"`
	AlasanTerkunci string  `json:"alasan_terkunci,omitempty"`
}

type GantiKaitanRequest struct {
	Mode         string `json:"mode"` // 'agen' | 'tanpa_agen'
	AgenJamaahID *int64 `json:"agen_jamaah_id"`
	Alasan       string `json:"alasan"`
}

type KaitanLog struct {
	ID           int64     `json:"id"`
	AgenLamaNama *string   `json:"agen_lama_nama"`
	AgenBaruNama *string   `json:"agen_baru_nama"`
	StatusLama   string    `json:"kaitan_status_lama"`
	StatusBaru   string    `json:"kaitan_status_baru"`
	DigantiOleh  string    `json:"diganti_oleh"`
	DigantiAt    time.Time `json:"diganti_at"`
	Alasan       string    `json:"alasan"`
}

// alasanTerkunci mengembalikan error gate C4, atau nil bila boleh diganti.
func alasanTerkunci(j JamaahKaitan) error {
	switch {
	case j.KaitanSumber == nil || *j.KaitanSumber != SumberJalur3:
		return ErrKaitanBukanJalur3
	case j.StatusAgen != "tidak_aktif":
		return ErrKaitanJamaahAgen
	case j.PunyaKomisi:
		return ErrKaitanSudahKomisi
	}
	return nil
}

const qJamaahKaitan = `
	SELECT j.id, COALESCE(j.id_jamaah,''), j.nama_lengkap, j.no_hp, j.brand_id, b.name, j.status_agen,
	       j.kaitan_status, j.kaitan_sumber, j.direkrut_oleh_jamaah_id, ag.nama_lengkap,
	       EXISTS (SELECT 1 FROM transaksi_komisi tk WHERE tk.jamaah_sumber_id = j.id)
	FROM jamaah j
	JOIN brands b ON b.id = j.brand_id
	LEFT JOIN jamaah ag ON ag.id = j.direkrut_oleh_jamaah_id`

func scanJamaahKaitan(sc interface{ Scan(...any) error }) (JamaahKaitan, error) {
	var j JamaahKaitan
	var agen sql.NullInt64
	err := sc.Scan(&j.JamaahID, &j.IDJamaah, &j.NamaLengkap, &j.NoHP, &j.BrandID, &j.BrandName, &j.StatusAgen,
		&j.KaitanStatus, &j.KaitanSumber, &agen, &j.AgenNama, &j.PunyaKomisi)
	if err != nil {
		return j, err
	}
	if agen.Valid {
		j.AgenID = &agen.Int64
	}
	if e := alasanTerkunci(j); e != nil {
		j.AlasanTerkunci = e.Error()
	} else {
		j.BisaDiganti = true
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
	j, err := scanJamaahKaitan(tx.QueryRowContext(ctx, qJamaahKaitan+` WHERE j.id = ? FOR UPDATE`, jamaahID))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("agen: kunci jamaah kaitan: %w", err)
	}
	// Cek ulang di dalam transaksi: komisi bisa tercatat setelah halaman dibuka.
	if e := alasanTerkunci(j); e != nil {
		return e
	}

	var agenBaru any
	statusBaru := "tanpa_agen"
	switch req.Mode {
	case "agen":
		if req.AgenJamaahID == nil || *req.AgenJamaahID == jamaahID {
			return ErrAgenTidakValid
		}
		if err := ValidasiAgenAktif(ctx, tx, j.BrandID, *req.AgenJamaahID); err != nil {
			return err
		}
		agenBaru = *req.AgenJamaahID
		statusBaru = "terikat_agen"
		if j.AgenID != nil && *j.AgenID == *req.AgenJamaahID {
			return ErrKaitanTidakBerubah
		}
	case "tanpa_agen":
		if j.KaitanStatus == "tanpa_agen" {
			return ErrKaitanTidakBerubah
		}
	default:
		return ErrKaitanModeTidakSah
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE jamaah SET direkrut_oleh_jamaah_id = ?, kaitan_status = ? WHERE id = ?`, agenBaru, statusBaru, jamaahID); err != nil {
		return fmt.Errorf("agen: ganti kaitan: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO jamaah_kaitan_log (jamaah_id, direkrut_oleh_jamaah_id_lama, direkrut_oleh_jamaah_id_baru,
		                               kaitan_status_lama, kaitan_status_baru, diganti_oleh, alasan)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, jamaahID, j.AgenID, agenBaru, j.KaitanStatus, statusBaru, adminID, req.Alasan); err != nil {
		return fmt.Errorf("agen: log kaitan: %w", err)
	}
	return nil
}

func listKaitanLog(ctx context.Context, q querier, jamaahID int64) ([]KaitanLog, error) {
	rows, err := q.QueryContext(ctx, `
		SELECT l.id, lama.nama_lengkap, baru.nama_lengkap, l.kaitan_status_lama, l.kaitan_status_baru,
		       COALESCE(au.display_name, au.email), l.diganti_at, l.alasan
		FROM jamaah_kaitan_log l
		LEFT JOIN jamaah lama ON lama.id = l.direkrut_oleh_jamaah_id_lama
		LEFT JOIN jamaah baru ON baru.id = l.direkrut_oleh_jamaah_id_baru
		JOIN admin_users au ON au.id = l.diganti_oleh
		WHERE l.jamaah_id = ? ORDER BY l.id DESC`, jamaahID)
	if err != nil {
		return nil, fmt.Errorf("agen: log kaitan: %w", err)
	}
	defer rows.Close()
	items := []KaitanLog{}
	for rows.Next() {
		var l KaitanLog
		if err := rows.Scan(&l.ID, &l.AgenLamaNama, &l.AgenBaruNama, &l.StatusLama, &l.StatusBaru, &l.DigantiOleh, &l.DigantiAt, &l.Alasan); err != nil {
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
