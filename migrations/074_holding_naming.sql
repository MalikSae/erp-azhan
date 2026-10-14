-- 074_holding_naming.sql
-- Keputusan 2 Okt 2026: istilah level atas baku memakai "Holding"
-- (nama perusahaan "Azhan Grup" tidak berubah). Slug tetap.

UPDATE roles SET name = 'Super Admin Holding' WHERE slug = 'super_admin_grup';
UPDATE roles SET name = 'Finance Holding' WHERE slug = 'finance_grup';
