-- 100_hapus_gudang_pusat.sql
-- Tidak ada gudang pusat: barang yang dibeli langsung masuk ke stok brand.
-- Menghapus stok pusat & seluruh baris buku besar yang bersangkutan (pusat/transfer),
-- lalu menjadikan brand_id wajib pada perlengkapan_mutasi.
-- Catatan: jalankan file ini sekali saja.

DELETE FROM perlengkapan_mutasi WHERE brand_id IS NULL OR jenis = 'transfer';

-- Sisa data uji dari fitur gudang pusat.
DELETE FROM perlengkapan_mutasi WHERE catatan IN ('2 rusak', 'reset data uji');

ALTER TABLE perlengkapan_mutasi
  MODIFY brand_id BIGINT UNSIGNED NOT NULL,
  MODIFY jenis ENUM('pembelian','distribusi','retur','koreksi') NOT NULL;

DROP TABLE IF EXISTS perlengkapan_stok_pusat;
