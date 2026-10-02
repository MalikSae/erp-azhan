-- 078_drop_konten_seo_role.sql
-- Konsistensi model: "Konten & SEO" adalah TIM (mkt_teams, Marketing Hub),
-- bukan role — permission-nya kembaran Marketing Member (+ analytics).
-- Pemegang lama dipindah ke marketing_member.

INSERT IGNORE INTO user_roles (admin_user_id, role_id)
SELECT ur.admin_user_id, (SELECT id FROM roles WHERE slug = 'marketing_member')
FROM user_roles ur JOIN roles old ON old.id = ur.role_id
WHERE old.slug = 'konten_seo';

DELETE FROM roles WHERE slug = 'konten_seo';
