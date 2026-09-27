# Audit Konsep & Sprint Plan — Fitur Agen Umroh ("Syiar")

**Sumber yang diaudit:** `agen-azhan.md` (status FINAL) dan `screen-agen.md`
**Diaudit terhadap:** kode `erp-azhan` dan `azhan-microsite` di branch `dev-malik` per 27 September 2026
**Metode:** setiap asumsi teknis di dokumen dicocokkan langsung ke kode, migrasi, dan halaman yang ada. Temuan di bawah menyebut lokasi file sebagai bukti.

---

## 1. Ringkasan

Secara bisnis konsepnya kuat dan konsisten:
- Komisi (ledger) dipisah dari pencairan.
- Kaitan agen melekat per jamaah, bukan per booking.
- Ada audit trail untuk penggantian kaitan agen.
- Biaya pendaftaran di-snapshot saat pengajuan.
- Approval keagenan independen dari verifikasi pembayaran, dengan alasan yang jelas.

Namun **ada 8 asumsi teknis di dokumen yang tidak sesuai kode**. Tiga di antaranya mengubah desain secara mendasar:

1. Tabel `paket` tidak ada. Paket di ERP adalah `schedules`.
2. Constraint `UNIQUE(brand_id, no_hp)` tidak ada.
3. Tidak ada verifikasi OTP/WA. Portal jamaah memakai PIN 6 digit, bukan password.

Selain itu ada **4 celah logika bisnis** yang akan menghasilkan komisi salah atau hilang kalau dibangun apa adanya. Yang terberat: status `lunas` bisa dicapai lewat beberapa jalur dan bisa mundur lalu maju lagi, sehingga komisi berisiko **tercatat dua kali**.

Rekomendasi: **jangan mulai coding fitur sebelum Sprint 0 selesai.** Sprint 0 berisi keputusan bisnis (§4) dan fondasi data (§5).

---

## 2. Asumsi dokumen yang tidak sesuai kode

