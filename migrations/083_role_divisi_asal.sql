-- 083_role_divisi_asal.sql
-- Divisi asal role (SPEK-PENGGUNA-ORGANISASI §R10): role menentukan unit mana
-- yang boleh dipilih di Penugasan. Role lalu tidak bisa bertentangan dengan
-- penugasan (mis. Finance di Marketing). NULL = role tanpa divisi Holding;
-- pengecualian: Super Admin boleh di unit mana pun (ditegakkan di service).
-- Catatan: ALTER TABLE tidak idempotent — jalankan file ini sekali saja.

ALTER TABLE roles
  ADD COLUMN divisi_id BIGINT UNSIGNED NULL AFTER scope,
  ADD CONSTRAINT fk_roles_divisi FOREIGN KEY (divisi_id) REFERENCES org_unit(id) ON DELETE RESTRICT;

UPDATE roles r
JOIN org_unit u ON u.parent_id IS NULL AND u.jenis = 'divisi' AND u.nama = 'Manajemen'
SET r.divisi_id = u.id WHERE r.slug = 'direktur';

UPDATE roles r
JOIN org_unit u ON u.parent_id IS NULL AND u.jenis = 'divisi' AND u.nama = 'Keuangan'
SET r.divisi_id = u.id WHERE r.slug = 'finance';

UPDATE roles r
JOIN org_unit u ON u.parent_id IS NULL AND u.jenis = 'divisi' AND u.nama = 'Produk'
SET r.divisi_id = u.id WHERE r.slug = 'product_manager';

UPDATE roles r
JOIN org_unit u ON u.parent_id IS NULL AND u.jenis = 'divisi' AND u.nama = 'Marketing & Growth'
SET r.divisi_id = u.id WHERE r.slug IN ('cmo', 'marketing_lead', 'marketing_member');
