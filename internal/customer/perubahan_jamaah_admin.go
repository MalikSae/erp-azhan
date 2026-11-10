package customer

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"erp-azhan/api/internal/identity"
	"erp-azhan/api/internal/mailer"
	"github.com/go-chi/chi/v5"
)

// Persetujuan admin atas perubahan data jamaah dari pelanggan. Izin: jamaah.perubahan.approve.

type perubahanBaris struct {
	ID          int64      `json:"id"`
	JamaahID    int64      `json:"jamaah_id"`
	JamaahNama  string     `json:"jamaah_nama"`
	AkunID      *int64     `json:"akun_id"`
	AkunNama    string     `json:"akun_nama"`
	Sumber      string     `json:"sumber"`
	Pelaku      string     `json:"pelaku"` // email admin bila sumber admin
	Status      string     `json:"status"`
	DibuatAt    time.Time  `json:"dibuat_at"`
	DiputusAt   *time.Time `json:"diputus_at"`
	DiputusOleh string     `json:"diputus_oleh"`
	Catatan     string     `json:"catatan"`
	JumlahItem  int        `json:"jumlah_item"`
}

type perubahanDetail struct {
	perubahanBaris
	Perubahan []perubahanItem `json:"perubahan"`
}

const perubahanSelect = `
	SELECT p.id, p.jamaah_id, j.nama_lengkap, p.akun_id, COALESCE(a.nama_lengkap,''), p.sumber, COALESCE(pa.email,''), p.status,
	       p.dibuat_at, p.diputus_at, COALESCE(pd.email,''), COALESCE(p.catatan,''),
	       (SELECT COUNT(*) FROM jamaah_perubahan_item i WHERE i.perubahan_id = p.id)
	FROM jamaah_perubahan p
	JOIN jamaah j ON j.id = p.jamaah_id
	LEFT JOIN customer_accounts a ON a.id = p.akun_id
	LEFT JOIN admin_users pa ON pa.id = p.admin_id
	LEFT JOIN admin_users pd ON pd.id = p.diputus_oleh
	WHERE 1=1`

func scanPerubahan(rows interface{ Scan(...any) error }) (*perubahanBaris, error) {
	var b perubahanBaris
	var akun sql.NullInt64
	var diputus sql.NullTime
	if err := rows.Scan(&b.ID, &b.JamaahID, &b.JamaahNama, &akun, &b.AkunNama, &b.Sumber, &b.Pelaku, &b.Status, &b.DibuatAt, &diputus,
		&b.DiputusOleh, &b.Catatan, &b.JumlahItem); err != nil {
		return nil, err
	}
	if akun.Valid {
		b.AkunID = &akun.Int64
	}
	if diputus.Valid {
		b.DiputusAt = &diputus.Time
	}
	return &b, nil
}

// ListAdmin GET /api/admin/perubahan-jamaah?status=menunggu|semua&jamaah_id=&brand_id=
func (h *JamaahAkun) ListAdmin(w http.ResponseWriter, r *http.Request) {
	qs := r.URL.Query()
	query := perubahanSelect
	var args []any
	if brand := adminBrandScope(r); brand != nil {
		query += ` AND j.brand_id = ?`
		args = append(args, *brand)
	}
	if qs.Get("status") != "semua" {
		query += ` AND p.status = 'menunggu'`
	}
	if id, err := strconv.ParseInt(qs.Get("jamaah_id"), 10, 64); err == nil && id > 0 {
		query += ` AND p.jamaah_id = ?`
		args = append(args, id)
	}
	query += ` ORDER BY p.id DESC LIMIT 200`
	rows, err := h.db.QueryContext(r.Context(), query, args...)
	if err != nil {
		log.Printf("[ERROR] admin list perubahan jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil daftar perubahan")
		return
	}
	defer rows.Close()
	out := []perubahanBaris{}
	for rows.Next() {
		b, err := scanPerubahan(rows)
		if err != nil {
			log.Printf("[ERROR] admin list perubahan jamaah scan: %v", err)
			writeError(w, http.StatusInternalServerError, "gagal mengambil daftar perubahan")
			return
		}
		out = append(out, *b)
	}
	writeJSON(w, http.StatusOK, out)
}

func idPerubahan(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "id perubahan tidak valid")
		return 0, false
	}
	return id, true
}

