-- 151: Jam buka tempat Jelajah Haramain dihapus (keputusan 17 November 2026).
--
-- Kolom dibuang tanpa migrasi data (belum rilis production).
ALTER TABLE haramain_tempat DROP COLUMN jam_buka;
