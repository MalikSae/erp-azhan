-- 131: admin pencatat pembayaran (code review auth dan izin, 11 November 2026, temuan 7).
--
-- Pembayaran yang dicatat admin langsung terkonfirmasi, dan verified_by terisi otomatis dengan admin yang
-- sama. Tanpa kolom pencatat tersendiri, "dicatat A lalu diverifikasi B" tidak bisa dibedakan dari "A
-- melakukan keduanya". Selama belum ada staf Finance, Admin Brand boleh mencatat dan memverifikasi
-- pembayaran; kolom ini menjadi jejak untuk pengawasan Holding.
-- Pembayaran dari portal dan CRM tidak punya admin pencatat (NULL).

ALTER TABLE payments
  ADD COLUMN dicatat_oleh BIGINT UNSIGNED NULL AFTER source,
  ADD CONSTRAINT fk_payments_dicatat_oleh FOREIGN KEY (dicatat_oleh) REFERENCES admin_users (id) ON DELETE SET NULL;

-- Data lama: pembayaran sumber admin dicatat oleh admin yang tercatat sebagai verifikatornya.
UPDATE payments SET dicatat_oleh = verified_by WHERE source = 'admin' AND verified_by IS NOT NULL;