// DetailAdmin GET /api/admin/perubahan-jamaah/{id}. Nilai sensitif tampil utuh selama menunggu (admin perlu memeriksa),
// dan tersamar sesudah diputuskan.
func (h *JamaahAkun) DetailAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := idPerubahan(w, r)
	if !ok {
		return
	}
	query := perubahanSelect + ` AND p.id = ?`
	args := []any{id}
	if brand := adminBrandScope(r); brand != nil {
		query += ` AND j.brand_id = ?`
		args = append(args, *brand)
	}
	b, err := scanPerubahan(h.db.QueryRowContext(r.Context(), query, args...))
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "perubahan tidak ditemukan")
		return
	}
	if err != nil {
		log.Printf("[ERROR] admin detail perubahan jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil perubahan")
		return
	}
	items, err := muatItem(r.Context(), h.db, id, b.Status != "menunggu")
	if err != nil {
		log.Printf("[ERROR] admin detail perubahan jamaah item: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal mengambil perubahan")
		return
	}
	writeJSON(w, http.StatusOK, perubahanDetail{perubahanBaris: *b, Perubahan: items})
}

var (
	errPerubahanSelesai = errors.New("pengajuan ini sudah diputuskan atau digantikan")
	errCatatanWajib     = errors.New("catatan penolakan wajib diisi (minimal 3 karakter)")
)

// errDataBerubah: isian jamaah sudah berbeda dari nilai "sebelumnya" yang tercatat pada pengajuan.
type errDataBerubah struct{ Label string }

func (e errDataBerubah) Error() string {
	return "data jamaah sudah berubah sejak pengajuan ini dibuat (" + e.Label + "); pengajuan tidak dapat disetujui. Tolak pengajuan ini dan minta pelanggan mengajukan ulang bila masih perlu"
}

type putusRequest struct {
	Catatan string `json:"catatan"`
}

type pengajuanTerkunci struct {
	JamaahID int64
	AkunID   int64
}

// kunciPengajuan mengunci baris pengajuan yang masih menunggu (dalam brand scope admin).
func kunciPengajuan(ctx context.Context, tx *sql.Tx, id int64, brand *int64) (*pengajuanTerkunci, error) {
	var p pengajuanTerkunci
	var akun sql.NullInt64
	var status string
	query := `SELECT p.jamaah_id, p.akun_id, p.status FROM jamaah_perubahan p JOIN jamaah j ON j.id = p.jamaah_id WHERE p.id = ?`
	args := []any{id}
	if brand != nil {
		query += ` AND j.brand_id = ?`
		args = append(args, *brand)
	}
	if err := tx.QueryRowContext(ctx, query+` FOR UPDATE`, args...).Scan(&p.JamaahID, &akun, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if status != "menunggu" {
		return nil, errPerubahanSelesai
	}
	p.AkunID = akun.Int64
	return &p, nil
}

func tulisGalatPutus(w http.ResponseWriter, err error) {
	var val *ErrValidasi
	var hp errBentrokHP
	var berubah errDataBerubah
	switch {
	case errors.As(err, &berubah):
		writeError(w, http.StatusConflict, berubah.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "perubahan tidak ditemukan")
	case errors.Is(err, errPerubahanSelesai):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, errCatatanWajib):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.As(err, &hp):
		writeError(w, http.StatusConflict, hp.Error()+"; pengajuan tidak dapat disetujui")
	case errors.As(err, &val):
		writeError(w, http.StatusBadRequest, val.Pesan)
	default:
		log.Printf("[ERROR] putus perubahan jamaah: %v", err)
		writeError(w, http.StatusInternalServerError, "gagal memproses pengajuan")
	}
}

// setujuiTx menerapkan semua isian pengajuan ke seluruh baris jamaah orang itu lalu menandainya disetujui.
func setujuiTx(ctx context.Context, tx *sql.Tx, id, adminID int64, brand *int64) (*pengajuanTerkunci, error) {
	p, err := kunciPengajuan(ctx, tx, id, brand)
	if err != nil {
		return nil, err
	}
	ids := []int64{p.JamaahID}
	if grup, err := kelompokDari(ctx, tx, p.AkunID, p.JamaahID); err == nil {
		ids = idGrup(grup)
	}
	items, err := muatItem(ctx, tx, id, false)
	if err != nil {
		return nil, err
	}
	// Nilai "sebelumnya" pada pengajuan harus masih sama dengan data sekarang. Bila data sudah diubah
	// sesudah pengajuan dibuat (mis. diedit admin), menyetujui akan menimpa perubahan yang lebih baru.
	if _, err := tx.ExecContext(ctx, `SELECT id FROM jamaah WHERE id = ? FOR UPDATE`, p.JamaahID); err != nil {
		return nil, err
	}
	sekarang, err := muatDataOrang(ctx, tx, ids)
	if err != nil {
		return nil, err
	}
	for _, it := range items {
		if k, ok := kolomIndeks[it.Kolom]; ok && sekarang.Nilai[it.Kolom] != it.Lama {
			return nil, errDataBerubah{Label: k.Label}
		}
	}
	for _, it := range items {
		k, ok := kolomIndeks[it.Kolom]
		if !ok {
			continue
		}
		if err := terapkanKolom(ctx, tx, ids, k, it.Baru, false); err != nil {
			return nil, err
		}
	}
	_, err = tx.ExecContext(ctx, `UPDATE jamaah_perubahan SET status='disetujui', diputus_oleh=?, diputus_at=NOW() WHERE id=?`, adminID, id)
	return p, err
}

