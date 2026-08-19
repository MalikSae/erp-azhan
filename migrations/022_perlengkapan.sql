CREATE TABLE perlengkapan_items (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id        BIGINT UNSIGNED NOT NULL,
  nama            VARCHAR(150) NOT NULL,
  stok_tersedia   INT NOT NULL DEFAULT 0,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY unique_brand_nama (brand_id, nama),
  FOREIGN KEY (brand_id) REFERENCES brands(id)
) ENGINE=InnoDB;

CREATE TABLE perlengkapan_set_items (
  brand_id                BIGINT UNSIGNED NOT NULL,
  perlengkapan_item_id    BIGINT UNSIGNED NOT NULL,
  qty                     INT NOT NULL DEFAULT 1,
  PRIMARY KEY (brand_id, perlengkapan_item_id),
  FOREIGN KEY (brand_id) REFERENCES brands(id),
  FOREIGN KEY (perlengkapan_item_id) REFERENCES perlengkapan_items(id) ON DELETE CASCADE
) ENGINE=InnoDB;

ALTER TABLE bookings
  ADD COLUMN perlengkapan_status ENUM('belum_diberikan','sudah_diberikan') NOT NULL DEFAULT 'belum_diberikan',
  ADD COLUMN perlengkapan_tanggal DATE NULL,
  ADD COLUMN perlengkapan_diberikan_oleh BIGINT UNSIGNED NULL,
  ADD FOREIGN KEY (perlengkapan_diberikan_oleh) REFERENCES admin_users(id);
