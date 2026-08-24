-- Lengkapi aset visual hotel hasil seeder yang sudah ada.
-- photo_url dipakai sebagai kolom visual hotel oleh API saat ini.
-- Aset yang telah diatur manual oleh admin tidak ditimpa.

UPDATE hotels SET photo_url = '/uploads/hotel-logos/fairmont.png'
WHERE UPPER(name) = UPPER('Fairmont Makkah Clock Royal Tower')
  AND (photo_url IS NULL OR photo_url = '');

UPDATE hotels SET photo_url = '/uploads/hotel-logos/doubletree.png'
WHERE UPPER(name) = UPPER('DoubleTree by Hilton Makkah Jabal Omar')
  AND (photo_url IS NULL OR photo_url = '');

UPDATE hotels SET photo_url = '/uploads/hotel-logos/anjum.png'
WHERE UPPER(name) = UPPER('Anjum Hotel Makkah')
  AND (photo_url IS NULL OR photo_url = '');

UPDATE hotels SET photo_url = '/uploads/hotel-logos/voco.png'
WHERE UPPER(name) = UPPER('voco Makkah by IHG')
  AND (photo_url IS NULL OR photo_url = '');

UPDATE hotels SET photo_url = '/uploads/hotel-logos/movenpick.png'
WHERE UPPER(name) = UPPER('Anwar Al Madinah Mövenpick')
  AND (photo_url IS NULL OR photo_url = '');

UPDATE hotels SET photo_url = '/uploads/hotel-logos/dallah-taibah.png'
WHERE UPPER(name) = UPPER('Dallah Taibah Hotel')
  AND (photo_url IS NULL OR photo_url = '');
