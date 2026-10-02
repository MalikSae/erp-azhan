-- 072_rbac_fase2.sql
-- RBAC Fase 2: penyesuaian seed sebelum alih guard route ke RequirePermission.
-- Prinsip Fase 2: pemetaan permission MENIRU akses yang berlaku hari ini;
-- pengetatan kebijakan adalah keputusan terpisah (lihat SPEK-RBAC §8).

-- Permission baru yang dibutuhkan pemetaan route.
INSERT IGNORE INTO permissions (code, description) VALUES
  ('payment.edit', 'Input/catat pembayaran booking'),
  ('agen.kaitan.edit', 'Ganti kaitan agen jamaah (admin master)');

-- Permission baru wajib ikut diberikan ke super_admin_grup (grant "semua"
-- di 071 adalah snapshot saat migrasi, bukan aturan dinamis).
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'super_admin_grup' AND p.code IN ('payment.edit', 'agen.kaitan.edit');

-- payment.edit untuk fungsi yang hari ini menginput pembayaran.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('finance_grup', 'finance_travel', 'admin_travel')
  AND p.code = 'payment.edit';

-- CS: allowlist CRM mengizinkan GET /bookings/{id}/payments → butuh payment.view.
-- (Path lain tetap tertutup oleh RequireAdminOrCRMAccess.)
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'cs' AND p.code = 'payment.view';

-- admin_travel: hari ini (guard lama) bisa membaca master data, mengelola
-- jadwal, memverifikasi pembayaran, refund, stok perlengkapan, dan memproses
-- deal CRM. Pemetaan setia → tambahkan permission-nya.
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'admin_travel' AND p.code IN (
  'masterdata.view', 'paket.view', 'schedule.edit', 'payment.verify',
  'refund.view', 'refund.edit', 'perlengkapan.edit', 'crmdeal.process');

-- Koreksi 071: analytics lintas brand bukan hak Admin Travel (lihat modul.md:
-- Admin Travel = Laporan Operasional Brand, bukan lintas brand).
DELETE rp FROM role_permissions rp
JOIN roles r ON r.id = rp.role_id
JOIN permissions p ON p.id = rp.permission_id
WHERE r.slug = 'admin_travel' AND p.code = 'report.analytics.view';

-- Koreksi 071: finance_travel juga melihat jadwal & inventory (modul.md).
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'finance_travel' AND p.code IN ('schedule.view', 'perlengkapan.view');
