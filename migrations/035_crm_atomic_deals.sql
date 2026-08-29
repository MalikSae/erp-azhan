-- Dukungan transaksi Deal CRM yang atomik, idempotent, dan aware jumlah pax.

ALTER TABLE bookings
  ADD COLUMN seat_count SMALLINT UNSIGNED NOT NULL DEFAULT 1 AFTER room_type,
  ADD COLUMN seat_hold_expires_at DATETIME NULL AFTER is_seat_blocked,
  ADD COLUMN seat_hold_key CHAR(36) NULL AFTER seat_hold_expires_at,
  ADD UNIQUE KEY uq_bookings_seat_hold_key (seat_hold_key),
  ADD KEY idx_bookings_expired_seat_hold (is_seat_blocked, seat_hold_expires_at, status);

ALTER TABLE payments
  MODIFY COLUMN source ENUM('admin','portal','crm') NOT NULL DEFAULT 'admin';

CREATE TABLE crm_deal_requests (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id BIGINT UNSIGNED NOT NULL,
  crm_lead_id CHAR(36) NOT NULL,
  idempotency_key CHAR(36) NOT NULL,
  request_hash CHAR(64) NOT NULL,
  status ENUM('processing','completed') NOT NULL DEFAULT 'processing',
  jamaah_id BIGINT UNSIGNED NULL,
  booking_id BIGINT UNSIGNED NULL,
  payment_id BIGINT UNSIGNED NULL,
  response_payload JSON NULL,
  created_by BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  UNIQUE KEY uq_crm_deal_idempotency (idempotency_key),
  UNIQUE KEY uq_crm_deal_lead (brand_id, crm_lead_id),
  CONSTRAINT fk_crm_deal_brand FOREIGN KEY (brand_id) REFERENCES brands(id),
  CONSTRAINT fk_crm_deal_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah(id),
  CONSTRAINT fk_crm_deal_booking FOREIGN KEY (booking_id) REFERENCES bookings(id),
  CONSTRAINT fk_crm_deal_payment FOREIGN KEY (payment_id) REFERENCES payments(id),
  CONSTRAINT fk_crm_deal_created_by FOREIGN KEY (created_by) REFERENCES admin_users(id)
) ENGINE=InnoDB;
