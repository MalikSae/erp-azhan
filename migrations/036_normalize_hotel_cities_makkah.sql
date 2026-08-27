-- Migrasi 036: Normalisasi ejaan kota hotel menjadi Makkah dan Madinah
UPDATE hotels SET city = 'Makkah' WHERE UPPER(TRIM(city)) IN ('MEKKAH', 'MAKKAH', 'MECCA');
UPDATE hotels SET city = 'Madinah' WHERE UPPER(TRIM(city)) IN ('MADINAH', 'MEDINA');
