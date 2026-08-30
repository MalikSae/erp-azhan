-- 045_draft_status.sql
-- Menambahkan status draft pada tabel jamaah dan memodifikasi tabel bookings untuk mendukung draft booking

-- 1. Tambahkan status pada tabel jamaah
ALTER TABLE jamaah ADD COLUMN status ENUM('draft','aktif') NOT NULL DEFAULT 'aktif' AFTER catatan;

-- 2. Modifikasi kolom status pada tabel bookings agar menerima 'draft'
ALTER TABLE bookings MODIFY COLUMN status ENUM('draft','baru','dp','lunas','batal') NOT NULL DEFAULT 'baru';

-- 3. Modifikasi kolom pic_jamaah_id pada tabel bookings agar bisa NULL (saat booking masih draft)
ALTER TABLE bookings DROP FOREIGN KEY fk_bookings_pic_jamaah;
ALTER TABLE bookings MODIFY COLUMN pic_jamaah_id BIGINT UNSIGNED NULL;
ALTER TABLE bookings ADD CONSTRAINT fk_bookings_pic_jamaah FOREIGN KEY (pic_jamaah_id) REFERENCES jamaah(id) ON DELETE RESTRICT;

-- 4. Modifikasi kolom id_booking pada tabel bookings agar bisa NULL (saat booking masih draft)
ALTER TABLE bookings MODIFY COLUMN id_booking VARCHAR(6) NULL;
