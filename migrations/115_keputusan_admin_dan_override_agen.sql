-- 115_keputusan_admin_dan_override_agen.sql
-- Spek keagenan akun, bagian 7 dan 4.4 (butir 8-9): catatan admin pada keputusan pengajuan, log override agen akun,
-- dan catatan pemindahan komisi pada log kaitan jamaah.

ALTER TABLE pembayaran_pendaftaran_agen
  ADD COLUMN catatan_admin TEXT NULL;

ALTER TABLE jamaah_kaitan_log
  ADD COLUMN komisi_dipindah INT NOT NULL DEFAULT 0,
  ADD COLUMN komisi_dihapus INT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS akun_kaitan_log (
  id                       BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  akun_id                  BIGINT UNSIGNED NOT NULL,
  direkrut_oleh_akun_id_lama BIGINT UNSIGNED NULL,
  direkrut_oleh_akun_id_baru BIGINT UNSIGNED NULL,
  upline_diubah            TINYINT(1) NOT NULL DEFAULT 0,
  diganti_oleh             BIGINT UNSIGNED NOT NULL,
  diganti_at               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alasan                   TEXT NOT NULL,
  KEY idx_akun_kaitan_log_akun (akun_id),
  CONSTRAINT fk_akun_kaitan_log_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id),
  CONSTRAINT fk_akun_kaitan_log_admin FOREIGN KEY (diganti_oleh) REFERENCES admin_users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
