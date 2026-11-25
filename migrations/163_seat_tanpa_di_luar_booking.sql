-- 163_seat_tanpa_di_luar_booking.sql
-- Tidak ada booking di luar sistem (keputusan 2026-11-25): sisa seat selalu
-- seat_total − seat yang dipegang booking. Koreksi sisa seat manual
-- (PUT /schedules/{id}/seat) dihapus.
-- Catatan: ALTER TABLE tidak idempotent, jalankan sekali. Data riwayat lama dihapus
-- permanen (masih data dummy, belum rilis production).

-- 1. Riwayat seat lama dihapus. Ke depan hanya perubahan kuota dan seat tampil.
DELETE FROM schedule_seat_adjustments;

ALTER TABLE schedule_seat_adjustments
  MODIFY COLUMN jenis ENUM('kuota','tampil') NOT NULL DEFAULT 'kuota';

-- 2. Hitung ulang sisa seat semua paket dari booking yang memegang seat.
UPDATE schedules s
LEFT JOIN (
  SELECT schedule_id, SUM(seat_count) AS dipegang
  FROM bookings
  WHERE is_seat_blocked = 1 AND status <> 'batal'
  GROUP BY schedule_id
) b ON b.schedule_id = s.id
SET s.seat_sisa = GREATEST(0, CAST(s.seat_total AS SIGNED) - COALESCE(b.dipegang, 0));
