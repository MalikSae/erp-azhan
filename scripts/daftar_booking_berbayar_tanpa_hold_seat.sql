-- Daftar booking yang sudah dibayar (dp atau lunas) tetapi seat-nya tidak di-hold.
--
-- Latar: sampai 11 November 2026 tombol "Cancel Block Seat" (sekarang "Lepas Seat") bisa dipakai pada
-- booking yang sudah dibayar. Seat booking itu kembali ke sisa seat jadwal dan bisa terjual ke booking
-- lain, sementara bookingnya tetap berstatus dp atau lunas. Mulai rilis itu pelepasan pada booking
-- berbayar ditolak, tetapi data lama tidak diperbaiki otomatis: bila sisa seat jadwal sudah habis,
-- siapa yang mendapat seat adalah keputusan manusia.
--
-- Cara pakai: jalankan (hanya membaca), lalu untuk tiap baris buka booking di dashboard dan tekan
-- "Hold Seat". Kolom sisa_seat_jadwal menunjukkan apakah seat masih tersedia; bila kurang dari
-- seat_dibutuhkan, tambah kuota seat jadwal atau putuskan bersama brand.

SELECT
  br.name                                   AS brand,
  b.id_booking                              AS kode_booking,
  b.status,
  s.jadwal_nama                             AS jadwal,
  DATE_FORMAT(s.berangkat_tanggal, '%Y-%m-%d') AS berangkat,
  (SELECT COUNT(*) FROM booking_pax bp
    WHERE bp.booking_id = b.id AND bp.counts_for_seat = TRUE AND bp.pax_status = 'aktif') AS seat_dibutuhkan,
  s.seat_sisa                               AS sisa_seat_jadwal,
  b.total_harga,
  (SELECT COALESCE(SUM(p.jumlah), 0) FROM payments p
    WHERE p.booking_id = b.id AND p.status = 'confirmed') AS terbayar
FROM bookings b
JOIN schedules s ON s.id = b.schedule_id
JOIN brands br ON br.id = s.brand_id
WHERE b.status IN ('dp', 'lunas')
  AND b.is_seat_blocked = FALSE
ORDER BY s.berangkat_tanggal, br.name, b.id_booking;
