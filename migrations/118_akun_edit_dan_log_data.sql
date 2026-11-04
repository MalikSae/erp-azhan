-- 118_akun_edit_dan_log_data.sql
-- Admin boleh mengubah nama, nomor WhatsApp, domisili, dan NIK agen sebuah akun (layar Akun Pelanggan).
-- Izin baru akun.edit untuk nama, WhatsApp, dan domisili; perubahan NIK agen memakai izin yang sudah ada
-- (agen.approve untuk pengajuan, agen.kaitan.edit untuk agen aktif). Tiap perubahan tercatat di akun_data_log.

INSERT IGNORE INTO permissions (code, description) VALUES
  ('akun.edit', 'Ubah nama, WhatsApp, dan domisili akun pelanggan');

-- Grant "semua" di 071 hanyalah snapshot, jadi super_admin_grup diberi eksplisit.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('super_admin_grup', 'admin_travel') AND p.code = 'akun.edit';

CREATE TABLE IF NOT EXISTS akun_data_log (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  akun_id     BIGINT UNSIGNED NOT NULL,
  kolom       VARCHAR(30) NOT NULL,
  nilai_lama  VARCHAR(190) NULL,
  nilai_baru  VARCHAR(190) NULL,
  diubah_oleh BIGINT UNSIGNED NOT NULL,
  diubah_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alasan      TEXT NOT NULL,
  KEY idx_akun_data_log_akun (akun_id),
  CONSTRAINT fk_akun_data_log_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id),
  CONSTRAINT fk_akun_data_log_admin FOREIGN KEY (diubah_oleh) REFERENCES admin_users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
