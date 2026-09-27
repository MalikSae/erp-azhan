-- 061: Model data fitur Agen Umroh "Syiar" (agen-azhan.md Bagian 7, revisi D1-D10).
-- Nominal uang memakai DECIMAL(12,0) mengikuti kolom harga yang sudah ada.

-- 7.1 Status & kaitan agen pada jamaah (+ rekening pencairan, L5)
ALTER TABLE jamaah
  ADD COLUMN status_agen ENUM('tidak_aktif','pengajuan','aktif','nonaktif') NOT NULL DEFAULT 'tidak_aktif',
  ADD COLUMN kode_referral VARCHAR(20) NULL,
  ADD COLUMN upline_jamaah_id BIGINT UNSIGNED NULL,
  ADD COLUMN direkrut_oleh_jamaah_id BIGINT UNSIGNED NULL,
  ADD COLUMN kaitan_status ENUM('belum_ditentukan','terikat_agen','tanpa_agen') NOT NULL DEFAULT 'belum_ditentukan',
  ADD COLUMN diajukan_agen_at DATETIME NULL,
  ADD COLUMN disetujui_agen_at DATETIME NULL,
  ADD COLUMN disetujui_agen_oleh BIGINT UNSIGNED NULL,
  ADD COLUMN foto_agen_url VARCHAR(500) NULL,
  ADD COLUMN domisili VARCHAR(100) NULL,
  ADD COLUMN menyetujui_syarat_ketentuan_agen_at DATETIME NULL,
  ADD COLUMN rekening_bank_agen VARCHAR(100) NULL,
  ADD COLUMN rekening_nomor_agen VARCHAR(50) NULL,
  ADD COLUMN rekening_atas_nama_agen VARCHAR(150) NULL,
  ADD UNIQUE KEY uq_jamaah_brand_kode_referral (brand_id, kode_referral),
  ADD KEY idx_jamaah_direkrut (direkrut_oleh_jamaah_id),
  ADD KEY idx_jamaah_upline (upline_jamaah_id),
  ADD KEY idx_jamaah_brand_status_agen (brand_id, status_agen),
  ADD CONSTRAINT fk_jamaah_upline FOREIGN KEY (upline_jamaah_id) REFERENCES jamaah (id),
  ADD CONSTRAINT fk_jamaah_direkrut FOREIGN KEY (direkrut_oleh_jamaah_id) REFERENCES jamaah (id),
  ADD CONSTRAINT fk_jamaah_disetujui_agen_oleh FOREIGN KEY (disetujui_agen_oleh) REFERENCES admin_users (id);

-- 7.2 Nominal komisi per jadwal (D1) dan snapshot di booking (D9)
ALTER TABLE schedules
  ADD COLUMN nominal_komisi_langsung DECIMAL(12,0) NULL,
  ADD COLUMN nominal_bonus_pembinaan DECIMAL(12,0) NULL;

ALTER TABLE bookings
  ADD COLUMN snapshot_komisi_langsung DECIMAL(12,0) NULL,
  ADD COLUMN snapshot_bonus_pembinaan DECIMAL(12,0) NULL;

-- 7.9 Pengaturan agen per brand
ALTER TABLE brands
  ADD COLUMN biaya_pendaftaran_agen DECIMAL(12,0) NOT NULL DEFAULT 0,
  ADD COLUMN no_wa_admin_travel VARCHAR(20) NULL;

