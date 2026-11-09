-- 122_perubahan_data_jamaah.sql
-- Pelanggan mengubah data jamaah dari halaman Data Jamaah. Mengisi isian kosong langsung tersimpan; mengubah isian yang sudah terisi
-- menjadi pengajuan yang menunggu persetujuan admin. Semua perubahan (langsung, pengajuan, edit admin) tercatat di dua tabel ini:
-- jamaah_perubahan (satu baris per simpan) dan jamaah_perubahan_item (tiap isian: nilai lama dan baru).

INSERT IGNORE INTO permissions (code, description) VALUES
  ('jamaah.perubahan.approve', 'Setujui atau tolak perubahan data jamaah dari pelanggan');

-- Grant "semua" di 071 hanyalah snapshot, jadi diberikan eksplisit.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('super_admin_grup', 'admin_travel') AND p.code = 'jamaah.perubahan.approve';

CREATE TABLE IF NOT EXISTS jamaah_perubahan (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  jamaah_id    BIGINT UNSIGNED NOT NULL,
  akun_id      BIGINT UNSIGNED NULL,
  admin_id     BIGINT UNSIGNED NULL,
  sumber       ENUM('pelanggan','admin') NOT NULL,
  status       ENUM('langsung','menunggu','disetujui','ditolak','diganti') NOT NULL,
  dibuat_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diputus_oleh BIGINT UNSIGNED NULL,
  diputus_at   DATETIME NULL,
  catatan      TEXT NULL,
  KEY idx_jamaah_perubahan_jamaah (jamaah_id, dibuat_at),
  KEY idx_jamaah_perubahan_status (status, dibuat_at),
  CONSTRAINT fk_jamaah_perubahan_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah (id),
  CONSTRAINT fk_jamaah_perubahan_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id),
  CONSTRAINT fk_jamaah_perubahan_admin FOREIGN KEY (admin_id) REFERENCES admin_users (id),
  CONSTRAINT fk_jamaah_perubahan_putus FOREIGN KEY (diputus_oleh) REFERENCES admin_users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS jamaah_perubahan_item (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  perubahan_id BIGINT UNSIGNED NOT NULL,
  kolom        VARCHAR(40) NOT NULL,
  nilai_lama   VARCHAR(255) NULL,
  nilai_baru   VARCHAR(255) NULL,
  KEY idx_jamaah_perubahan_item (perubahan_id),
  CONSTRAINT fk_jamaah_perubahan_item FOREIGN KEY (perubahan_id) REFERENCES jamaah_perubahan (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
