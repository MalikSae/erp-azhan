-- 015_booking_addons_diskon.sql
-- Menambahkan kolom harga_dasar, diskon, diskon_keterangan pada bookings dan tabel booking_addons

ALTER TABLE bookings 
  ADD COLUMN harga_dasar DECIMAL(12,0) NULL AFTER room_type,
  ADD COLUMN diskon DECIMAL(12,0) NOT NULL DEFAULT 0 AFTER total_harga,
  ADD COLUMN diskon_keterangan VARCHAR(255) NULL AFTER diskon;

CREATE TABLE IF NOT EXISTS booking_addons (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  booking_id  BIGINT UNSIGNED NOT NULL,
  nama        VARCHAR(255) NOT NULL,
  nominal     DECIMAL(12,0) NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Backfill harga_dasar untuk booking yang sudah ada
UPDATE bookings b
JOIN schedules s ON s.id = b.schedule_id
SET b.harga_dasar = CASE 
  WHEN b.room_type = 'Quad' THEN s.harga_quad
  WHEN b.room_type = 'Triple' THEN s.harga_triple
  WHEN b.room_type = 'Double' THEN s.harga_double
  ELSE b.total_harga
END
WHERE b.harga_dasar IS NULL;
