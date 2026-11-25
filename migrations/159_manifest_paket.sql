-- 159: Manifest paket Fase 1 (docs/spek/SPEK-MANIFEST-PAKET-2026-11-25.md).
--
-- Nama paspor jamaah (kosong = pakai nama_lengkap), data manifest per paket (PNR dan detail penerbangan
-- yang ditempel dari maskapai), jejak unduhan, dan izin Kelola manifest.
ALTER TABLE jamaah ADD COLUMN nama_paspor VARCHAR(255) NULL AFTER nama_lengkap;

CREATE TABLE IF NOT EXISTS manifest_paket (
  schedule_id         BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  pnr                 VARCHAR(100) NULL,
  detail_penerbangan  TEXT NULL,
  diubah_oleh         BIGINT UNSIGNED NULL,
  diubah_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  CONSTRAINT fk_manifest_paket_schedule FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE CASCADE,
  CONSTRAINT fk_manifest_paket_admin FOREIGN KEY (diubah_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS manifest_unduhan (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  schedule_id    BIGINT UNSIGNED NOT NULL,
  admin_user_id  BIGINT UNSIGNED NULL,
  jumlah_pax     INT UNSIGNED NOT NULL,
  dibuat_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_manifest_unduhan_paket (schedule_id, dibuat_at),
  CONSTRAINT fk_manifest_unduhan_schedule FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE CASCADE,
  CONSTRAINT fk_manifest_unduhan_admin FOREIGN KEY (admin_user_id) REFERENCES admin_users (id) ON DELETE SET NULL
);

INSERT IGNORE INTO permissions (code, description) VALUES
  ('manifest.manage', 'Kelola manifest paket: PNR, detail penerbangan, dan unduh');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('super_admin_grup', 'product_manager', 'admin_travel') AND p.code = 'manifest.manage';

UPDATE admin_users u
JOIN user_roles ur ON ur.admin_user_id = u.id
JOIN roles r ON r.id = ur.role_id AND r.slug IN ('super_admin_grup', 'product_manager', 'admin_travel')
SET u.perm_version = u.perm_version + 1;
