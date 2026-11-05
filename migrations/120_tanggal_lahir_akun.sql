-- 120_tanggal_lahir_akun.sql
-- Tanggal lahir akun pelanggan (opsional, diisi pemilik akun di Profil). Kota domisili memakai kolom domisili yang sudah ada.
ALTER TABLE customer_accounts ADD COLUMN tanggal_lahir DATE NULL AFTER domisili;
