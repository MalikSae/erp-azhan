-- 114_agen_ke_akun.sql
-- Keagenan dipindah dari jamaah ke akun pelanggan (docs/spek/SPEK-KEAGENAN-AKUN-2026-11-03.md, bagian 10 butir 1, 2, 11).
-- Agen adalah AKUN (customer_accounts); jamaah hanya menyimpan kunci ke akun agen.
--
-- Tidak ada migrasi data: sistem belum pernah deploy production dan tidak ada agen lama (spek bagian 1).
-- Baris yang menunjuk jamaah sebagai agen/penerima komisi (ledger komisi, pembayaran pendaftaran, pencairan,
-- log kaitan) dihapus, dan kunci jamaah ke agen lama dilepas, supaya kunci asing ke akun bisa dipasang.
-- Hanya relevan untuk database pengembangan.

-- 1. Kolom agen di akun. Rekening pencairan ikut pindah karena pemilik saldo kini akun.
ALTER TABLE customer_accounts
  ADD COLUMN status_agen ENUM('tidak_aktif','pengajuan','aktif','nonaktif') NOT NULL DEFAULT 'tidak_aktif',
  ADD COLUMN kode_referral VARCHAR(20) NULL,
  ADD COLUMN foto_agen_url VARCHAR(500) NULL,
  ADD COLUMN domisili VARCHAR(100) NULL,
  ADD COLUMN menyetujui_syarat_ketentuan_agen_at DATETIME NULL,
  ADD COLUMN diajukan_agen_at DATETIME NULL,
  ADD COLUMN disetujui_agen_at DATETIME NULL,
  ADD COLUMN disetujui_agen_oleh BIGINT UNSIGNED NULL,
  ADD COLUMN direkrut_oleh_akun_id BIGINT UNSIGNED NULL,
  ADD COLUMN upline_akun_id BIGINT UNSIGNED NULL,
  ADD COLUMN rekening_bank_agen VARCHAR(100) NULL,
  ADD COLUMN rekening_nomor_agen VARCHAR(50) NULL,
  ADD COLUMN rekening_atas_nama_agen VARCHAR(150) NULL,
  ADD UNIQUE KEY uq_customer_brand_kode_referral (brand_id, kode_referral),
  ADD KEY idx_customer_direkrut (direkrut_oleh_akun_id),
  ADD KEY idx_customer_upline (upline_akun_id),
  ADD KEY idx_customer_brand_status_agen (brand_id, status_agen),
  ADD CONSTRAINT fk_customer_direkrut FOREIGN KEY (direkrut_oleh_akun_id) REFERENCES customer_accounts (id),
  ADD CONSTRAINT fk_customer_upline FOREIGN KEY (upline_akun_id) REFERENCES customer_accounts (id),
  ADD CONSTRAINT fk_customer_disetujui_agen_oleh FOREIGN KEY (disetujui_agen_oleh) REFERENCES admin_users (id);

-- 2. Bersihkan baris yang menunjuk jamaah sebagai agen.
DELETE FROM transaksi_komisi;
DELETE FROM pengajuan_pencairan;
DELETE FROM pembayaran_pendaftaran_agen_upload;
DELETE FROM pembayaran_pendaftaran_agen;
DELETE FROM jamaah_kaitan_log;
DELETE FROM pemakaian_cashback;
UPDATE jamaah SET direkrut_oleh_jamaah_id = NULL, kaitan_status = 'belum_ditentukan', kaitan_sumber = NULL
  WHERE kaitan_status = 'terikat_agen';
UPDATE jamaah SET direkrut_oleh_jamaah_id = NULL WHERE direkrut_oleh_jamaah_id IS NOT NULL;

-- 3. Jamaah: kolom agen dihapus; penunjuk kunci menjadi penunjuk ke akun agen.
-- uq_jamaah_brand_kode_referral dan idx_jamaah_brand_status_agen dihapus eksplisit: tanpa itu MySQL
-- menyusutkan indeks ke (brand_id) dan keunikan kode referral berubah menjadi satu baris per brand.
ALTER TABLE jamaah
  DROP FOREIGN KEY fk_jamaah_upline,
  DROP FOREIGN KEY fk_jamaah_direkrut,
  DROP FOREIGN KEY fk_jamaah_disetujui_agen_oleh;

ALTER TABLE jamaah
  DROP INDEX uq_jamaah_brand_kode_referral,
  DROP INDEX idx_jamaah_upline,
  DROP INDEX idx_jamaah_brand_status_agen,
  DROP COLUMN status_agen,
  DROP COLUMN kode_referral,
  DROP COLUMN upline_jamaah_id,
  DROP COLUMN foto_agen_url,
  DROP COLUMN domisili,
  DROP COLUMN diajukan_agen_at,
  DROP COLUMN disetujui_agen_at,
  DROP COLUMN disetujui_agen_oleh,
  DROP COLUMN menyetujui_syarat_ketentuan_agen_at,
  DROP COLUMN rekening_bank_agen,
  DROP COLUMN rekening_nomor_agen,
  DROP COLUMN rekening_atas_nama_agen,
  CHANGE COLUMN direkrut_oleh_jamaah_id direkrut_oleh_akun_id BIGINT UNSIGNED NULL,
  RENAME INDEX idx_jamaah_direkrut TO idx_jamaah_direkrut_akun,
  ADD CONSTRAINT fk_jamaah_direkrut_akun FOREIGN KEY (direkrut_oleh_akun_id) REFERENCES customer_accounts (id);