-- 7.5 Ledger komisi (immutable, idempoten per pax per jenis per penerima)
CREATE TABLE IF NOT EXISTS transaksi_komisi (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id            BIGINT UNSIGNED NOT NULL,
  jamaah_penerima_id  BIGINT UNSIGNED NOT NULL,
  jamaah_sumber_id    BIGINT UNSIGNED NOT NULL,
  booking_id          BIGINT UNSIGNED NOT NULL,
  booking_pax_id      BIGINT UNSIGNED NOT NULL,
  jenis               ENUM('langsung','pembinaan','repeat_order','cashback') NOT NULL,
  nominal             DECIMAL(12,0) NOT NULL,
  created_at          TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_komisi_pax_jenis_penerima (booking_pax_id, jenis, jamaah_penerima_id),
  KEY idx_komisi_penerima (jamaah_penerima_id),
  KEY idx_komisi_sumber (jamaah_sumber_id),
  KEY idx_komisi_booking (booking_id),
  CONSTRAINT fk_komisi_brand FOREIGN KEY (brand_id) REFERENCES brands (id),
  CONSTRAINT fk_komisi_penerima FOREIGN KEY (jamaah_penerima_id) REFERENCES jamaah (id),
  CONSTRAINT fk_komisi_sumber FOREIGN KEY (jamaah_sumber_id) REFERENCES jamaah (id),
  CONSTRAINT fk_komisi_booking FOREIGN KEY (booking_id) REFERENCES bookings (id),
  CONSTRAINT fk_komisi_booking_pax FOREIGN KEY (booking_pax_id) REFERENCES booking_pax (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7.6 Pengajuan pencairan (+ rekening snapshot & bukti transfer keluar, L5)
CREATE TABLE IF NOT EXISTS pengajuan_pencairan (
  id                         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id                   BIGINT UNSIGNED NOT NULL,
  jamaah_agen_id             BIGINT UNSIGNED NOT NULL,
  nominal_diajukan           DECIMAL(12,0) NOT NULL,
  status                     ENUM('pending','disetujui','ditolak') NOT NULL DEFAULT 'pending',
  rekening_bank              VARCHAR(100) NOT NULL,
  rekening_nomor             VARCHAR(50) NOT NULL,
  rekening_atas_nama         VARCHAR(150) NOT NULL,
  bukti_transfer_keluar_url  VARCHAR(500) NULL,
  diajukan_at                DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diproses_oleh              BIGINT UNSIGNED NULL,
  diproses_at                DATETIME NULL,
  catatan_penolakan          TEXT NULL,
  KEY idx_pencairan_agen_status (jamaah_agen_id, status),
  KEY idx_pencairan_status (status),
  CONSTRAINT fk_pencairan_brand FOREIGN KEY (brand_id) REFERENCES brands (id),
  CONSTRAINT fk_pencairan_agen FOREIGN KEY (jamaah_agen_id) REFERENCES jamaah (id),
  CONSTRAINT fk_pencairan_diproses_oleh FOREIGN KEY (diproses_oleh) REFERENCES admin_users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7.8 Audit trail ganti kaitan agen (Jalur 3)
CREATE TABLE IF NOT EXISTS jamaah_kaitan_log (
  id                            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  jamaah_id                     BIGINT UNSIGNED NOT NULL,
  direkrut_oleh_jamaah_id_lama  BIGINT UNSIGNED NULL,
  direkrut_oleh_jamaah_id_baru  BIGINT UNSIGNED NULL,
  kaitan_status_lama            ENUM('belum_ditentukan','terikat_agen','tanpa_agen') NOT NULL,
  kaitan_status_baru            ENUM('belum_ditentukan','terikat_agen','tanpa_agen') NOT NULL,
  diganti_oleh                  BIGINT UNSIGNED NOT NULL,
  diganti_at                    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  alasan                        TEXT NOT NULL,
  KEY idx_kaitan_log_jamaah (jamaah_id),
  CONSTRAINT fk_kaitan_log_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah (id),
  CONSTRAINT fk_kaitan_log_admin FOREIGN KEY (diganti_oleh) REFERENCES admin_users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7.10 Pembayaran pendaftaran agen (1 baris per siklus pengajuan)
CREATE TABLE IF NOT EXISTS pembayaran_pendaftaran_agen (
  id                   BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  jamaah_id            BIGINT UNSIGNED NOT NULL,
  brand_id             BIGINT UNSIGNED NOT NULL,
  nominal_tagihan      DECIMAL(12,0) NOT NULL DEFAULT 0,
  bukti_transfer_url   VARCHAR(500) NULL,
  status               ENUM('menunggu_verifikasi','terverifikasi','ditolak') NOT NULL DEFAULT 'menunggu_verifikasi',
  diverifikasi_oleh    BIGINT UNSIGNED NULL,
  diverifikasi_at      DATETIME NULL,
  catatan_penolakan    TEXT NULL,
  keputusan_agen       ENUM('menunggu','disetujui','ditolak') NOT NULL DEFAULT 'menunggu',
  keputusan_agen_at    DATETIME NULL,
  alasan_ditolak_agen  TEXT NULL,
  created_at           TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  KEY idx_ppa_jamaah (jamaah_id),
  KEY idx_ppa_brand_status (brand_id, status),
  CONSTRAINT fk_ppa_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah (id),
  CONSTRAINT fk_ppa_brand FOREIGN KEY (brand_id) REFERENCES brands (id),
  CONSTRAINT fk_ppa_diverifikasi_oleh FOREIGN KEY (diverifikasi_oleh) REFERENCES admin_users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7.11 Riwayat upload bukti transfer pendaftaran agen (tidak pernah ditimpa)
CREATE TABLE IF NOT EXISTS pembayaran_pendaftaran_agen_upload (
  id                               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  pembayaran_pendaftaran_agen_id   BIGINT UNSIGNED NOT NULL,
  bukti_transfer_url               VARCHAR(500) NOT NULL,
  diupload_oleh_tipe               ENUM('jamaah','admin') NOT NULL,
  diupload_oleh_id                 BIGINT UNSIGNED NOT NULL,
  created_at                       TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  KEY idx_ppa_upload_ppa (pembayaran_pendaftaran_agen_id),
  CONSTRAINT fk_ppa_upload_ppa FOREIGN KEY (pembayaran_pendaftaran_agen_id) REFERENCES pembayaran_pendaftaran_agen (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- 7.13 Pemakaian kredit cashback (D4)
CREATE TABLE IF NOT EXISTS pemakaian_cashback (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id      BIGINT UNSIGNED NOT NULL,
  jamaah_id     BIGINT UNSIGNED NOT NULL,
  booking_id    BIGINT UNSIGNED NOT NULL,
  nominal       DECIMAL(12,0) NOT NULL,
  dipakai_oleh  BIGINT UNSIGNED NOT NULL,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  KEY idx_cashback_jamaah (jamaah_id),
  CONSTRAINT fk_cashback_brand FOREIGN KEY (brand_id) REFERENCES brands (id),
  CONSTRAINT fk_cashback_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah (id),
  CONSTRAINT fk_cashback_booking FOREIGN KEY (booking_id) REFERENCES bookings (id),
  CONSTRAINT fk_cashback_admin FOREIGN KEY (dipakai_oleh) REFERENCES admin_users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
