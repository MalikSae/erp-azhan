-- Migration 019: Tambah kode_jamaah 6 karakter unik untuk autentikasi Portal Jamaah
ALTER TABLE jamaah ADD COLUMN kode_jamaah VARCHAR(6) NOT NULL UNIQUE AFTER brand_id;
