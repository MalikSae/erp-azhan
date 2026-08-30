-- 044_jamaah_paspor_issued_date.sql
-- Menambahkan kolom tanggal_paspor_keluar (issued date) pada tabel jamaah

ALTER TABLE jamaah 
ADD COLUMN tanggal_paspor_keluar DATE NULL AFTER tempat_paspor_keluar;
