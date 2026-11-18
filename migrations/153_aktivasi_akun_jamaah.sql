-- 153: Aktivasi akun jamaah (keputusan 18 November 2026).
--
-- Satu jamaah hanya boleh tertaut ke satu akun: customer_accounts.jamaah_id (kolom dari migrasi 104)
-- menjadi unik. Pemindahan tautan hanya lewat admin (Lepas Tautan). Anonimkan akun mengosongkan kolom ini.
-- Tanpa backfill: belum rilis production.

ALTER TABLE customer_accounts
  ADD UNIQUE KEY uq_customer_jamaah (jamaah_id),
  DROP INDEX idx_customer_jamaah;
