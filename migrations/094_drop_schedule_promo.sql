-- 094_drop_schedule_promo.sql
-- Fitur harga promo dihapus: kolom promo dan harga coret tidak lagi dipakai.
-- Data lama hilang permanen.
DROP INDEX idx_schedules_promo ON schedules;
ALTER TABLE schedules
  DROP COLUMN is_promo,
  DROP COLUMN promo_until,
  DROP COLUMN harga_coret;
