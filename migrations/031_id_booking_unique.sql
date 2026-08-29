ALTER TABLE bookings MODIFY COLUMN id_booking VARCHAR(6) NOT NULL, ADD UNIQUE KEY unique_id_booking (id_booking);
