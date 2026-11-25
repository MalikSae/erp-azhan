# Spek: Menu Jamaah per peran dan istilah Booking/Jamaah (2026-11-25)

Status: disepakati dan dibangun 2026-11-25 (ERP dev-malik, web dan app branch agen-fitur).

## Masalah
- Web punya dua daftar orang dengan nama mirip: "Data Jamaah" (`/akun/jamaah`, jamaah yang akun daftarkan) dan "Jamaah Saya" (`/akun/agen/jamaah`, jamaah agen). Jamaah yang dibookingkan agen muncul di keduanya.
- App hanya punya layar agen, berjudul "Data Jamaah"; menu itu tampil untuk semua akun, sehingga akun biasa membuka layar agen yang kosong. App tidak punya daftar jamaah milik akun.
- Daftar transaksi bernama "Booking Saya" di web dan "Perjalanan" di app.

## Analisa pengguna
- **Akun biasa**: ingin berangkat atau memesankan keluarga; kebutuhan = melengkapi data orang yang ia daftarkan.
- **Agen**: mencari penghasilan; kebutuhan = siapa yang perlu dihubungi, siapa yang sudah aman, kapan komisi cair, dan melengkapi data jamaah yang ia bookingkan. Bagi agen, "dibuat oleh saya" dan "terikat ke saya" adalah istilah teknis, bukan dua hal.

## Keputusan
**Istilah (web dan app)**
1. Daftar orang = **"Jamaah"**; daftar transaksi = **"Booking"**; tanpa akhiran "Saya"/"Data".
   - Web: menu "Booking Saya" -> "Booking", "Data Jamaah" -> "Jamaah".
   - App: tab "Perjalanan" -> "Booking", menu "Data Jamaah" -> "Jamaah". Alamat halaman tetap.
2. `docs/ux-copy.md` (erp-azhan dan azhan-booking) dan GLOSSARY mengikuti.

**Menu Jamaah menurut peran**
3. Akun biasa: jamaah yang ia daftarkan (GET /api/account/jamaah), untuk melengkapi data. App mendapat layar baru setara web.
4. Agen: satu menu Jamaah dengan segmen **Jamaah | Prospek**. Segmen Jamaah menggabungkan:
   - jamaah dari booking buatan agen (`dibuat_oleh_agen_akun_id`), termasuk yang belum DP (status "Belum DP");
   - jamaah terikat ke agen dari booking mulai DP;
   - agen sendiri bila ikut berangkat (lencana **"Anda"**). Keluarga atau orang lain yang dibookingkan agen = jamaah biasa (tetap berkomisi).
5. Saringan: **Semua / Perlu tindakan / Lunas / Berangkat**. Perlu tindakan = belum DP, belum lunas, atau data/dokumen belum lengkap.
6. Ketuk baris -> halaman detail jamaah agen. Bila booking dibuat agen sendiri, ada tautan **"Isi data"** ke halaman ubah data; selain itu hanya baca.
7. Kotak "Jamaah" di dashboard web dan app = angka saringan "Semua".
8. Agen tidak punya dua menu jamaah lagi. Kotak dashboard dan notifikasi jamaah_referral, jamaah_aktivasi, prospek_* membuka menu Jamaah.

## Rencana kerja
1. ERP (dev-malik): perluas GET /api/portal/agen/data-jamaah dan detailnya; field `saya`, `bisa_isi_data`, `perlu_tindakan`; status `belum_dp`; dashboard `jumlah_jamaah` = jumlah baris. Tes integrasi.
2. Web (agen-fitur): label menu dan judul, menu Jamaah per peran, saringan baru, tautan Isi data.
3. App (agen-fitur): tab Booking, layar Jamaah akun biasa, layar Jamaah agen dengan saringan baru dan tautan Isi data.
4. Dokumen: ux-copy kedua repo, GLOSSARY, memori. Cek simulasi 375/1440, cabut sebelum commit.
