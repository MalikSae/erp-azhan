-- 067_booking_pax_perlengkapan_status.sql
-- Menambahkan kolom status distribusi perlengkapan per pax pada booking_pax jika belum ada
-- (idempotent untuk MySQL 8.0+, pola sama dengan 016).
--
-- Kolom ini dipakai kode sejak perlengkapan dipindah per pax (048), tetapi tidak pernah
-- dibuat oleh migrasi mana pun. Tanpa kolom ini detail booking gagal:
-- "Unknown column 'bp.perlengkapan_status'". Environment yang sudah menambahkannya manual
-- tidak berubah.

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'booking_pax'
    AND COLUMN_NAME = 'perlengkapan_status'
);

SET @sql = IF(
  @col_exists = 0,
  'ALTER TABLE booking_pax ADD COLUMN perlengkapan_status ENUM(''belum_diberikan'',''sudah_diberikan'') NOT NULL DEFAULT ''belum_diberikan'' AFTER progress_vaksin_meningitis',
  'DO 0'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'booking_pax'
    AND COLUMN_NAME = 'perlengkapan_tanggal'
);

SET @sql = IF(
  @col_exists = 0,
  'ALTER TABLE booking_pax ADD COLUMN perlengkapan_tanggal DATE NULL AFTER perlengkapan_status',
  'DO 0'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
