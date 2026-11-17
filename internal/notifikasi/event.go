package notifikasi

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strconv"
	"time"
)

// Fungsi di file ini dipanggil paket lain tepat setelah perubahan data, di transaksi yang sama bila ada.
// Semuanya "aman": kegagalan hanya dicatat di log supaya perubahan utama tetap tersimpan.

// RefJamaah: ref_id = jamaah id (data jamaah di app).
const RefJamaah = "jamaah"

var labelDokumen = map[string]string{
	"pas_foto":   "Pas foto",
	"paspor":     "Paspor",
	"ktp":        "KTP",
	"kk":         "Kartu keluarga",
	"buku_nikah": "Buku nikah",
	"akte_lahir": "Akta lahir",
}

func catat(jenis string, err error) {
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		log.Printf("[WARN] notifikasi %s: %v", jenis, err)
	}
}

// infoBooking: pemilik booking dan nama jadwalnya.
func infoBooking(ctx context.Context, q Querier, bookingID int64) (akun int64, paket string, berangkat time.Time, err error) {
	var a sql.NullInt64
	err = q.QueryRowContext(ctx, `SELECT b.customer_account_id, s.jadwal_nama, s.berangkat_tanggal
		FROM bookings b JOIN schedules s ON s.id = b.schedule_id WHERE b.id = ?`, bookingID).Scan(&a, &paket, &berangkat)
	return a.Int64, paket, berangkat, err
}

// Pembayaran: pembayaran diverifikasi (diterima) atau ditolak admin.
func Pembayaran(ctx context.Context, q Querier, bookingID int64, diterima bool, jumlah float64) {
	akun, paket, _, err := infoBooking(ctx, q, bookingID)
	if err != nil {
		catat("pembayaran", err)
		return
	}
	n := Notif{Jenis: JenisPembayaranDiterima, Judul: "Pembayaran diterima", RefTipe: RefBooking, RefID: bookingID,
		Isi: Gabung(Rupiah(jumlah), paket)}
	if !diterima {
		n.Jenis, n.Judul = JenisPembayaranDitolak, "Pembayaran ditolak"
	}
	KirimAman(ctx, q, akun, n)
}

// BookingBatalJatuhTempo: booking dibatalkan otomatis karena lewat batas pelunasan.
func BookingBatalJatuhTempo(ctx context.Context, q Querier, bookingID int64) {
	akun, paket, _, err := infoBooking(ctx, q, bookingID)
	if err != nil {
		catat("booking batal", err)
		return
	}
	KirimAman(ctx, q, akun, Notif{Jenis: JenisBookingBatal, Judul: "Booking dibatalkan", RefTipe: RefBooking, RefID: bookingID,
		Isi: Gabung(paket, "Lewat batas pelunasan"), Kunci: "batal:" + strconv.FormatInt(bookingID, 10)})
}

func memegangKursi(status string) bool { return status == "dp" || status == "lunas" }

// StatusBooking: dipanggil saat status booking berubah dari dari ke ke. Lunas memberi tahu pemilik
// booking; naik ke dp/lunas pertama kali memberi tahu agen perekrut tiap jamaah ("Jamaah baru").
func StatusBooking(ctx context.Context, q Querier, bookingID int64, dari, ke string) {
	if dari == ke {
		return
	}
	akun, paket, berangkat, err := infoBooking(ctx, q, bookingID)
	if err != nil {
		catat("status booking", err)
		return
	}
	if ke == "lunas" {
		KirimAman(ctx, q, akun, Notif{Jenis: JenisBookingLunas, Judul: "Booking lunas", RefTipe: RefBooking, RefID: bookingID,
			Isi: Gabung(paket, Tanggal(berangkat))})
	}
	if memegangKursi(ke) && !memegangKursi(dari) {
		_, err := q.ExecContext(ctx, `INSERT IGNORE INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, kunci)
			SELECT j.direkrut_oleh_akun_id, ?, 'Jamaah baru', LEFT(CONCAT(j.nama_lengkap, ' · ', ?), 300), ?, CONCAT('referral:', bp.id)
			FROM booking_pax bp
			JOIN jamaah j ON j.id = bp.jamaah_id
			JOIN customer_accounts a ON a.id = j.direkrut_oleh_akun_id AND a.status_agen = 'aktif' AND a.dihapus_at IS NULL
			WHERE bp.booking_id = ? AND bp.pax_status = 'aktif'`, JenisJamaahReferral, paket, RefAgen, bookingID)
		catat("jamaah referral", err)
	}
}

