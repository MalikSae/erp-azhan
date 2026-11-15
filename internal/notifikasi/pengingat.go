package notifikasi

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// JamPengingat: pengingat harian dikirim mulai pukul 08.00 WIB. Loop berjalan tiap menit; kunci unik
// membuat tiap pengingat hanya terkirim sekali.
const JamPengingat = 8

// JalankanPengingat menjalankan loop latar: pengingat berbasis waktu, pengiriman pengumuman, dan
// pembersihan notifikasi lama. Berhenti saat ctx selesai.
func JalankanPengingat(ctx context.Context, db *sql.DB, jeda time.Duration) {
	jalan := func() {
		if err := Pengingat(ctx, db, time.Now()); err != nil {
			log.Printf("[ERROR] notifikasi pengingat: %v", err)
		}
	}
	jalan()
	t := time.NewTicker(jeda)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			jalan()
		}
	}
}

// Pengingat menjalankan satu putaran. now menentukan "hari ini" (WIB) dan jam pengingat harian.
func Pengingat(ctx context.Context, db Querier, now time.Time) error {
	if err := holdSeat(ctx, db, now); err != nil {
		return err
	}
	if err := KirimPengumuman(ctx, db, now); err != nil {
		return err
	}
	if now.In(ZonaWIB).Hour() >= JamPengingat {
		hari := now.In(ZonaWIB).Format("2006-01-02")
		for _, f := range []func(context.Context, Execer, string) error{pelunasan, berangkat, komisiTersedia} {
			if err := f(ctx, db, hari); err != nil {
				return err
			}
		}
	}
	return Bersihkan(ctx, db, now)
}

// holdSeat: booking baru yang kursinya ditahan dan hold-nya berakhir dalam 3 jam. Kunci memuat waktu
// berakhir, sehingga hold yang diperpanjang mendapat pengingat baru.
func holdSeat(ctx context.Context, db Execer, now time.Time) error {
	_, err := db.ExecContext(ctx, `INSERT IGNORE INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, ref_id, kunci)
		SELECT b.customer_account_id, ?, 'Hold seat berakhir 3 jam lagi', LEFT(CONCAT(s.jadwal_nama, ' · Bayar DP'), 300), ?, b.id,
		       CONCAT('hold:', b.id, ':', DATE_FORMAT(b.seat_hold_expires_at, '%Y%m%d%H%i'))
		FROM bookings b JOIN schedules s ON s.id = b.schedule_id
		WHERE b.status = 'baru' AND b.is_seat_blocked AND b.customer_account_id IS NOT NULL
		  AND b.seat_hold_expires_at > ? AND b.seat_hold_expires_at <= ?`,
		JenisHoldSeat, RefBooking, now, now.Add(3*time.Hour))
	return err
}

// pelunasan: booking dp dengan sisa tagihan yang batas pelunasannya 7 hari lagi. Batas diambil dari
// checkout booking; booking tanpa checkout memakai tanggal berangkat dikurangi batas pelunasan brand.
func pelunasan(ctx context.Context, db Execer, hari string) error {
	_, err := db.ExecContext(ctx, `INSERT IGNORE INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, ref_id, kunci)
		SELECT x.akun, ?, 'Pelunasan H-7', LEFT(CONCAT('Sisa Rp', FORMAT(x.sisa, 0, 'id_ID'), ' · ', x.paket), 300), ?, x.id,
		       CONCAT('pelunasan:', x.id, ':', x.batas)
		FROM (
			SELECT b.id, b.customer_account_id AS akun, s.jadwal_nama AS paket,
			       COALESCE(DATE(c.due_at), DATE_SUB(s.berangkat_tanggal, INTERVAL br.batas_pelunasan_hari DAY)) AS batas,
			       COALESCE(b.total_harga, 0) - COALESCE((SELECT SUM(p.jumlah) FROM payments p WHERE p.booking_id = b.id AND p.status = 'confirmed'), 0) AS sisa
			FROM bookings b
			JOIN schedules s ON s.id = b.schedule_id
			JOIN brands br ON br.id = s.brand_id
			LEFT JOIN booking_checkout c ON c.booking_id = b.id
			WHERE b.status = 'dp' AND b.customer_account_id IS NOT NULL
		) x
		WHERE x.sisa > 0 AND x.batas = DATE_ADD(?, INTERVAL 7 DAY)`,
		JenisPelunasan, RefBooking, hari)
	return err
}

// berangkat: booking dp/lunas yang berangkat 7 hari lagi atau besok.
func berangkat(ctx context.Context, db Execer, hari string) error {
	for _, h := range []struct {
		hari  int
		judul string
	}{{7, "Berangkat 7 hari lagi"}, {1, "Berangkat besok"}} {
		_, err := db.ExecContext(ctx, `INSERT IGNORE INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, ref_id, kunci)
			SELECT b.customer_account_id, ?, ?, LEFT(CONCAT(s.jadwal_nama, ' · ', CONCAT(DAY(s.berangkat_tanggal), ' ', ELT(MONTH(s.berangkat_tanggal), 'Jan','Feb','Mar','Apr','Mei','Jun','Jul','Agu','Sep','Okt','Nov','Des'), ' ', YEAR(s.berangkat_tanggal))), 300), ?, b.id,
			       CONCAT('berangkat', ?, ':', b.id)
			FROM bookings b JOIN schedules s ON s.id = b.schedule_id
			WHERE b.status IN ('dp','lunas') AND b.customer_account_id IS NOT NULL
			  AND s.berangkat_tanggal = DATE_ADD(?, INTERVAL ? DAY)`,
			JenisBerangkat, h.judul, RefPerjalanan, h.hari, hari, h.hari)
		if err != nil {
			return err
		}
	}
	return nil
}

// komisiTersedia: komisi aktif (bukan cashback) yang mulai bisa dicairkan, yaitu H+3 setelah tanggal
// berangkat (komisi.SQLTersediaMulai). Jendela 3 hari menutup jeda bila server sempat mati, tanpa
// membanjiri agen dengan komisi lama.
func komisiTersedia(ctx context.Context, db Execer, hari string) error {
	_, err := db.ExecContext(ctx, `INSERT IGNORE INTO notifikasi (akun_id, jenis, judul, isi, ref_tipe, kunci)
		SELECT t.akun_penerima_id, ?, 'Komisi tersedia', LEFT(CONCAT('Rp', FORMAT(t.nominal, 0, 'id_ID'), ' · ', j.nama_lengkap), 300), ?,
		       CONCAT('komisi:', t.id)
		FROM transaksi_komisi t
		JOIN bookings b ON b.id = t.booking_id
		JOIN schedules s ON s.id = b.schedule_id
		JOIN booking_pax bp ON bp.id = t.booking_pax_id
		JOIN jamaah j ON j.id = bp.jamaah_id
		WHERE t.dibatalkan_at IS NULL AND t.jenis <> 'cashback' AND b.status = 'lunas'
		  AND DATE_ADD(s.berangkat_tanggal, INTERVAL 3 DAY) BETWEEN DATE_SUB(?, INTERVAL 3 DAY) AND ?`,
		JenisKomisiTersedia, RefDompet, hari, hari)
	return err
}