| # | Asumsi di dokumen | Fakta di kode | Dampak & usulan |
|---|---|---|---|
| T1 | Ada tabel `paket`, komisi ditambahkan ke sana (§7.2, screen C2) | Tidak ada tabel `paket`. Modul `internal/paket/model.go` hanya placeholder. "Paket" di ERP adalah `schedules` (per keberangkatan, dengan harga per tipe kamar), yang dikelola di `ScheduleFormPage.jsx`. | Kolom `nominal_komisi_langsung` dan `nominal_bonus_pembinaan` dipasang di `schedules`. C2 berarti memodifikasi `ScheduleFormPage.jsx`. Keputusan D1: disetel per jadwal atau per kategori paket. |
| T2 | Constraint `UNIQUE(brand_id, no_hp)` sudah ada (§2.2, §3.1, §3.6.2) | **[Koreksi Sprint 0]** Tidak ada di folder `migrations/`, tapi database dev ternyata punya key `uq_jamaah_brand_phone (brand_id, no_hp mentah)` yang dulu ditambahkan manual (tercatat di `ANALISIS-portal-jamaah.md`). Environment baru tidak akan memilikinya, dan key itu tidak menangkap format nomor berbeda (`08…` vs `+62 …`). Ditangani migrasi 060. `no_hp` bebas duplikat, dan login portal (`portal/handler.go`) sengaja menolak kalau ada lebih dari 1 nomor cocok. Sebaliknya, `nik` berstatus `UNIQUE` **global lintas brand** (`013_jamaah_booking.sql:9`). | Semua aturan "nomor HP sudah terdaftar pakai baris lama" tidak punya dasar. Perlu migrasi normalisasi dan deduplikasi `no_hp`, lalu unique index per brand. `nik` global-unique bertentangan dengan §2.2 (orang yang sama di brand lain dianggap jamaah baru). Lihat D2. |
| T3 | Ada verifikasi OTP/WA di self-booking yang bisa di-reuse untuk `/daftar-agen` (A0, §3.6.2, §9) | Tidak ada OTP sama sekali (hasil pencarian `otp` di kedua repo kosong). Akun portal = baris `jamaah` + `portal_pin_hash` (PIN 6 digit). Akun diaktifkan lewat PIN saat self-booking atau lewat link aktivasi dari admin. | A0 tidak bisa "reuse OTP". Pilihannya: bangun OTP WA (butuh vendor gateway, biaya, dan dependensi baru) atau A0 tanpa verifikasi nomor (risiko: orang mendaftarkan nomor milik orang lain). Field "password" di dokumen harus diganti menjadi PIN 6 digit. Lihat D3. |
| T4 | Mungkin ada registrasi akun jamaah mandiri di microsite (§9) | Tidak ada. Akun jamaah baru tercipta hanya lewat self-booking (PIC dan anggota) atau lewat admin. | Resolusi cookie referral (Jalur 2) dilakukan di `selfbooking.ProcessBooking` saat baris `jamaah` baru dibuat, plus di endpoint baru A0. |
| T5 | Jalur 3 terjadi di dalam transaksi booking. "Belum dikonfirmasi apakah ada form jamaah standalone" (§9, screen B4) | **Terkonfirmasi ada dua pintu:** (a) form standalone `/jamaah/new` (`JamaahFormPage`) dan (b) modal "Tambah Jamaah Baru" di `BookingFormPage.jsx:784` yang memanggil `POST /api/admin/jamaah` **sebagai request terpisah sebelum booking disimpan**. | Picker "Kaitkan ke Agen / Tanpa Agen" dipasang di pembuatan jamaah (form dan modal, lewat satu komponen shared), bukan di transaksi booking. Granularitas per-pax jadi alami. Pre-fill "Sama seperti PIC" perlu konteks PIC dikirim ke modal. |
| T6 | Jalur 1 cukup memakai alur booking yang ada (`/api/public/book`) (§7.3) | `selfbooking.ProcessBooking` memaksa jamaah yang login menjadi PIC (`isAuthPIC`, `selfbooking/repository.go:145`). Nomor yang sudah terdaftar dan punya PIN wajib memasukkan PIN pemiliknya. Endpoint publik juga terkena captcha dan rate limit (20 per IP per jam, 3 per nomor per hari). | Agen bukan PIC. Perlu endpoint baru `POST /api/portal/agen/bookings`, dengan inti `ProcessBooking` di-refactor agar menerima parameter "inisiator". UI tetap reuse `BookingWizard`. Lihat D8 soal privasi jamaah existing. |
| T7 | Komisi cukup di-hook di `syncBookingStatusTx` (§7.4) | Status `lunas` bisa dicapai lewat **2 jalur**: (1) `payment.syncBookingStatusTx` dan (2) `booking.recalculateTotalTx` (`booking/repository.go:1449`, dipanggil saat add-on atau diskon berubah). Ubah status manual hanya boleh `batal` (`booking/handler.go:527`). Status juga bisa **mundur** dari `lunas` ke `dp` (`booking/repository.go:1452`, dan saat pembayaran dihapus atau ditolak) lalu naik lagi. | Tanpa perlindungan, komisi **tercatat ganda**. Wajib ada satu fungsi terpusat (mis. `komisi.OnBookingLunas(tx, bookingID)`) yang dipanggil dari kedua jalur, plus `UNIQUE(booking_pax_id, jenis, jamaah_penerima_id)` di `transaksi_komisi` (insert idempoten). |
| T8 | Admin Travel **dan CS** memakai form booking dengan picker agen (screen B4) | Role `cs` diblokir dari seluruh backoffice ERP kecuali gateway CRM (`identity/middleware.go`, `RequireAdminOrCRMAccess`). CS membuat booking lewat `POST /api/admin/crm/deals`. | Untuk CS, kaitan agen masuk lewat field `kode_referral` di CRM deal (sudah disebut di §7.3), bukan lewat B4. Perbaiki teks B4. |

