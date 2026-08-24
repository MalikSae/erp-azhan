-- Lengkapi logo untuk maskapai hasil seeder yang sudah ada.
-- Logo yang telah diatur manual oleh admin tidak ditimpa.

UPDATE airlines
SET logo_url = '/uploads/airline-logos/248e6902-020f-4062-914b-e0dd44d8e5f8.webp'
WHERE UPPER(name) = UPPER('Garuda Indonesia')
  AND (logo_url IS NULL OR logo_url = '');

UPDATE airlines
SET logo_url = '/uploads/airline-logos/7426450f-d7a1-4486-9623-2e9d3b8b192a.webp'
WHERE UPPER(name) = UPPER('Saudia')
  AND (logo_url IS NULL OR logo_url = '');

UPDATE airlines
SET logo_url = '/uploads/airline-logos/181837fa-a756-456f-891c-6f3af052723e.webp'
WHERE UPPER(name) = UPPER('Emirates')
  AND (logo_url IS NULL OR logo_url = '');

UPDATE airlines
SET logo_url = '/uploads/airline-logos/49bdd0d0-f27a-4102-8435-d65ffbaad645.webp'
WHERE UPPER(name) = UPPER('Oman Air')
  AND (logo_url IS NULL OR logo_url = '');

UPDATE airlines
SET logo_url = '/uploads/airline-logos/f28d8537-8eab-43e4-a265-ca08a3609a7b.webp'
WHERE UPPER(name) = UPPER('Lion Air')
  AND (logo_url IS NULL OR logo_url = '');
