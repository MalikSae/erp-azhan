# Perbaikan self-booking — 28 September 2026

## Status penerapan

Perubahan kode ERP dan microsite tersedia pada working tree `dev-malik`. Belum commit, push, atau deploy. Setelah pengguna memberikan izin eksplisit, migrasi `065_booking_checkout.sql` berhasil diterapkan pada 28 September 2026 pukul 14:06 WIB ke database lokal `127.0.0.1:3306/erp_azhan_dev` (MySQL 8.4.3).

Backup penuh sebelum migrasi: `scratch/backups/erp_azhan_dev_pre065_20260928_140533.sql` (167095 byte), SHA256 `c355395fe981309ae8934b9818070bcfcbc57a1fb1d629fcacdb5273921c7f5e`. Backup berada pada direktori scratch yang diabaikan Git. Pembuatan backup berhasil; uji restore belum dilakukan.

```text
Migrasi: kedua statement berhasil, exit 0.
Bookings: 62; checkout table exists: true
Snapshots: 62; legacy: 62; missing: 0; invalid: 0; duplicate tokens: 0
Verification PASS
```

Tes integrasi menggunakan transaksi yang di-rollback dan tabel `booking_checkout` sementara pada koneksi pengujian. Hasil tes tidak membuktikan bahwa migrasi/backfill permanen telah diterapkan.

## Pemetaan temuan

| Temuan | Perubahan kode |
| --- | --- |
| SB01 | Pemeriksaan duplikasi mencakup seluruh anggota aktif pada jadwal yang sama, di dalam transaksi booking. |
| SB02 | Tarif infant memakai tanggal lahir identitas tersimpan. Tanggal lahir wajib, bukan di masa depan, dan usia di bawah dua tahun saat keberangkatan. |
| SB03 | Snapshot DP, deadline, dan aturan full payment pada checkout; invoice dan pembayaran membaca snapshot. Nol berbeda dari NULL. |
| SB04 | Bukti pending tepat waktu memberi satu tambahan hold 24 jam setelah batas awal. Unggah ulang tidak memperpanjang tanpa batas; bukti terlambat tidak menghidupkan hold. |
| SB05 | Invoice memakai token acak 256-bit dan scope brand, no-store, noindex, serta no-referrer. Kode booking pendek tidak dapat dipakai sebagai tautan publik. Portal menyediakan tautan bagi PIC atau agen aktif yang berhak. |
| SB06 | Konfirmasi mengunci booking lalu pembayaran, menghitung ulang pembayaran terkonfirmasi lainnya, dan menolak jumlah gabungan di atas tagihan. Verifikasi bersamaan tidak menimpa keputusan yang sudah selesai. |
| SB07 | Status dihitung dari pembayaran terkonfirmasi: nol kembali ke baru. Perubahan total administratif memakai sinkronisasi yang sama. Hold sementara diperbarui ketika status turun ke baru. |
| SB08 | Captcha tanpa secret ditolak. Bypass hanya berlaku dengan APP_ENV=development dan ALLOW_DEV_CAPTCHA=true secara eksplisit. |
| SB09 | CF-Connecting-IP tidak lagi dipercaya/diforward otomatis. IP mengikuti rantai X-Forwarded-For dari proxy tepercaya. |
| SB10 | Invoice membedakan held, expired, cancelled, dan confirmed; rekening/ajakan transfer disembunyikan untuk reservasi tidak aktif. Badge portal DP mengikuti status terverifikasi. |
| SB11 | Izin dan akreditasi invoice berasal dari brand; nilai rekaan dihapus. |
| SB12 | Halaman ketentuan dapat dibaca. Server mewajibkan persetujuan versi terbaru dan menyimpan versi/waktu penerimaan. |
| SB13 | Request key dan fingerprint mengembalikan booking/invoice yang sama pada retry tanpa reservasi kedua. Perubahan payload ditolak. Kegagalan menyimpan token lokal tidak mengubah keberhasilan booking menjadi kegagalan. Retry akses portal tetap memeriksa PIN terkini atau identitas login. |
| SB14 | Label kontrol, native select, fokus langkah/modal, Escape, dan validasi PIN sebelum konfirmasi. Verifikasi browser langkah berikutnya masih menunggu backend terbaru. |
| SB15 | PIN server harus enam digit angka. |
| SB16 | Respons pemeriksaan nomor yang basi diabaikan; hasil nomor sama digunakan ulang; kesalahan/rate limit ditampilkan. Batas telepon frontend/backend 10–15 digit. |
| SB17 | Invoice menjelaskan aktivasi PIC tanpa PIN/booking agen, kelengkapan tanggal lahir, dan bantuan admin. Jalur ini tetap membutuhkan admin; bukan aktivasi otomatis berdasarkan nama saja. |
| SB18 | Checkout dalam 45 hari mewajibkan pelunasan penuh dalam hold awal 24 jam. Checkout lebih awal tetap memakai H-45. |
| SB19 | Server membandingkan total/DP yang disetujui dengan harga aktual. Konflik meminta pemeriksaan ulang dan memuat ulang harga, termasuk jalur agen. |
| SB20 | PIC invoice diambil langsung dari booking; pax batal ditandai dan tidak menambah minimum DP. Selisih biaya administratif tampil sebagai penyesuaian agregat. |
| SB21 | Draf opsional 30 menit dalam tab, dipisahkan brand/jadwal/pemesan, tanpa PIN atau persetujuan. Hasil lama tidak dipulihkan sebagai keberhasilan baru; draf dibersihkan saat logout. |

