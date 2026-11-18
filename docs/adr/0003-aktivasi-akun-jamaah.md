# 0003. Aktivasi akun jamaah lewat kode booking dan NIK, tautan satu jamaah satu akun

Tanggal: 18 November 2026
Status: diterima. Tahap ERP pertama (E2) dibangun; komisi (E3) dan pandangan jamaah (E4) menyusul. Flag `AKTIVASI_JAMAAH_AKTIF` mati sampai keduanya rilis.

## Konteks

Jamaah sering didaftarkan orang lain: agen membuatkan booking, atau anggota keluarga memesan untuk jamaah. Jamaah itu tidak punya akun, jadi tidak bisa melihat perjalanannya sendiri. Sebaliknya, agen yang sudah menjadi pemilik kunci jamaah (`jamaah.direkrut_oleh_akun_id`) perlu tahu bahwa jamaahnya aktif, dan upline akun baru harus mengikuti kunci itu, bukan tebakan dari cookie referral.

Kolom `customer_accounts.jamaah_id` sudah ada sejak migrasi 104 tetapi tidak lagi ditulis dan tidak unik.

## Keputusan

1. **Pembuktian: kode booking + NIK.** Keduanya harus cocok pada pax aktif non-bayi yang sama di booking berstatus baru, dp, atau lunas, pada brand yang sama, dan jamaah itu belum tertaut ke akun mana pun. Booking draft dan batal ditolak.
2. **Satu fungsi inti, dua pintu.** `customer.Aktivasi.Jalankan` stateless dengan penanda `simpan` (false = cek, true = simpan). Pintu publik (`POST /api/public/auth/aktivasi`) membuat akun baru; pintu akun masuk (`POST /api/account/jamaah/tautkan`) menautkan akun yang sudah ada. Tanpa Google, captcha, atau OTP.
3. **Tautan satu jamaah satu akun** lewat `customer_accounts.jamaah_id` dengan `UNIQUE uq_customer_jamaah` (migrasi 153). Tanpa kolom baru dan tanpa backfill karena belum rilis production. Hapus akun mengosongkan `jamaah_id`. Memindahkan tautan hanya lewat admin: aksi **Lepas Tautan Jamaah** (izin `akun.edit`, alasan wajib, tercatat, semua sesi akun dicabut).
4. **Upline akun = pemilik kunci jamaah.** `agen.IkatAkunDariKunciTx` mengisi `direkrut_oleh_akun_id` dari kunci bila jamaah `terikat_agen`. Pemilik kunci aktif maupun nonaktif tetap mengikat. Kode referral diabaikan bila jamaah terikat; untuk `belum_ditentukan` dan `tanpa_agen` kode berlaku seperti daftar biasa dan jamaah tidak diikat saat aktivasi. Tidak ada klaim upline belakangan. Akun yang sudah punya upline berbeda dari pemilik kunci ditolak dengan galat generik; admin menyelesaikannya lewat Ganti Upline atau Lepas Tautan.
5. **Satu galat generik.** Kode salah, NIK salah, bukan peserta, sudah tertaut, booking batal, upline bentrok: semuanya `400 AKTIVASI_GAGAL` dengan teks sama. Sebab sebenarnya hanya masuk `jamaah_aktivasi_log.alasan` untuk admin. Jalur gagal diberi jeda acak 150-400 ms. Respons tidak membawa `kolom_kode_agen` dan tidak pernah memuat NIK, kode, atau data orang lain.
6. **Batas percobaan atomik di database.** Tabel `jamaah_aktivasi_hitung` (migrasi 154) dinaikkan dengan `INSERT .. ON DUPLICATE KEY UPDATE` sebelum evaluasi dan dikembalikan saat sukses. Batas gagal 5 per 15 menit per IP (IPv6 dipotong ke blok /64), perangkat, akun, dan per pasangan kode+NIK (pengaman utama; kegagalan satu pasangan tidak mengunci kode atau NIK orang lain). Per kode booking dan per NIK 100 per hari: tinggi, hanya menahan tebakan terdistribusi; kode booking yang valid formatnya selalu kena penghitung, ada atau tidak, dan semua 429 memakai Retry-After tetap 15 menit supaya keberadaan kode tidak terbaca. Per brand 300 per jam hanya alarm log, tidak memperketat batas. Admin membuka kunci harian lewat POST /api/admin/akun/aktivasi/reset-hitung (akun.edit; kode_booking atau nik; dicatat reset_admin). Kode, NIK, IP, dan perangkat disimpan sebagai HMAC dengan `AKTIVASI_PEPPER` (minimal 32 byte, server gagal start bila flag menyala tanpa pepper).
7. **Pembersihan.** Penghitung dibuang setelah kedaluwarsa, log kegagalan setelah 90 hari, jejak aktivasi, tautan, dan lepas tautan setelah 2 tahun. Penulisan log kegagalan dibatasi 100 baris per kode per 24 jam.
8. **Akses terbatas di tahap pertama.** Akun tertaut hanya melihat: label paket dan tanggal berangkat (`jamaah_tertaut` dan `perjalanan_tertaut` di `/me`), data jamaah mode lihat, dokumen mode lihat. Tidak ada kode booking mentah, harga, pax lain, ubah data, atau unduh dokumen. `sqlAksesPenuh` dan `dibuat_oleh_akun_id` tidak diubah.
9. **Penggunaan tautan oleh komisi (E3).** `akunJamaah` hanya diperluas untuk jamaah tertaut berstatus `terikat_agen` dengan direkrut terisi dan upline akun sama dengan direkrut itu. `tanpa_agen` dan `belum_ditentukan` tetap lewat persetujuan admin. `nik_agen` akun tertaut dipaksa sama dengan NIK jamaah tertaut; bila sudah dipegang akun lain, admin memakai **Kosongkan NIK Agen**. Lepas Tautan ditolak untuk akun agen aktif dan nonaktif (admin menonaktifkan agen dulu).

