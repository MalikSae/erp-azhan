-- 090_brand_booking_terms.sql
-- S&K Booking per brand: batas pelunasan (H- sebelum keberangkatan) dan ketentuan pembatalan.
-- Default 45 hari mempertahankan perilaku lama (pelunasan H-45).
ALTER TABLE brands
  ADD COLUMN batas_pelunasan_hari INT UNSIGNED NOT NULL DEFAULT 45 AFTER minimal_dp,
  ADD COLUMN ketentuan_pembatalan TEXT NULL AFTER batas_pelunasan_hari;
