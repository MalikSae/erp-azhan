package customer

// Hapus akun dari app (keputusan 15 November 2026). Akun dianonimkan, bukan dihapus barisnya: booking, jamaah, dan
// pembayaran tetap merujuk akun yang sama. Ditolak selama masih ada urusan berjalan (booking aktif, keagenan aktif,
// pencairan diproses); app lalu menampilkan tombol WhatsApp admin.

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
)

// HapusAkunView: apakah akun boleh dihapus sendiri, dan alasannya bila tidak.
type HapusAkunView struct {
	Bisa   bool    `json:"bisa"`
	Alasan *string `json:"alasan"`
}

type pembacaBaris interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// alasanTolakHapus: alasan pertama akun tidak boleh dihapus sendiri, atau "" bila boleh.
func alasanTolakHapus(ctx context.Context, q pembacaBaris, akunID int64) (string, error) {
	var bookingAktif, agenAktif, pencairan bool
	err := q.QueryRowContext(ctx, `
		SELECT
			EXISTS (SELECT 1 FROM bookings b JOIN schedules s ON s.id = b.schedule_id
				WHERE b.customer_account_id = ? AND b.status IN ('dp', 'lunas') AND s.berangkat_tanggal >= CURDATE()),
			EXISTS (SELECT 1 FROM customer_accounts WHERE id = ? AND status_agen = 'aktif'),
			EXISTS (SELECT 1 FROM pengajuan_pencairan WHERE akun_agen_id = ? AND status = 'pending')`,
		akunID, akunID, akunID).Scan(&bookingAktif, &agenAktif, &pencairan)
	if err != nil {
		return "", fmt.Errorf("customer: cek hapus akun: %w", err)
	}
	switch {
	case bookingAktif:
		return "Masih ada booking aktif.", nil
	case agenAktif:
		return "Akun agen aktif.", nil
	case pencairan:
		return "Masih ada pencairan diproses.", nil
	}
	return "", nil
}

// anonimkanAkun mengosongkan data pribadi akun dan mencabut semua sesi serta token. Email diganti alamat unik yang
// tidak bisa dipakai masuk; nomor dikosongkan (no_hp_normal menjadi NULL, bebas dipakai akun lain).
func anonimkanAkun(ctx context.Context, tx *sql.Tx, akunID int64) error {
	email := fmt.Sprintf("dihapus-%d@akun.invalid", akunID)
	if _, err := tx.ExecContext(ctx, `
		UPDATE customer_accounts SET
			nama_lengkap = 'Akun dihapus', email = ?, email_normal = ?, no_hp = '', password_hash = '!',
			foto_url = NULL, domisili = NULL, tanggal_lahir = NULL, jenis_kelamin = NULL, nik_agen = NULL,
			kode_referral = NULL, rekening_bank_agen = NULL, rekening_nomor_agen = NULL, rekening_atas_nama_agen = NULL,
			email_verified_at = NULL, google_sub = NULL, dihapus_at = NOW()
		WHERE id = ?`, email, email, akunID); err != nil {
		return fmt.Errorf("customer: anonimkan akun: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_sessions SET revoked_at = NOW() WHERE account_id = ? AND revoked_at IS NULL`, akunID); err != nil {
		return fmt.Errorf("customer: cabut sesi: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE customer_tokens SET used_at = NOW() WHERE account_id = ? AND used_at IS NULL`, akunID); err != nil {
		return fmt.Errorf("customer: cabut token: %w", err)
	}
	return nil
}

// CekHapusAkun GET /api/account/hapus: {bisa, alasan}.
func (h *Handler) CekHapusAkun(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	alasan, err := alasanTolakHapus(r.Context(), h.repo.db, acc.ID)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	v := HapusAkunView{Bisa: alasan == ""}
	if alasan != "" {
		v.Alasan = &alasan
	}
	writeJSON(w, http.StatusOK, v)
}

type hapusAkunRequest struct {
	PasswordSaatIni string `json:"password_saat_ini"`
	// IDToken: konfirmasi untuk akun tanpa sandi (dibuat lewat Google), berupa ID token akun Google yang tertaut.
	IDToken string `json:"id_token"`
}

// cekKonfirmasiHapus: sandi saat ini, atau ID token Google milik akun Google yang tertaut ke akun ini.
func (h *Handler) cekKonfirmasiHapus(w http.ResponseWriter, r *http.Request, acc *Account, req hapusAkunRequest) bool {
	if req.IDToken == "" {
		return h.cekPasswordSaatIni(w, acc, req.PasswordSaatIni)
	}
	p, ok := h.verifikasiGoogle(w, r, req.IDToken)
	if !ok {
		return false
	}
	var sub string
	if err := h.repo.db.QueryRowContext(r.Context(), `SELECT COALESCE(google_sub, '') FROM customer_accounts WHERE id = ?`, acc.ID).Scan(&sub); err != nil {
		log.Printf("[ERROR] customer hapus akun google: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return false
	}
	if sub == "" || sub != p.Sub {
		writeError(w, http.StatusForbidden, "Akun Google tidak sesuai")
		return false
	}
	return true
}

// HapusAkun POST /api/account/hapus: {password_saat_ini}. 204 bila berhasil; 409 dengan alasan bila masih ada urusan
// berjalan. Sesi yang dipakai ikut dicabut.
func (h *Handler) HapusAkun(w http.ResponseWriter, r *http.Request) {
	acc := FromContext(r.Context())
	var req hapusAkunRequest
	if !decode(w, r, &req) {
		return
	}
	if !h.cekKonfirmasiHapus(w, r, acc, req) {
		return
	}
	tx, err := h.repo.db.BeginTx(r.Context(), nil)
	if err != nil {
		log.Printf("[ERROR] customer hapus akun: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	defer func() { _ = tx.Rollback() }()
	// Kunci baris akun supaya booking atau pencairan baru tidak menyelinap di antara pengecekan dan anonimisasi.
	var x int64
	if err := tx.QueryRowContext(r.Context(), `SELECT id FROM customer_accounts WHERE id = ? FOR UPDATE`, acc.ID).Scan(&x); err != nil {
		log.Printf("[ERROR] customer hapus akun kunci: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	alasan, err := alasanTolakHapus(r.Context(), tx, acc.ID)
	if err == nil && alasan != "" {
		writeError(w, http.StatusConflict, alasan)
		return
	}
	if err == nil {
		err = anonimkanAkun(r.Context(), tx, acc.ID)
	}
	if err == nil {
		err = tx.Commit()
	}
	if err != nil {
		log.Printf("[ERROR] customer hapus akun: %v", err)
		writeError(w, http.StatusInternalServerError, "terjadi kesalahan, silakan coba lagi")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
