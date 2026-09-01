-- 048_perlengkapan_pax.sql

CREATE TABLE IF NOT EXISTS booking_pax_perlengkapan_logs (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    booking_pax_id BIGINT UNSIGNED NOT NULL,
    perlengkapan_item_id BIGINT UNSIGNED NOT NULL,
    qty INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (booking_pax_id) REFERENCES booking_pax(id) ON DELETE CASCADE,
    FOREIGN KEY (perlengkapan_item_id) REFERENCES perlengkapan_items(id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE bookings
DROP FOREIGN KEY bookings_ibfk_4;

ALTER TABLE bookings
DROP COLUMN perlengkapan_status,
DROP COLUMN perlengkapan_tanggal,
DROP COLUMN perlengkapan_diberikan_oleh,
DROP COLUMN perlengkapan_jumlah_pax;
