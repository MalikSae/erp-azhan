# Spek: Konsultan dari tautan agen (2026-11-27)

Status: disepakati 2026-11-27 (brainstorming dengan pemilik). Belum dibangun.

## Masalah
Agen membagikan tautan paket berkode `?ref`, tetapi tombol WhatsApp di halaman paket tetap mengarah ke nomor brand. Calon jamaah yang datang dari tautan agen tidak melihat siapa agennya, dan pertanyaannya masuk ke brand, bukan ke agen.

## Fakta kode
- Web: `src/proxy.js` menyimpan `?ref=KODE` (6 karakter A–Z0–9) di cookie httpOnly `azhan_ref` selama 90 hari. Klik terakhir menang. Klik dicatat ke `POST /api/public/referral/klik`. Cookie hanya dibaca saat daftar akun (`src/app/api/auth/[action]/route.js`).
- ERP: belum ada endpoint publik yang mengembalikan nama atau WA agen dari kode. Nama dan WA agen sekarang hanya terlihat oleh akun yang diajaknya (`GET /api/account/upline`, `internal/customer/upline.go`, kartu "Konsultan Anda").
- Data agen ada di `customer_accounts`: `nama_lengkap`, `no_hp`, `status_agen`, `kode_referral`, `brand_id`.
- Halaman web sudah dirender dinamis karena membaca header `x-brand`, jadi tidak ada cache halaman bersama antarpengunjung.
- App: tautan `/paket` tidak membuka app. Kode tautan (90 hari) dan install referrer Android disimpan di `apps/hana/src/data/kodeReferralInstall.ts`. Kartu "Konsultasi gratis" di detail paket (`src/components/paket/detail/Pesan.tsx`) memakai `KONTAK_WA` brand.
- S&K agen adalah isian brand di ERP (`brands.syarat_ketentuan_agen`) yang diubah admin.

## Keputusan
**Persetujuan agen**
1. Berlaku otomatis untuk semua agen aktif, tanpa sakelar. Admin menambah satu poin di S&K agen di ERP, misalnya: "Nama dan nomor WhatsApp Anda tampil di web dan app bagi pengunjung yang membuka tautan Anda." Perubahan ini adalah konten, bukan kode.

**Siapa konsultannya** (aturan sama di web dan app)
2. Akun login yang bukan agen dan punya upline aktif: konsultannya upline akun. Kode tautan diabaikan, karena upline tidak bisa diklaim belakangan.
3. Akun login yang agen, atau akun login tanpa upline aktif: WA brand.
4. Belum login: agen dari kode tautan terakhir yang masih dalam 90 hari. Di web sumbernya cookie `azhan_ref`. Di app sumbernya kode tautan, lalu install referrer.
5. Kode kosong, salah, milik agen nonaktif, atau milik brand lain: WA brand, tanpa pesan galat.

**Di mana**
6. Hanya halaman paket.
   - Detail paket web (`/paket/...`): di kartu harga, di bawah tombol utama, tampil "Konsultan Anda", lalu nama agen, lalu tautan "Tanya via WhatsApp" ke konsultan. Bilah "Tanya via WhatsApp" saat paket penuh atau ditutup juga ke konsultan.
   - Daftar paket web (`/paket`, hub, landing): tautan WA ke konsultan, tanpa baris nama.
   - App Hana, detail paket: kartu "Konsultasi gratis" menampilkan "Konsultan Anda" dan nama, dan WA-nya ke konsultan. "Pesan via WhatsApp" (booking online sudah ditutup) tetap ke tim travel brand.
7. Tetap ke brand: header, footer, beranda, jadwal, kalkulator, kontak, cabang, jadi agen, checkout, bayar, invoice, dan halaman akun.
8. Isi pesan WA ke konsultan: "Halo {nama agen}, saya ingin bertanya tentang paket {paket} (berangkat {tanggal})." Tanpa nama paket: "Halo {nama agen}, saya ingin berkonsultasi tentang paket umroh."

**Endpoint ERP (baru; endpoint lama tidak berubah)**
9. `GET /api/public/v1/agen/konsultan?brand_id={id}&kode={KODE}` mengembalikan
   `200 {"konsultan": {"nama_lengkap": "...", "whatsapp": "62..."}}` atau `200 {"konsultan": null}`.
   - Hanya agen `status_agen = 'aktif'` dengan `brand_id` yang sama. Kode dinormalisasi UPPERCASE dan harus 6 karakter A–Z0–9. Selain itu `null`.
   - Hanya nama dan WA, tanpa foto, domisili, atau kode.
   - Batas jumlah kode berbeda per IP per jam, untuk mencegah pengumpulan nomor WA. IP diambil dengan `shared.ClientIP` (header proxy hanya dipercaya dari `TRUSTED_PROXIES`). Lewat batas: `null`, bukan galat.
   - Respons untuk `null` dan berisi tidak boleh dibedakan selain isinya (status sama).
10. Tanpa migrasi.

**Web azhan-booking**
11. Proxy meneruskan kode, dari `?ref` atau cookie, ke render lewat header internal baru, yang ditambahkan ke `INTERNAL_REQUEST_HEADERS` dan selalu ditimpa. Dengan begitu kunjungan pertama dari tautan langsung menampilkan agen.
12. Fetch konsultan di server di-cache per URL (per kode) selama 60 detik. Agen yang dinonaktifkan berhenti tampil paling lambat setelah masa cache itu habis.
13. Nama agen tidak masuk metadata, Open Graph, atau JSON-LD. Canonical tetap tanpa `?ref`.

## Di luar lingkup
- Pencatatan klik tombol WA ke agen.
- Sakelar agen untuk menyembunyikan diri.
- Header, footer, dan halaman lain selain halaman paket.

## Risiko yang disadari
- Tautan yang diteruskan ke grup ikut menyebarkan nama dan WA agen. Ini diterima melalui poin S&K.
- Pengunjung yang belum login dan membuka tautan dua agen melihat agen terakhir, sama dengan agen yang akan mengikatnya saat daftar.
