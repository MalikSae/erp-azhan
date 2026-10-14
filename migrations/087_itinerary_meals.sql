-- 087_itinerary_meals.sql
-- Format itinerary mengikuti dokumen operasional: per hari ada rute, kode makan
-- (B/L/D, MOB) dan daftar poin kegiatan tanpa jam. Kode makan disimpan per hari.
ALTER TABLE itinerary_days
  ADD COLUMN meals VARCHAR(20) NOT NULL DEFAULT '' AFTER location;
