-- 082_hapus_permission_crmuser.sql
-- Endpoint /api/admin/crm/users & modul crmuser dihapus (SPEK-PENGGUNA-
-- ORGANISASI §5): tidak dipanggil frontend ERP maupun repo CRM, dan menjadi
-- pintu kedua pembuatan akun. Akun CS kini dibuat lewat halaman Pengguna.
-- role_permissions ikut terhapus lewat FK ON DELETE CASCADE.
DELETE FROM permissions WHERE code = 'crmuser.admin';
