-- 154: Log dan penghitung aktivasi akun jamaah (keputusan 18 November 2026).
--
-- jamaah_aktivasi_log: jejak untuk admin. Hanya kegagalan di kode booking yang ada dan keberhasilan yang
-- ditulis. Kode, NIK, IP, dan perangkat disimpan sebagai HMAC (pepper AKTIVASI_PEPPER), tidak pernah mentah.
-- alasan: sebab sebenarnya (kode_salah, nik_salah, bukan_peserta, sudah_tertaut, booking_batal, dan
-- seterusnya); pengguna selalu menerima satu pesan galat yang sama.
-- jamaah_aktivasi_hitung: penghitung percobaan gagal yang atomik (INSERT .. ON DUPLICATE KEY UPDATE).
-- jenis: ip, dev, pasangan, kode_24, nik_24, brand, simpan_ip. jendela: nomor jendela waktu tetap.
-- Baris kedaluwarsa dibersihkan berkala (kedaluwarsa_at).

CREATE TABLE IF NOT EXISTS jamaah_aktivasi_log (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id      BIGINT UNSIGNED NOT NULL,
  aksi          ENUM('cek_gagal','cek_ok','aktivasi','lepas_admin','kode_dikunci','reset_admin') NOT NULL,
  jamaah_id     BIGINT UNSIGNED NULL,
  akun_id       BIGINT UNSIGNED NULL,
  kode_hash     CHAR(64) NULL,
  nik_hash      CHAR(64) NULL,
  ip_hash       CHAR(64) NULL,
  device_hash   CHAR(64) NULL,
  alasan        VARCHAR(40) NOT NULL DEFAULT '',
  oleh_admin_id BIGINT UNSIGNED NULL,
  catatan       VARCHAR(300) NOT NULL DEFAULT '',
  dibuat_pada   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_aktivasi_log_kode (kode_hash, dibuat_pada),
  KEY idx_aktivasi_log_nik (nik_hash, dibuat_pada),
  KEY idx_aktivasi_log_ip (ip_hash, dibuat_pada),
  KEY idx_aktivasi_log_jamaah (jamaah_id),
  KEY idx_aktivasi_log_akun (akun_id),
  KEY idx_aktivasi_log_waktu (dibuat_pada)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS jamaah_aktivasi_hitung (
  kunci_hash     CHAR(64) NOT NULL,
  jenis          VARCHAR(12) NOT NULL,
  jendela        BIGINT UNSIGNED NOT NULL,
  jumlah         INT UNSIGNED NOT NULL DEFAULT 0,
  kedaluwarsa_at DATETIME NOT NULL,
  PRIMARY KEY (kunci_hash, jenis, jendela),
  KEY idx_aktivasi_hitung_kedaluwarsa (kedaluwarsa_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