// Dokumen: dokumen jamaah diverifikasi atau ditolak. Penerima: akun pembuat data jamaah dan setiap akun
// yang punya booking aktif berisi jamaah itu.
func Dokumen(ctx context.Context, q Querier, jamaahID int64, jenisDok string, diverifikasi bool) {
	jenis, judul := JenisDokumenDiverifikasi, "Dokumen diverifikasi"
	if !diverifikasi {
		jenis, judul = JenisDokumenDitolak, "Dokumen ditolak"
	}
	label := labelDokumen[jenisDok]
	if label == "" {
		label = jenisDok
	}
	_, err := q.ExecContext(ctx, `INSERT INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, ref_id)
		SELECT x.akun, ?, ?, LEFT(CONCAT(?, ' · ', j.nama_lengkap), 300), ?, j.id
		FROM (
			SELECT dibuat_oleh_akun_id AS akun FROM jamaah WHERE id = ? AND dibuat_oleh_akun_id IS NOT NULL
			UNION
			SELECT b.customer_account_id FROM booking_pax bp JOIN bookings b ON b.id = bp.booking_id
			WHERE bp.jamaah_id = ? AND bp.pax_status = 'aktif' AND b.status NOT IN ('draft','batal') AND b.customer_account_id IS NOT NULL
		) x
		JOIN jamaah j ON j.id = ?
		JOIN customer_accounts a ON a.id = x.akun AND a.dihapus_at IS NULL`,
		jenis, judul, label, RefJamaah, jamaahID, jamaahID, jamaahID)
	catat("dokumen", err)
}

// PerubahanData: pengajuan perubahan data jamaah disetujui atau ditolak.
func PerubahanData(ctx context.Context, q Querier, akunID, jamaahID int64, disetujui bool) {
	var nama string
	if err := q.QueryRowContext(ctx, `SELECT nama_lengkap FROM jamaah WHERE id = ?`, jamaahID).Scan(&nama); err != nil {
		catat("perubahan data", err)
		return
	}
	n := Notif{Jenis: JenisPerubahanDisetujui, Judul: "Perubahan data disetujui", Isi: nama, RefTipe: RefJamaah, RefID: jamaahID}
	if !disetujui {
		n.Jenis, n.Judul = JenisPerubahanDitolak, "Perubahan data ditolak"
	}
	KirimAman(ctx, q, akunID, n)
}

// AgenDisetujui: akun resmi jadi agen; upline-nya (bila ada) mendapat "Agen binaan baru".
func AgenDisetujui(ctx context.Context, q Querier, akunID int64) {
	var nama string
	var kode sql.NullString
	var upline sql.NullInt64
	if err := q.QueryRowContext(ctx, `SELECT nama_lengkap, kode_referral, upline_akun_id FROM customer_accounts WHERE id = ?`, akunID).
		Scan(&nama, &kode, &upline); err != nil {
		catat("agen disetujui", err)
		return
	}
	isi := ""
	if kode.String != "" {
		isi = "Kode " + kode.String
	}
	KirimAman(ctx, q, akunID, Notif{Jenis: JenisAgenDisetujui, Judul: "Anda resmi jadi agen", Isi: isi, RefTipe: RefAgen,
		Kunci: "agen-disetujui"})
	if upline.Valid && upline.Int64 != akunID {
		KirimAman(ctx, q, upline.Int64, Notif{Jenis: JenisAgenBinaan, Judul: "Agen binaan baru", Isi: nama, RefTipe: RefAgen,
			Kunci: "binaan:" + strconv.FormatInt(akunID, 10)})
	}
}

// Pencairan: pengajuan pencairan komisi diproses (disetujui) atau ditolak.
func Pencairan(ctx context.Context, q Querier, pencairanID int64, diproses bool) {
	var agenID int64
	var nominal float64
	if err := q.QueryRowContext(ctx, `SELECT akun_agen_id, nominal_diajukan FROM pengajuan_pencairan WHERE id = ?`, pencairanID).
		Scan(&agenID, &nominal); err != nil {
		catat("pencairan", err)
		return
	}
	n := Notif{Jenis: JenisPencairanDiproses, Judul: "Pencairan diproses", Isi: Rupiah(nominal), RefTipe: RefDompet}
	if !diproses {
		n.Jenis, n.Judul = JenisPencairanDitolak, "Pencairan ditolak"
	}
	KirimAman(ctx, q, agenID, n)
}

// KomisiBatal: dipanggil sebelum komisi aktif sebuah booking dibatalkan. Satu notifikasi per penerima
// per jamaah. Cashback tidak diberitahukan.
func KomisiBatal(ctx context.Context, q Querier, bookingID int64) {
	_, err := q.ExecContext(ctx, `INSERT INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe)
		SELECT t.akun_penerima_id, ?, 'Komisi batal', LEFT(CONCAT(?, FORMAT(SUM(t.nominal), 0, 'id_ID'), ' · ', MAX(j.nama_lengkap)), 300), ?
		FROM transaksi_komisi t
		JOIN booking_pax bp ON bp.id = t.booking_pax_id
		JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE t.booking_id = ? AND t.dibatalkan_at IS NULL AND t.jenis <> 'cashback'
		GROUP BY t.akun_penerima_id, t.booking_pax_id`, JenisKomisiBatal, "Rp", RefDompet, bookingID)
	catat("komisi batal", err)
}
