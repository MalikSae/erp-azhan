-- 099_perlengkapan_mutasi_stok.sql
-- Perlengkapan dibeli terpusat oleh holding lalu dikirim ke brand:
--   * perlengkapan_stok_pusat : stok gudang pusat (holding) per item,
--   * perlengkapan_mutasi     : buku besar semua pergerakan stok (pusat & brand) — siapa,
--                               kapan, berapa, dan stok setelahnya.
-- perlengkapan_stok (per brand) tetap menjadi saldo berjalan; tiap perubahan kini
-- tercatat di perlengkapan_mutasi. varian_id disiapkan (NULL) untuk varian dinamis.
-- Catatan: jalankan file ini sekali saja.

CREATE TABLE IF NOT EXISTS perlengkapan_stok_pusat (
  perlengkapan_item_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  stok_tersedia        INT NOT NULL DEFAULT 0,
  CONSTRAINT fk_stok_pusat_item FOREIGN KEY (perlengkapan_item_id) REFERENCES perlengkapan_items(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS perlengkapan_mutasi (
  id                   BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  perlengkapan_item_id BIGINT UNSIGNED NOT NULL,
  varian_id            BIGINT UNSIGNED NULL,
  brand_id             BIGINT UNSIGNED NULL COMMENT 'NULL = gudang pusat',
  jenis                ENUM('pembelian','transfer','distribusi','retur','koreksi') NOT NULL,
  qty                  INT NOT NULL COMMENT 'positif = masuk, negatif = keluar',
  stok_setelah         INT NOT NULL,
  harga_satuan         DECIMAL(15,2) NULL,
  vendor               VARCHAR(150) NULL,
  catatan              VARCHAR(255) NULL,
  referensi_tipe       VARCHAR(30) NULL,
  referensi_id         BIGINT UNSIGNED NULL,
  created_by           BIGINT UNSIGNED NULL,
  created_at           TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_mutasi_item (perlengkapan_item_id, created_at),
  KEY idx_mutasi_brand (brand_id, created_at),
  KEY idx_mutasi_ref (referensi_tipe, referensi_id),
  CONSTRAINT fk_mutasi_item FOREIGN KEY (perlengkapan_item_id) REFERENCES perlengkapan_items(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Saldo awal dari stok yang sudah ada, agar riwayat tidak dimulai dari nol.
INSERT INTO perlengkapan_mutasi (perlengkapan_item_id, brand_id, jenis, qty, stok_setelah, catatan)
SELECT s.perlengkapan_item_id, s.brand_id, 'koreksi', s.stok_tersedia, s.stok_tersedia, 'Saldo awal'
FROM perlengkapan_stok s
WHERE s.stok_tersedia <> 0;
