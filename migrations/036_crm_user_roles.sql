-- CRM role foundation. Existing brand users remain admins for backward compatibility.
ALTER TABLE admin_users
  ADD COLUMN display_name VARCHAR(120) NULL AFTER email,
  ADD COLUMN role ENUM('admin', 'cs') NOT NULL DEFAULT 'admin' AFTER brand_id,
  ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT TRUE AFTER role;

UPDATE admin_users
SET display_name = SUBSTRING_INDEX(email, '@', 1)
WHERE display_name IS NULL OR display_name = '';

CREATE INDEX idx_admin_users_brand_role_active
  ON admin_users (brand_id, role, is_active);
