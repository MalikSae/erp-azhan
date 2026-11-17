-- 140: menu Manasik (keputusan 15 November 2026, menggantikan desain manasik per jadwal tanggal 14 November).
--
-- Satu manasik milik satu brand dan dikaitkan ke satu atau lebih paket (jadwal keberangkatan); satu paket paling banyak
-- satu manasik. App Hana menampilkan manasik paket sebuah booking dan menerima konfirmasi hadir (RSVP) per jamaah dewasa.
-- Kehadiran nyata dicatat petugas di halaman detail manasik dan disimpan di penanda progres booking_pax.progress_manasik
-- (satu sumber dengan centang di detail booking), ditambah siapa dan kapan.
CREATE TABLE IF NOT EXISTS manasik (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id     BIGINT UNSIGNED NOT NULL,
  judul        VARCHAR(150) NOT NULL,
  tanggal      DATE NOT NULL,
  jam_mulai    TIME NOT NULL,
  jam_selesai  TIME NOT NULL,
  -- Agenda: satu butir per baris, mis. "08.00 Pembukaan".
  agenda       TEXT NULL,
  ustadz       VARCHAR(150) NULL,
  tempat       VARCHAR(150) NOT NULL,
  alamat       VARCHAR(500) NOT NULL,
  gmaps_url    VARCHAR(500) NULL,
  dibuat_oleh  BIGINT UNSIGNED NULL,
  diubah_oleh  BIGINT UNSIGNED NULL,
  dibuat_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diubah_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_manasik_brand_tanggal (brand_id, tanggal),
  CONSTRAINT fk_manasik_brand FOREIGN KEY (brand_id) REFERENCES brands (id),
  CONSTRAINT fk_manasik_dibuat FOREIGN KEY (dibuat_oleh) REFERENCES admin_users (id) ON DELETE SET NULL,
  CONSTRAINT fk_manasik_diubah FOREIGN KEY (diubah_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
);

-- Paket yang ikut sebuah manasik. Primary key schedule_id: satu paket paling banyak satu manasik.
CREATE TABLE IF NOT EXISTS manasik_jadwal (
  schedule_id  BIGINT UNSIGNED NOT NULL PRIMARY KEY,
  manasik_id   BIGINT UNSIGNED NOT NULL,
  dikaitkan_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_manasik_jadwal_manasik (manasik_id),
  CONSTRAINT fk_manasik_jadwal_manasik FOREIGN KEY (manasik_id) REFERENCES manasik (id) ON DELETE CASCADE,
  CONSTRAINT fk_manasik_jadwal_schedule FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS manasik_rsvp (
  manasik_id  BIGINT UNSIGNED NOT NULL,
  jamaah_id   BIGINT UNSIGNED NOT NULL,
  booking_id  BIGINT UNSIGNED NOT NULL,
  hadir       BOOLEAN NOT NULL,
  -- Akun pelanggan yang terakhir mengisi konfirmasi.
  dijawab_oleh BIGINT UNSIGNED NULL,
  dijawab_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (manasik_id, jamaah_id),
  KEY idx_manasik_rsvp_booking (booking_id),
  CONSTRAINT fk_manasik_rsvp_manasik FOREIGN KEY (manasik_id) REFERENCES manasik (id) ON DELETE CASCADE,
  CONSTRAINT fk_manasik_rsvp_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah (id) ON DELETE CASCADE,
  CONSTRAINT fk_manasik_rsvp_booking FOREIGN KEY (booking_id) REFERENCES bookings (id) ON DELETE CASCADE
);

-- Jejak penanda "sudah manasik": admin dan waktu terakhir penanda diubah (dari absensi manasik atau detail booking).
ALTER TABLE booking_pax
  ADD COLUMN progress_manasik_oleh BIGINT UNSIGNED NULL AFTER progress_manasik,
  ADD COLUMN progress_manasik_at DATETIME NULL AFTER progress_manasik_oleh,
  ADD CONSTRAINT fk_booking_pax_manasik_oleh FOREIGN KEY (progress_manasik_oleh) REFERENCES admin_users (id) ON DELETE SET NULL;

INSERT IGNORE INTO permissions (code, description) VALUES
  ('manasik.view', 'Lihat manasik dan rekap kehadiran'),
  ('manasik.edit', 'Kelola manasik, paket terkait, dan absensi');

-- Super Admin, Product Manager (pengelola paket di holding), dan Admin Brand.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('super_admin_grup', 'product_manager', 'admin_travel') AND p.code IN ('manasik.view', 'manasik.edit');

-- Sesi yang sedang berjalan memuat ulang izinnya.
UPDATE admin_users u
JOIN user_roles ur ON ur.admin_user_id = u.id
JOIN roles r ON r.id = ur.role_id AND r.slug IN ('super_admin_grup', 'product_manager', 'admin_travel')
SET u.perm_version = u.perm_version + 1;
