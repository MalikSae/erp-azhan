-- 129_hotel_unik_per_kota.sql
-- Audit menu Paket 10 November 2026 (S-12), keputusan pemilik proyek.
--
-- Nama hotel sebelumnya unik untuk seluruh sistem, sehingga hotel bernama sama di Makkah dan Madinah
-- tidak bisa didaftarkan keduanya. Kunci unik diganti menjadi (name, city).
--
-- Aman dijalankan ulang.

SET @ada_lama = (
  SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'hotels' AND INDEX_NAME = 'name' AND NON_UNIQUE = 0
);
SET @sql = IF(@ada_lama > 0, 'ALTER TABLE hotels DROP INDEX name', 'DO 0');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ada_baru = (
  SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'hotels' AND INDEX_NAME = 'uq_hotels_name_city'
);
SET @sql = IF(@ada_baru = 0, 'ALTER TABLE hotels ADD UNIQUE KEY uq_hotels_name_city (name, city)', 'DO 0');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
