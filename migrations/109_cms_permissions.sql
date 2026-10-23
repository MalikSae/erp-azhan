-- 109_cms_permissions.sql
-- Izin CMS SEO (SPEK-WEB-PUBLIK-CMS-2026-10-20.md §4.4). Tim SEO bukan role (lihat 078):
-- kemampuan anggotanya mengikuti Marketing Member/Lead, cakupan brand dari Akses Brand.
--   cms.view    : melihat admin CMS
--   cms.edit    : mengubah SEO, slug, dan konten (draft)
--   cms.publish : menerbitkan perubahan, menyetujui landing
INSERT IGNORE INTO permissions (code, description) VALUES
  ('cms.view', 'Lihat admin CMS web publik'),
  ('cms.edit', 'Ubah SEO, slug, dan konten CMS (draft)'),
  ('cms.publish', 'Terbitkan perubahan CMS dan setujui landing SEO');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'super_admin_grup' AND p.code LIKE 'cms.%';

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('direktur', 'product_manager') AND p.code = 'cms.view';

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'cmo' AND p.code LIKE 'cms.%';

-- Daftar eksplisit (tinjauan keamanan 2026-10-22), bukan LIKE 'cms.%': bila 109 dijalankan ulang
-- setelah 110, LIKE ikut memberi marketing_lead izin Fase 2 di luar rancangan (mis.
-- cms.tracking.manage). Untuk database yang sudah menjalankan 109 sebelum 110, hasil akhirnya
-- sama persis (saat itu hanya ada tiga izin ini; INSERT IGNORE, idempoten).
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'marketing_lead' AND p.code IN ('cms.view', 'cms.edit', 'cms.publish');

INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'marketing_member' AND p.code IN ('cms.view', 'cms.edit');
