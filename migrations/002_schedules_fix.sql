-- migrations/002_schedules_fix.sql
-- Perbaikan skema tabel schedules:
-- 1. Hapus kolom jadwal_id (identifier kini HANYA pakai id auto-increment)
-- 2. Tambahkan kolom status ENUM untuk lifecycle management paket

ALTER TABLE schedules DROP COLUMN jadwal_id;
ALTER TABLE schedules ADD COLUMN status ENUM('draft','published','archived') NOT NULL DEFAULT 'draft' AFTER jadwal_nama;
