-- 161_hapus_crm_deal.sql
-- Modul crmdeal dan endpoint POST /api/admin/crm/deals dihapus. Deal CRM dan
-- fitur CS dibahas ulang saat integrasi dengan CRM dibangun.
-- Kolom bookings.seat_count, seat_hold_expires_at, dan seat_hold_key (migrasi
-- 035) tetap dipakai booking admin, self-booking, dan hold seat manual.
-- Catatan: MODIFY COLUMN gagal bila masih ada payments.source = 'crm'.

DROP TABLE IF EXISTS crm_deal_requests;

-- user_permissions dan role_permissions ikut terhapus lewat FK ON DELETE CASCADE.
DELETE FROM permissions WHERE code = 'crmdeal.process';

ALTER TABLE payments
  MODIFY COLUMN source ENUM('admin','portal') NOT NULL DEFAULT 'admin';
