-- 073_role_display_names.sql
-- Audit nama role (2 Okt 2026): selaraskan dengan istilah "brand" yang dipakai
-- UI & arsitektur. Hanya nama tampilan — slug tidak berubah.

UPDATE roles SET name = 'Admin Brand' WHERE slug = 'admin_travel';
UPDATE roles SET name = 'Finance Brand' WHERE slug = 'finance_travel';
