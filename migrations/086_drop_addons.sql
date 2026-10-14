-- 086_drop_addons.sql
-- Fitur Add-On dihapus: master add-on, pilihan add-on per paket, dan biaya
-- tambahan per booking tidak lagi dipakai. Data lama hilang permanen.
--
-- PERHATIAN: bookings.total_harga yang sudah tersimpan masih memuat nominal
-- booking_addons lama. Nilainya tidak diubah di sini; saat sebuah booking
-- dihitung ulang (ubah pax/diskon), total = pax - diskon sehingga turun sebesar
-- add-on yang dulu ditagihkan.
DROP TABLE IF EXISTS schedule_add_ons;
DROP TABLE IF EXISTS booking_addons;
DROP TABLE IF EXISTS add_ons;

UPDATE permissions
SET description = 'Lihat master data (hotel/maskapai/airport/kategori/itinerary)'
WHERE code = 'masterdata.view';