**Open item dokumen yang sekarang terjawab:**
- **Komponen instruksi pembayaran (§9, A1b):** sudah ada di `azhan-microsite/src/app/portal/pembayaran/page.jsx` (daftar rekening brand, upload bukti lewat `uploadPortalMedia`, form pengirim). Komponennya masih menempel di halaman, jadi perlu diekstrak menjadi komponen reusable untuk A1b.
- **Placeholder** `KomisiReferralPage.jsx` (Master) dan `/portal/syiar` (microsite) memang ada dan kosong.

---

## 3. Celah logika bisnis

| # | Celah | Kenapa bermasalah | Usulan |
|---|---|---|---|
| L1 | §7.4 "skip `transaksi_komisi` untuk penerima yang `status_agen != 'aktif'`" | Penerima **cashback** adalah jamaah biasa (`status_agen = 'tidak_aktif'`). Aturan ini membuat cashback **tidak pernah tercatat**. | Aturan skip hanya untuk jenis `langsung`, `pembinaan`, dan `repeat_order`. Cashback dikecualikan. |
| L2 | Cashback masuk "saldo" jamaah | Pencairan (A7) hanya untuk agen aktif. Jamaah biasa tidak punya jalan untuk memakai saldonya. | Keputusan D4: cashback dicairkan (jamaah boleh A7), dipotong ke tagihan booking berikutnya, atau hanya dicatat. |
| L3 | Definisi "repeat order" dan "jamaah baru" tidak didefinisikan secara operasional (§4) | Kode tidak bisa menentukan kapan sebuah pax dianggap repeat. | Usulan definisi: pax dianggap **repeat** jika jamaah itu sudah punya minimal 1 `booking_pax` aktif di booking lain yang pernah `lunas` sebelumnya. Selain itu dianggap **baru**. Perlu konfirmasi (D5). |
| L4 | Komisi langsung dan cashback bisa langsung dicairkan, **tanpa reversal** (§5 poin 8) | Celah fraud: buat booking, lunasi, komisi masuk, ajukan pencairan, lalu batalkan dan minta refund. | Keputusan D6. Usulan: saldo baru **boleh dicairkan** setelah keberangkatan booking sumbernya (ledger tetap tanpa reversal, hanya ketersediaan pencairan yang ditunda). |
| L5 | Pencairan tidak punya data rekening agen | Admin Master menyetujui pencairan, tapi tidak ada nomor rekening tujuan dan tidak ada bukti transfer keluar. | Tambah data rekening agen (bank, nomor, atas nama) dan bukti transfer saat pencairan disetujui. Keputusan D7: pajak (PPh) atas komisi. |
| L6 | §3.1 bilang Jalur 1 mengikat "pax yang **memang baru**", sedangkan §3.4 bilang auto-bind **semua pax yang masih NULL** | Dengan aturan §3.4, agen bisa mengklaim jamaah organik lama (yang sudah punya akun tapi belum punya agen) hanya dengan membuatkan booking. Ditambah pengecekan nomor HP, ini juga membocorkan keberadaan jamaah. | Keputusan D8. Usulan: Jalur 1 hanya mengikat jamaah yang **dibuat** di booking itu. |
| L7 | Snapshot nominal komisi | §7.2 bilang 50% "saat booking terjadi", tapi komisi dihitung saat `lunas`. Nominal jadwal bisa diedit di antara kedua waktu itu. | Keputusan D9. Usulan: snapshot nominal ke `bookings` saat booking dibuat, supaya agen terlindungi dari perubahan setelahnya. |
| L8 | Kode referral milik agen yang sudah nonaktif atau beda brand | Dokumen tidak mengatur apa yang terjadi. | Saat resolve, kode diabaikan jika agennya tidak `aktif` atau `brand_id` berbeda. Tidak ada error ke pengguna. |
| L9 | Agen nonaktif tidak menerima komisi "selama nonaktif" | Tidak jelas apakah komisinya hilang permanen atau ditahan. | Ikuti bunyi dokumen (hilang permanen), tapi tulis eksplisit di S&K agen. |
| L10 | Kolom `domisili` baru | `jamaah.kota` sudah ada (`042_jamaah_kota.sql`). Dokumen jamaah juga sudah punya jenis `pas_foto`. | Pertimbangkan reuse `kota` dan pas foto yang ada, atau tegaskan bedanya. Foto agen wajib disimpan di kategori media **terproteksi**. |

