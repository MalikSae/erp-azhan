-- 147: Masuk dengan Google di app (keputusan 16 November 2026).
--
-- google_sub: ID akun Google (klaim "sub" pada ID token) yang tertaut ke akun pelanggan. Unik per brand,
-- karena akun pelanggan berlaku per brand. Terisi saat akun dibuat lewat Google, atau otomatis saat akun
-- lama dengan email yang sama masuk lewat Google untuk pertama kali.
-- password_hash boleh NULL: akun yang dibuat lewat Google belum punya sandi sampai pemiliknya membuat
-- sandi di Pengaturan atau lewat lupa password.

ALTER TABLE customer_accounts
  MODIFY COLUMN password_hash VARCHAR(100) NULL,
  ADD COLUMN google_sub VARCHAR(64) NULL,
  ADD UNIQUE KEY uq_customer_brand_google (brand_id, google_sub);
