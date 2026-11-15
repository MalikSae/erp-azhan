-- 146: notifikasi akun untuk kotak masuk app (keputusan 16 November 2026).
--
-- Satu baris per akun per kejadian: pembayaran, booking, dokumen, perlengkapan, perubahan data,
-- agen, komisi, pencairan, pengingat berbasis waktu, dan pengumuman yang mulai tayang.
-- kunci: penanda unik per akun supaya pengingat dari ticker dan pengumuman tidak terkirim dua kali.
-- ref_tipe + ref_id: layar app yang dibuka saat notifikasi ditap.
-- pengumuman_id: notifikasi pengumuman ikut terhapus bila pengumumannya dihapus; judul dan isi
-- dibaca dari pengumuman supaya perubahan admin langsung terlihat.
-- Notifikasi lebih dari 90 hari dihapus otomatis oleh API.

CREATE TABLE IF NOT EXISTS notifikasi (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  akun_id        BIGINT UNSIGNED NOT NULL,
  jenis          VARCHAR(40) NOT NULL,
  judul          VARCHAR(150) NOT NULL,
  isi            VARCHAR(300) NOT NULL DEFAULT '',
  ref_tipe       VARCHAR(30) NULL,
  ref_id         BIGINT UNSIGNED NULL,
  pengumuman_id  BIGINT UNSIGNED NULL,
  kunci          VARCHAR(120) NULL,
  dibaca_at      DATETIME NULL,
  dibuat_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_notifikasi_akun_kunci (akun_id, kunci),
  KEY idx_notifikasi_akun_dibuat (akun_id, dibuat_at),
  KEY idx_notifikasi_dibuat (dibuat_at),
  CONSTRAINT fk_notifikasi_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE,
  CONSTRAINT fk_notifikasi_pengumuman FOREIGN KEY (pengumuman_id) REFERENCES pengumuman (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Penanda pengumuman yang sudah dikirim ke kotak masuk akun sasaran.
ALTER TABLE pengumuman ADD COLUMN notif_dikirim_at DATETIME NULL;
