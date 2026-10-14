-- 088_drop_itinerary_meals.sql
-- Keterangan makan (B/L/D/MOB) per hari itinerary tidak dipakai; kolom dihapus
-- (ditambahkan sebelumnya di 087). Nilai lama hilang permanen.
ALTER TABLE itinerary_days DROP COLUMN meals;
