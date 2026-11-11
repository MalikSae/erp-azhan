-- 133: penanda paket pernah tayang (code review Paket dan master, 11 November 2026).
--
-- Hanya paket draft yang boleh dihapus permanen; paket yang pernah tayang diarsipkan, supaya tautan yang
-- sudah tersebar tetap menjawab "sudah tidak tersedia". Status saja tidak cukup: paket tayang bisa
-- dikembalikan ke draft lalu dihapus. Kolom ini mencatat kapan paket pertama kali berstatus published dan
-- tidak pernah dikosongkan lagi.

ALTER TABLE schedules
  ADD COLUMN pernah_tayang_at DATETIME NULL AFTER status;

-- Data lama: paket yang sekarang published atau archived dianggap pernah tayang. Paket draft yang dulu
-- pernah tayang tidak bisa dikenali dari data, jadi tetap kosong.
UPDATE schedules SET pernah_tayang_at = COALESCE(updated_at, created_at, NOW())
WHERE status IN ('published', 'archived') AND pernah_tayang_at IS NULL;
