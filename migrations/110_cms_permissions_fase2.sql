-- 110_cms_permissions_fase2.sql
-- Izin CMS Fase 2 yang lebih halus (SPEK-WEB-PUBLIK-CMS-2026-10-20.md §14.2 D4, Appendix F spek SEO).
-- Menambah izin di atas cms.view/edit/publish (109). Tim SEO tetap bukan role (078): kemampuan mengikuti role yang ada.
--   cms.blog.write      : menulis dan menyunting artikel blog (draft, review)
--   cms.blog.publish    : menerbitkan, menjadwalkan, dan menarik artikel blog
--   cms.schema.approve  : menyetujui JSON-LD kustom sebelum tayang
--   cms.tracking.manage : mengatur ID tracking (GTM, GA4, Meta Pixel, kode verifikasi); berdampak ke seluruh situs
INSERT IGNORE INTO permissions (code, description) VALUES
  ('cms.blog.write', 'Tulis dan sunting artikel blog (draft dan review)'),
  ('cms.blog.publish', 'Terbitkan, jadwalkan, dan tarik artikel blog'),
  ('cms.schema.approve', 'Setujui JSON-LD kustom CMS sebelum tayang'),
  ('cms.tracking.manage', 'Atur ID tracking dan kode verifikasi situs (GTM, GA4, Meta Pixel, Search Console)');

-- Super Admin Grup: semua izin cms.* termasuk yang baru (109 memakai LIKE, tetapi hanya untuk izin yang ada saat itu).
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'super_admin_grup' AND p.code LIKE 'cms.%';

-- CMO: seluruh izin cms.* (penerbitan, schema, dan tracking).
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'cmo' AND p.code LIKE 'cms.%';

-- Marketing Lead: menulis dan menerbitkan blog, menyetujui schema; BUKAN tracking (berdampak seluruh situs).
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'marketing_lead' AND p.code IN ('cms.blog.write', 'cms.blog.publish', 'cms.schema.approve');

-- Marketing Member: hanya menulis blog (tanpa menerbitkan).
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'marketing_member' AND p.code = 'cms.blog.write';
