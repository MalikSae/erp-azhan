-- 046_unique_booking_jamaah.sql
-- Membersihkan duplikasi data historis pada booking_pax sebelum menambahkan unique constraint (booking_id, jamaah_id)
DELETE bp1 FROM booking_pax bp1
INNER JOIN booking_pax bp2 
WHERE bp1.id > bp2.id 
  AND bp1.booking_id = bp2.booking_id 
  AND bp1.jamaah_id = bp2.jamaah_id;

-- Menambahkan UNIQUE KEY pada pasangan (booking_id, jamaah_id)
ALTER TABLE booking_pax ADD UNIQUE KEY uq_booking_jamaah (booking_id, jamaah_id);
