-- 143: hapus akun dari app Hana (keputusan 15 November 2026).
--
-- Hapus akun = anonimkan: nama, email, WhatsApp, foto, biodata, dan rekening dikosongkan, semua sesi dicabut, dan akun
-- tidak bisa masuk lagi. Baris akun tetap ada supaya booking, jamaah, dan pembayaran yang merujuknya utuh. Email dan
-- nomor lama bebas dipakai mendaftar ulang.
ALTER TABLE customer_accounts ADD COLUMN dihapus_at DATETIME NULL;
