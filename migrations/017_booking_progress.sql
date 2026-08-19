-- 017_booking_progress.sql
-- Menambahkan 8 checklist progress keberangkatan pada tabel bookings

ALTER TABLE bookings
  ADD COLUMN progress_paspor BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN progress_visa BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN progress_tiket BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN progress_hotel BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN progress_land_arrangement BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN progress_manasik BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN progress_perlengkapan BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN progress_vaksin_meningitis BOOLEAN NOT NULL DEFAULT FALSE;
