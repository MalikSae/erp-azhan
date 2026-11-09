-- 123_pengumuman.sql
-- Pengumuman untuk pelanggan, tampil di dashboard akun (web). Ditulis admin per brand dengan masa tayang; bila schedule_id terisi, hanya
-- tampil untuk akun yang punya booking di jadwal keberangkatan itu, bila kosong tampil untuk semua pelanggan brand.

INSERT IGNORE INTO permissions (code, description) VALUES
  ('pengumuman.manage', 'Kelola pengumuman untuk pelanggan (tambah, ubah, hapus)');

-- Grant "semua" di 071 hanyalah snapshot, jadi diberikan eksplisit.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('super_admin_grup', 'admin_travel') AND p.code = 'pengumuman.manage';

CREATE TABLE IF NOT EXISTS pengumuman (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id    BIGINT UNSIGNED NOT NULL,
  schedule_id BIGINT UNSIGNED NULL,
  judul       VARCHAR(120) NOT NULL,
  isi         TEXT NOT NULL,
  mulai_tayang DATE NOT NULL,
  akhir_tayang DATE NOT NULL,
  dibuat_oleh BIGINT UNSIGNED NULL,
  dibuat_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diubah_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_pengumuman_brand_tayang (brand_id, mulai_tayang, akhir_tayang),
  KEY idx_pengumuman_schedule (schedule_id),
  CONSTRAINT fk_pengumuman_brand FOREIGN KEY (brand_id) REFERENCES brands (id),
  CONSTRAINT fk_pengumuman_schedule FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE CASCADE,
  CONSTRAINT fk_pengumuman_admin FOREIGN KEY (dibuat_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
