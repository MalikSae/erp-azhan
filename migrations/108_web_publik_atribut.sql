-- 108_web_publik_atribut.sql
-- Atribut klasifikasi paket untuk web publik/CMS SEO (SPEK-WEB-PUBLIK-CMS-2026-10-20.md §4.1).
-- ERP hanya memegang fakta klasifikasi; slug URL dan konten SEO dimiliki CMS.
--   - destinations: master destinasi (Umroh Plus), global seperti maskapai/bandara.
--   - schedules.journey_type / program_slug / destination_id: klasifikasi per keberangkatan.
--   - package_categories.seo_tier: pemetaan kategori ke tier SEO (hemat/reguler/comfort/premium).
-- Durasi tidak disimpan: dihitung dari berangkat_tanggal dan pulang_tanggal.
-- Catatan: ALTER TABLE ... ADD COLUMN tidak idempotent; jalankan sekali.
CREATE TABLE IF NOT EXISTS destinations (
  id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name       VARCHAR(100) NOT NULL,
  slug       VARCHAR(100) NOT NULL,
  is_active  BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_destination_slug (slug)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

ALTER TABLE schedules
  ADD COLUMN journey_type   ENUM('umroh','haji','wisata_halal') NOT NULL DEFAULT 'umroh' AFTER category_id,
  ADD COLUMN program_slug   VARCHAR(50) NULL AFTER journey_type,
  ADD COLUMN destination_id BIGINT UNSIGNED NULL AFTER program_slug,
  ADD KEY idx_schedules_destination (destination_id),
  ADD CONSTRAINT fk_schedules_destination FOREIGN KEY (destination_id) REFERENCES destinations (id);

ALTER TABLE package_categories
  ADD COLUMN seo_tier ENUM('hemat','reguler','comfort','premium') NULL AFTER slug;
