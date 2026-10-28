-- 111_akun_pelanggan_hp_unik.sql
-- Nomor WhatsApp akun pelanggan unik per brand, sama seperti email. Penyamaan format memakai aturan yang sama
-- dengan jamaah (migrasi 060): angka saja, awalan 0 menjadi 62, sehingga 0812..., 62812..., dan +62 812... dianggap satu nomor.
-- Prasyarat: tidak ada dua akun di brand yang sama dengan nomor ternormalisasi sama (bereskan dulu bila ada).

ALTER TABLE customer_accounts
  ADD COLUMN no_hp_normal VARCHAR(20) GENERATED ALWAYS AS (
    CASE
      WHEN REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', '') = '' THEN NULL
      WHEN REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', '') LIKE '0%'
        THEN CONCAT('62', SUBSTRING(REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', ''), 2))
      ELSE REGEXP_REPLACE(COALESCE(no_hp, ''), '[^0-9]', '')
    END
  ) STORED AFTER no_hp,
  ADD UNIQUE KEY uq_customer_brand_hp (brand_id, no_hp_normal);
