-- 081_pengguna_organisasi.sql
-- Pengguna & Struktur Organisasi (SPEK-PENGGUNA-ORGANISASI-2026-10-03.md §3).
-- Satu orang = satu Pengguna (admin_users). Keanggotaan unit untuk penugasan;
-- jabatan hanya label tampilan. Catatan: ALTER TABLE tidak idempotent —
-- jalankan file ini sekali saja.

-- Profil pengguna (display_name sudah ada; dipakai sebagai "Nama").
ALTER TABLE admin_users
  ADD COLUMN no_hp   VARCHAR(32)  NULL AFTER display_name,
  ADD COLUMN jabatan VARCHAR(100) NULL AFTER no_hp;

-- Backfill nama kosong dari bagian lokal email (tampilan saja; bisa diedit).
UPDATE admin_users SET display_name = SUBSTRING_INDEX(email, '@', 1)
WHERE display_name IS NULL OR display_name = '';

-- Struktur organisasi: pohon 2 tingkat (divisi -> tim). Aturan tingkat
-- ditegakkan di service (R8).
CREATE TABLE IF NOT EXISTS org_unit (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  parent_id  BIGINT UNSIGNED NULL,
  jenis      ENUM('divisi','tim') NOT NULL,
  nama       VARCHAR(120) NOT NULL,
  urutan     INT NOT NULL DEFAULT 0,
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_org_parent FOREIGN KEY (parent_id) REFERENCES org_unit(id) ON DELETE RESTRICT
) ENGINE=InnoDB;

-- Keanggotaan: many-to-many, posisi per keanggotaan (boleh banyak pemimpin).
CREATE TABLE IF NOT EXISTS org_member (
  admin_user_id BIGINT UNSIGNED NOT NULL,
  org_unit_id   BIGINT UNSIGNED NOT NULL,
  posisi        ENUM('pemimpin','anggota') NOT NULL DEFAULT 'anggota',
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (admin_user_id, org_unit_id),
  CONSTRAINT fk_om_user FOREIGN KEY (admin_user_id) REFERENCES admin_users(id) ON DELETE CASCADE,
  CONSTRAINT fk_om_unit FOREIGN KEY (org_unit_id) REFERENCES org_unit(id) ON DELETE RESTRICT
) ENGINE=InnoDB;

-- ── Data awal (seed sekali; setelahnya dikelola lewat UI) ────────────────────
INSERT INTO org_unit (parent_id, jenis, nama, urutan)
SELECT NULL, 'divisi', s.nama, s.urutan FROM (
  SELECT 'Manajemen' nama, 1 urutan
  UNION ALL SELECT 'Marketing & Growth', 2
  UNION ALL SELECT 'Keuangan', 3
  UNION ALL SELECT 'Operasional', 4
  UNION ALL SELECT 'Produk', 5
  UNION ALL SELECT 'IT', 6
) s
WHERE NOT EXISTS (SELECT 1 FROM org_unit u WHERE u.jenis = 'divisi' AND u.nama = s.nama);

-- Enam tim Marketing & Growth sesuai SOP M&G A3.
INSERT INTO org_unit (parent_id, jenis, nama, urutan)
SELECT d.id, 'tim', s.nama, s.urutan
FROM org_unit d
JOIN (
  SELECT 'Brand & Community' nama, 1 urutan
  UNION ALL SELECT 'Creative Production', 2
  UNION ALL SELECT 'SEO', 3
  UNION ALL SELECT 'Performance Marketing', 4
  UNION ALL SELECT 'Sales & Customer Acquisition', 5
  UNION ALL SELECT 'Web Dev & MarTech', 6
) s
WHERE d.jenis = 'divisi' AND d.nama = 'Marketing & Growth' AND d.parent_id IS NULL
  AND NOT EXISTS (SELECT 1 FROM org_unit u WHERE u.parent_id = d.id AND u.nama = s.nama);
