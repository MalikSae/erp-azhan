CREATE TABLE add_ons (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name          VARCHAR(150) UNIQUE NOT NULL,
  created_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE schedule_add_ons (
  schedule_id   BIGINT UNSIGNED NOT NULL,
  add_on_id     BIGINT UNSIGNED NOT NULL,
  PRIMARY KEY (schedule_id, add_on_id),
  FOREIGN KEY (schedule_id) REFERENCES schedules(id) ON DELETE CASCADE,
  FOREIGN KEY (add_on_id) REFERENCES add_ons(id)
) ENGINE=InnoDB;