## Kebijakan bisnis

- Reservasi awal 24 jam. DP nol berarti tidak ada minimum nominal, bukan perjalanan gratis. Konfirmasi kursi tetap memerlukan pembayaran terverifikasi positif.
- Bukti sebelum batas awal memberikan satu masa review tambahan 24 jam; keputusan admin tetap diperlukan.
- Keberangkatan dalam 45 hari memerlukan pelunasan penuh dalam 24 jam. Pendaftaran online ditutup ketika kurang dari 14 hari menuju keberangkatan.
- Pembatalan/refund ditangani admin berdasarkan ketentuan tertulis travel; sistem tidak menetapkan biaya refund rekaan.
- Snapshot booking lama memakai konfigurasi saat migrasi sebagai baseline legacy. Ketentuan historis yang sebelumnya tidak tersimpan tidak dapat direkonstruksi.

## Bukti verifikasi

```text
go test -p 1 ./...
Seluruh package lulus; exit 0.

ERP_TEST_DB=1 go test -p 1 ./internal/selfbooking ./internal/payment ./internal/booking ./internal/schedule
ok erp-azhan/api/internal/selfbooking 6.776s
ok erp-azhan/api/internal/payment 2.921s
ok erp-azhan/api/internal/booking 4.025s
ok erp-azhan/api/internal/schedule 7.915s
exit 0

node --test tests/*.test.mjs
tests 10; pass 10; fail 0
```

Build production microsite final lulus:

```text
npm run build
Compiled successfully in 10.3s
Generating static pages (32/32)
exit 0
```

Next.js masih memberi peringatan konvensi middleware yang deprecated; ini tidak menggagalkan build dan migrasi konvensinya berada di luar perubahan self-booking.

Browser membuka `http://hana.azhan.test/paket/45/book?room=quad`: aturan pelunasan 45 hari tampil. Viewport 375: scrollWidth=360. Viewport 1440: scrollWidth=1425. Langkah pertama tidak meluber horizontal. Form menampilkan `Konfigurasi paket belum tersedia. Muat ulang atau hubungi admin.` dan tombol lanjut dinonaktifkan karena API aktif belum memuat kontrak harga terbaru. Tidak ada booking/transfer nyata yang dikirim. Langkah data, submit, invoice setelah booking, dan pembayaran belum dinyatakan lulus browser end-to-end.

## Urutan penerapan setelah izin

1. Verifikasi target tetap database development `erp_azhan_dev`; siapkan backup.
2. Jalankan migrasi 065 dari ERP. Migrasi membuat tabel checkout dan mengisi snapshot/token untuk booking lama; tidak menghapus booking.
3. Konfigurasikan Turnstile. Jangan mengaktifkan bypass development pada production. Proxy harus menambahkan/membersihkan X-Forwarded-For dengan benar; TRUSTED_PROXIES hanya memuat jaringan proxy yang dikendalikan.
4. Jalankan backend terbaru, lalu microsite terbaru sebagai satu rilis kontrak API. Tautan invoice pendek lama tidak berlaku; tautan baru diperoleh melalui portal berizin.
5. Uji browser lengkap pada 375 lalu 1440: pengunjung baru, PIC login, PIC tanpa PIN, agen, infant, harga berubah, retry, pembayaran pending, expiry, dan invoice lintas brand. Gunakan data uji development yang disetujui dan catat hasil aktual.

Migrasi dan pemeriksaan backfill selesai. Restart backend terbaru dan acceptance test setelah penerapan masih terbuka. Jangan menandai fitur sudah aktif sepenuhnya berdasarkan migrasi/build/test fixture saja.
