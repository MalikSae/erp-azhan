-- 140: manasik per keberangkatan dan konfirmasi hadir jamaah (keputusan 14 November 2026).
--
-- Satu jadwal keberangkatan punya tepat satu manasik yang membahas seluruh tata cara umroh. Admin mengisinya di halaman
-- detail jadwal; app Hana menampilkan detailnya dan menerima konfirmasi hadir (RSVP) per jamaah. RSVP tidak mengubah
-- penanda progres "sudah manasik" di booking_pax: penanda itu tetap diisi petugas setelah jamaah benar-benar hadir.
CREATE TABLE IF NOT EXISTS manasik (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  schedule_id  BIGINT UNSIGNED NOT NULL,
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
  diubah_oleh  BIGINT UNSIGNED NULL,
  dibuat_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  diubah_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_manasik_schedule (schedule_id),
  CONSTRAINT fk_manasik_schedule FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE CASCADE,
  CONSTRAINT fk_manasik_admin FOREIGN KEY (diubah_oleh) REFERENCES admin_users (id) ON DELETE SET NULL
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
