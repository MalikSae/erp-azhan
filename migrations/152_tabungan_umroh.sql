-- 152: Tabungan Umroh di app (keputusan 17 November 2026).
--
-- Catatan target menabung pribadi calon jamaah. Hana tidak menyimpan uang: tabel ini hanya
-- mencatat target dan angka yang dicatat sendiri oleh akun.
-- tabungan_target: satu target per akun. Uang bilangan bulat rupiah.
-- schedule_id: paket (jadwal) yang dipilih; null untuk perkiraan. nama_paket dan dp_minimal adalah
-- snapshot supaya target tidak berubah saat paket diubah admin.
-- tabungan_catatan: satu baris per catatan. Mengganti target tidak menghapus catatan.
-- perangkat_push: token push Expo per perangkat, maks 10 per akun (diatur API). Pengiriman push
-- belum dibuat di tahap ini.
-- Ketiganya ikut terhapus saat akun dihapus (anonimkan) dan lewat ON DELETE CASCADE.

CREATE TABLE IF NOT EXISTS tabungan_target (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  akun_id           BIGINT UNSIGNED NOT NULL,
  jenis             VARCHAR(10) NOT NULL,
  schedule_id       BIGINT UNSIGNED NULL,
  nama_paket        VARCHAR(150) NOT NULL DEFAULT '',
  bulan_berangkat   CHAR(7) NOT NULL,
  jumlah_dewasa     TINYINT UNSIGNED NOT NULL,
  jumlah_bayi       TINYINT UNSIGNED NOT NULL DEFAULT 0,
  tipe_kamar        VARCHAR(10) NOT NULL,
  target_total      BIGINT UNSIGNED NOT NULL,
  dp_minimal        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  pengingat_aktif   BOOLEAN NOT NULL DEFAULT FALSE,
  pengingat_tanggal TINYINT UNSIGNED NOT NULL DEFAULT 1,
  pengingat_jam     TINYINT UNSIGNED NOT NULL DEFAULT 9,
  dibuat_at         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diperbarui_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_tabungan_target_akun (akun_id),
  CONSTRAINT fk_tabungan_target_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE,
  CONSTRAINT fk_tabungan_target_jadwal FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tabungan_catatan (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  akun_id    BIGINT UNSIGNED NOT NULL,
  jumlah     BIGINT UNSIGNED NOT NULL,
  tanggal    DATE NOT NULL,
  dibuat_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_tabungan_catatan_akun_tanggal (akun_id, tanggal),
  CONSTRAINT fk_tabungan_catatan_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS perangkat_push (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  akun_id       BIGINT UNSIGNED NOT NULL,
  token         VARCHAR(255) NOT NULL,
  platform      VARCHAR(10) NOT NULL,
  dibuat_at     DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  diperbarui_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_perangkat_push_token (token),
  KEY idx_perangkat_push_akun (akun_id, diperbarui_at),
  CONSTRAINT fk_perangkat_push_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
