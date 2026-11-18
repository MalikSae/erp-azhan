-- 155: Banner Promosi bisa tampil di website, app, atau keduanya (landing v1 web memakai banner yang sama).
-- app | web | keduanya. Banner lama tetap tampil di app dan web (keduanya).
ALTER TABLE banner_promosi
  ADD COLUMN tampil_di VARCHAR(10) NOT NULL DEFAULT 'keduanya' AFTER tujuan_nilai;
