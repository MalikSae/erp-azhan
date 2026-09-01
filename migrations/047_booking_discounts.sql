-- 047_booking_discounts.sql
-- Memisahkan diskon menjadi tabel sendiri (one-to-many) seperti addons

CREATE TABLE IF NOT EXISTS booking_discounts (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  booking_id  BIGINT UNSIGNED NOT NULL,
  nama        VARCHAR(255) NOT NULL,
  nominal     DECIMAL(12,0) NOT NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Pindahkan data diskon yang sudah ada ke tabel baru
INSERT INTO booking_discounts (booking_id, nama, nominal, created_at)
SELECT 
    id, 
    COALESCE(diskon_keterangan, 'Diskon Khusus'), 
    diskon,
    created_at
FROM bookings 
WHERE diskon > 0;

-- Hapus kolom lama dari bookings
ALTER TABLE bookings
  DROP COLUMN diskon,
  DROP COLUMN diskon_keterangan;
