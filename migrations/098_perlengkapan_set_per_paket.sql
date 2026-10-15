-- 098_perlengkapan_set_per_paket.sql
-- Isi perlengkapan tidak lagi satu set global untuk semua paket:
--   1. set bawaan per KATEGORI paket (Low Cost, Reguler, Plus, VIP),
--   2. bisa ditimpa per PAKET (schedules.perlengkapan_custom = TRUE),
--   3. tiap baris set berlaku untuk semua / pria / wanita (jenis kelamin jamaah).
-- perlengkapan_set_template (global) tetap ada sebagai cadangan terakhir bagi paket
-- tanpa kategori dan kategori yang set-nya kosong.
-- Catatan: ALTER TABLE tidak idempotent — jalankan file ini sekali saja.

CREATE TABLE IF NOT EXISTS package_category_perlengkapan (
  category_id          BIGINT UNSIGNED NOT NULL,
  perlengkapan_item_id BIGINT UNSIGNED NOT NULL,
  qty                  INT NOT NULL DEFAULT 1,
  untuk                ENUM('semua','pria','wanita') NOT NULL DEFAULT 'semua',
  PRIMARY KEY (category_id, perlengkapan_item_id),
  KEY idx_pcp_item (perlengkapan_item_id),
  CONSTRAINT fk_pcp_category FOREIGN KEY (category_id) REFERENCES package_categories(id) ON DELETE CASCADE,
  CONSTRAINT fk_pcp_item FOREIGN KEY (perlengkapan_item_id) REFERENCES perlengkapan_items(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS schedule_perlengkapan (
  schedule_id          BIGINT UNSIGNED NOT NULL,
  perlengkapan_item_id BIGINT UNSIGNED NOT NULL,
  qty                  INT NOT NULL DEFAULT 1,
  untuk                ENUM('semua','pria','wanita') NOT NULL DEFAULT 'semua',
  PRIMARY KEY (schedule_id, perlengkapan_item_id),
  KEY idx_sp_item (perlengkapan_item_id),
  CONSTRAINT fk_sp_schedule FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE,
  CONSTRAINT fk_sp_item FOREIGN KEY (perlengkapan_item_id) REFERENCES perlengkapan_items(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE schedules
  ADD COLUMN perlengkapan_custom BOOLEAN NOT NULL DEFAULT FALSE;

-- Set bawaan tiap kategori = set global yang berlaku sekarang (perilaku tidak berubah).
INSERT IGNORE INTO package_category_perlengkapan (category_id, perlengkapan_item_id, qty, untuk)
SELECT c.id, t.perlengkapan_item_id, t.qty, 'semua'
FROM package_categories c
CROSS JOIN perlengkapan_set_template t;
