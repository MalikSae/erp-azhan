package jamaah

import (
	"context"
	"log"
	"strings"

	"erp-azhan/api/internal/identity"
)

// Edit data jamaah oleh admin dicatat di jamaah_perubahan (sumber admin, status langsung) bersama perubahan dari pelanggan,
// sehingga satu riwayat memuat semua perubahan. Pencatatan gagal hanya dicatat di log server; edit admin tetap berlaku.

type isianLog struct {
	kolom string
	lama  string
	baru  string
}

func tanggal10(s string) string {
	if len(s) > 10 {
		return s[:10]
	}
	return s
}

func nilaiPtr(v *string) string { return isiDari(v) }

// bedaIsian membandingkan jamaah lama dengan hasil edit admin pada isian identitas dan data lengkap.
func bedaIsian(lama *Jamaah, req *UpdateJamaahRequest) []isianLog {
	pasangan := []isianLog{
		{"nama_lengkap", strings.TrimSpace(lama.NamaLengkap), strings.TrimSpace(req.NamaLengkap)},
		{"jenis_kelamin", nilaiPtr(lama.JenisKelamin), nilaiPtr(req.JenisKelamin)},
		{"nama_ayah_kandung", nilaiPtr(lama.NamaAyahKandung), nilaiPtr(req.NamaAyahKandung)},
		{"nik", nilaiPtr(lama.NIK), nilaiPtr(req.NIK)},
		{"tempat_lahir", nilaiPtr(lama.TempatLahir), nilaiPtr(req.TempatLahir)},
		{"tanggal_lahir", tanggal10(nilaiPtr(lama.TanggalLahir)), tanggal10(nilaiPtr(req.TanggalLahir))},
		{"no_paspor", nilaiPtr(lama.NoPaspor), nilaiPtr(req.NoPaspor)},
		{"tempat_paspor_keluar", nilaiPtr(lama.TempatPasporKeluar), nilaiPtr(req.TempatPasporKeluar)},
		{"tanggal_paspor_keluar", tanggal10(nilaiPtr(lama.TanggalPasporKeluar)), tanggal10(nilaiPtr(req.TanggalPasporKeluar))},
		{"paspor_berlaku_sampai", tanggal10(nilaiPtr(lama.PasporBerlakuSampai)), tanggal10(nilaiPtr(req.PasporBerlakuSampai))},
		{"no_hp", nilaiPtr(lama.NoHP), nilaiPtr(req.NoHP)},
		{"email", nilaiPtr(lama.Email), nilaiPtr(req.Email)},
		{"pekerjaan", nilaiPtr(lama.Pekerjaan), nilaiPtr(req.Pekerjaan)},
		{"pendidikan_terakhir", nilaiPtr(lama.PendidikanTerakhir), nilaiPtr(req.PendidikanTerakhir)},
		{"penjamin_kesehatan", nilaiPtr(lama.PenjaminKesehatan), nilaiPtr(req.PenjaminKesehatan)},
		{"no_asuransi_bpjs", nilaiPtr(lama.NoAsuransiBPJS), nilaiPtr(req.NoAsuransiBPJS)},
		{"alamat", nilaiPtr(lama.Alamat), nilaiPtr(req.Alamat)},
		{"kota", nilaiPtr(lama.Kota), nilaiPtr(req.Kota)},
		{"emergency_nama", nilaiPtr(lama.EmergencyNama), nilaiPtr(req.EmergencyNama)},
		{"emergency_hp", nilaiPtr(lama.EmergencyHP), nilaiPtr(req.EmergencyHP)},
		{"emergency_hubungan", nilaiPtr(lama.EmergencyHubungan), nilaiPtr(req.EmergencyHubungan)},
		{"emergency_alamat", nilaiPtr(lama.EmergencyAlamat), nilaiPtr(req.EmergencyAlamat)},
	}
	var beda []isianLog
	for _, p := range pasangan {
		if p.lama != p.baru {
			beda = append(beda, p)
		}
	}
	return beda
}

// catatEditAdmin menulis satu baris jamaah_perubahan beserta item-itemnya bila ada isian yang berubah.
func (r *Repository) catatEditAdmin(ctx context.Context, jamaahID int64, lama *Jamaah, req *UpdateJamaahRequest) {
	beda := bedaIsian(lama, req)
	if len(beda) == 0 {
		return
	}
	adminID := identity.GetAdminUserID(ctx)
	var admin any
	if adminID > 0 {
		admin = adminID
	}
	res, err := r.db.ExecContext(ctx, `INSERT INTO jamaah_perubahan (jamaah_id, admin_id, sumber, status) VALUES (?,?,'admin','langsung')`, jamaahID, admin)
	if err != nil {
		log.Printf("[ERROR] catat edit admin jamaah %d: %v", jamaahID, err)
		return
	}
	pid, err := res.LastInsertId()
	if err != nil {
		log.Printf("[ERROR] catat edit admin jamaah %d: %v", jamaahID, err)
		return
	}
	for _, b := range beda {
		var l, n any
		if b.lama != "" {
			l = b.lama
		}
		if b.baru != "" {
			n = b.baru
		}
		if _, err := r.db.ExecContext(ctx, `INSERT INTO jamaah_perubahan_item (perubahan_id, kolom, nilai_lama, nilai_baru) VALUES (?,?,?,?)`, pid, b.kolom, l, n); err != nil {
			log.Printf("[ERROR] catat item edit admin jamaah %d: %v", jamaahID, err)
		}
	}
}
