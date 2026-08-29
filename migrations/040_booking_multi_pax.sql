-- 040_booking_multi_pax.sql
-- Migrasi struktur booking menuju arsitektur Header + Detail (booking_pax)

-- 1a. Buat tabel detail booking_pax
CREATE TABLE IF NOT EXISTS booking_pax (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  booking_id BIGINT UNSIGNED NOT NULL,
  jamaah_id BIGINT UNSIGNED NOT NULL,
  pax_type ENUM('reguler','infant') NOT NULL DEFAULT 'reguler',
  room_type ENUM('Quad','Triple','Double') NULL,
  harga_pax DECIMAL(12,0) NOT NULL DEFAULT 0,
  counts_for_seat BOOLEAN NOT NULL DEFAULT TRUE,
  pax_status ENUM('aktif','batal') NOT NULL DEFAULT 'aktif',
  progress_visa BOOLEAN NOT NULL DEFAULT FALSE,
  progress_siskopatuh BOOLEAN NOT NULL DEFAULT FALSE,
  progress_manasik BOOLEAN NOT NULL DEFAULT FALSE,
  progress_vaksin_meningitis BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE,
  FOREIGN KEY (jamaah_id) REFERENCES jamaah(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 1b. Migrasi data existing ke booking_pax
INSERT INTO booking_pax (
  booking_id, 
  jamaah_id, 
  pax_type, 
  room_type, 
  harga_pax, 
  counts_for_seat, 
  pax_status, 
  progress_visa, 
  progress_siskopatuh, 
  progress_manasik, 
  progress_vaksin_meningitis, 
  created_at
)
SELECT 
  id, 
  jamaah_id, 
  'reguler', 
  room_type, 
  COALESCE(harga_dasar, 0), 
  TRUE, 
  CASE WHEN status = 'batal' THEN 'batal' ELSE 'aktif' END,
  progress_visa, 
  progress_siskopatuh, 
  progress_manasik, 
  progress_vaksin_meningitis, 
  created_at
FROM bookings;

-- 1c. ALTER TABLE bookings — drop FK lama, rename jamaah_id ke pic_jamaah_id, add FK baru, dan drop kolom yang pindah ke detail
ALTER TABLE bookings DROP FOREIGN KEY bookings_ibfk_2;

ALTER TABLE bookings
  CHANGE COLUMN jamaah_id pic_jamaah_id BIGINT UNSIGNED NOT NULL,
  ADD CONSTRAINT fk_bookings_pic_jamaah FOREIGN KEY (pic_jamaah_id) REFERENCES jamaah(id) ON DELETE RESTRICT,
  DROP COLUMN room_type,
  DROP COLUMN harga_dasar,
  DROP COLUMN progress_paspor,
  DROP COLUMN progress_tiket,
  DROP COLUMN progress_visa,
  DROP COLUMN progress_siskopatuh,
  DROP COLUMN progress_manasik,
  DROP COLUMN progress_vaksin_meningitis;
