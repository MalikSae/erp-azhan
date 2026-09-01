-- Menambahkan minimal_dp ke brands (global setting)
ALTER TABLE brands ADD COLUMN minimal_dp DOUBLE NOT NULL DEFAULT 0;

-- Menambahkan minimal_dp ke schedules (override per paket)
ALTER TABLE schedules ADD COLUMN minimal_dp DOUBLE NULL;

-- Menambahkan portal_pin_hash ke jamaah untuk keamanan Portal Jamaah
ALTER TABLE jamaah ADD COLUMN portal_pin_hash VARCHAR(100) NULL AFTER status;
