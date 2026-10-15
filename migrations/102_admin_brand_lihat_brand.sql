-- 102_admin_brand_lihat_brand.sql
-- Halaman bersama (paket, jamaah, booking) memuat nama/warna brand lewat daftar brand.
-- Admin Brand diberi brand.view; API membatasinya ke brand sendiri (list & detail).
-- Menu "Kelola Brand" tetap hanya untuk brand.edit (holding).
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'admin_travel' AND p.code = 'brand.view';
