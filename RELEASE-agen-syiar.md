# Catatan Rilis & Panduan Deploy — Agen Syiar

**Cakupan:** fitur Agen Umroh "Syiar" (Sprint 0–5, `AUDIT-SPRINT-agen-syiar.md`) beserta perbaikan audit keamanan sebelumnya.
**Repo:** `erp-azhan` dan `azhan-microsite`, branch `dev-malik`. Keduanya harus dirilis bersamaan.

---

## 1. Ringkasan fitur

| Permukaan | Screen | Ringkasan |
|---|---|---|
| Microsite | A0 `/daftar-agen` | Daftar akun calon agen (PIN 6 digit, captcha, rate limit), login otomatis, lanjut ke A1a |
| Microsite | A1–A4 `/portal/syiar` | Ajukan jadi agen, lengkapi data, bayar pendaftaran, status menunggu/aktif/nonaktif |
| Microsite | A3 | Dashboard agen: saldo tersedia & tertahan, kredit cashback, jamaah, komisi terbaru |
| Microsite | A5 `/portal/syiar/booking-baru` | Agen membuat booking untuk jamaahnya (Jalur 1) dengan `BookingWizard` mode agen |
| Microsite | A6 `/portal/syiar/riwayat-komisi` | Riwayat komisi dengan status ketersediaan |
| Microsite | A7 `/portal/syiar/pencairan` | Tarik saldo (min. Rp500.000, satu pengajuan diproses), rekening tersimpan, bukti transfer |
| Microsite | A9 | Link `?ref=KODE` disimpan di cookie httpOnly 90 hari (last-click wins) |
| Travel Dashboard | B1 | Persetujuan agen & verifikasi pembayaran pendaftaran |
| Travel Dashboard | B2, B2a | Daftar & detail agen (8 kelompok data), toggle aktif/nonaktif |
| Travel Dashboard | B3 | Riwayat komisi brand (filter agen, jenis, tanggal) |
| Shared (kedua dashboard) | B4/C3 | Picker "Kaitkan ke Agen / Tanpa Agen" wajib saat membuat jamaah |
| Shared (kedua dashboard) | B5 | Pakai kredit cashback sebagai potongan booking |
| Master Dashboard | C1 `/komisi` | Persetujuan pencairan lintas brand, wajib unggah bukti transfer keluar |
| Master Dashboard | C2, C5 | Nominal komisi per jadwal; biaya pendaftaran agen & WA Admin Travel per brand |
| Master Dashboard | C4 `/komisi/ganti-kaitan` | Ganti kaitan agen hasil Jalur 3, dengan gate komisi dan log audit |

Komisi dihitung otomatis saat booking pertama kali `lunas` (langsung, pembinaan 1 tier, repeat order 50%, cashback 50% sebagai kredit). Ledger tidak pernah dibalik; komisi baru bisa dicairkan setelah tanggal berangkat booking sumbernya.

---

## 2. Migrasi (jalankan berurutan)

| File | Isi | Catatan deploy |
|---|---|---|
| `059_admin_refresh_tokens.sql` | Refresh token admin disimpan & bisa dicabut | **Semua admin perlu login ulang** setelah deploy |
| `060_jamaah_unique_per_brand.sql` | Unique `no_hp` (dinormalisasi) & `nik` per brand | **Gagal bila masih ada duplikat.** Jalankan laporan duplikat di produksi dan bersihkan dulu |
| `061_agen_syiar.sql` | Kolom agen di `jamaah`, komisi di `schedules`/`bookings`, kolom `brands`, tabel ledger, pencairan, log kaitan, pembayaran pendaftaran, pemakaian cashback | — |
| `062_booking_pertama_lunas.sql` | `bookings.pertama_lunas_at` (baru vs repeat order) | Mengisi booking yang sudah `lunas` dengan `created_at` |
| `063_pemakaian_cashback_diskon.sql` | Tautan `pemakaian_cashback` ↔ `booking_discounts` (ON DELETE CASCADE) | — |
| `064_jamaah_kaitan_sumber.sql` | `jamaah.kaitan_sumber` (jalur1/jalur2/jalur3/ikut_pic) | Data lama bernilai NULL, tidak bisa diganti lewat C4 |

