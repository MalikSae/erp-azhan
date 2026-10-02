-- 077_role_scope.sql
-- Role diberi scope supaya kombinasi tak bermakna (mis. Super Admin + akses
-- brand A) tidak bisa dipilih: 'holding' = hanya user akses holding,
-- 'brand' = hanya user terikat brand, 'semua' = keduanya.

ALTER TABLE roles ADD COLUMN scope ENUM('holding','brand','semua') NOT NULL DEFAULT 'semua';

UPDATE roles SET scope = 'holding' WHERE slug IN
  ('super_admin_grup','direktur','product_manager','konten_seo','cmo','marketing_lead','marketing_member');
UPDATE roles SET scope = 'brand' WHERE slug IN ('admin_travel','cs','admin_cabang');
UPDATE roles SET scope = 'semua' WHERE slug = 'finance';
