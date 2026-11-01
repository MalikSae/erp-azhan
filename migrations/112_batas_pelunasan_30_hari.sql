-- 112_batas_pelunasan_30_hari.sql
-- Aturan pelunasan: paling lambat H-30. Booking lebih dari 30 hari sebelum berangkat boleh DP minimal; booking di H-30 atau kurang
-- wajib lunas saat booking (selfbooking/repository.go membaca brands.batas_pelunasan_hari). Brand yang masih memakai bawaan lama (45)
-- diseragamkan ke 30; nilai yang sudah diubah manual tidak disentuh.

ALTER TABLE brands MODIFY COLUMN batas_pelunasan_hari INT UNSIGNED NOT NULL DEFAULT 30;
UPDATE brands SET batas_pelunasan_hari = 30 WHERE batas_pelunasan_hari = 45;
