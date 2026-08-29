-- 043_jamaah_relasi.sql
-- Tabel hubungan kekerabatan/mahram antar jamaah (dua arah) & field catatan bebas jamaah

CREATE TABLE IF NOT EXISTS jamaah_relasi (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  jamaah_id BIGINT UNSIGNED NOT NULL,
  relasi_jamaah_id BIGINT UNSIGNED NOT NULL,
  hubungan ENUM('Pasangan','Orang Tua','Anak','Saudara Kandung','Mahram','Kerabat Lain') NOT NULL,
  keterangan VARCHAR(255) NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (jamaah_id) REFERENCES jamaah(id) ON DELETE CASCADE,
  FOREIGN KEY (relasi_jamaah_id) REFERENCES jamaah(id) ON DELETE CASCADE,
  UNIQUE KEY uq_jamaah_relasi (jamaah_id, relasi_jamaah_id),
  CHECK (jamaah_id <> relasi_jamaah_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE jamaah ADD COLUMN catatan TEXT NULL;
