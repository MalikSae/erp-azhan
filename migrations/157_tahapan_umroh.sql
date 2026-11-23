-- 157: Tahapan umroh yang dilihat jamaah (Perlengkapan, Paspor, SISKOPATUH, Visa, Tiket). Tanggal selesai
-- SISKOPATUH dan visa per jamaah, tiket per booking; diisi saat admin mencentang, dikosongkan saat dicabut.
-- Tanggal perlengkapan memakai perlengkapan_diserahkan_at, paspor memakai dokumen_jamaah.updated_at.
-- Belum rilis production: centang lama tidak diisi tanggalnya (tampil Selesai tanpa tanggal).
ALTER TABLE booking_pax
  ADD COLUMN progress_siskopatuh_at DATETIME NULL AFTER progress_siskopatuh,
  ADD COLUMN progress_visa_at DATETIME NULL AFTER progress_visa;

ALTER TABLE bookings
  ADD COLUMN progress_tiket_at DATETIME NULL AFTER progress_tiket;
