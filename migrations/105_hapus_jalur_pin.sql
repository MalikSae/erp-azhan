-- Akun pelanggan email + password (104) menggantikan akun portal nomor HP + PIN.
-- Jalur PIN dihapus total: kolom PIN, token aktivasi dari admin, dan daftar token sesi portal yang dicabut.
-- Tidak ada migrasi data lama (platform masih tahap dev, tidak ada pelanggan nyata).
-- Terapkan SETELAH 104 dan setelah kode backend yang tidak lagi membaca kolom ini dideploy.

DROP TABLE IF EXISTS jamaah_activation_tokens;
DROP TABLE IF EXISTS portal_revoked_tokens;
ALTER TABLE jamaah DROP COLUMN portal_pin_hash;
