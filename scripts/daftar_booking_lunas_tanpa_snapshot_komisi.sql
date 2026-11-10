-- Baca-saja. Audit menu Paket 10 November 2026 (T-01).
--
-- Daftar pax pada booking 'lunas' yang snapshot komisinya kosong, padahal paketnya kini punya nominal
-- efektif (nilai paket, selain itu bawaan brand). Booking seperti ini tidak mencatat komisi saat lunas.
-- Migrasi 127 sengaja tidak mengubahnya: pemilik proyek memutuskan per booking.
--
-- Satu baris per pax aktif non-infant. Kolom penerima kosong berarti jamaah tanpa agen, sehingga tidak
-- ada komisi yang terlewat untuk pax itu. Pax "diri agen" (booking pribadi agen) tidak dibedakan di sini;
-- periksa manual bila nama jamaah sama dengan nama agen.
--
-- Jalankan: mysql -u<user> -p <database> -t < scripts/daftar_booking_lunas_tanpa_snapshot_komisi.sql

SELECT
  b.id                                        AS booking_id,
  b.id_booking                                AS kode_booking,
  br.name                                     AS brand,
  s.id                                        AS paket_id,
  s.jadwal_nama                               AS paket,
  DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d') AS berangkat,
  DATE_FORMAT(b.pertama_lunas_at, '%Y-%m-%d')  AS lunas_pertama,
  j.nama_lengkap                              AS jamaah,
  a.nama_lengkap                              AS agen_penerima_langsung,
  a.status_agen                               AS status_agen,
  COALESCE(s.nominal_komisi_langsung, NULLIF(br.ujroh_closing, 0))   AS komisi_langsung_seharusnya,
  u.nama_lengkap                              AS upline_penerima_pembinaan,
  u.status_agen                               AS status_upline,
  COALESCE(s.nominal_bonus_pembinaan, NULLIF(br.ujroh_pembinaan, 0)) AS bonus_pembinaan_seharusnya
FROM bookings b
JOIN schedules s ON s.id = b.schedule_id
JOIN brands br ON br.id = s.brand_id
JOIN booking_pax bp ON bp.booking_id = b.id AND bp.pax_status = 'aktif' AND bp.pax_type <> 'infant'
JOIN jamaah j ON j.id = bp.jamaah_id
LEFT JOIN customer_accounts a ON a.id = j.direkrut_oleh_akun_id
LEFT JOIN customer_accounts u ON u.id = a.upline_akun_id
WHERE b.status = 'lunas'
  AND b.snapshot_komisi_langsung IS NULL
  AND b.snapshot_bonus_pembinaan IS NULL
  AND (COALESCE(s.nominal_komisi_langsung, NULLIF(br.ujroh_closing, 0)) > 0
       OR COALESCE(s.nominal_bonus_pembinaan, NULLIF(br.ujroh_pembinaan, 0)) > 0)
ORDER BY b.id, bp.id;