**Risiko yang disengaja (sudah diputuskan bisnis, dicatat saja):** agen bisa aktif sebelum membayar pendaftaran (§5 poin 20), dan biaya pendaftaran hangus kalau keagenan ditolak (§5 poin 22).

**Keamanan untuk endpoint baru:**
- A0 membuat akun publik, jadi wajib captcha dan rate limit (pola dari perbaikan audit sebelumnya).
- Semua endpoint `/api/portal/agen/*` wajib memeriksa `status_agen = 'aktif'` di server, bukan hanya di UI.
- Pengajuan pencairan wajib memakai row lock (§7.6).

---

## 4. Keputusan sebelum coding (Sprint 0)

**Status: DIPUTUSKAN 27 September 2026 — semua mengikuti rekomendasi (★).** `agen-azhan.md` (Bagian 10) dan `screen-agen.md` sudah diperbarui sesuai keputusan ini.

| ID | Pertanyaan | Opsi (★ = rekomendasi) |
|---|---|---|
| D1 | Komisi disetel di mana? | ★ Per `schedules` (sejalan dengan harga) · per kategori paket |
| D2 | Keunikan jamaah | ★ `UNIQUE(brand_id, no_hp_normal)` + ubah `nik` menjadi `UNIQUE(brand_id, nik)` · biarkan apa adanya |
| D3 | Verifikasi nomor di `/daftar-agen` | ★ Tanpa OTP di MVP (captcha, rate limit, persetujuan admin sudah menjadi gerbang), OTP menyusul · bangun OTP WA sekarang |
| D4 | Nasib saldo cashback jamaah | Dicairkan (min. Rp500rb) · ★ potong tagihan booking berikutnya · hanya dicatat |
| D5 | Definisi repeat order | ★ Sudah punya booking `lunas` sebelumnya · definisi lain |
| D6 | Kapan saldo boleh dicairkan | Langsung · ★ setelah tanggal berangkat booking sumbernya |
| D7 | Pajak komisi | Dipotong sistem · ★ di luar sistem untuk MVP (dicatat manual) |
| D8 | Jalur 1 boleh mengikat jamaah existing yang belum punya agen? | ★ Tidak, hanya jamaah yang dibuat di booking itu · ya (sesuai §3.4) |
| D9 | Snapshot nominal komisi | ★ Saat booking dibuat · saat `lunas` |
| D10 | Reuse `kota` dan pas foto untuk domisili dan foto agen? | ★ Kolom terpisah (sesuai dokumen), foto di kategori media terproteksi · reuse |

---

## 5. Sprint Plan

**Asumsi:**
- Sprint 2 minggu.
- 1–2 developer full-stack (Go + React + Next.js).
- Semua kerja di `dev-malik` sesuai AGENTS.md.
- Ukuran task: S ≈ ½–1 hari, M ≈ 2–3 hari, L ≈ 4–5 hari.
- Urutan dibuat supaya setiap sprint menghasilkan sesuatu yang bisa diuji end-to-end.

### Sprint 0 — Keputusan & fondasi (1 minggu)
**Tujuan:** menutup D1–D10 dan menyiapkan pondasi data serta titik hook yang aman.

| Task | Ukuran |
|---|---|
| ~~Rapat keputusan D1–D10, lalu perbarui `agen-azhan.md` dan `screen-agen.md` (termasuk koreksi T1–T8)~~ **Selesai 27 Sep 2026** | S |
| ~~Laporan duplikat `no_hp` dan `nik` per brand di data yang ada, lalu tentukan cara pembersihannya~~ **Selesai**: dev 48 jamaah, 0 duplikat. Laporan ulang wajib dijalankan di production sebelum migrasi 060. | S |
| ~~Migrasi `no_hp` ternormalisasi + unique per brand (dan `nik` per brand, sesuai D2)~~ **Selesai**: `060_jamaah_unique_per_brand.sql` (`no_hp_normal` generated column, cocok 48/48 dengan `shared.PhoneVariants`), error API dibedakan NIK vs nomor HP. | M |
| ~~Refactor: satu fungsi terpusat untuk "booking menjadi lunas" yang dipanggil dari kedua jalur (T7). Belum ada logika komisi.~~ **Selesai**: `internal/komisi.OnBookingLunas` (no-op), dipanggil dari `syncBookingStatusTx` dan `recalculateTotalTx`. | M |
| ~~Test otomatis transisi status booking (baru/dp/lunas/batal, termasuk lunas→dp→lunas).~~ **Selesai**: `internal/testdb` + test integrasi di `payment` dan `booking` (jalan dengan `ERP_TEST_DB=1`, transaksi di-rollback). | M |

