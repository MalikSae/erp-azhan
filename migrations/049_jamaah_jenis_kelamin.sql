-- 049_jamaah_jenis_kelamin.sql
-- Menambahkan kolom jenis_kelamin pada tabel jamaah

ALTER TABLE jamaah 
ADD COLUMN jenis_kelamin ENUM('L', 'P') NULL AFTER nama_ayah_kandung;
