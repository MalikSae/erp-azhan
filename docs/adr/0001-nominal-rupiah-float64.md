# 0001. Nominal rupiah disimpan sebagai float64, status ditulis langsung di SQL

Tanggal: 10 November 2026
Status: diterima

## Konteks

Code review area uang (`internal/payment`, `internal/komisi`, refund, cashback, pencairan) menandai dua pola sebagai code smell:

1. Nominal uang memakai `float64` di Go, bukan tipe uang tersendiri. Di MySQL, kolom buku besar (pembayaran, refund, harga pax, total booking, DP per pax, komisi, pencairan) bertipe `DECIMAL(12,0)`, jadi database sendiri tidak menyimpan pecahan; `DOUBLE` hanya dipakai kolom pengaturan `minimal_dp`.
2. Nilai status (`'confirmed'`, `'lunas'`, `'batal'`, dan seterusnya) tertulis langsung di banyak kueri SQL.

Keduanya sudah ada sejak awal dan dipakai di hampir seluruh backend.

## Keputusan

**Nominal tetap `float64`.** Rupiah tidak punya pecahan, dan `float64` menyimpan bilangan bulat tanpa galat sampai sekitar 9 kuadriliun, jauh di atas nominal mana pun di sistem ini. Mengganti tipe menyentuh hampir seluruh backend, tes, dan kolom database, dengan risiko lebih besar daripada manfaatnya.

Penjaganya:

- Setiap nominal yang masuk dari luar diperiksa bilangan bulat dengan `shared.RupiahBulat` sebelum disimpan.
- Hasil pembagian atau persentase atas nominal wajib dibulatkan secara eksplisit di tempat perhitungannya. Saat ini satu-satunya pembagian ada di cabang repeat order (50%), yang dibulatkan ke bawah dan sedang dimatikan.

**Status di dalam teks SQL tetap ditulis langsung.** Kueri tetap bisa dibaca dan disalin ke klien MySQL saat menyelidiki masalah. Salah ketik di SQL tertangkap tes integrasi.

Yang wajib memakai konstanta adalah perbandingan status di kode Go, karena salah ketik di sana tidak tertangkap kompilator maupun tes. Untuk pembayaran dan status booking, konstantanya ada di `internal/payment/status.go`; aturan baru/dp/lunas hanya ditulis di `statusBookingDari`.

## Konsekuensi

- Review berikutnya tidak perlu menandai `float64` untuk uang atau status literal di SQL sebagai temuan, selama dua penjaga di atas dipatuhi.
- Bila kelak ada mata uang berpecahan, bunga, atau pembagian komisi persentase yang luas, keputusan ini perlu ditinjau ulang.
