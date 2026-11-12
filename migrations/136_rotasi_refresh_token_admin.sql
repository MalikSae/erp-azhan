-- 136: rotasi refresh token admin (temuan audit keamanan, 12 November 2026).
--
-- Dulu satu refresh token bisa dipakai berulang selama masa berlakunya (7 hari). Sekarang setiap
-- /api/auth/refresh mencabut token lama dan menerbitkan yang baru. diganti_at menandai token yang
-- dicabut karena rotasi (bukan logout/reset password), supaya:
--   - dua tab yang refresh hampir bersamaan tetap jalan (token lama masih diterima 60 detik setelah diganti);
--   - token lama yang dipakai lagi setelah itu dianggap bocor, dan semua refresh token user dicabut.

ALTER TABLE admin_refresh_tokens
  ADD COLUMN diganti_at DATETIME NULL AFTER revoked_at;
