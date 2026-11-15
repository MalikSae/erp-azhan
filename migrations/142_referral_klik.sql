-- 142: klik link referral agen (keputusan 15 November 2026, bagian agen di app Hana).
--
-- Situs publik memanggil POST /api/public/referral/klik saat halaman dibuka dengan ?ref=KODE. Satu pengunjung dihitung
-- satu kali per agen per hari (WIB); pengunjung disimpan sebagai hash, bukan data mentah. App menampilkan jumlah klik
-- bulan berjalan.
CREATE TABLE IF NOT EXISTS referral_klik (
  akun_id     BIGINT UNSIGNED NOT NULL,
  tanggal     DATE NOT NULL,
  pengunjung  CHAR(64) NOT NULL,
  dibuat_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (akun_id, tanggal, pengunjung),
  CONSTRAINT fk_referral_klik_akun FOREIGN KEY (akun_id) REFERENCES customer_accounts (id) ON DELETE CASCADE
);
