-- migrations/001_init.sql
-- Migrasi awal: schema umroh microsite ERP Azhan
-- Jalankan ke database erp_azhan_dev yang sudah dibuat secara manual.

CREATE TABLE airlines (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name            VARCHAR(150) UNIQUE NOT NULL,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE hotels (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  name            VARCHAR(255) UNIQUE NOT NULL,
  city            VARCHAR(100) NOT NULL,
  star_rating     TINYINT CHECK (star_rating BETWEEN 1 AND 5),
  distance_m      INT,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE itineraries (
  id              BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  title           VARCHAR(255) NOT NULL,
  created_at      TIMESTAMP DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE itinerary_days (
  id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  itinerary_id  BIGINT UNSIGNED NOT NULL,
  day_number    INT NOT NULL,
  title         VARCHAR(255),
  location      VARCHAR(255),
  activities    JSON NOT NULL,
  FOREIGN KEY (itinerary_id) REFERENCES itineraries(id) ON DELETE CASCADE
) ENGINE=InnoDB;

CREATE TABLE schedules (
  id                          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  jadwal_id                   VARCHAR(50) UNIQUE NOT NULL,
  jadwal_nama                 VARCHAR(255) NOT NULL,
  is_promo                    BOOLEAN DEFAULT FALSE,
  seat_total                  INT NOT NULL,
  seat_sisa                   INT NOT NULL,
  maskapai_id                 BIGINT UNSIGNED,
  berangkat_tanggal           DATE NOT NULL,
  berangkat_jam                TIME,
  berangkat_kode_penerbangan   VARCHAR(50),
  pulang_tanggal               DATE NOT NULL,
  pulang_jam                   VARCHAR(20),
  pulang_kode_penerbangan      VARCHAR(50),
  hotel_mekkah_id               BIGINT UNSIGNED,
  hotel_madinah_id              BIGINT UNSIGNED,
  harga_quad                    DECIMAL(12,0) NOT NULL,
  harga_triple                  DECIMAL(12,0) NOT NULL,
  harga_double                  DECIMAL(12,0) NOT NULL,
  itinerary_id                   BIGINT UNSIGNED,
  include_items                  JSON,
  exclude_items                  JSON,
  brosur_url                    VARCHAR(500),
  brosur_thumb_url              VARCHAR(500),
  created_at                    TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at                    TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  FOREIGN KEY (maskapai_id) REFERENCES airlines(id),
  FOREIGN KEY (hotel_mekkah_id) REFERENCES hotels(id),
  FOREIGN KEY (hotel_madinah_id) REFERENCES hotels(id),
  FOREIGN KEY (itinerary_id) REFERENCES itineraries(id)
) ENGINE=InnoDB;

CREATE INDEX idx_schedules_berangkat ON schedules(berangkat_tanggal);
CREATE INDEX idx_schedules_promo ON schedules(is_promo);
CREATE INDEX idx_hotels_name ON hotels(name);
CREATE INDEX idx_airlines_name ON airlines(name);
