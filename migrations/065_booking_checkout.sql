-- Checkout snapshots and retry identity. Existing bookings retain current terms
-- as a legacy baseline; historical terms cannot be reconstructed automatically.
CREATE TABLE IF NOT EXISTS booking_checkout (
 booking_id BIGINT UNSIGNED NOT NULL PRIMARY KEY,
 dp_per_pax DOUBLE NOT NULL,
 full_payment BOOLEAN NOT NULL DEFAULT FALSE,
 due_at DATETIME NOT NULL,
 original_expires_at DATETIME NULL,
 review_extended BOOLEAN NOT NULL DEFAULT FALSE,
 invoice_token CHAR(64) NOT NULL UNIQUE,
 terms_version VARCHAR(40) NOT NULL,
 accepted_at DATETIME NULL,
 request_key_hash CHAR(64) NULL UNIQUE,
 request_hash CHAR(64) NULL,
 response_json JSON NULL,
 FOREIGN KEY (booking_id) REFERENCES bookings(id) ON DELETE CASCADE
);
INSERT IGNORE INTO booking_checkout
 (booking_id,dp_per_pax,full_payment,due_at,original_expires_at,invoice_token,terms_version)
SELECT b.id,COALESCE(s.minimal_dp,br.minimal_dp,0),FALSE,
 GREATEST(DATE_SUB(s.berangkat_tanggal,INTERVAL 45 DAY),DATE_ADD(b.created_at,INTERVAL 1 DAY)),
 b.seat_hold_expires_at,LOWER(HEX(RANDOM_BYTES(32))),'legacy-2026-09-28'
FROM bookings b JOIN schedules s ON s.id=b.schedule_id JOIN brands br ON br.id=s.brand_id;
