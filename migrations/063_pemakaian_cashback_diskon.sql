-- 063: Kaitkan pemakaian kredit cashback (screen B5, keputusan D4) ke baris
-- booking_discounts yang dibuatnya. Bila Admin Travel menghapus diskon itu
-- lewat fitur diskon booking, baris pemakaian ikut terhapus (ON DELETE CASCADE)
-- sehingga kredit cashback jamaah kembali utuh.
ALTER TABLE pemakaian_cashback
  ADD COLUMN booking_discount_id BIGINT UNSIGNED NULL AFTER booking_id,
  ADD UNIQUE KEY uq_cashback_discount (booking_discount_id),
  ADD CONSTRAINT fk_cashback_discount FOREIGN KEY (booking_discount_id)
    REFERENCES booking_discounts (id) ON DELETE CASCADE;
