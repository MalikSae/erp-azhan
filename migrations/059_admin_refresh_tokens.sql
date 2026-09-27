-- 059: Simpan refresh token admin di server agar bisa dicabut (logout,
-- reset/ganti password, akun dinonaktifkan). Refresh token yang tidak
-- tercatat di tabel ini ditolak oleh /api/auth/refresh.
CREATE TABLE IF NOT EXISTS admin_refresh_tokens (
  jti            CHAR(36) PRIMARY KEY,
  admin_user_id  BIGINT UNSIGNED NOT NULL,
  expires_at     DATETIME NOT NULL,
  revoked_at     DATETIME NULL,
  created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_admin_refresh_user (admin_user_id),
  CONSTRAINT fk_admin_refresh_user FOREIGN KEY (admin_user_id)
    REFERENCES admin_users(id) ON DELETE CASCADE
) ENGINE=InnoDB;
