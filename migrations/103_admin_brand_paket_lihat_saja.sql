-- 103_admin_brand_paket_lihat_saja.sql
-- Paket (jadwal, harga, kuota seat) dikelola holding. Admin Brand hanya melihat paket brand-nya.
DELETE rp FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE r.slug = 'admin_travel' AND p.code = 'schedule.edit';
