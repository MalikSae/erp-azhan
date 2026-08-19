-- 1. Tabel stok baru (per brand per item)
CREATE TABLE perlengkapan_stok (
  brand_id                BIGINT UNSIGNED NOT NULL,
  perlengkapan_item_id    BIGINT UNSIGNED NOT NULL,
  stok_tersedia           INT NOT NULL DEFAULT 0,
  PRIMARY KEY (brand_id, perlengkapan_item_id),
  FOREIGN KEY (brand_id) REFERENCES brands(id),
  FOREIGN KEY (perlengkapan_item_id) REFERENCES perlengkapan_items(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- 2. Pindahkan data stok existing ke tabel baru (brand_id + stok_tersedia dari baris lama)
INSERT INTO perlengkapan_stok (brand_id, perlengkapan_item_id, stok_tersedia)
SELECT brand_id, id, stok_tersedia FROM perlengkapan_items;

-- 3. Drop tabel set lama jika ada
DROP TABLE IF EXISTS perlengkapan_set_items;
DROP TABLE IF EXISTS perlengkapan_set_template;

-- 4. perlengkapan_items jadi GLOBAL — hapus kolom brand_id & stok_tersedia
ALTER TABLE perlengkapan_items DROP FOREIGN KEY perlengkapan_items_ibfk_1;
ALTER TABLE perlengkapan_items DROP INDEX unique_brand_nama;
ALTER TABLE perlengkapan_items DROP COLUMN brand_id;
ALTER TABLE perlengkapan_items DROP COLUMN stok_tersedia;
ALTER TABLE perlengkapan_items ADD UNIQUE KEY unique_nama_item (nama);

-- 5. Set template: ganti referensi jadi perlengkapan_item_id (FK)
CREATE TABLE perlengkapan_set_template (
  perlengkapan_item_id  BIGINT UNSIGNED PRIMARY KEY,
  qty                   INT NOT NULL DEFAULT 1,
  FOREIGN KEY (perlengkapan_item_id) REFERENCES perlengkapan_items(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- 6. Seed perlengkapan_set_template dari data existing
INSERT INTO perlengkapan_set_template (perlengkapan_item_id, qty)
SELECT id, 1 FROM perlengkapan_items;

-- 7. Backfill perlengkapan_stok untuk SEMUA kombinasi brand x item yang belum ada (stok_tersedia=0)
INSERT INTO perlengkapan_stok (brand_id, perlengkapan_item_id, stok_tersedia)
SELECT b.id, i.id, 0
FROM brands b CROSS JOIN perlengkapan_items i
WHERE NOT EXISTS (
  SELECT 1 FROM perlengkapan_stok s WHERE s.brand_id = b.id AND s.perlengkapan_item_id = i.id
);
