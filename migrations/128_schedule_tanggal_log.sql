-- 128_schedule_tanggal_log.sql
-- Audit menu Paket 10 November 2026 (S-04), keputusan pemilik proyek.
--
-- Tanggal berangkat paket yang punya booking aktif tetap boleh diubah, tetapi wajib beralasan dan
-- tercatat. Tabel ini menyimpan riwayatnya: tanggal lama dan baru, jumlah booking aktif saat itu,
-- alasan, dan admin yang mengubah.
--
-- Aman dijalankan ulang.

CREATE TABLE IF NOT EXISTS schedule_tanggal_log (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  schedule_id BIGINT NOT NULL,
  tanggal_lama DATE NOT NULL,
  tanggal_baru DATE NOT NULL,
  booking_aktif INT NOT NULL,
  alasan VARCHAR(500) NOT NULL,
  diubah_oleh BIGINT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_schedule_tanggal_log_schedule (schedule_id, created_at)
);
