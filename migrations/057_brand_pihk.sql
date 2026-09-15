-- Migration 057: Tambah kolom pihk_number ke tabel brands.
ALTER TABLE brands
    ADD COLUMN pihk_number VARCHAR(50) NULL AFTER ppiu_number;
