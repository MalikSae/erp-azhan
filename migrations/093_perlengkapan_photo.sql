-- 093_perlengkapan_photo.sql
-- Foto item perlengkapan di katalog (opsional).
ALTER TABLE perlengkapan_items
  ADD COLUMN photo_url VARCHAR(500) NULL AFTER nama;
