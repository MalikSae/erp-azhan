-- Apply after 065. Existing sessions without PIN binding must log in again.
CREATE TABLE IF NOT EXISTS portal_revoked_tokens (
 jti VARCHAR(64) PRIMARY KEY,
 expires_at DATETIME NOT NULL,
 INDEX idx_portal_revoked_expiry (expires_at)
);

CREATE TABLE IF NOT EXISTS media_uploads (
 path VARCHAR(500) NOT NULL PRIMARY KEY,
 brand_id BIGINT UNSIGNED NULL,
 jamaah_id BIGINT UNSIGNED NULL,
 admin_user_id BIGINT UNSIGNED NULL,
 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
 INDEX idx_media_owner (jamaah_id),
 INDEX idx_media_brand (brand_id)
);

ALTER TABLE dokumen_jamaah
 ADD COLUMN version BIGINT UNSIGNED NOT NULL DEFAULT 1,
 ADD COLUMN rejection_reason VARCHAR(500) NULL;

CREATE TABLE IF NOT EXISTS dokumen_jamaah_versions (
 dokumen_id BIGINT UNSIGNED NOT NULL,
 version BIGINT UNSIGNED NOT NULL,
 file_url VARCHAR(500) NULL,
 status VARCHAR(30) NOT NULL,
 rejection_reason VARCHAR(500) NULL,
 recorded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
 PRIMARY KEY (dokumen_id, version)
);