Perintah per file:

```bash
go run -buildvcs=false migrations/run.go migrations/0XX_nama.sql
```

Backup database sebelum migrasi. Migrasi 061–064 hanya menambah kolom/tabel; tidak ada data yang dihapus.

---

## 3. Konfigurasi environment

| Variabel | Repo | Nilai |
|---|---|---|
| `TRUSTED_PROXIES` | erp-azhan | IP/CIDR server microsite dan reverse proxy. Tanpa ini rate limit per IP berlaku bersama untuk semua pengunjung microsite |
| `TURNSTILE_SECRET_KEY` | erp-azhan | **Biarkan kosong** sampai widget Turnstile asli terpasang di microsite (saat ini `BookingWizard` dan `/daftar-agen` masih mengirim token demo). Mengisinya sekarang akan membuat booking publik dan daftar agen selalu gagal |
| `API_BASE_URL_INTERNAL`, `NEXT_PUBLIC_API_BASE_URL` | azhan-microsite | Tidak berubah |

---

## 4. Urutan deploy

1. Backup database produksi.
2. Jalankan laporan duplikat `no_hp`/`nik` per brand; bersihkan sampai kosong.
3. Deploy backend `erp-azhan` (belum dinyalakan), jalankan migrasi 059–064 berurutan.
4. Nyalakan backend, cek `GET /api/health`.
5. Build & deploy `frontend/master-dashboard` dan `frontend/travel-dashboard`.
6. Deploy `azhan-microsite`.
7. Umumkan ke admin: login ulang (migrasi 059).
8. Admin Master mengisi C5 (biaya pendaftaran & WA Admin Travel) dan C2 (nominal komisi) untuk jadwal yang ikut program. Jadwal dengan nominal kosong tidak ikut program Syiar.

**Rollback:** kembalikan binary & frontend versi lama. Kolom/tabel baru tidak mengganggu kode lama, jadi migrasi tidak perlu dibalik. Jangan menghapus tabel ledger yang sudah berisi data.

---

## 5. Pemetaan 20 skenario §6 ke test otomatis

Semua test integrasi berjalan di dalam transaksi yang selalu di-rollback (`ERP_TEST_DB=1 go test ./internal/...`).

| Langkah | Test |
|---|---|
| 1, 3b, 17 | `agen.TestLangkah3JalurReferralDanUpline`, `agen.TestLangkah17ApproveTidakMenungguPembayaran` |
| 2 | `selfbooking.TestBookingAgenJalur1`, `komisi.TestSimulasiBagian6` |
| 3 | `agen.TestLangkah3JalurReferralDanUpline`, `agen.TestResolveKodeReferral` |
| 4 (backend) | `komisi.TestSimulasiBagian6`; last-click cookie diuji E2E di microsite |
| 5, 6, 7 | `komisi.TestSimulasiBagian6` |
| 8, 9 | `agen.TestLangkah8dan9Pencairan` |
| 10, 11, 12, 13 | `komisi.TestSimulasiBagian6` |
| 14 | `agen.TestLangkah14Jalur3PerPax` |
| 15, 16 | `agen.TestLangkah15dan16GantiKaitan` |
| 18 | `agen.TestLangkah18DitolakLaluAjukanUlang` |
| 19 | `agen.TestLangkah19BrandTanpaBiaya` |
| 20 | `agen.TestLangkah20RevisiPembayaranDitolak` |

Test pendukung: idempotensi komisi, agen nonaktif, snapshot nominal, saldo & kredit cashback, booking agen (D8), regresi booking publik + referral, kinerja agen, cashback B5.

