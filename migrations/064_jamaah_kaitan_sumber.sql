-- 064: Catat dari jalur mana kaitan agen seorang jamaah ditetapkan.
-- Dibutuhkan screen C4 (Ganti Kaitan Agen): hanya kaitan hasil Jalur 3 (input
-- Admin) yang boleh diganti Admin Master (agen-azhan.md 3.5, §5 poin 17);
-- kaitan Jalur 1 (booking agen) dan Jalur 2 (link referral) tetap immutable.
--   jalur1   : jamaah dibuat oleh agen lewat booking agen (Portal Syiar)
--   jalur2   : kode referral (link agen) saat daftar/booking mandiri atau CRM
--   jalur3   : pilihan eksplisit Admin saat membuat jamaah
--   ikut_pic : auto-bind mengikuti agen PIC dalam satu booking (3.4)
-- Data lama (sebelum kolom ini ada) bernilai NULL: asal tidak diketahui,
-- sehingga tidak bisa diganti lewat C4.
ALTER TABLE jamaah
  ADD COLUMN kaitan_sumber ENUM('jalur1','jalur2','jalur3','ikut_pic') NULL AFTER kaitan_status;
