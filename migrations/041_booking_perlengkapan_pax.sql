-- 041_booking_perlengkapan_pax.sql
-- Tambahkan kolom perlengkapan_jumlah_pax pada tabel bookings untuk mencatat jumlah pax saat distribusi perlengkapan

ALTER TABLE bookings ADD COLUMN perlengkapan_jumlah_pax INT UNSIGNED NULL AFTER perlengkapan_diberikan_oleh;
