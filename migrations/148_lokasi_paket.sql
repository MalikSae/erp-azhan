-- 148: Lokasi Dikunjungi per paket (keputusan 17 November 2026).
--
-- Paket = satu baris schedules. Lokasi diambil dari master Jelajah Haramain (global, migrasi 145), opsional,
-- maksimal 12 per paket, urutan dari admin. Tempat nonaktif tetap tertaut tetapi tidak tampil di publik.
-- Tempat yang dipakai paket tidak bisa dihapus dari master (ON DELETE RESTRICT); menghapus paket ikut
-- menghapus tautannya.
CREATE TABLE IF NOT EXISTS schedule_lokasi (
  schedule_id  BIGINT UNSIGNED NOT NULL,
  tempat_id    BIGINT UNSIGNED NOT NULL,
  urutan       SMALLINT NOT NULL,
  PRIMARY KEY (schedule_id, tempat_id),
  KEY idx_schedule_lokasi_tempat (tempat_id),
  CONSTRAINT fk_schedule_lokasi_schedule FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE CASCADE,
  CONSTRAINT fk_schedule_lokasi_tempat FOREIGN KEY (tempat_id) REFERENCES haramain_tempat (id) ON DELETE RESTRICT
);
