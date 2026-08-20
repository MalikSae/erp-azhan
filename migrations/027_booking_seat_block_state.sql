ALTER TABLE bookings
  ADD COLUMN is_seat_blocked BOOLEAN NOT NULL DEFAULT FALSE AFTER status;

UPDATE bookings
SET is_seat_blocked = TRUE
WHERE status IN ('dp', 'lunas', 'dokumen_lengkap', 'siap_berangkat');
