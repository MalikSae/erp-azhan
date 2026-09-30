-- 068_backfill_booking_seat_count.sql
-- Menyelaraskan bookings.seat_count dengan jumlah pax reguler aktif (JB-02).
--
-- Booking admin (POST /bookings) dan finalisasi draft sebelumnya tidak mengisi
-- seat_count sehingga tersimpan default 1 untuk berapa pun pax regulernya.
-- Nilai ini dipakai blokir ulang kursi, worker hold kedaluwarsa, dan batas
-- penyesuaian kuota manual. Aman dijalankan ulang (hasil selalu sama).
--
-- Migrasi ini TIDAK mengubah schedules.seat_sisa. Selisih seat_sisa yang sudah
-- terjadi akibat bug lama perlu direkonsiliasi manual oleh admin (lihat
-- AUDIT-JAMAAH-BOOKING-2026-09-30.md, JB-02).

UPDATE bookings b
JOIN (
  SELECT booking_id, SUM(counts_for_seat = TRUE AND pax_status = 'aktif') AS reguler_aktif
  FROM booking_pax
  GROUP BY booking_id
) p ON p.booking_id = b.id
SET b.seat_count = p.reguler_aktif
WHERE b.status <> 'batal'
  AND b.seat_count <> p.reguler_aktif;
