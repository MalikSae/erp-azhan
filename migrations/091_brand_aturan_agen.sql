-- 091_brand_aturan_agen.sql
-- Aturan agen per brand: nominal ujroh (Rp per jamaah) dan syarat & ketentuan agen.
-- Biaya pendaftaran agen memakai kolom brands.biaya_pendaftaran_agen yang sudah ada (0 = gratis).
ALTER TABLE brands
  ADD COLUMN ujroh_closing DECIMAL(12,0) NOT NULL DEFAULT 0 AFTER biaya_pendaftaran_agen,
  ADD COLUMN ujroh_pembinaan DECIMAL(12,0) NOT NULL DEFAULT 0 AFTER ujroh_closing,
  ADD COLUMN ujroh_repeat_booking DECIMAL(12,0) NOT NULL DEFAULT 0 AFTER ujroh_pembinaan,
  ADD COLUMN syarat_ketentuan_agen TEXT NULL AFTER ujroh_repeat_booking;
