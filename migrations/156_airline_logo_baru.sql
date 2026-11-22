-- Ganti logo enam maskapai utama dengan logo baru (folder IT/maskapai, 2026-11-22).
-- Hanya mengganti logo kosong atau logo bawaan seeder/lama; logo yang diunggah admin di luar daftar itu tidak ditimpa.
-- Aman dijalankan ulang.

UPDATE airlines
SET logo_url = '/uploads/airline-logos/garuda-indonesia.webp'
WHERE UPPER(name) = 'GARUDA INDONESIA'
  AND (logo_url IS NULL OR logo_url = ''
       OR logo_url IN ('/uploads/airline-logos/248e6902-020f-4062-914b-e0dd44d8e5f8.webp',
                       '/uploads/airline-logos/3b04bc74-6d1b-4ce6-9dda-205d0b3f84db.webp'));

UPDATE airlines
SET logo_url = '/uploads/airline-logos/saudia.jpeg'
WHERE UPPER(name) = 'SAUDIA'
  AND (logo_url IS NULL OR logo_url = ''
       OR logo_url = '/uploads/airline-logos/7426450f-d7a1-4486-9623-2e9d3b8b192a.webp');

UPDATE airlines
SET logo_url = '/uploads/airline-logos/emirates.webp'
WHERE UPPER(name) = 'EMIRATES'
  AND (logo_url IS NULL OR logo_url = ''
       OR logo_url = '/uploads/airline-logos/181837fa-a756-456f-891c-6f3af052723e.webp');

UPDATE airlines
SET logo_url = '/uploads/airline-logos/etihad-airways.webp'
WHERE UPPER(name) = 'ETIHAD AIRWAYS'
  AND (logo_url IS NULL OR logo_url = ''
       OR logo_url = '/uploads/airline-logos/95392369-fe88-4207-87f8-b7bd9074b765.webp');

UPDATE airlines
SET logo_url = '/uploads/airline-logos/oman-air.webp'
WHERE UPPER(name) = 'OMAN AIR'
  AND (logo_url IS NULL OR logo_url = ''
       OR logo_url = '/uploads/airline-logos/49bdd0d0-f27a-4102-8435-d65ffbaad645.webp');

UPDATE airlines
SET logo_url = '/uploads/airline-logos/lion-air.webp'
WHERE UPPER(name) = 'LION AIR'
  AND (logo_url IS NULL OR logo_url = ''
       OR logo_url = '/uploads/airline-logos/f28d8537-8eab-43e4-a265-ca08a3609a7b.webp');
