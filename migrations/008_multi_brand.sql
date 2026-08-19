CREATE TABLE brands (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  whatsapp_number VARCHAR(20),
  logo_url VARCHAR(500),
  primary_color VARCHAR(7),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

INSERT INTO brands (name) VALUES ('Travel Pertama');

CREATE TABLE brand_domains (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id BIGINT UNSIGNED NOT NULL,
  domain VARCHAR(255) UNIQUE NOT NULL,
  is_primary BOOLEAN DEFAULT TRUE,
  FOREIGN KEY (brand_id) REFERENCES brands(id)
) ENGINE=InnoDB;

ALTER TABLE admin_users ADD COLUMN brand_id BIGINT UNSIGNED NULL AFTER password_hash;
ALTER TABLE admin_users ADD FOREIGN KEY (brand_id) REFERENCES brands(id);

ALTER TABLE schedules ADD COLUMN brand_id BIGINT UNSIGNED NULL AFTER id;
UPDATE schedules SET brand_id = (SELECT id FROM brands WHERE name = 'Travel Pertama' LIMIT 1);
ALTER TABLE schedules MODIFY COLUMN brand_id BIGINT UNSIGNED NOT NULL;
ALTER TABLE schedules ADD FOREIGN KEY (brand_id) REFERENCES brands(id);
