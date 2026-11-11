-- 132: log kaitan akun boleh dicatat oleh sistem (code review Agen, 11 November 2026, temuan C4).
--
-- Akun bukan agen yang ikut berangkat dan jamaah dirinya sudah terikat ke seorang agen otomatis mendapat
-- agen itu sebagai upline (agen.IkatAkunDariJamaah). Perubahan ini dulu tidak dicatat. Kolom diganti_oleh
-- wajib berisi admin, sedangkan pengikatan otomatis tidak punya admin: NULL berarti dicatat oleh sistem.

ALTER TABLE akun_kaitan_log MODIFY diganti_oleh BIGINT UNSIGNED NULL;
