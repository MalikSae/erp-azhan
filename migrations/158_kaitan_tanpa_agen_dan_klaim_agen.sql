-- 158: Satu status jamaah tanpa agen (keputusan 2026-11-24). 'belum_ditentukan' digabung ke 'tanpa_agen'.
-- Jamaah baru (belum pernah booking, kaitan belum pernah ditetapkan) tetap terikat otomatis ke agen yang
-- membawanya; jamaah 'tanpa_agen' lain hanya dikaitkan admin. Agen yang membawa jamaah 'tanpa_agen' dicatat di
-- bookings.agen_klaim_akun_id agar admin bisa mengaitkannya dari Detail Booking.
-- Belum rilis production: tanpa backfill.

UPDATE jamaah SET kaitan_status = 'tanpa_agen' WHERE kaitan_status = 'belum_ditentukan';
UPDATE jamaah_kaitan_log SET kaitan_status_lama = 'tanpa_agen' WHERE kaitan_status_lama = 'belum_ditentukan';
UPDATE jamaah_kaitan_log SET kaitan_status_baru = 'tanpa_agen' WHERE kaitan_status_baru = 'belum_ditentukan';

ALTER TABLE jamaah
  MODIFY COLUMN kaitan_status ENUM('terikat_agen','tanpa_agen') NOT NULL DEFAULT 'tanpa_agen';

ALTER TABLE jamaah_kaitan_log
  MODIFY COLUMN kaitan_status_lama ENUM('terikat_agen','tanpa_agen') NOT NULL,
  MODIFY COLUMN kaitan_status_baru ENUM('terikat_agen','tanpa_agen') NOT NULL;

ALTER TABLE bookings
  ADD COLUMN agen_klaim_akun_id BIGINT UNSIGNED NULL AFTER dibuat_oleh_agen_akun_id,
  ADD KEY idx_bookings_agen_klaim (agen_klaim_akun_id),
  ADD CONSTRAINT fk_bookings_agen_klaim FOREIGN KEY (agen_klaim_akun_id) REFERENCES customer_accounts (id);
