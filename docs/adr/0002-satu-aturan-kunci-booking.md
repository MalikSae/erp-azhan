# 0002. Satu aturan kunci jamaah untuk semua jalur booking

Tanggal: 18 November 2026
Status: diterima (tahap E1)

## Konteks

Booking self-service punya dua pintu dengan aturan kunci jamaah berbeda: `POST /api/account/bookings` dan `/bookings-app` (akun biasa: kunci lewat upline akun) dan `POST /api/portal/agen/bookings` (agen: kunci ke agen pemesan). Agen aktif yang memesan lewat app jatuh ke aturan akun biasa, sehingga jamaahnya tidak terkunci ke dirinya, batasnya salah, dan penanda diri tidak tertulis.

## Keputusan

- Mode ditentukan status akun pemesan, dibaca di transaksi booking: agen aktif di brand yang sama memesan sebagai agen; agen nonaktif, pengajuan, dan akun biasa memesan sebagai akun biasa. `ProcessBookingAgen` dan `agenID` dihapus.
- `kunciJamaahPemesan` (`internal/selfbooking/kunci.go`) satu-satunya pintu kunci, selalu lewat `agen.IkatJamaah` (first-set wins). Jamaah yang sudah terkunci tidak berpindah; hanya Admin.
  - Pemesan agen: kunci ke pemesan. Jamaah diri dikunci ke `upline_akun_id` agen, aktif maupun nonaktif. Diri = jamaah tertaut ke akun, atau NIK sama dengan `nik_agen` selama jamaah itu tidak tertaut ke akun lain dan tidak dibuat akun lain (`nik_agen` tidak diverifikasi, jadi NIK jamaah orang lain tidak membuatnya diri agen; pemesan ikut berangkat dengan jamaah itu ditolak `ErrSayaSendiriBukanAnda`).
  - Pemesan bukan agen: `IkatAkunDariJamaah`, lalu `UplineAkun` dan `IkatRombongan`, seperti sebelumnya.
- Pemesan agen yang ikut berangkat: NIK diisi server dari `nik_agen`, NIK request yang berbeda ditolak, `nik_agen` kosong memberi `ErrNIKAgenBelumAda`. `booking_pax.saya_sendiri_akun_id` terisi di semua jalur. `bookings.dibuat_oleh_agen_akun_id` terisi untuk agen aktif di jalur mana pun.
- Batas menurut status akun: agen aktif memakai hold seat agen (20 per jadwal, 45 total, override); akun lain 9/18 seat dan 3 booking per 24 jam. Satu limiter percobaan per akun (20/jam) dan per IP (120/jam); percobaan identitas gagal satu kunci per akun.
- `jamaah_id` (Jamaah Saya) hanya untuk agen aktif dan wajib cocok NIK dan nama; akun biasa ditolak `ErrInputTidakValid`.
- `peringatan[]` jamaah terkunci dihapus dari respons: oracle status kunci. Aturan kunci dijamin SQL `IkatJamaah`.
- Fingerprint idempotensi memakai akun pemesan saja; `request_key` yang sama lintas jalur adalah jawaban ulang. Akun yang menjadi agen antar retry dengan payload berbeda mendapat `ErrRequestConflict`.
- `POST /api/portal/agen/bookings` tetap sebagai alias tipis: `pic.saya_sendiri` diterjemahkan ke `pemesan_tidak_ikut=false`, bawaan portal pemesan tidak ikut, bentuk respons lama dikunci tes kontrak. Bawaan `/bookings-app`: ikut.

## Konsekuensi

- Upline punya dua kolom (`direkrut_oleh_akun_id`, `upline_akun_id`), satu konsep. Cabang agen membaca `upline_akun_id`, cabang akun biasa membaca `direkrut_oleh_akun_id`; komentar di `kunciJamaahPemesan` menjaga keduanya tidak tertukar.
- Komisi langsung ke upline nonaktif hangus tanpa backfill saat diaktifkan lagi.
- Perilaku agen di app berubah (kunci ke diri, batas 20/45, tanpa batas harian); belum rilis production.
- Prasyarat rilis: PR mobile M0 (azhan-mobile `packages/core/src/booking.ts` mengisi NIK dari profil agen dan menampilkan galat "Lengkapi NIK di profil agen.") dirilis bersama E1. Tanpa itu agen yang ikut berangkat lewat `/bookings-app` ditolak bila NIK ketikan berbeda dari `nik_agen` atau `nik_agen` kosong; kontrak ini dikunci tes handler.
- Penyesuaian mesin komisi (`loadPax`, `akunJamaah`, NIK-sama) bukan bagian tahap ini (E3).
