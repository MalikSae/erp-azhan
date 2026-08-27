CREATE TABLE schedule_transit_hotels (
  id           BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  schedule_id  BIGINT UNSIGNED NOT NULL,
  hotel_id     BIGINT UNSIGNED NOT NULL,
  urutan       INT NOT NULL DEFAULT 0,
  FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE,
  FOREIGN KEY (hotel_id) REFERENCES hotels(id)
) ENGINE=InnoDB;
