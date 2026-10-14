-- 085_drop_ticket_confirmed.sql
-- Fitur "Tiket Confirmed" dihapus: flag per paket tidak lagi dipakai admin,
-- travel dashboard, microsite, maupun progres booking (langkah Tiket pada header
-- progres ikut dihapus). Nilai lama hilang permanen; kolom bookings.progress_tiket
-- (data legacy, lihat migrasi 040) sengaja tidak disentuh.
ALTER TABLE schedules DROP COLUMN is_ticket_confirmed;