// tolakTx menandai pengajuan ditolak beserta alasannya; data jamaah tidak berubah.
func tolakTx(ctx context.Context, tx *sql.Tx, id, adminID int64, brand *int64, catatan string) (*pengajuanTerkunci, error) {
	p, err := kunciPengajuan(ctx, tx, id, brand)
	if err != nil {
		return nil, err
	}
	_, err = tx.ExecContext(ctx, `UPDATE jamaah_perubahan SET status='ditolak', diputus_oleh=?, diputus_at=NOW(), catatan=? WHERE id=?`, adminID, catatan, id)
	return p, err
}

// Setujui POST /api/admin/perubahan-jamaah/{id}/setujui: menerapkan semua isian ke seluruh baris jamaah orang itu.
func (h *JamaahAkun) Setujui(w http.ResponseWriter, r *http.Request) {
	id, ok := idPerubahan(w, r)
	if !ok {
		return
	}
	adminID := identity.GetAdminUserID(r.Context())
	var p *pengajuanTerkunci
	err := dalamTx(r.Context(), h.db, func(tx *sql.Tx) error {
		var err error
		p, err = setujuiTx(r.Context(), tx, id, adminID, adminBrandScope(r))
		return err
	})
	if err != nil {
		tulisGalatPutus(w, err)
		return
	}
	log.Printf("[AUDIT] perubahan_jamaah_disetujui id=%d jamaah_id=%d admin_id=%d", id, p.JamaahID, adminID)
	h.kabari(p.AkunID, p.JamaahID, true, "")
	writeJSON(w, http.StatusOK, map[string]string{"message": "perubahan disetujui dan diterapkan"})
}

// Tolak POST /api/admin/perubahan-jamaah/{id}/tolak {catatan}
func (h *JamaahAkun) Tolak(w http.ResponseWriter, r *http.Request) {
	id, ok := idPerubahan(w, r)
	if !ok {
		return
	}
	var req putusRequest
	if !decode(w, r, &req) {
		return
	}
	req.Catatan = strings.TrimSpace(req.Catatan)
	if len([]rune(req.Catatan)) < 3 {
		tulisGalatPutus(w, errCatatanWajib)
		return
	}
	if len([]rune(req.Catatan)) > 500 {
		req.Catatan = string([]rune(req.Catatan)[:500])
	}
	adminID := identity.GetAdminUserID(r.Context())
	var p *pengajuanTerkunci
	err := dalamTx(r.Context(), h.db, func(tx *sql.Tx) error {
		var err error
		p, err = tolakTx(r.Context(), tx, id, adminID, adminBrandScope(r), req.Catatan)
		return err
	})
	if err != nil {
		tulisGalatPutus(w, err)
		return
	}
	log.Printf("[AUDIT] perubahan_jamaah_ditolak id=%d jamaah_id=%d admin_id=%d", id, p.JamaahID, adminID)
	h.kabari(p.AkunID, p.JamaahID, false, req.Catatan)
	writeJSON(w, http.StatusOK, map[string]string{"message": "pengajuan ditolak"})
}

// kabari mengirim email keputusan ke pemilik akun (tanpa menunggu; gagal kirim hanya dicatat).
func (h *JamaahAkun) kabari(akunID, jamaahID int64, setuju bool, catatan string) {
	if akunID == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var email, nama, namaJamaah string
	var brandID int64
	if err := h.db.QueryRowContext(ctx, `SELECT a.email, a.nama_lengkap, a.brand_id, j.nama_lengkap FROM customer_accounts a, jamaah j WHERE a.id=? AND j.id=?`,
		akunID, jamaahID).Scan(&email, &nama, &brandID, &namaJamaah); err != nil {
		log.Printf("[ERROR] email perubahan jamaah: %v", err)
		return
	}
	brand, domain, err := h.repo.BrandSite(ctx, brandID)
	if err != nil || domain == "" {
		return
	}
	subject, text := mailer.PerubahanDataJamaah(brand, nama, namaJamaah, setuju, catatan, siteLink(domain, "/akun/jamaah/"+strconv.FormatInt(jamaahID, 10), nil))
	h.sendAsync(mailer.Message{To: email, Subject: subject, Text: text})
}
