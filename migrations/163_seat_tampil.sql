-- 163_seat_tampil.sql
-- Scarcity seat: angka seat yang tampil di publik (keputusan 2026-11-25).
-- NULL = nonaktif, publik melihat seat_sisa. Angka tampil = LEAST(seat_tampil, seat_sisa).
-- Hanya label: booking, batas pax, dan cek penuh tetap memakai seat_sisa.
-- Catatan: ALTER TABLE tidak idempotent, jalankan sekali.

ALTER TABLE schedules
  ADD COLUMN seat_tampil INT UNSIGNED NULL AFTER seat_sisa;

-- Riwayat seat membedakan koreksi sisa seat dan perubahan angka tampil.
-- Untuk jenis 'tampil', seat_before/seat_after 0 berarti nonaktif.
ALTER TABLE schedule_seat_adjustments
  ADD COLUMN jenis ENUM('sisa','tampil') NOT NULL DEFAULT 'sisa' AFTER schedule_id;
