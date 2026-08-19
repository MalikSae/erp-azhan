-- 016_payment_bukti_url.sql
-- Menambahkan kolom bukti_url pada tabel payments jika belum ada (idempotent untuk MySQL 8.0+)

SET @col_exists = (
  SELECT COUNT(*) 
  FROM information_schema.COLUMNS 
  WHERE TABLE_SCHEMA = DATABASE() 
    AND TABLE_NAME = 'payments' 
    AND COLUMN_NAME = 'bukti_url'
);

SET @sql = IF(
  @col_exists = 0,
  'ALTER TABLE payments ADD COLUMN bukti_url VARCHAR(500) NULL AFTER status',
  'DO 0'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
