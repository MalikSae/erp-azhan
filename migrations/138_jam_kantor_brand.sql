-- 138: jam kantor brand (keputusan 14 November 2026).
--
-- App menampilkan jam kerja kantor di layar pembayaran tunai. Teks bebas, mis. "Senin-Sabtu, 08.00-17.00 WIB".
ALTER TABLE brands ADD COLUMN jam_kantor VARCHAR(100) NULL AFTER gmaps_url;
