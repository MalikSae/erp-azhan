ALTER TABLE bookings MODIFY COLUMN status ENUM('baru', 'dp', 'lunas', 'batal') NOT NULL DEFAULT 'baru';
