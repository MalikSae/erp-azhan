-- 075_merge_finance_role.sql
-- Audit model RBAC (2 Okt 2026): Finance Holding & Finance Brand dilebur jadi
-- satu role "Finance". Role = kemampuan (permission); cakupan data diatur
-- Akses Brand (brand_id), sehingga pasangan per-level redundan dan membuka
-- kombinasi membingungkan (mis. Finance Holding + akses brand A).

INSERT IGNORE INTO roles (slug, name, is_system) VALUES ('finance', 'Finance', TRUE);

-- Union permission kedua role lama.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'finance' AND (
  p.code LIKE 'payment.%' OR p.code LIKE 'refund.%' OR p.code LIKE 'bankaccount.%'
  OR p.code IN ('report.finance.view', 'booking.view', 'paket.view',
                'dokumen.view', 'schedule.view', 'perlengkapan.view'));

-- Pindahkan user yang terlanjur memegang role lama.
INSERT IGNORE INTO user_roles (admin_user_id, role_id)
SELECT ur.admin_user_id, (SELECT id FROM roles WHERE slug = 'finance')
FROM user_roles ur JOIN roles old ON old.id = ur.role_id
WHERE old.slug IN ('finance_grup', 'finance_travel');

-- Hapus role lama (role_permissions & user_roles lama ikut lewat FK cascade).
DELETE FROM roles WHERE slug IN ('finance_grup', 'finance_travel');
