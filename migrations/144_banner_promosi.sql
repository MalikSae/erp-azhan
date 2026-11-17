-- 144: Banner Promosi untuk Beranda app Hana (keputusan 15 November 2026).
--
-- Banner: gambar jadi buatan desainer (rasio 2:1, lebar minimal 1200 px, maksimal 1 MB), satu tujuan saat ditekan,
-- jadwal tayang, urutan manual, dan hitungan klik. App menampilkan maksimal 5 banner aktif per brand.
-- Strip oranye di atas carousel tidak dikelola di sini: app menyusunnya dari kategori low cost (slug hemat).
CREATE TABLE IF NOT EXISTS banner_promosi (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id     BIGINT UNSIGNED NOT NULL,
  judul        VARCHAR(100) NOT NULL,
  gambar_url   VARCHAR(500) NOT NULL,
  -- tidak_ada | paket (id jadwal) | jenis_paket (umroh, low-cost, private, wisata, haji) | whatsapp (pesan opsional) | tautan (https)
  tujuan       VARCHAR(20) NOT NULL DEFAULT 'tidak_ada',
  tujuan_nilai VARCHAR(500) NULL,
  mulai        DATE NULL,
  selesai      DATE NULL,
  aktif        BOOLEAN NOT NULL DEFAULT TRUE,
  urutan       INT NOT NULL DEFAULT 0,
  klik         INT UNSIGNED NOT NULL DEFAULT 0,
  dibuat_oleh  BIGINT UNSIGNED NULL,
  diubah_oleh  BIGINT UNSIGNED NULL,
  dibuat_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diubah_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_banner_brand_urutan (brand_id, urutan),
  CONSTRAINT fk_banner_brand FOREIGN KEY (brand_id) REFERENCES brands (id),
  CONSTRAINT fk_banner_dibuat FOREIGN KEY (dibuat_oleh) REFERENCES admin_users (id) ON DELETE SET NULL,
  CONSTRAINT fk_banner_diubah FOREIGN KEY (diubah_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
);

INSERT IGNORE INTO permissions (code, description) VALUES
  ('banner.manage', 'Kelola banner promosi app');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('super_admin_grup', 'admin_travel', 'cmo', 'marketing_lead') AND p.code = 'banner.manage';

UPDATE admin_users u
JOIN user_roles ur ON ur.admin_user_id = u.id
JOIN roles r ON r.id = ur.role_id AND r.slug IN ('super_admin_grup', 'admin_travel', 'cmo', 'marketing_lead')
SET u.perm_version = u.perm_version + 1;
