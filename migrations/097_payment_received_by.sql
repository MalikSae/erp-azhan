-- 097_payment_received_by.sql
-- Nama staff yang menerima pembayaran tunai (wajib untuk metode cash yang dicatat admin).
-- Untuk pembayaran transfer kolom ini kosong; rekening tujuan yang menjadi penandanya.
-- Catatan: ALTER TABLE tidak idempotent — jalankan file ini sekali saja.
ALTER TABLE payments
  ADD COLUMN received_by VARCHAR(150) NULL AFTER sender_bank;
