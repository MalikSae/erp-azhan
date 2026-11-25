# Spek: Prospek agen (2026-11-25)

Status: disepakati 2026-11-25 (brainstorming + grilling), dibangun 2026-11-25 (ERP dev-malik, web dan app branch agen-fitur). Pengingat hold untuk agen memakai jenis notifikasi `prospek_hold`.

## Masalah
Spek app (SPEK-APP-AZHAN-BOOKING-2026-11-12, cerita pengguna jamaah-agen) meminta agen melihat "siapa yang sudah booking lewat link saya dan status pembayarannya". Keputusan dompet 2026-11-15 membatasi Data Jamaah mulai DP, sehingga booking yang belum DP (hold 24 jam) dan akun yang baru daftar lewat tautan agen tidak terlihat. Agen tidak bisa menindaklanjuti pada saat paling berguna.

## Fakta kode
- Jamaah diikat ke agen saat booking dibuat (status `baru`), bukan saat DP (`internal/selfbooking/kunci.go`). Batas DP hanya di tampilan Data Jamaah agen (`internal/agen/portal_app.go`).
- Hold yang habis hanya melepas seat; booking tetap `baru` dan masih bisa DP bila seat tersedia (`internal/booking/seat_hold_expiry.go`).
- Akun yang daftar lewat tautan agen tercatat di `customer_accounts.direkrut_oleh_akun_id`.

## Keputusan
**Isi**
1. Prospek = akun dengan `direkrut_oleh_akun_id` = agen, dalam tiga tahap:
   - **Belum booking**: akun tanpa booking. Akun yang sudah jadi agen tetap ikut (diputuskan pemilik 2026-11-26).
   - **Belum DP**: booking `baru` dengan hold berjalan (sisa waktu tampil).
   - **Expired** (tahap `hold_habis`): booking `baru` tanpa hold. Bila kuota jadwal penuh atau pendaftaran ditutup, label **"Kuota penuh"** / **"Pendaftaran tutup"** (agen menawarkan paket lain).
2. Booking yang dibuat agen sendiri tidak ikut (sudah di Booking Saya).
3. Booking oleh akun lain yang berisi jamaah terikat ke agen tidak ikut (kontak orang lain tidak dibuka).
4. Keluar dari daftar saat DP (pindah ke Jamaah Saya), saat batal, atau setelah 30 hari tanpa kabar (sejak booking dibuat atau sejak akun daftar tanpa booking).

**Data dan tindakan**
5. Per prospek: nama akun pemesan, paket dan tanggal berangkat, tanggal booking, sisa hold, jumlah jamaah. Tanpa NIK, nama tiap jamaah, atau dokumen.
6. Satu tindakan: **Chat WhatsApp** ke nomor akun pemesan, pesan awal per tahap (bisa diubah agen):
   - Belum DP: "Assalamu'alaikum {nama}, saya {agen} dari {brand}. Booking {paket} Anda belum DP. Ada yang bisa saya bantu?"
   - Hold habis: sama, dengan "masa tahan seat sudah habis".
   - Belum booking: "Assalamu'alaikum {nama}, saya {agen} dari {brand}. Ada paket umroh yang ingin Anda tanyakan?"
   Agen tidak bisa membuka detail booking, mengunggah bukti, atau membatalkan.

**Tampilan (web dan app)**
7. Halaman Jamaah Saya: segmen **"Jamaah"** dan **"Prospek (N)"**. Dashboard tetap tiga kotak.
8. Urutan: hold paling cepat habis, lalu "Expired" terbaru, lalu "Belum booking" terbaru. Label lencana: "Sisa N jam", "Expired", "Kuota penuh", "Pendaftaran tutup", "Belum booking" (diputuskan pemilik 2026-11-25).

**Notifikasi**
9. **"Prospek baru"** saat akun lewat tautan agen membuat booking; membuka segmen Prospek.
10. Pengingat **"Hold berakhir 3 jam lagi"** juga ke agen perekrut. Akun baru tanpa booking tidak dinotifikasi.

**Aturan lama dan privasi**
11. Keputusan dompet 2026-11-15 tetap: Data Jamaah dan "Jumlah Jamaah" mulai DP. Prospek tidak masuk komisi, peringkat, maupun KPI.
12. Kebijakan Privasi (konten CMS, diisi tim legal/admin) ditambah satu kalimat: agen yang tautannya dipakai pelanggan dapat menghubungi pelanggan terkait pendaftaran dan booking. Usulan: "Bila Anda mendaftar lewat tautan agen kami, agen tersebut dapat menghubungi Anda terkait pendaftaran dan booking."

## Rencana kerja
1. **ERP (dev-malik):** `GET /api/portal/agen/prospek` (RequireAgenAktif); jenis notifikasi `prospek_baru` (ref agen); agen perekrut ditambah sebagai penerima pengingat hold. Tes integrasi: tiap tahap, batas 30 hari, jadwal penuh/tutup, booking buatan agen tidak ikut, akun lain tidak bocor, agen nonaktif 403.
2. **Web dan app (branch agen-fitur):** segmen Prospek di Jamaah Saya, tombol WA dengan pesan awal, tujuan notifikasi `prospek_baru`. Cek dengan simulasi 375/1440, lalu simulasi dicabut.
3. **GLOSSARY:** istilah "prospek". Kalimat Kebijakan Privasi diserahkan ke tim legal untuk CMS.
