ALTER TABLE airlines ADD COLUMN code VARCHAR(3) NULL AFTER name;

UPDATE airlines SET code = 'EK' WHERE name = 'EMIRATES';
UPDATE airlines SET code = 'EY' WHERE name = 'ETIHAD AIRWAYS';
UPDATE airlines SET code = 'GA' WHERE name = 'GARUDA INDONESIA';
UPDATE airlines SET code = 'JT' WHERE name = 'LION AIR';
UPDATE airlines SET code = 'WY' WHERE name = 'OMAN AIR';
UPDATE airlines SET code = 'SV' WHERE name = 'SAUDIA';
