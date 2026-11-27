-- 165: layanan tambahan per booking (SPEK-LAYANAN-TAMBAHAN-BOOKING-2026-11-27).
--
-- Biaya layanan yang diinput admin ke sebuah booking, misalnya kereta cepat: nama teks bebas dan nominal
-- total untuk booking itu. Kebalikan booking_discounts:
--   total_harga = GREATEST(0, SUM(harga_pax) + SUM(booking_layanan.nominal) - SUM(booking_discounts.nominal))
-- dibuat_oleh: admin yang menambahkan (jejak). Layanan yang dihapus tidak disimpan.

CREATE TABLE IF NOT EXISTS booking_layanan (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  booking_id  BIGINT UNSIGNED NOT NULL,
  nama        VARCHAR(255) NOT NULL,
  nominal     DECIMAL(12,0) NOT NULL,
  dibuat_oleh BIGINT UNSIGNED NULL,
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_booking_layanan_booking FOREIGN KEY (booking_id) REFERENCES bookings (id) ON DELETE CASCADE,
  CONSTRAINT fk_booking_layanan_dibuat_oleh FOREIGN KEY (dibuat_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB;
