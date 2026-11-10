-- 125_riwayat_keputusan_dan_status_agen.sql
-- Audit menu Agen 10 November 2026.
--
-- keputusan_agen_log: satu baris per keputusan admin atas pengajuan agen (setujui, bebaskan, minta unggah
-- ulang, tolak, setujui bayar menyusul) beserta catatan dan pelakunya. Kolom catatan_admin di
-- pembayaran_pendaftaran_agen hanya memuat catatan terakhir dan tertimpa keputusan berikutnya; riwayat ini
-- menyimpan semuanya, termasuk siapa yang menolak.
--
-- agen_status_log: perubahan status agen antara aktif dan nonaktif, dengan alasan dan pelaku.
--
-- Aman dijalankan ulang.

CREATE TABLE IF NOT EXISTS keputusan_agen_log (
  id                              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  pembayaran_pendaftaran_agen_id  BIGINT UNSIGNED NOT NULL,
  akun_id                         BIGINT UNSIGNED NOT NULL,
  aksi                            VARCHAR(40) NOT NULL,
  catatan                         TEXT NULL,
  admin_id                        BIGINT UNSIGNED NULL,
  created_at                      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_keputusan_agen_log_siklus (pembayaran_pendaftaran_agen_id),
  KEY idx_keputusan_agen_log_akun (akun_id),
  CONSTRAINT fk_keputusan_agen_log_siklus FOREIGN KEY (pembayaran_pendaftaran_agen_id) REFERENCES pembayaran_pendaftaran_agen (id) ON DELETE CASCADE,
  CONSTRAINT fk_keputusan_agen_log_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE,
  CONSTRAINT fk_keputusan_agen_log_admin FOREIGN KEY (admin_id) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS agen_status_log (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  akun_id      BIGINT UNSIGNED NOT NULL,
  status_lama  VARCHAR(20) NOT NULL,
  status_baru  VARCHAR(20) NOT NULL,
  alasan       TEXT NOT NULL,
  diubah_oleh  BIGINT UNSIGNED NULL,
  diubah_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_agen_status_log_akun (akun_id),
  CONSTRAINT fk_agen_status_log_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE,
  CONSTRAINT fk_agen_status_log_admin FOREIGN KEY (diubah_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
