-- 160_akses_pengguna.sql
-- Akses pengguna disederhanakan (SPEK-AKSES-PENGGUNA-2026-11-25.md): dua jenis
-- akses, Super Admin dan Tim; izin Tim disimpan per pengguna. Role, struktur
-- organisasi, jabatan, dan kolom role lama dihapus.
-- Catatan: ALTER TABLE dan DROP TABLE tidak idempotent — jalankan file ini
-- sekali saja. Data disalin dulu sebelum tabel lama dihapus.

ALTER TABLE admin_users
  ADD COLUMN is_super_admin BOOLEAN NOT NULL DEFAULT FALSE AFTER brand_id;

CREATE TABLE IF NOT EXISTS user_permissions (
  admin_user_id BIGINT UNSIGNED NOT NULL,
  permission_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (admin_user_id, permission_id),
  CONSTRAINT fk_up_user FOREIGN KEY (admin_user_id) REFERENCES admin_users(id) ON DELETE CASCADE,
  CONSTRAINT fk_up_perm FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Izin yang tidak menjaga route apa pun (role_permissions ikut lewat FK cascade).
DELETE FROM permissions
WHERE code LIKE 'marketing.%' OR code LIKE 'compliance.%'
   OR code IN ('paket.view', 'paket.edit', 'paket.approve', 'audit.view', 'report.finance.view');

-- Pemegang role Super Admin menjadi Super Admin (hanya pengguna Holding).
UPDATE admin_users u
JOIN user_roles ur ON ur.admin_user_id = u.id
JOIN roles r ON r.id = ur.role_id
SET u.is_super_admin = TRUE
WHERE r.slug = 'super_admin_grup' AND u.brand_id IS NULL;

-- Pengguna lain menjadi Tim dengan gabungan izin role lamanya, kecuali izin
-- yang kini hanya milik Super Admin.
INSERT IGNORE INTO user_permissions (admin_user_id, permission_id)
SELECT DISTINCT ur.admin_user_id, rp.permission_id
FROM user_roles ur
JOIN admin_users u ON u.id = ur.admin_user_id
JOIN role_permissions rp ON rp.role_id = ur.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE u.is_super_admin = FALSE
  AND p.code NOT IN ('user.admin', 'brand.edit', 'crmdeal.process');

-- Pengguna terikat brand hanya boleh memegang izin brand (A5; daftar sama dengan
-- izinBrand di internal/akses). Pemegang role Holding yang terikat brand tidak
-- membawa izin Holding-nya.
DELETE up FROM user_permissions up
JOIN admin_users u ON u.id = up.admin_user_id
JOIN permissions p ON p.id = up.permission_id
WHERE u.brand_id IS NOT NULL
  AND p.code NOT IN (
    'jamaah.view', 'jamaah.edit', 'dokumen.view', 'dokumen.edit',
    'booking.view', 'booking.edit',
    'payment.view', 'payment.edit', 'refund.view', 'refund.edit',
    'akun.view', 'akun.edit',
    'agen.view', 'agen.edit', 'komisi.view',
    'schedule.view', 'manifest.manage', 'manasik.view', 'manasik.edit',
    'masterdata.view', 'perlengkapan.view', 'perlengkapan.edit',
    'bankaccount.view', 'bankaccount.edit', 'brand.view',
    'pengumuman.manage', 'banner.manage',
    'payment.verify', 'payment.void', 'agen.approve', 'jamaah.perubahan.approve');

ALTER TABLE roles DROP FOREIGN KEY fk_roles_divisi;
DROP TABLE user_roles;
DROP TABLE role_permissions;
DROP TABLE roles;
DROP TABLE org_member;
DROP TABLE org_unit;

ALTER TABLE admin_users
  DROP INDEX idx_admin_users_brand_role_active,
  ADD INDEX idx_admin_users_brand_active (brand_id, is_active),
  DROP COLUMN role,
  DROP COLUMN jabatan;

-- Klaim token lama (role, roles) tidak berlaku lagi: paksa muat ulang.
UPDATE admin_users SET perm_version = perm_version + 1;
