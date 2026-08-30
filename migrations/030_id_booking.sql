ALTER TABLE bookings ADD COLUMN id_booking VARCHAR(6) NULL AFTER id;

-- Backfill booking lama sebelum kolom dijadikan NOT NULL oleh migrasi 031.
-- Format mengikuti kode brand dua karakter + empat karakter base-36 dari ID booking.
UPDATE bookings b
JOIN schedules s ON s.id = b.schedule_id
JOIN brands br ON br.id = s.brand_id
SET b.id_booking = CONCAT(
  LEFT(RPAD(UPPER(COALESCE(NULLIF(TRIM(br.kode_brand), ''), 'BK')), 2, 'X'), 2),
  LPAD(UPPER(CONV(b.id, 10, 36)), 4, '0')
)
WHERE b.id_booking IS NULL OR b.id_booking = '';