## Alasan OTP ditolak

Nomor WhatsApp jamaah di booking diketik oleh pemesan, bukan jamaah. OTP ke nomor itu hanya membuktikan penerima nomor yang diketik pemesan, jadi bukan faktor kedua yang independen. Menambah OTP menambah biaya kirim dan titik gagal tanpa menutup celah yang sebenarnya. Faktor kedua yang berarti (misalnya konfirmasi lewat kontak yang tercatat di profil jamaah sendiri) diputuskan terpisah.

Yang dilakukan sebagai gantinya: akses tahap pertama dibuat terbatas supaya tautan palsu tidak membuka dokumen KTP atau paspor dan tidak bisa mengubah data; kode booking tidak dikirim ke peran jamaah (separuh kredensial untuk menyerang pax lain); pembatas atomik; notifikasi kotak masuk ke pemilik kunci setiap kali jamaahnya aktivasi.

## Konsekuensi dan risiko yang diterima

- Orang yang memegang kode booking dan NIK bisa aktivasi lebih dulu daripada jamaah asli. Jamaah asli ditolak generik. Jalan keluar: Bantuan WhatsApp brand dan Lepas Tautan oleh admin. Residu ini diterima selama akses terbatas.
- Pihak ketiga bisa mengunci kode yang ada selama 1 sampai 24 jam lewat percobaan gagal. Diterima; kode acak tidak ikut terkunci.
- Akun bisa menjadi agen otomatis lewat tautan lalu booking pendukungnya batal. Agen tidak dicabut otomatis. Admin memantau lewat daftar **agen dari tautan** (`GET /api/admin/akun/agen-dari-tautan`, memuat penanda booking pendukung masih aktif atau tidak) dan menonaktifkan manual.
- Upline pemilik kunci yang nonaktif tetap mengikat jamaahnya. Komisi langsung ke pemilik kunci nonaktif hangus, tanpa backfill.
- NIK yang sama dengan `nik_agen` tidak lagi menjadi penentu "diri agen" untuk komisi (E3). Booking admin atau CRM untuk pax ber-NIK agen tidak dihitung sebagai diri agen; admin menyelesaikan lewat tautan atau Ganti Upline.
- Limiter memori per instance tetap dipakai sebagai lapis depan murah (60 permintaan per 15 menit per IP). Di jaringan seluler dengan IP bersama, batas per pasangan kode+NIK yang menjadi kontrol utama; header `X-Perangkat-Id` bisa dipalsukan dan hanya pelengkap.
