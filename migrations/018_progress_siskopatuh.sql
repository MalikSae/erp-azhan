ALTER TABLE bookings
  DROP COLUMN progress_perlengkapan,
  ADD COLUMN progress_siskopatuh BOOLEAN NOT NULL DEFAULT FALSE;