-- 4. Nomor WhatsApp jamaah menjadi data kontak biasa (spek 4.3, butir 11): batas unik dicabut.
-- uq_jamaah_brand_phone (nomor mentah) hanya ada di sebagian database karena ditambahkan manual,
-- jadi dicabut bila ada. Indeks biasa menggantikan untuk pencarian.
SET @ada_phone = (
  SELECT COUNT(*) FROM information_schema.STATISTICS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'jamaah' AND INDEX_NAME = 'uq_jamaah_brand_phone'
);
SET @sql = IF(@ada_phone > 0, 'ALTER TABLE jamaah DROP INDEX uq_jamaah_brand_phone', 'DO 0');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

ALTER TABLE jamaah
  DROP INDEX uq_jamaah_brand_hp,
  ADD KEY idx_jamaah_brand_hp (brand_id, no_hp_normal);

-- 5. Log kaitan: penunjuk agen lama/baru kini id akun.
ALTER TABLE jamaah_kaitan_log
  RENAME COLUMN direkrut_oleh_jamaah_id_lama TO direkrut_oleh_akun_id_lama,
  RENAME COLUMN direkrut_oleh_jamaah_id_baru TO direkrut_oleh_akun_id_baru;

-- 6. Ledger komisi: penerima adalah akun. Sumber tetap jamaah (pemilik pax).
ALTER TABLE transaksi_komisi DROP FOREIGN KEY fk_komisi_penerima;
ALTER TABLE transaksi_komisi
  RENAME COLUMN jamaah_penerima_id TO akun_penerima_id,
  ADD CONSTRAINT fk_komisi_penerima_akun FOREIGN KEY (akun_penerima_id) REFERENCES customer_accounts (id);

-- 7. Pembayaran pendaftaran agen: milik akun; status baru 'dibebaskan' (biaya dibebaskan admin).
ALTER TABLE pembayaran_pendaftaran_agen DROP FOREIGN KEY fk_ppa_jamaah;
ALTER TABLE pembayaran_pendaftaran_agen
  RENAME COLUMN jamaah_id TO akun_id,
  RENAME INDEX idx_ppa_jamaah TO idx_ppa_akun,
  MODIFY COLUMN status ENUM('menunggu_verifikasi','terverifikasi','ditolak','dibebaskan') NOT NULL DEFAULT 'menunggu_verifikasi',
  ADD CONSTRAINT fk_ppa_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id);

-- Pengunggah bukti adalah akun pemohon atau admin (tabel sudah dikosongkan di langkah 2).
ALTER TABLE pembayaran_pendaftaran_agen_upload
  MODIFY COLUMN diupload_oleh_tipe ENUM('akun','admin') NOT NULL;

-- 8. Pencairan: pemohon adalah akun agen.
ALTER TABLE pengajuan_pencairan DROP FOREIGN KEY fk_pencairan_agen;
ALTER TABLE pengajuan_pencairan
  RENAME COLUMN jamaah_agen_id TO akun_agen_id,
  ADD CONSTRAINT fk_pencairan_akun_agen FOREIGN KEY (akun_agen_id) REFERENCES customer_accounts (id);

-- 9. Pemakaian kredit cashback mengikuti penerimanya: akun. Fitur dinonaktifkan di tahap awal, skema dijaga konsisten.
ALTER TABLE pemakaian_cashback DROP FOREIGN KEY fk_cashback_jamaah;
ALTER TABLE pemakaian_cashback
  RENAME COLUMN jamaah_id TO akun_id,
  RENAME INDEX idx_cashback_jamaah TO idx_cashback_akun,
  ADD CONSTRAINT fk_cashback_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id);

-- 10. Agen pembuat booking dicatat terpisah dari penerima komisi (spek 8, butir 6): booking portal agen tetap
-- milik agen yang membuatnya walau komisi mengalir ke pemilik kunci jamaah. Dasar peringkat pembuat booking.
ALTER TABLE bookings
  ADD COLUMN dibuat_oleh_agen_akun_id BIGINT UNSIGNED NULL,
  ADD KEY idx_bookings_dibuat_oleh_agen (dibuat_oleh_agen_akun_id),
  ADD CONSTRAINT fk_bookings_dibuat_oleh_agen FOREIGN KEY (dibuat_oleh_agen_akun_id) REFERENCES customer_accounts (id);

-- 11. Pilihan "Saya sendiri" di portal agen (K13): pax itu menghasilkan komisi langsung untuk agen yang memilihnya,
-- walau jamaahnya terkunci ke agen lain. Penanda per pax karena akun tidak lagi terhubung ke satu jamaah.
ALTER TABLE booking_pax
  ADD COLUMN saya_sendiri_akun_id BIGINT UNSIGNED NULL,
  ADD CONSTRAINT fk_booking_pax_saya_sendiri FOREIGN KEY (saya_sendiri_akun_id) REFERENCES customer_accounts (id);
