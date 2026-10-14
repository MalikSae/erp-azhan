-- 071_rbac.sql
-- RBAC Fase 1 (SPEK-RBAC-2026-10-02.md): role & permission sebagai data.
-- Catatan: ALTER TABLE perm_version tidak idempotent (MySQL tidak mendukung
-- ADD COLUMN IF NOT EXISTS) — jalankan file ini sekali saja.

CREATE TABLE IF NOT EXISTS roles (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  slug VARCHAR(64) UNIQUE NOT NULL,
  name VARCHAR(128) NOT NULL,
  is_system BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS permissions (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  code VARCHAR(96) UNIQUE NOT NULL,
  description VARCHAR(255) NOT NULL
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS role_permissions (
  role_id BIGINT UNSIGNED NOT NULL,
  permission_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (role_id, permission_id),
  CONSTRAINT fk_rp_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
  CONSTRAINT fk_rp_perm FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS user_roles (
  admin_user_id BIGINT UNSIGNED NOT NULL,
  role_id BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (admin_user_id, role_id),
  CONSTRAINT fk_ur_user FOREIGN KEY (admin_user_id) REFERENCES admin_users(id) ON DELETE CASCADE,
  CONSTRAINT fk_ur_role FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Paksa refresh klaim saat role/permission user berubah.
ALTER TABLE admin_users ADD COLUMN perm_version INT NOT NULL DEFAULT 0;

-- ── Seed permissions ─────────────────────────────────────────────────────────
INSERT IGNORE INTO permissions (code, description) VALUES
  ('paket.view', 'Lihat master paket'),
  ('paket.edit', 'Kelola master paket'),
  ('paket.approve', 'Approve paket/distribusi ke brand'),
  ('schedule.view', 'Lihat jadwal keberangkatan'),
  ('schedule.edit', 'Kelola jadwal keberangkatan'),
  ('masterdata.view', 'Lihat master data (hotel/maskapai/airport/kategori/itinerary/addon)'),
  ('masterdata.edit', 'Kelola master data'),
  ('brand.view', 'Lihat brand'),
  ('brand.edit', 'Kelola brand & domain'),
  ('booking.view', 'Lihat booking'),
  ('booking.edit', 'Kelola booking'),
  ('jamaah.view', 'Lihat data jamaah'),
  ('jamaah.edit', 'Kelola data jamaah'),
  ('dokumen.view', 'Lihat dokumen jamaah'),
  ('dokumen.edit', 'Kelola dokumen jamaah'),
  ('payment.view', 'Lihat pembayaran'),
  ('payment.verify', 'Verifikasi/tolak pembayaran'),
  ('refund.view', 'Lihat refund'),
  ('refund.edit', 'Kelola refund'),
  ('bankaccount.view', 'Lihat rekening bank'),
  ('bankaccount.edit', 'Kelola rekening bank'),
  ('report.finance.view', 'Lihat laporan keuangan'),
  ('perlengkapan.view', 'Lihat inventory perlengkapan'),
  ('perlengkapan.edit', 'Kelola inventory perlengkapan'),
  ('agen.view', 'Lihat agen Syiar'),
  ('agen.edit', 'Kelola agen Syiar'),
  ('agen.approve', 'Approve pengajuan agen'),
  ('agen.payout.approve', 'Approve pencairan komisi agen'),
  ('komisi.view', 'Lihat komisi & referral'),
  ('crmuser.admin', 'Kelola akun CS untuk CRM'),
  ('crmdeal.process', 'Proses deal dari CRM'),
  ('marketing.work.view', 'Lihat work item Marketing Hub'),
  ('marketing.work.edit', 'Kelola work item Marketing Hub'),
  ('marketing.approve.lead', 'Approval operasional level Lead (Marketing Hub)'),
  ('marketing.approve.cmo', 'Approval eskalasi level CMO (Marketing Hub)'),
  ('marketing.metric.edit', 'Input/impor metrik marketing harian'),
  ('marketing.report.view', 'Lihat laporan marketing'),
  ('report.analytics.view', 'Lihat analytics lintas brand'),
  ('user.admin', 'Kelola user, role, dan permission'),
  ('compliance.view', 'Lihat arsip legalitas/compliance'),
  ('compliance.edit', 'Kelola arsip legalitas/compliance'),
  ('audit.view', 'Lihat audit log');

-- ── Seed roles (dari modul.md) ───────────────────────────────────────────────
INSERT IGNORE INTO roles (slug, name, is_system) VALUES
  ('direktur', 'Direktur / Owner', TRUE),
  ('super_admin_grup', 'Super Admin Grup', TRUE),
  ('product_manager', 'Product Manager', TRUE),
  ('finance_grup', 'Finance Grup', TRUE),
  ('konten_seo', 'Konten & SEO', TRUE),
  ('cmo', 'CMO / Head of Marketing & Growth', TRUE),
  ('marketing_lead', 'Marketing Lead', TRUE),
  ('marketing_member', 'Marketing Member', TRUE),
  ('admin_travel', 'Admin Travel', TRUE),
  ('finance_travel', 'Finance Travel', TRUE),
  ('cs', 'Customer Service', TRUE),
  ('admin_cabang', 'Admin Cabang', TRUE);

-- ── Role ↔ permission ────────────────────────────────────────────────────────
-- super_admin_grup: semua permission
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p WHERE r.slug = 'super_admin_grup';

-- direktur: semua *.view + report.* + compliance.view
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'direktur' AND (p.code LIKE '%.view' OR p.code LIKE 'report.%');

-- product_manager
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'product_manager' AND (
  p.code LIKE 'paket.%' OR p.code LIKE 'schedule.%' OR p.code LIKE 'masterdata.%'
  OR p.code LIKE 'perlengkapan.%' OR p.code = 'report.analytics.view');

-- finance_grup
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'finance_grup' AND (
  p.code LIKE 'payment.%' OR p.code LIKE 'refund.%' OR p.code LIKE 'bankaccount.%'
  OR p.code IN ('report.finance.view', 'booking.view', 'paket.view'));

-- konten_seo
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'konten_seo' AND (p.code LIKE 'marketing.work.%' OR p.code = 'report.analytics.view');

-- cmo
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'cmo' AND (
  p.code LIKE 'marketing.%' OR p.code LIKE 'report.%' OR p.code IN ('agen.view', 'booking.view'));

-- marketing_lead
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'marketing_lead' AND (
  p.code LIKE 'marketing.work.%'
  OR p.code IN ('marketing.approve.lead', 'marketing.metric.edit', 'marketing.report.view'));

-- marketing_member
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'marketing_member' AND p.code IN ('marketing.work.view', 'marketing.work.edit');

-- admin_travel
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'admin_travel' AND (
  p.code LIKE 'booking.%' OR p.code LIKE 'jamaah.%' OR p.code LIKE 'dokumen.%'
  OR p.code IN ('schedule.view', 'perlengkapan.view', 'payment.view',
                'agen.view', 'agen.edit', 'agen.approve', 'komisi.view',
                'crmuser.admin', 'report.analytics.view'));

-- finance_travel
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'finance_travel' AND (
  p.code LIKE 'payment.%' OR p.code LIKE 'refund.%'
  OR p.code IN ('booking.view', 'dokumen.view', 'bankaccount.view', 'report.finance.view'));

-- cs
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'cs' AND p.code IN ('crmdeal.process', 'schedule.view');

-- admin_cabang (aktif penuh saat scope cabang dibangun)
INSERT IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'admin_cabang' AND p.code IN (
  'booking.view', 'booking.edit', 'jamaah.view', 'jamaah.edit',
  'dokumen.view', 'dokumen.edit', 'schedule.view', 'perlengkapan.view');

-- ── Pemetaan akun lama → role baru ───────────────────────────────────────────
INSERT IGNORE INTO user_roles (admin_user_id, role_id)
SELECT u.id, r.id
FROM admin_users u
JOIN roles r ON r.slug = CASE
  WHEN u.role = 'cs' THEN 'cs'
  WHEN u.role = 'admin' AND u.brand_id IS NULL THEN 'super_admin_grup'
  ELSE 'admin_travel'
END;
