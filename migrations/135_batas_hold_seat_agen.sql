-- 135: batas hold seat agen (keputusan 12 November 2026, temuan audit keamanan).
--
-- Booking dari portal agen dulu tidak dibatasi jumlah seat yang ditahan, sehingga satu agen bisa menahan
-- ratusan seat dengan NIK asal sampai jadwal tampak penuh. Batas bawaan (20 seat per jadwal, 45 total)
-- ada di kode (selfbooking.batasSeatAgenDefault). Kolom di bawah adalah batas khusus per agen yang diisi
-- admin (izin agen.edit, maksimal 200 per jadwal dan 500 total); NULL berarti memakai batas bawaan.
--
-- agen_batas_seat_log: setiap perubahan batas khusus beserta nilai lama, nilai baru, alasan, dan pelakunya.

ALTER TABLE customer_accounts
  ADD COLUMN batas_seat_agen_per_jadwal INT UNSIGNED NULL,
  ADD COLUMN batas_seat_agen_total INT UNSIGNED NULL;

CREATE TABLE IF NOT EXISTS agen_batas_seat_log (
  id                 BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  akun_id            BIGINT UNSIGNED NOT NULL,
  per_jadwal_lama    INT UNSIGNED NULL,
  per_jadwal_baru    INT UNSIGNED NULL,
  total_lama         INT UNSIGNED NULL,
  total_baru         INT UNSIGNED NULL,
  alasan             TEXT NOT NULL,
  diubah_oleh        BIGINT UNSIGNED NULL,
  diubah_at          DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_agen_batas_seat_log_akun (akun_id),
  CONSTRAINT fk_agen_batas_seat_log_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE,
  CONSTRAINT fk_agen_batas_seat_log_admin FOREIGN KEY (diubah_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