---

## 6. Review keamanan endpoint baru

| Area | Kontrol |
|---|---|
| `/api/portal/agen/*` | Token portal wajib; fitur agen aktif dicek di server; brand selalu dari akun agen, bukan dari body |
| Booking agen | Jamaah existing hanya milik agen sendiri; nomor milik pihak lain ditolak dengan pesan umum (D8); 20 booking per agen per jam |
| Bukti pencairan (portal) | Hanya pengajuan milik agen itu yang sudah disetujui; nama file divalidasi regex (tanpa path traversal) |
| `/api/admin/agen/*`, B5 | `RequireAdminRole` + brand scope di repository; CS diblokir |
| C1, C4 | `RequireAdminRole` + `RequireSuperAdmin`; Admin Travel mendapat 403 (diverifikasi) |
| Pencairan | Row lock agen; satu pending; saldo dicek saat diajukan dan saat disetujui; bukti transfer wajib dari upload terproteksi |
| C4 | Row lock jamaah; gate Jalur 3, belum ada komisi, bukan agen; log audit wajib alasan |
| Publik | `/api/public/agen/daftar` captcha + 10 percobaan per IP per jam; kode referral hanya dari cookie httpOnly |

Tidak ada temuan keamanan tingkat tinggi pada endpoint baru.

---

## 7. Checklist UAT (Admin Travel satu brand)

UAT belum dilakukan; perlu pengguna nyata. Gunakan brand uji dengan jadwal yang nominal komisinya diisi.

- [ ] A0/A1: daftar agen dari link, lengkapi data, bayar (brand berbayar & gratis)
- [ ] B1: setujui/tolak keagenan, verifikasi/tolak pembayaran, unggah bukti atas nama pemohon
- [ ] A9/A8: jamaah baru daftar lewat link agen, kaitan terisi
- [ ] B4: buat jamaah dengan pilihan agen/tanpa agen; pre-fill "Sama seperti PIC"
- [ ] A5: agen membuat booking; Admin Travel follow up pembayaran sampai lunas
- [ ] A3/A6/B3: komisi tampil tertahan sebelum berangkat
- [ ] B2/B2a: detail agen lengkap; nonaktifkan lalu aktifkan kembali
- [ ] B5: kredit cashback dipakai, lalu diskonnya dihapus (kredit kembali)
- [ ] A7/C1: ajukan pencairan, Admin Master transfer & unggah bukti, agen melihat bukti
- [ ] C4: koreksi kaitan Jalur 3; percobaan pada jamaah yang sudah pernah lunas ditolak

---

## 8. Keterbatasan & temuan di luar scope

- **Turnstile:** widget asli belum dipasang di microsite (lihat §3).
- **Pajak komisi (D7):** diproses manual di luar sistem.
- **OTP (D3):** belum ada; daftar agen hanya dilindungi captcha dan rate limit.
- **Tanggal lahir infant** pada booking publik tidak tersimpan (validasi ada, insert tidak mengisi kolom). Belum diperbaiki.
- **Zona waktu:** beberapa tanggal tampil maju satu hari/berjam-jam (mis. `created_at` komisi, `diganti_at`). Kemungkinan waktu DB dikirim sebagai UTC lalu browser menambah zona waktu lagi; perlu dicek konfigurasi DSN (`loc`). Belum diperbaiki.
- **Pindah brand jamaah:** `PUT /api/admin/jamaah/{id}` masih menerima `brand_id` baru dari Super Admin (UI menguncinya). Untuk jamaah yang terkait agen, perpindahan brand melanggar aturan "semua perhitungan per brand". Disarankan menolak perubahan brand bila jamaah punya kaitan agen, status agen, atau riwayat komisi. Belum diperbaiki.
- **Komisi infant:** mengikuti spesifikasi (per pax aktif, termasuk infant). Perlu konfirmasi bisnis.
