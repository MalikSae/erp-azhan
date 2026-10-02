-- 076_super_admin_name.sql
-- Konsistensi model dua sumbu: nama role tidak membawa kata cakupan.
-- "Super Admin Holding" -> "Super Admin" (cakupan tetap dari Akses Brand).
UPDATE roles SET name = 'Super Admin' WHERE slug = 'super_admin_grup';
