-- NIK agen (K24): wajib saat mengajukan agen. Jadi pembanding "Saya sendiri" di booking portal agen:
-- hanya NIK ini yang boleh dipakai agen untuk komisi atas dirinya sendiri. Satu NIK satu akun per brand,
-- supaya agen tidak bisa mendaftar dengan NIK orang lain.
ALTER TABLE customer_accounts
  ADD COLUMN nik_agen CHAR(16) NULL AFTER domisili,
  ADD UNIQUE KEY uq_customer_brand_nik_agen (brand_id, nik_agen);
