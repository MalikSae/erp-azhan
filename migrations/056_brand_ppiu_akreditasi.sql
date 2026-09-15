-- Migration 056: Tambah kolom ppiu_number dan akreditasi ke tabel brands.
-- Menggantikan nilai hardcode di selfbooking/repository.go.
ALTER TABLE brands
    ADD COLUMN ppiu_number  VARCHAR(50)  NULL AFTER legalitas,
    ADD COLUMN akreditasi   VARCHAR(10)  NULL AFTER ppiu_number;
