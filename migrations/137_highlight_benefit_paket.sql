-- 137: highlight benefit paket (keputusan 13 November 2026).
--
-- Tiap paket menampilkan 4 highlight benefit (ikon + teks pendek) di aplikasi. Benefit diambil dari master
-- per brand supaya teksnya seragam antar paket dan cukup diubah di satu tempat.
--
-- paket_benefits: master benefit per brand. Teks maksimal 30 karakter dan tidak boleh kembar dalam satu
-- brand; kolasi utf8mb4_unicode_ci membuat indeks unik (brand_id, teks) tidak membedakan huruf besar-kecil.
-- Kunci ikon divalidasi di kode (benefit.DaftarIkon). Benefit nonaktif tidak ditawarkan untuk pilihan baru,
-- tetapi tetap tampil di paket yang sudah memakainya.
--
-- schedule_benefits: benefit pilihan tiap paket beserta urutannya (1 sampai 4). Benefit yang masih dipakai
-- paket tidak bisa dihapus (ON DELETE RESTRICT); menghapus paket ikut menghapus tautannya.
--
-- Bibit: empat benefit awal untuk setiap brand. Paket yang sudah ada TIDAK ditautkan ke benefit mana pun;
-- paket itu tetap tayang tanpa benefit sampai admin mengeditnya.

CREATE TABLE IF NOT EXISTS paket_benefits (
  id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  brand_id    BIGINT UNSIGNED NOT NULL,
  teks        VARCHAR(30) NOT NULL,
  ikon        VARCHAR(64) NOT NULL,
  is_active   TINYINT(1) NOT NULL DEFAULT 1,
  created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_paket_benefits_brand_teks (brand_id, teks),
  CONSTRAINT fk_paket_benefits_brand FOREIGN KEY (brand_id) REFERENCES brands (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS schedule_benefits (
  schedule_id  BIGINT UNSIGNED NOT NULL,
  benefit_id   BIGINT UNSIGNED NOT NULL,
  urutan       TINYINT UNSIGNED NOT NULL,
  PRIMARY KEY (schedule_id, urutan),
  UNIQUE KEY uq_schedule_benefits_benefit (schedule_id, benefit_id),
  KEY idx_schedule_benefits_benefit (benefit_id),
  CONSTRAINT fk_schedule_benefits_schedule FOREIGN KEY (schedule_id) REFERENCES schedules (id) ON DELETE CASCADE,
  CONSTRAINT fk_schedule_benefits_benefit FOREIGN KEY (benefit_id) REFERENCES paket_benefits (id) ON DELETE RESTRICT
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

INSERT IGNORE INTO paket_benefits (brand_id, teks, ikon)
SELECT b.id, d.teks, d.ikon
FROM brands b
CROSS JOIN (
  SELECT 'Hotel dekat Haram' AS teks, 'solar:bed-bold-duotone' AS ikon
  UNION ALL SELECT 'Ustadz pembimbing', 'solar:user-speak-bold-duotone'
  UNION ALL SELECT 'Transportasi bus', 'solar:bus-bold-duotone'
  UNION ALL SELECT 'Makan 3x sehari', 'solar:chef-hat-bold-duotone'
) d;
