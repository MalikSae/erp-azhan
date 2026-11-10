-- 126_komisi_batal_dan_lunas_terakhir.sql
-- Audit menu Agen 10 November 2026, keputusan pemilik proyek (revisi K20, lihat K25-K27 di spek keagenan).
--
-- Komisi mengikuti lunas terakhir. Saat booking turun dari 'lunas' ke 'dp' atau 'baru', baris komisinya
-- ditandai batal (tidak dihapus); saat lunas lagi, baris baru dicatat dengan penerima saat itu.
--
-- 1. Kolom dibatalkan_at dan sebab_batal.
-- 2. Kolom turunan `aktif` (1 bila belum batal, NULL bila batal) untuk kunci unik: hanya satu baris aktif
--    per pax dan jenis komisi, siapa pun penerimanya, sedangkan baris batal boleh berapa pun (NULL tidak
--    saling bertabrakan). Ini menggantikan uq_komisi_pax_jenis_penerima, yang memuat penerima sehingga
--    penerima baru menghasilkan baris kedua untuk pax yang sama.
-- 3. Data lama: komisi booking yang kini berstatus 'dp' atau 'baru' ditandai batal; pasangan pax dan jenis
--    yang masih punya lebih dari satu baris aktif hanya menyisakan baris terbaru.
--
-- Aman dijalankan ulang.

SET @ada_kolom = (
  SELECT COUNT(*) FROM information_schema.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'transaksi_komisi' AND COLUMN_NAME = 'dibatalkan_at'
);
SET @sql = IF(
  @ada_kolom = 0,
  'ALTER TABLE transaksi_komisi
     ADD COLUMN dibatalkan_at DATETIME NULL,
     ADD COLUMN sebab_batal VARCHAR(40) NULL,
     ADD COLUMN aktif TINYINT GENERATED ALWAYS AS (IF(dibatalkan_at IS NULL, 1, NULL)) STORED',
  'DO 0'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- Data lama: booking yang tidak lunas tidak punya komisi aktif. Booking batal tidak disentuh (K20).
UPDATE transaksi_komisi tk
JOIN bookings b ON b.id = tk.booking_id
SET tk.dibatalkan_at = NOW(), tk.sebab_batal = 'penyesuaian_aturan'
WHERE tk.dibatalkan_at IS NULL AND b.status IN ('dp', 'baru');

-- Data lama: sisakan satu baris aktif per pax dan jenis (yang terbaru).
UPDATE transaksi_komisi tk
JOIN (
  SELECT booking_pax_id, jenis, MAX(id) AS terbaru
  FROM transaksi_komisi
  WHERE dibatalkan_at IS NULL
  GROUP BY booking_pax_id, jenis
  HAVING COUNT(*) > 1
) ganda ON ganda.booking_pax_id = tk.booking_pax_id AND ganda.jenis = tk.jenis
SET tk.dibatalkan_at = NOW(), tk.sebab_batal = 'penyesuaian_aturan'
WHERE tk.dibatalkan_at IS NULL AND tk.id <> ganda.terbaru;

-- Kunci unik baru dibuat dulu supaya foreign key booking_pax_id tetap punya indeks saat kunci lama dihapus.
SET @ada_kunci_baru = (
  SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'transaksi_komisi' AND INDEX_NAME = 'uq_komisi_pax_jenis_aktif'
);
SET @sql = IF(
  @ada_kunci_baru = 0,
  'ALTER TABLE transaksi_komisi ADD UNIQUE KEY uq_komisi_pax_jenis_aktif (booking_pax_id, jenis, aktif)',
  'DO 0'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @ada_kunci_lama = (
  SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'transaksi_komisi' AND INDEX_NAME = 'uq_komisi_pax_jenis_penerima'
);
SET @sql = IF(
  @ada_kunci_lama > 0,
  'ALTER TABLE transaksi_komisi DROP INDEX uq_komisi_pax_jenis_penerima',
  'DO 0'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
