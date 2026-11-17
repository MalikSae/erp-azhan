-- 150: Program dan destinasi paket pindah ke CMS (keputusan 17 November 2026).
--
-- ERP tidak lagi menyimpan program dan destinasi; keduanya diatur di CMS web publik (azhan-booking).
-- schedules.journey_type tetap. Kolom dan tabel dibuang tanpa migrasi data (belum rilis production).
ALTER TABLE schedules DROP FOREIGN KEY fk_schedules_destination;
ALTER TABLE schedules
  DROP INDEX idx_schedules_destination,
  DROP COLUMN destination_id,
  DROP COLUMN program_slug;
DROP TABLE IF EXISTS destinations;
