-- 167_indeks_performa.sql
-- Indeks komposit untuk query yang sering jalan (audit performa loading 2026-11-28, A12,
-- docs/arsip/AUDIT-PERFORMA-LOADING-2026-11-28.md). Hanya menambah indeks: tidak mengubah data atau kolom.
-- Catatan: ADD KEY tidak idempotent, jalankan sekali. InnoDB menambah indeks secara online.

-- Daftar paket publik per brand (status published, urut tanggal berangkat), feed sitemap, komisi paket agen.
ALTER TABLE schedules
  ADD KEY idx_schedules_brand_status_berangkat (brand_id, status, berangkat_tanggal);

-- Pengingat pelunasan/berangkat dan jatuh tempo menyaring status; hitungan seat per jadwal
-- (SeatDipegangBooking, booking aktif) menyaring schedule_id + status + is_seat_blocked.
ALTER TABLE bookings
  ADD KEY idx_bookings_status_customer (status, customer_account_id),
  ADD KEY idx_bookings_schedule_status_seat (schedule_id, status, is_seat_blocked);

-- SUM pembayaran terkonfirmasi per booking (daftar pesanan akun, pengingat pelunasan): cukup dari indeks.
ALTER TABLE payments
  ADD KEY idx_payments_booking_status (booking_id, status, jumlah);

-- Perjalanan dan data jamaah per akun/agen menyaring jamaah_id + pax_status.
ALTER TABLE booking_pax
  ADD KEY idx_booking_pax_jamaah_status (jamaah_id, pax_status);

-- Saldo dan total komisi agen per jenis, tanpa yang dibatalkan.
ALTER TABLE transaksi_komisi
  ADD KEY idx_komisi_penerima_jenis (akun_penerima_id, jenis, dibatalkan_at);
