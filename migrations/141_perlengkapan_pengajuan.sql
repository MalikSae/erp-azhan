-- 141: pengajuan dan konfirmasi terima perlengkapan per jamaah (keputusan 14 November 2026).
--
-- Alur per jamaah: belum diajukan -> diajukan (akun, dari app) -> diserahkan (petugas menekan Serahkan) -> diterima
-- (akun pemilik booking mengonfirmasi). Petugas boleh menyerahkan tanpa pengajuan. Penyerahan hanya boleh dibatalkan
-- selama belum dikonfirmasi. perlengkapan_status tetap menjadi penanda "sudah_diberikan" seperti sebelumnya.
-- Nama petugas disalin saat menyerahkan agar log tetap terbaca bila akun admin dihapus.
ALTER TABLE booking_pax
  ADD COLUMN perlengkapan_metode         VARCHAR(20)     NULL AFTER perlengkapan_tanggal,
  ADD COLUMN perlengkapan_diajukan_at    DATETIME        NULL AFTER perlengkapan_metode,
  ADD COLUMN perlengkapan_diajukan_oleh  BIGINT UNSIGNED NULL AFTER perlengkapan_diajukan_at,
  ADD COLUMN perlengkapan_diserahkan_at  DATETIME        NULL AFTER perlengkapan_diajukan_oleh,
  ADD COLUMN perlengkapan_diserahkan_oleh BIGINT UNSIGNED NULL AFTER perlengkapan_diserahkan_at,
  ADD COLUMN perlengkapan_diserahkan_nama VARCHAR(150)   NULL AFTER perlengkapan_diserahkan_oleh,
  ADD COLUMN perlengkapan_diterima_at    DATETIME        NULL AFTER perlengkapan_diserahkan_nama,
  ADD COLUMN perlengkapan_diterima_oleh  BIGINT UNSIGNED NULL AFTER perlengkapan_diterima_at,
  ADD KEY idx_booking_pax_perlengkapan_diajukan (perlengkapan_diajukan_at),
  ADD CONSTRAINT fk_booking_pax_perl_diajukan FOREIGN KEY (perlengkapan_diajukan_oleh) REFERENCES customer_accounts (id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_booking_pax_perl_diserahkan FOREIGN KEY (perlengkapan_diserahkan_oleh) REFERENCES admin_users (id) ON DELETE SET NULL,
  ADD CONSTRAINT fk_booking_pax_perl_diterima FOREIGN KEY (perlengkapan_diterima_oleh) REFERENCES customer_accounts (id) ON DELETE SET NULL;
