-- 062: Catat kapan booking PERTAMA KALI mencapai 'lunas'.
-- Dipakai komisi Syiar untuk membedakan pax "baru" dan "repeat order" (keputusan D5):
-- status booking bisa mundur dari lunas ke dp, jadi status saat ini tidak cukup.
-- Diisi sekali oleh komisi.OnBookingLunas dan tidak pernah diubah lagi.
ALTER TABLE bookings ADD COLUMN pertama_lunas_at DATETIME NULL;

-- Booking yang sudah lunas sebelum kolom ini ada: pakai created_at sebagai perkiraan urutan.
UPDATE bookings SET pertama_lunas_at = created_at WHERE status = 'lunas' AND pertama_lunas_at IS NULL;
