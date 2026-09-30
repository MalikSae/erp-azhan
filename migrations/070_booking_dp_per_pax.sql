-- 070_booking_dp_per_pax.sql
-- Snapshot DP per pax pada booking (keputusan audit master paket MP-02, 2026-10-01).
--
-- Sebelumnya booking admin/CRM tidak menyimpan DP, sehingga minimal DP dibaca dari
-- paket/brand saat ini: menaikkan DP paket menurunkan status booking lama.
-- Booking baru mengisi kolom ini saat dibuat; self-booking tetap memakai
-- booking_checkout.dp_per_pax yang didahulukan.
--
-- Backfill: booking lama (non-draft, tanpa snapshot checkout) dibekukan ke DP efektif
-- saat migrasi dijalankan. DP historis saat booking dibuat tidak tersimpan sehingga
-- tidak dapat direkonstruksi (pola sama dengan backfill 065). Aman dijalankan ulang.

SET @col_exists = (
  SELECT COUNT(*)
  FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'bookings'
    AND COLUMN_NAME = 'dp_per_pax'
);

SET @sql = IF(
  @col_exists = 0,
  'ALTER TABLE bookings ADD COLUMN dp_per_pax DECIMAL(12,0) NULL AFTER total_harga',
  'DO 0'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE bookings b
JOIN schedules s ON s.id = b.schedule_id
JOIN brands br ON br.id = s.brand_id
LEFT JOIN booking_checkout c ON c.booking_id = b.id
SET b.dp_per_pax = COALESCE(s.minimal_dp, br.minimal_dp, 0)
WHERE b.dp_per_pax IS NULL
  AND b.status <> 'draft'
  AND c.booking_id IS NULL;
