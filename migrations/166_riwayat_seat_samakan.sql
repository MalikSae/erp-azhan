-- 166_riwayat_seat_samakan.sql
-- Riwayat seat mencatat "Samakan dengan booking" (keputusan audit scarcity seat 2026-11-27).
-- seat_before/seat_after = sisa seat sebelum dan sesudah disamakan.
-- Catatan: ALTER TABLE tidak idempotent, jalankan sekali. Nomor 165 dipakai migrasi layanan booking.

ALTER TABLE schedule_seat_adjustments
  MODIFY COLUMN jenis ENUM('kuota','tampil','samakan') NOT NULL DEFAULT 'kuota';
