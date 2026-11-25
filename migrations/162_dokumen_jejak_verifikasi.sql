-- 162: Jejak verifikasi Dokumen Digital (siapa dan kapan menyetujui/menolak).
--
-- Diisi saat admin mengubah status dokumen, dikosongkan saat berkas diganti. Versi lama di
-- dokumen_jamaah_versions menyimpan peninjaunya sendiri. Nomor 160/161 dipakai branch lain.
ALTER TABLE dokumen_jamaah
  ADD COLUMN verified_by BIGINT UNSIGNED NULL AFTER rejection_reason,
  ADD COLUMN verified_at DATETIME NULL AFTER verified_by,
  ADD CONSTRAINT fk_dokumen_jamaah_verified_by FOREIGN KEY (verified_by) REFERENCES admin_users (id) ON DELETE SET NULL;

ALTER TABLE dokumen_jamaah_versions
  ADD COLUMN verified_by BIGINT UNSIGNED NULL AFTER rejection_reason,
  ADD COLUMN verified_at DATETIME NULL AFTER verified_by,
  ADD CONSTRAINT fk_dokumen_versions_verified_by FOREIGN KEY (verified_by) REFERENCES admin_users (id) ON DELETE SET NULL;
