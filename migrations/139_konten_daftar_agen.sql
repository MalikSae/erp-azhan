-- 139: konten dan aturan pendaftaran agen dari app (keputusan 14 November 2026).
--
-- Brand: gambar rekrutmen agen, daftar benefit agen (satu benefit per baris), dan draft MoU agen (HTML terbatas dari
-- editor admin). syarat_ketentuan_agen yang sudah ada kini juga berisi HTML terbatas; teks lama tetap terbaca.
-- Pengajuan (siklus pembayaran_pendaftaran_agen): salinan MoU yang disetujui calon agen, waktunya, dan penanda
-- pengajuan yang disetujui otomatis karena akunnya jamaah (punya booking minimal DP).
ALTER TABLE brands
  ADD COLUMN gambar_rekrutmen_agen VARCHAR(500) NULL AFTER syarat_ketentuan_agen,
  ADD COLUMN benefit_agen TEXT NULL AFTER gambar_rekrutmen_agen,
  ADD COLUMN mou_agen MEDIUMTEXT NULL AFTER benefit_agen;

ALTER TABLE pembayaran_pendaftaran_agen
  ADD COLUMN mou_disetujui MEDIUMTEXT NULL,
  ADD COLUMN mou_disetujui_at DATETIME NULL,
  ADD COLUMN disetujui_otomatis TINYINT(1) NOT NULL DEFAULT 0;
