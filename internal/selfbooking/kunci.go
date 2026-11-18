package selfbooking

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"erp-azhan/api/internal/agen"
)

// pemesan adalah akun yang membuat booking, dibaca di dalam transaksi booking. Mode ditentukan status akunnya,
// bukan endpoint yang dipanggil: agen aktif di brand yang sama memesan sebagai agen, selain itu akun biasa.
type pemesan struct {
	id       int64
	agen     bool
	nikAgen  string
	jamaahID int64 // jamaah tertaut ke akun; 0 bila tidak ada
	// uplineID: upline_akun_id akun (kolom keagenan, bukan direkrut_oleh_akun_id), aktif atau nonaktif.
	// Satu semantik dengan agen.UplineAkun. 0 bila tidak ada.
	uplineID int64
}

// bacaPemesan membaca dan mengunci baris akun pemesan. Akun tidak ada atau beda brand: ErrAkunTidakValid.
func bacaPemesan(ctx context.Context, tx *sql.Tx, brandID, akunID int64) (pemesan, error) {
	p := pemesan{id: akunID}
	if akunID <= 0 {
		return p, ErrAkunTidakValid
	}
	var status string
	var nik sql.NullString
	var jamaah, upline sql.NullInt64
	err := tx.QueryRowContext(ctx, `
		SELECT c.status_agen, c.nik_agen, c.jamaah_id, u.id
		FROM customer_accounts c
		LEFT JOIN customer_accounts u ON u.id = c.upline_akun_id AND u.status_agen IN ('aktif','nonaktif')
		WHERE c.id = ? AND c.brand_id = ?
		FOR UPDATE OF c`, akunID, brandID).Scan(&status, &nik, &jamaah, &upline)
	if errors.Is(err, sql.ErrNoRows) {
		return p, ErrAkunTidakValid
	}
	if err != nil {
		return p, fmt.Errorf("baca pemesan: %w", err)
	}
	p.agen = status == "aktif"
	p.nikAgen = nik.String
	p.jamaahID = jamaah.Int64
	p.uplineID = upline.Int64
	return p, nil
}

// jamaahDiriAgen: jamaah itu diri pemesan agen. Jamaah tertaut ke akunnya selalu diri, sama dengan penentu di
// komisi.loadPax (tertaut atau saya_sendiri_akun_id). NIK sama dengan nik_agen hanya berlaku bila jamaah tidak
// tertaut ke akun lain dan tidak dibuat akun lain: nik_agen tidak diverifikasi, jadi NIK jamaah orang lain tidak
// boleh menjadikannya diri agen. Cabang NIK hanya memilih pemilik kunci (upline, bukan agen); di komisi jamaah
// itu terikat ke upline, sehingga penerimanya sama. Komisi tidak membaca NIK-sama (ADR 0004).
func jamaahDiriAgen(ctx context.Context, tx *sql.Tx, p pemesan, jamaahID int64) (bool, error) {
	if p.jamaahID > 0 && p.jamaahID == jamaahID {
		return true, nil
	}
	if p.nikAgen == "" {
		return false, nil
	}
	var nik sql.NullString
	var pembuat sql.NullInt64
	var tertautLain bool
	err := tx.QueryRowContext(ctx, `
		SELECT j.nik, j.dibuat_oleh_akun_id,
			EXISTS(SELECT 1 FROM customer_accounts c WHERE c.jamaah_id = j.id AND c.id <> ?)
		FROM jamaah j WHERE j.id = ?`, p.id, jamaahID).Scan(&nik, &pembuat, &tertautLain)
	if err != nil {
		return false, fmt.Errorf("baca jamaah diri agen: %w", err)
	}
	if !nik.Valid || nik.String != p.nikAgen || tertautLain {
		return false, nil
	}
	return !pembuat.Valid || pembuat.Int64 == p.id, nil
}

// isiNIKPemesanAgen mengisi NIK jamaah pertama dari akun agen yang ikut berangkat. NIK utuh tidak dikirim
// ke browser; NIK request yang berbeda ditolak.
func isiNIKPemesanAgen(ctx context.Context, tx *sql.Tx, p pemesan, req *BookingRequest) error {
	nik := p.nikAgen
	if nik == "" && p.jamaahID > 0 {
		var n sql.NullString
		if err := tx.QueryRowContext(ctx, `SELECT nik FROM jamaah WHERE id = ?`, p.jamaahID).Scan(&n); err != nil {
			return fmt.Errorf("baca NIK jamaah tertaut: %w", err)
		}
		nik = n.String
	}
	if nik == "" {
		return ErrNIKAgenBelumAda
	}
	if dikirim := nikBersih(req.PIC.NIK); dikirim != "" && dikirim != nik {
		return ErrSayaSendiriBukanAnda
	}
	req.PIC.NIK = &nik
	return nil
}

// kunciJamaahPemesan adalah satu-satunya pintu kunci jamaah booking self-service (first-set wins lewat
// agen.IkatJamaah; hanya jamaah belum_ditentukan yang terikat, yang sudah terikat tidak pernah berpindah).
//
// Pemesan agen aktif: pemilik kunci = pemesan. Jamaah diri pemesan dikunci ke upline_akun_id pemesan, aktif atau
// nonaktif. Pemesan bukan agen: akun yang ikut berangkat lebih dulu mengikuti pemilik kunci jamaahnya
// (IkatAkunDariJamaah), lalu semua jamaah dikunci ke upline akun (UplineAkun).
//
// Awas: upline punya dua kolom (direkrut_oleh_akun_id dan upline_akun_id). Cabang agen membaca upline_akun_id,
// cabang akun biasa membaca direkrut_oleh_akun_id lewat UplineAkun; keduanya satu konsep.
func kunciJamaahPemesan(ctx context.Context, tx *sql.Tx, p pemesan, picJamaahID int64, pemesanIkut bool, jamaahIDs []int64) error {
	if p.agen {
		for _, id := range jamaahIDs {
			pemilik := p.id
			diri, err := jamaahDiriAgen(ctx, tx, p, id)
			if err != nil {
				return err
			}
			if diri {
				pemilik = p.uplineID
			}
			if err := agen.IkatJamaah(ctx, tx, id, pemilik, agen.SumberJalur1); err != nil {
				return err
			}
		}
		return nil
	}
	if pemesanIkut {
		if err := agen.IkatAkunDariJamaah(ctx, tx, p.id, picJamaahID); err != nil {
			return err
		}
	}
	upline, err := agen.UplineAkun(ctx, tx, p.id)
	if err != nil {
		return err
	}
	return agen.IkatRombongan(ctx, tx, jamaahIDs, upline, agen.SumberReferral)
}
