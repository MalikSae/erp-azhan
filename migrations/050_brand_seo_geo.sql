-- Migration 050: Tambah field SEO, GEO, dan Local NAP pada tabel brands
ALTER TABLE brands
ADD COLUMN meta_title VARCHAR(150) NULL AFTER primary_color,
ADD COLUMN meta_description VARCHAR(255) NULL AFTER meta_title,
ADD COLUMN og_image_url VARCHAR(255) NULL AFTER meta_description,
ADD COLUMN city VARCHAR(100) NULL AFTER address,
ADD COLUMN province VARCHAR(100) NULL AFTER city,
ADD COLUMN email VARCHAR(100) NULL AFTER whatsapp_number,
ADD COLUMN phone VARCHAR(50) NULL AFTER email,
ADD COLUMN google_verification_code VARCHAR(100) NULL AFTER og_image_url;
