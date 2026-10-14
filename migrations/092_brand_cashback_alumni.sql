-- 092_brand_cashback_alumni.sql
-- Cashback alumni (Rp) per brand, bagian dari S&K Booking.
ALTER TABLE brands
  ADD COLUMN cashback_alumni DECIMAL(12,0) NOT NULL DEFAULT 0 AFTER ketentuan_pembatalan;