**Selesai jika:** keputusan tertulis, migrasi berjalan di database dev dengan bukti, dan test transisi status lulus.

### Sprint 1 — Model data & mesin komisi
**Tujuan:** komisi tercatat benar dan idempoten, bisa diuji tanpa UI.

| Task | Ukuran |
|---|---|
| Migrasi: kolom `jamaah` (§7.1, + rekening agen), komisi di `schedules` dan snapshot di `bookings` (D1, D9), kolom `brands` (§7.9), `transaksi_komisi` (+ unique idempotensi), `pengajuan_pencairan` (+ rekening & bukti transfer keluar, L5), `jamaah_kaitan_log`, `pembayaran_pendaftaran_agen`, `pembayaran_pendaftaran_agen_upload`, `pemakaian_cashback` (D4) | M |
| Snapshot nominal komisi ke booking di semua jalur pembuatan booking (self-booking, admin, finalisasi draft, CRM) | S |
| Modul `internal/komisi`: kalkulasi langsung, pembinaan (1 tier), repeat, dan cashback per pax aktif. Mengikuti aturan L1, D5, D9. Dipasang di hook Sprint 0. | L |
| Query saldo tersedia, saldo tertahan (D6), dan kredit cashback (D4) | S |
| Test otomatis dengan skenario §6 langkah 2, 4, 5, 7, 10, 13, termasuk lunas ganda | M |
| Master Dashboard: 2 field komisi di `ScheduleFormPage` (C2) dan pengaturan agen per brand (C5) | M |

**Selesai jika:** skenario §6 menghasilkan ledger yang persis sama dengan tabel dokumen, dan status lunas ganda tidak menggandakan komisi.

**Status: SELESAI 27 Sep 2026.** Migrasi 061 (model data) dan 062 (`bookings.pertama_lunas_at`, tambahan untuk D5 karena status bisa mundur dari lunas); `komisi.SnapshotNominal` di 4 jalur pembuatan booking; `komisi.ProcessBookingLunas` + `HitungSaldo`; C2 dan C5 di Master Dashboard. Test: skenario §6 langkah 2, 4, 5, 7, 10, 11, 12, 13 + idempotensi + agen nonaktif + snapshot + saldo, semua lulus.

### Sprint 2 — Menjadi agen (jalur jamaah existing)
**Tujuan:** jamaah yang sudah punya akun bisa mengajukan diri, membayar, dan disetujui.

| Task | Ukuran |
|---|---|
| Backend modul `internal/agen`: submit kelengkapan, upload bukti (oleh pemohon atau admin), verifikasi/tolak pembayaran, approve/tolak keagenan (+ `keputusan_agen`), toggle aktif/nonaktif, generate `kode_referral`, set upline otomatis | L |
| Microsite: ekstrak komponen instruksi pembayaran dari `portal/pembayaran`, lalu buat A1, A1a, A1b (termasuk revisi saat ditolak), A2, A4 | L |
| Travel Dashboard: B1 Persetujuan Agen (approve tidak pernah dikunci oleh status pembayaran) | M |
| Test skenario §6 langkah 17, 18, 19 (dengan jalur existing), 20 | S |

**Selesai jika:** skenario 17–20 berjalan end-to-end, tampilan responsif (375 px & 1440 px), dengan screenshot sebagai bukti.

