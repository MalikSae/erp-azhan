-- 127_komisi_bawaan_brand.sql
-- Audit menu Paket 10 November 2026 (T-01), keputusan pemilik proyek.
--
-- Nominal komisi agen kini punya bawaan per brand, dan paket boleh menimpanya:
--   komisi langsung  = COALESCE(schedules.nominal_komisi_langsung, NULLIF(brands.ujroh_closing, 0))
--   bonus pembinaan  = COALESCE(schedules.nominal_bonus_pembinaan, NULLIF(brands.ujroh_pembinaan, 0))
-- Paket kosong (NULL) ikut brand; paket 0 berarti sengaja tanpa komisi; brand 0 berarti brand belum ikut
-- program. Kolom brands.ujroh_closing dan ujroh_pembinaan sudah ada sejak migrasi 091 tetapi belum dipakai.
--
-- 1. Tabel brand_komisi_log: riwayat perubahan bawaan brand.
-- 2. Nilai awal Hana (kode_brand 'HN'): 500.000 dan 100.000, hanya bila keduanya masih 0.
-- 3. Isi susulan: booking 'baru' dan 'dp' yang snapshot komisinya kosong diisi nominal efektif paketnya.
--    Booking 'lunas' sengaja tidak disentuh (lihat scripts/daftar_booking_lunas_tanpa_snapshot_komisi.sql).
--
-- Aman dijalankan ulang.

CREATE TABLE IF NOT EXISTS brand_komisi_log (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  brand_id BIGINT UNSIGNED NOT NULL,
  komisi_langsung_lama DECIMAL(12,0) NOT NULL,
  komisi_langsung_baru DECIMAL(12,0) NOT NULL,
  bonus_pembinaan_lama DECIMAL(12,0) NOT NULL,
  bonus_pembinaan_baru DECIMAL(12,0) NOT NULL,
  diubah_oleh BIGINT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_brand_komisi_log_brand (brand_id, created_at)
);

UPDATE brands
SET ujroh_closing = 500000, ujroh_pembinaan = 100000
WHERE kode_brand = 'HN' AND ujroh_closing = 0 AND ujroh_pembinaan = 0;

UPDATE bookings b
JOIN schedules s ON s.id = b.schedule_id
JOIN brands br ON br.id = s.brand_id
SET b.snapshot_komisi_langsung = COALESCE(s.nominal_komisi_langsung, NULLIF(br.ujroh_closing, 0)),
    b.snapshot_bonus_pembinaan = COALESCE(s.nominal_bonus_pembinaan, NULLIF(br.ujroh_pembinaan, 0))
WHERE b.status IN ('baru', 'dp')
  AND b.snapshot_komisi_langsung IS NULL
  AND b.snapshot_bonus_pembinaan IS NULL;
