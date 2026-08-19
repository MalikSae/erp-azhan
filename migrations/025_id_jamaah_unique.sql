ALTER TABLE jamaah 
  MODIFY COLUMN id_jamaah VARCHAR(20) NOT NULL,
  ADD UNIQUE KEY unique_id_jamaah (id_jamaah);