### Sprint 3 — Akuisisi jamaah (Jalur 2, Jalur 3, CRM) & `/daftar-agen`
**Tujuan:** kaitan agen terisi dengan benar dari semua pintu masuk.

| Task | Ukuran |
|---|---|
| Microsite: middleware `?ref=` (cookie 90 hari, last-click-wins) dan meneruskan kode ke API (route `book` dan A0) | S |
| Backend: resolve referral dan auto-bind multi-pax di `selfbooking.ProcessBooking` (L8), plus `kaitan_status` | M |
| Jalur 3: komponen picker agen di `JamaahFormPage` dan modal di `BookingFormPage` (shared), pre-fill dari PIC, validasi di backend `POST /api/admin/jamaah`. Auto-bind juga di pembuatan dan finalisasi booking admin. | L |
| CRM deal: field `kode_referral` | S |
| A0 `/daftar-agen` (D3: PIN 6 digit, tanpa OTP, dengan captcha dan rate limit) yang langsung lanjut ke A1a | M |
| Test skenario §6 langkah 3, 3b, 11, 12, 14, 19 | S |

### Sprint 4 — Jalur 1 & dashboard agen
**Tujuan:** agen aktif bisa membuat booking untuk jamaahnya dan melihat kinerjanya.

| Task | Ukuran |
|---|---|
| Refactor inti `ProcessBooking` agar menerima inisiator (publik atau agen), lalu endpoint `POST /api/portal/agen/bookings` (T6, D8) | L |
| Microsite: A3 Dashboard Agen, A5 (reuse `BookingWizard` dalam mode agen), A6 Riwayat Komisi | L |
| Travel Dashboard: B2 Daftar Agen, B2a Detail Agen (8 kelompok data), B3 Riwayat Komisi | L |
| Travel Dashboard: B5 Pakai Kredit Cashback di detail booking (D4) | S |
| Test skenario §6 langkah 2 lewat UI agen | S |

### Sprint 5 — Pencairan, kontrol Admin Master & rilis
**Tujuan:** siklus uang lengkap dan fitur siap UAT.

| Task | Ukuran |
|---|---|
| A7 Tarik Saldo (minimal Rp500rb, satu pengajuan pending, row lock) dan C1 Persetujuan Pencairan lintas brand (+ bukti transfer keluar) | M |
| C4 Ganti Kaitan Agen (gate `transaksi_komisi`, alasan wajib, log) | M |
| Test otomatis lengkap 20 skenario §6, review keamanan endpoint baru (otorisasi, brand scope, rate limit) | M |
| UAT dengan Admin Travel satu brand, perbaikan, catatan rilis dan panduan deploy | M |

**Selesai jika:** 20 skenario §6 lulus, tidak ada temuan keamanan tingkat tinggi, dan UAT disetujui.

---

## 6. Dependensi & risiko jadwal

- **Keputusan Sprint 0 menentukan segalanya.** D2 (keunikan jamaah) dan D8 (Jalur 1) mengubah skema dan endpoint. Menunda keputusan berarti rework.
- **OTP (D3):** diputuskan menyusul setelah MVP. Kalau nanti dibangun, perkirakan sekitar 1 sprint untuk integrasi vendor WA.
- **Pembersihan data duplikat (D2):** migrasi unique index gagal selama ada duplikat `no_hp`/`nik` per brand. Durasi pembersihan bergantung jumlah duplikat di data nyata; laporan duplikat di Sprint 0 menentukan apakah cukup 1 minggu.
- **Turnstile di microsite:** perbaikan audit sebelumnya menyisakan widget Turnstile asli yang belum terpasang di `BookingWizard`. A0 dan A5 memakai jalur yang sama, jadi sebaiknya dipasang paling lambat di Sprint 3.
- **Test booking dan payment belum ada.** Sprint 0 sengaja membangunnya lebih dulu, karena komisi bergantung penuh pada transisi status yang benar.
- **Kontrak microsite:** endpoint publik yang ada tidak diubah secara breaking. Semua kebutuhan agen memakai endpoint baru di `/api/portal/agen/*`.
