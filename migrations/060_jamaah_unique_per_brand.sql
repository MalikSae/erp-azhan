-- 060: Keunikan jamaah per brand (prasyarat fitur Agen/Syiar, keputusan D2).
-- - no_hp_normal: generated column dari no_hp (hanya digit, awalan 0 -> 62),
--   sama dengan shared.PhoneVariants. Otomatis konsisten di semua jalur insert.
-- - UNIQUE(brand_id, no_hp_normal): satu nomor HP = satu jamaah per brand.
--   no_hp kosong/NULL -> no_hp_normal NULL (boleh banyak).
-- - nik: dari UNIQUE global menjadi UNIQUE(brand_id, nik), karena orang yang
--   sama di brand lain adalah jamaah baru.
-- Migrasi gagal (1062) jika masih ada duplikat; bersihkan data dulu.
-- Catatan drift: sebagian database (termasuk erp_azhan_dev) sudah punya key
-- uq_jamaah_brand_phone (brand_id, no_hp mentah) yang dulu ditambahkan manual,
-- bukan lewat migrasi. Key itu tidak dihapus di sini (MySQL tidak punya
-- DROP INDEX IF EXISTS); keberadaannya aman karena uq_jamaah_brand_hp lebih ketat.
ALTER TABLE jamaah
  ADD COLUMN no_hp_normal VARCHAR(20) GENERATED ALWAYS AS (
    CASE
      WHEN REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', '') = '' THEN NULL
      WHEN REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', '') LIKE '0%'
        THEN CONCAT('62', SUBSTRING(REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', ''), 2))
      ELSE REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', '')
    END
  ) STORED AFTER no_hp;

ALTER TABLE jamaah ADD UNIQUE KEY uq_jamaah_brand_hp (brand_id, no_hp_normal);

ALTER TABLE jamaah DROP INDEX nik, ADD UNIQUE KEY uq_jamaah_brand_nik (brand_id, nik);
