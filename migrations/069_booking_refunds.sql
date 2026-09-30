-- 069_booking_refunds.sql
-- Catatan pengembalian dana untuk booking batal yang sudah punya pembayaran
-- terkonfirmasi (keputusan audit jamaah-booking 2026-09-30).
--
-- Hanya menambah data (tanpa update/hapus lewat API) agar jejak audit utuh.
-- Aturan di aplikasi: booking wajib berstatus 'batal' dan total refund tidak
-- boleh melebihi total pembayaran terkonfirmasi.

CREATE TABLE IF NOT EXISTS booking_refunds (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    booking_id BIGINT UNSIGNED NOT NULL,
    jumlah DECIMAL(12,0) NOT NULL,
    tanggal DATE NOT NULL,
    metode VARCHAR(50) NULL,
    catatan VARCHAR(500) NULL,
    created_by BIGINT UNSIGNED NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_booking_refunds_booking (booking_id),
    FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE RESTRICT,
    FOREIGN KEY (created_by) REFERENCES admin_users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
