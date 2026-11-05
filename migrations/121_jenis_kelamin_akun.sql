-- 121_jenis_kelamin_akun.sql
-- Jenis kelamin akun pelanggan (L atau P), diisi saat mendaftar agen. Tanggal lahir memakai kolom dari migrasi 120.
ALTER TABLE customer_accounts ADD COLUMN jenis_kelamin ENUM('L','P') NULL AFTER tanggal_lahir;
