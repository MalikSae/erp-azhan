-- 124_void_pembayaran_izin_diskon.sql
-- Audit menu Jamaah & Booking 10 November 2026 (T-02, T-03).
--
-- T-03: pembayaran terkonfirmasi yang salah catat bisa dibatalkan (void). Baris tidak dihapus:
-- statusnya menjadi 'void' beserta alasan, pelaku, dan waktu. Semua penjumlahan pembayaran
-- memakai status = 'confirmed', jadi pembayaran void otomatis tidak ikut dihitung.
--
-- T-02: total diskon satu booking di atas 10% jumlah harga pax hanya boleh oleh pemegang izin
-- booking.diskon.besar.
--
-- Aman dijalankan ulang.

SET @kolom_ada = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'payments'
    AND COLUMN_NAME = 'void_reason'
);

SET @sql = IF(
  @kolom_ada = 0,
  'ALTER TABLE payments
     MODIFY COLUMN status ENUM(''pending'',''confirmed'',''rejected'',''void'') NOT NULL DEFAULT ''pending'',
     ADD COLUMN void_reason VARCHAR(500) NULL AFTER verified_at,
     ADD COLUMN voided_by BIGINT UNSIGNED NULL AFTER void_reason,
     ADD COLUMN voided_at DATETIME NULL AFTER voided_by,
     ADD CONSTRAINT fk_payment_voided_by FOREIGN KEY (voided_by) REFERENCES admin_users(id) ON DELETE SET NULL',
  'DO 0'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

INSERT IGNORE INTO permissions (code, description) VALUES
  ('payment.void', 'Batalkan pembayaran yang sudah dikonfirmasi'),
  ('booking.diskon.besar', 'Beri diskon booking di atas 10% harga');

-- Super Admin memegang semua izin. Finance mengelola pembayaran, jadi ikut boleh membatalkannya.
-- Izin diskon besar hanya untuk Super Admin; peran lain diatur lewat layar peran.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'super_admin_grup' AND p.code IN ('payment.void', 'booking.diskon.besar');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'finance' AND p.code = 'payment.void';

-- Sesi yang sedang berjalan memuat ulang izinnya.
UPDATE admin_users u
JOIN user_roles ur ON ur.admin_user_id = u.id
JOIN roles r ON r.id = ur.role_id AND r.slug IN ('super_admin_grup', 'finance')
SET u.perm_version = u.perm_version + 1;
