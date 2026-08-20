-- Rekening transfer manual per brand dan alur konfirmasi pembayaran portal.
CREATE TABLE IF NOT EXISTS bank_accounts (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id BIGINT UNSIGNED NOT NULL,
  bank_name VARCHAR(100) NOT NULL,
  account_number VARCHAR(50) NOT NULL,
  account_holder VARCHAR(150) NOT NULL,
  instructions VARCHAR(500) NULL,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  sort_order INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_bank_account_brand_number (brand_id, account_number),
  CONSTRAINT fk_bank_account_brand FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE
) ENGINE=InnoDB;

INSERT IGNORE INTO bank_accounts (brand_id, bank_name, account_number, account_holder, sort_order)
SELECT id, bank_name, bank_account_number, bank_account_holder, 0
FROM brands
WHERE bank_name IS NOT NULL AND bank_name <> ''
  AND bank_account_number IS NOT NULL AND bank_account_number <> ''
  AND bank_account_holder IS NOT NULL AND bank_account_holder <> '';

ALTER TABLE payments
  MODIFY COLUMN status ENUM('pending','confirmed','rejected') NOT NULL DEFAULT 'pending',
  ADD COLUMN bank_account_id BIGINT UNSIGNED NULL AFTER booking_id,
  ADD COLUMN destination_bank_name VARCHAR(100) NULL AFTER bank_account_id,
  ADD COLUMN destination_account_number VARCHAR(50) NULL AFTER destination_bank_name,
  ADD COLUMN destination_account_holder VARCHAR(150) NULL AFTER destination_account_number,
  ADD COLUMN sender_name VARCHAR(150) NULL AFTER metode,
  ADD COLUMN sender_bank VARCHAR(100) NULL AFTER sender_name,
  ADD COLUMN notes VARCHAR(500) NULL AFTER bukti_url,
  ADD COLUMN source ENUM('admin','portal') NOT NULL DEFAULT 'admin' AFTER notes,
  ADD COLUMN rejection_reason VARCHAR(500) NULL AFTER source,
  ADD COLUMN verified_by BIGINT UNSIGNED NULL AFTER rejection_reason,
  ADD COLUMN verified_at DATETIME NULL AFTER verified_by,
  ADD CONSTRAINT fk_payment_bank_account FOREIGN KEY (bank_account_id) REFERENCES bank_accounts(id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_payment_verified_by FOREIGN KEY (verified_by) REFERENCES admin_users(id) ON DELETE SET NULL;
