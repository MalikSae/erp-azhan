-- Akun pelanggan (email + password) menggantikan akun portal berbasis nomor HP + PIN.
-- Tahap tambahan saja: tidak ada yang dihapus di sini. Penghapusan jalur PIN ada di 105.

CREATE TABLE IF NOT EXISTS customer_accounts (
  id                BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id          BIGINT UNSIGNED NOT NULL,
  nama_lengkap      VARCHAR(150) NOT NULL,
  email             VARCHAR(190) NOT NULL,
  email_normal      VARCHAR(190) NOT NULL,
  no_hp             VARCHAR(30)  NOT NULL,
  password_hash     VARCHAR(100) NOT NULL,
  email_verified_at DATETIME NULL,
  jamaah_id         BIGINT UNSIGNED NULL,
  last_login_at     DATETIME NULL,
  created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_customer_brand_email (brand_id, email_normal),
  KEY idx_customer_jamaah (jamaah_id),
  CONSTRAINT fk_customer_brand  FOREIGN KEY (brand_id)  REFERENCES brands (id),
  CONSTRAINT fk_customer_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah (id) ON DELETE SET NULL
) ENGINE=InnoDB;

-- Sesi buram: browser memegang token acak, database hanya menyimpan hash SHA-256.
CREATE TABLE IF NOT EXISTS customer_sessions (
  token_hash  CHAR(64) NOT NULL PRIMARY KEY,
  account_id  BIGINT UNSIGNED NOT NULL,
  expires_at  DATETIME NOT NULL,
  revoked_at  DATETIME NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_customer_sessions_account (account_id),
  KEY idx_customer_sessions_expiry (expires_at),
  CONSTRAINT fk_customer_sessions_account FOREIGN KEY (account_id) REFERENCES customer_accounts (id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Token sekali pakai: verifikasi email (24 jam) dan reset password (1 jam).
CREATE TABLE IF NOT EXISTS customer_tokens (
  token_hash  CHAR(64) NOT NULL PRIMARY KEY,
  account_id  BIGINT UNSIGNED NOT NULL,
  purpose     ENUM('verify_email','reset_password') NOT NULL,
  expires_at  DATETIME NOT NULL,
  used_at     DATETIME NULL,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_customer_tokens_account (account_id, purpose),
  CONSTRAINT fk_customer_tokens_account FOREIGN KEY (account_id) REFERENCES customer_accounts (id) ON DELETE CASCADE
) ENGINE=InnoDB;

-- Jamaah tersimpan: bahan prefill checkout, bukan sumber kebenaran data jamaah.
CREATE TABLE IF NOT EXISTS customer_travelers (
  id             BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  account_id     BIGINT UNSIGNED NOT NULL,
  nama_lengkap   VARCHAR(150) NOT NULL,
  nama_normal    VARCHAR(150) NOT NULL,
  jenis_kelamin  ENUM('L','P') NOT NULL,
  no_hp          VARCHAR(30) NULL,
  tanggal_lahir  DATE NULL,
  created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_traveler (account_id, nama_normal, tanggal_lahir),
  CONSTRAINT fk_traveler_account FOREIGN KEY (account_id) REFERENCES customer_accounts (id) ON DELETE CASCADE
) ENGINE=InnoDB;

ALTER TABLE bookings
  ADD COLUMN customer_account_id BIGINT UNSIGNED NULL,
  ADD KEY idx_bookings_customer (customer_account_id),
  ADD CONSTRAINT fk_bookings_customer FOREIGN KEY (customer_account_id) REFERENCES customer_accounts (id) ON DELETE SET NULL;

ALTER TABLE media_uploads
  ADD COLUMN customer_account_id BIGINT UNSIGNED NULL,
  ADD KEY idx_media_customer (customer_account_id);
