-- 134: catat apakah pemesan (pemilik akun) ikut berangkat (11 November 2026).
--
-- Detail booking menampilkan "tidak ikut berangkat" di samping nama pemesan. Dulu keterangan itu ditebak
-- dari customer_accounts.jamaah_id, padahal checkout web tidak pernah mengisi kolom tersebut, sehingga
-- pemesan yang ikut berangkat pun tampil "tidak ikut berangkat". Pilihan pemesan saat checkout sekarang
-- disimpan di booking.
--
-- NULL = tidak tercatat (booking lama, booking buatan admin, booking dari CRM).

ALTER TABLE bookings
  ADD COLUMN pemesan_ikut TINYINT(1) NULL AFTER customer_account_id;

-- Data lama: hanya yang bisa dipastikan ikut yang diisi. Sisanya tetap NULL, karena pilihan saat checkout
-- tidak tersimpan.
UPDATE bookings b
JOIN customer_accounts a ON a.id = b.customer_account_id
SET b.pemesan_ikut = 1
WHERE b.pemesan_ikut IS NULL
  AND EXISTS (
    SELECT 1 FROM booking_pax bp
    WHERE bp.booking_id = b.id
      AND (bp.jamaah_id = a.jamaah_id OR bp.saya_sendiri_akun_id = a.id)
  );
