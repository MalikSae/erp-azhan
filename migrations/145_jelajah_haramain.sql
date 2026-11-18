-- 145: Jelajah Haramain (keputusan 16 November 2026).
--
-- Tempat di Makkah dan Madinah, global untuk semua brand, dikelola holding. Web dan app membaca API publik yang sama.
-- Ulasan = pengalaman jamaah, tanpa bintang (teks + maks 3 foto), hanya dari akun yang jamaahnya sudah berangkat; satu
-- ulasan per akun per tempat; langsung tayang, bisa dilaporkan, admin bisa menyembunyikan.
CREATE TABLE IF NOT EXISTS haramain_tempat (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  nama         VARCHAR(100) NOT NULL,
  nama_lengkap VARCHAR(150) NULL,
  -- masjid | bersejarah | ziarah | belanja | kuliner | lainnya
  kategori     VARCHAR(20) NOT NULL,
  -- makkah | madinah
  kota         VARCHAR(10) NOT NULL,
  foto_url     VARCHAR(500) NOT NULL,
  -- JSON array URL foto galeri (maks 10), urut tampil.
  galeri       JSON NULL,
  ringkas      VARCHAR(100) NOT NULL,
  deskripsi    TEXT NULL,
  alamat       VARCHAR(300) NULL,
  jam_buka     VARCHAR(60) NULL,
  gmaps_url    VARCHAR(500) NULL,
  populer      BOOLEAN NOT NULL DEFAULT FALSE,
  aktif        BOOLEAN NOT NULL DEFAULT TRUE,
  urutan       INT NOT NULL DEFAULT 0,
  diubah_oleh  BIGINT UNSIGNED NULL,
  dibuat_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diubah_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_haramain_urutan (aktif, urutan),
  CONSTRAINT fk_haramain_diubah FOREIGN KEY (diubah_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
);

CREATE TABLE IF NOT EXISTS haramain_ulasan (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  tempat_id           BIGINT UNSIGNED NOT NULL,
  akun_id             BIGINT UNSIGNED NOT NULL,
  teks                TEXT NOT NULL,
  -- JSON array URL foto (maks 3).
  foto                JSON NULL,
  disembunyikan       BOOLEAN NOT NULL DEFAULT FALSE,
  disembunyikan_oleh  BIGINT UNSIGNED NULL,
  -- Laporan pengguna: jumlah pelapor dan waktu laporan terakhir; ditinjau_at = admin memilih Abaikan.
  jumlah_laporan      INT UNSIGNED NOT NULL DEFAULT 0,
  laporan_terakhir_at DATETIME NULL,
  ditinjau_at         DATETIME NULL,
  dibuat_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diubah_at           DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_haramain_ulasan_akun (tempat_id, akun_id),
  CONSTRAINT fk_haramain_ulasan_tempat FOREIGN KEY (tempat_id) REFERENCES haramain_tempat (id) ON DELETE CASCADE,
  CONSTRAINT fk_haramain_ulasan_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE,
  CONSTRAINT fk_haramain_ulasan_admin FOREIGN KEY (disembunyikan_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
);

-- Satu laporan per akun per ulasan.
CREATE TABLE IF NOT EXISTS haramain_laporan (
  ulasan_id  BIGINT UNSIGNED NOT NULL,
  akun_id    BIGINT UNSIGNED NOT NULL,
  dibuat_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (ulasan_id, akun_id),
  CONSTRAINT fk_haramain_laporan_ulasan FOREIGN KEY (ulasan_id) REFERENCES haramain_ulasan (id) ON DELETE CASCADE,
  CONSTRAINT fk_haramain_laporan_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE
);

INSERT IGNORE INTO permissions (code, description) VALUES
  ('haramain.manage', 'Kelola Jelajah Haramain dan ulasan jamaah');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('super_admin_grup', 'konten_seo', 'cmo') AND p.code = 'haramain.manage';

UPDATE admin_users u
JOIN user_roles ur ON ur.admin_user_id = u.id
JOIN roles r ON r.id = ur.role_id AND r.slug IN ('super_admin_grup', 'konten_seo', 'cmo')
SET u.perm_version = u.perm_version + 1;
