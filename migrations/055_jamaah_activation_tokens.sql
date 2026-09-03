-- Tabel token aktivasi akun Portal Jamaah
CREATE TABLE IF NOT EXISTS jamaah_activation_tokens (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    jamaah_id BIGINT UNSIGNED NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    expires_at DATETIME NOT NULL,
    used_at DATETIME NULL,
    created_by BIGINT UNSIGNED NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_activation_jamaah_id (jamaah_id),
    CONSTRAINT fk_activation_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah (id) ON DELETE CASCADE,
    CONSTRAINT fk_activation_created_by FOREIGN KEY (created_by) REFERENCES admin_users (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;