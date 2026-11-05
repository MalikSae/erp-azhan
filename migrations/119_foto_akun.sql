-- 119_foto_akun.sql
-- Satu foto per akun pelanggan, tidak peduli sudah menjadi agen atau belum. Kolom foto agen diganti namanya
-- (isi lama ikut terbawa); foto tidak lagi wajib saat mendaftar agen.
ALTER TABLE customer_accounts RENAME COLUMN foto_agen_url TO foto_url;
