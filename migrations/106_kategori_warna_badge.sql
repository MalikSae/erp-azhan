-- Warna badge kategori paket: hex #rrggbb dari color picker (NULL = otomatis oleh situs booking).
ALTER TABLE package_categories
    ADD COLUMN badge_color VARCHAR(20) NULL AFTER description;
