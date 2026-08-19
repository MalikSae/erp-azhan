-- 013_jamaah_booking.sql
-- Tabel jamaah (data per brand) dan bookings (relasi jamaah ↔ schedule)

CREATE TABLE jamaah (
  id                    BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id              BIGINT UNSIGNED NOT NULL,
  nama_lengkap          VARCHAR(255) NOT NULL,
  nama_ayah_kandung     VARCHAR(255),
  nik                   VARCHAR(20) UNIQUE,
  tempat_lahir          VARCHAR(100),
  tanggal_lahir         DATE,
  no_paspor             VARCHAR(50),
  tempat_paspor_keluar  VARCHAR(100),
  paspor_berlaku_sampai DATE,
  no_hp                 VARCHAR(20),
  email                 VARCHAR(255),
  pekerjaan             VARCHAR(100),
  pendidikan_terakhir   VARCHAR(50),
  penjamin_kesehatan    VARCHAR(100),
  no_asuransi_bpjs      VARCHAR(50),
  alamat                TEXT,
  emergency_nama        VARCHAR(255),
  emergency_nik         VARCHAR(20),
  emergency_hp          VARCHAR(20),
  emergency_hubungan    VARCHAR(50),
  emergency_alamat      TEXT,
  created_at            TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (brand_id) REFERENCES brands(id)
) ENGINE=InnoDB;

CREATE TABLE bookings (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  schedule_id   BIGINT UNSIGNED NOT NULL,
  jamaah_id     BIGINT UNSIGNED NOT NULL,
  room_type     ENUM('Quad','Triple','Double') NOT NULL,
  status        ENUM('baru','dp','lunas','dokumen_lengkap','siap_berangkat','batal') NOT NULL DEFAULT 'baru',
  total_harga   DECIMAL(12,0),
  created_by    BIGINT UNSIGNED,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (schedule_id) REFERENCES schedules(id),
  FOREIGN KEY (jamaah_id) REFERENCES jamaah(id),
  FOREIGN KEY (created_by) REFERENCES admin_users(id)
) ENGINE=InnoDB;
