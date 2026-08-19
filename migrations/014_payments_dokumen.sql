-- 014_payments_dokumen.sql
-- Tabel payments (ledger pembayaran per booking) dan dokumen_jamaah (dokumen per jamaah per jenis)

CREATE TABLE payments (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  booking_id    BIGINT UNSIGNED NOT NULL,
  jumlah        DECIMAL(12,0) NOT NULL,
  metode        VARCHAR(50),
  tanggal       DATE,
  status        ENUM('pending','confirmed') NOT NULL DEFAULT 'pending',
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (booking_id) REFERENCES bookings(id)
) ENGINE=InnoDB;

CREATE TABLE dokumen_jamaah (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  jamaah_id     BIGINT UNSIGNED NOT NULL,
  jenis         ENUM('pas_foto','paspor','ktp','kk','buku_nikah','akte_lahir') NOT NULL,
  file_url      VARCHAR(500),
  status        ENUM('belum_upload','submitted','approved','rejected') NOT NULL DEFAULT 'belum_upload',
  updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY unique_jamaah_jenis (jamaah_id, jenis),
  FOREIGN KEY (jamaah_id) REFERENCES jamaah(id)
) ENGINE=InnoDB;
