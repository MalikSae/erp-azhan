-- 101_admin_brand_lihat_rekening.sql
-- Admin Brand mencatat pembayaran transfer dan wajib memilih rekening tujuan milik brand-nya,
-- sehingga perlu melihat daftar rekening. API membatasinya ke brand sendiri.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'admin_travel' AND p.code = 'bankaccount.view';
