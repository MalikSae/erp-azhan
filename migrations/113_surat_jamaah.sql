-- 113_surat_jamaah.sql
-- Surat jamaah yang diunduh dari halaman booking (Surat Keterangan Umroh dan Surat Rekomendasi Paspor).
-- brand_surat: isian surat per brand yang bisa diubah di dashboard (penandatangan, nama perusahaan, kode surat, gambar kop, tanda tangan,
-- stempel) dan urutan nomor terakhir. Kota surat dan nomor izin PPIU/PIHK diambil dari tabel brands (city, ppiu_number, pihk_number).
-- surat_jamaah: satu baris per booking, jamaah, dan jenis surat. Nomor dibuat saat unduhan pertama dan tidak berubah; unduhan berikutnya
-- hanya menambah jumlah_unduh.

CREATE TABLE IF NOT EXISTS brand_surat (
    brand_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
    nama_perusahaan VARCHAR(150) NULL,
    kode_surat VARCHAR(10) NULL,
    penandatangan_nama VARCHAR(150) NULL,
    penandatangan_jabatan VARCHAR(100) NULL,
    kop_url VARCHAR(500) NULL,
    tanda_tangan_url VARCHAR(500) NULL,
    stempel_url VARCHAR(500) NULL,
    nomor_terakhir INT UNSIGNED NOT NULL DEFAULT 0,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_brand_surat_brand FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS surat_jamaah (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    booking_id BIGINT UNSIGNED NOT NULL,
    jamaah_id BIGINT UNSIGNED NOT NULL,
    jenis VARCHAR(30) NOT NULL,
    nomor_urut INT UNSIGNED NOT NULL,
    nomor VARCHAR(100) NOT NULL,
    terbit_at DATETIME NOT NULL,
    jumlah_unduh INT UNSIGNED NOT NULL DEFAULT 1,
    unduh_terakhir_at DATETIME NOT NULL,
    UNIQUE KEY uq_surat_jamaah (booking_id, jamaah_id, jenis),
    CONSTRAINT fk_surat_jamaah_booking FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE,
    CONSTRAINT fk_surat_jamaah_jamaah FOREIGN KEY (jamaah_id) REFERENCES jamaah(id) ON DELETE CASCADE
);

-- Isian awal brand Hana dari contoh surat (Hana Asia Wisata, kode HAW, Fikry Shahab); urutan dimulai dari nomor contoh terakhir.
INSERT IGNORE INTO brand_surat (brand_id, nama_perusahaan, kode_surat, penandatangan_nama, penandatangan_jabatan, nomor_terakhir)
SELECT id, 'PT. Hana Asia Wisata', 'HAW', 'Fikry Shahab', 'Direktur Utama', 100807 FROM brands WHERE kode_brand = 'HN';
