-- 130: akun pembuat jamaah (code review endpoint publik, 11 November 2026, temuan C1).
--
-- Jamaah dipakai ulang lintas akun lewat NIK + nama (spek keagenan 4.3). Tanpa catatan siapa yang
-- pertama memasukkan jamaah, akun mana pun yang tahu NIK + nama bisa membaca dan mengisi data lengkap
-- jamaah itu lewat booking miliknya sendiri. Kolom ini menjadi dasar akses penuh akun atas data jamaah:
-- hanya akun pembuat yang boleh melihat dan mengisi data lengkap; akun lain mendapat akses terbatas.
-- Jamaah yang dibuat admin tidak punya akun pembuat (NULL).

ALTER TABLE jamaah
  ADD COLUMN dibuat_oleh_akun_id BIGINT UNSIGNED NULL AFTER direkrut_oleh_akun_id,
  ADD CONSTRAINT fk_jamaah_dibuat_akun FOREIGN KEY (dibuat_oleh_akun_id) REFERENCES customer_accounts (id) ON DELETE SET NULL;

-- Data lama: pembuat = akun pemilik booking paling awal yang memuat jamaah itu, hanya bila baris jamaah
-- lahir bersama booking tersebut (selisih waktu paling lama satu menit; booking dari web membuat jamaah
-- dan booking dalam satu transaksi). Jamaah yang sudah ada sebelum booking itu dibuat admin: tetap NULL.
UPDATE jamaah j
JOIN (
  SELECT bp.jamaah_id, MIN(bp.booking_id) AS booking_awal
  FROM booking_pax bp
  JOIN bookings b ON b.id = bp.booking_id
  WHERE b.status <> 'draft'
  GROUP BY bp.jamaah_id
) awal ON awal.jamaah_id = j.id
JOIN bookings b ON b.id = awal.booking_awal
SET j.dibuat_oleh_akun_id = b.customer_account_id
WHERE b.customer_account_id IS NOT NULL
  AND TIMESTAMPDIFF(SECOND, j.created_at, b.created_at) BETWEEN 0 AND 60;

-- Klaim pembuat gugur bila jamaah sudah dipakai booking pihak lain (admin atau akun lain) sementara akun
-- pembuat belum pernah punya pembayaran terkonfirmasi pada booking yang memuat jamaah itu.
UPDATE jamaah j
SET j.dibuat_oleh_akun_id = NULL
WHERE j.dibuat_oleh_akun_id IS NOT NULL
  AND EXISTS (
    SELECT 1 FROM booking_pax bp
    JOIN bookings b ON b.id = bp.booking_id
    WHERE bp.jamaah_id = j.id AND b.status <> 'draft'
      AND NOT (b.customer_account_id <=> j.dibuat_oleh_akun_id)
  )
  AND NOT EXISTS (
    SELECT 1 FROM payments p
    JOIN bookings b2 ON b2.id = p.booking_id
    JOIN booking_pax bp2 ON bp2.booking_id = b2.id
    WHERE bp2.jamaah_id = j.id AND b2.customer_account_id = j.dibuat_oleh_akun_id AND p.status = 'confirmed'
  );
