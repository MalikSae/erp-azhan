---
trigger: always_on
---

# AGENTS.md — Project Rules for AI Coding Agents

File ini dibaca Antigravity (dan tool AI lain yang kompatibel AGENTS.md) di awal setiap sesi. Aturan di sini berlaku terus-menerus — tidak perlu diulang manual di tiap prompt.

Lihat juga `.agents/rules/design-system.md` untuk aturan khusus frontend UI (pelengkap, diterapkan setelah file ini).

---

## Project Overview

- **Nama**: ERP Azhan — dimulai sebagai microsite umroh single-travel, dirancang untuk tumbuh jadi ERP multi-brand Azhan Grup
- **Tipe**: Web app — backend API + beberapa frontend dashboard + microsite publik (menyusul)
- **Tahap**: MVP / Fase 1 (lihat `fase-development-microsite-travel.md` untuk roadmap lengkap)
- **Dokumen acuan**: `spesifikasi-teknis-sistem-umroh.md` (skema DB & kontrak API), `form-input-paket-umroh.md` (field form), `arsitektur-modular-erp-azhan-grup.md` (arah jangka panjang)

---

## Ekosistem Project — Repo Terkait (WAJIB dipahami sebelum ubah endpoint publik)

Ada project TERPISAH yang konsumsi API ini: **`azhan-microsite`** (`C:\laragon\www\azhan-microsite`, Next.js, tanpa database sendiri). Endpoint publik `GET /api/schedules?brand={id}`, `GET /api/public/brand?domain={hostname}`, `GET /api/itinerary/{id}` adalah KONTRAK dengan repo itu — ubah nama field/struktur nested (BUKAN nambah field baru) di endpoint ini bisa mematahkan microsite secara diam-diam. Kalau prompt minta ubah salah satu endpoint itu secara breaking, STOP dan laporkan dulu.

`azhan-microsite` punya `AGENTS.md` sendiri (kontrak API detail + cara jalankan Nginx multi-domain) — baca itu kalau kerja di repo itu, TIDAK otomatis mewarisi file ini.

---

## Tech Stack

- **Backend**: Golang (net/http + chi router), tanpa framework berat
- **Database**: MySQL 8+ — lokal via Laragon (Windows, native, tanpa Docker), production via VPS Hostinger (Docker + Caddy)
- **Auth**: JWT (access 15 menit + refresh 7 hari), bcrypt untuk password, HS256, wajib ada claim `jti` unik di setiap token
- **Frontend dashboard**: React + Vite (JavaScript, bukan TypeScript) + Tailwind CSS + react-router-dom + axios
- **Package manager**: `go mod` (backend), `npm` (frontend)

---

## Struktur & Arsitektur

- **Modular monolith** — backend dipisah per domain di `/internal/{nama_modul}` (hotel, airline, itinerary, schedule, identity, dst). Modul TIDAK boleh query tabel milik modul lain secara langsung.
- Pola tiap modul: `model.go` (struct), `repository.go` (query DB), `handler.go` (HTTP handler + validasi)
- Endpoint admin selalu prefix `/api/admin/*`, dibungkus middleware `requireDB` + `RequireAuth`
- Endpoint publik tanpa prefix admin, tanpa `RequireAuth`

---

## Konvensi API (WAJIB diikuti di endpoint baru)

- **Format response error**: selalu `{"error": "pesan dalam Bahasa Indonesia"}` — konsisten dengan endpoint yang sudah ada, jangan campur bahasa
- **Format response list kosong**: array kosong `[]`, JANGAN `null`
- **Urutan validasi**: gate dicek berurutan, stop di gate pertama yang gagal — jangan validasi semua sekaligus lalu gabung pesan error
- **Uniqueness check**: case-insensitive, data dinormalisasi UPPERCASE sebelum simpan (pola sudah dipakai di hotel, airline)
- **Foreign key error (MySQL 1451)**: selalu ditangkap eksplisit dan diubah jadi `409 {"error": "tidak bisa dihapus, masih dipakai oleh ..."}` — JANGAN biarkan raw SQL error bocor ke response
- **Operasi multi-tabel** (contoh: itinerary + itinerary_days): WAJIB pakai database transaction (BEGIN/COMMIT/ROLLBACK), rollback total kalau ada error di tengah, jangan sisakan data parsial
- **Update itinerary_days**: pola replace-all (DELETE semua lalu INSERT ulang), bukan diff/merge — pola ini sudah disepakati, ikuti kalau ada kasus serupa di modul lain
- **Endpoint sensitif** (ubah status, ubah kuota kursi) dipisah dari update biasa (`PUT /schedules/{id}/status`, `PUT /schedules/{id}/seat`) — bukan digabung ke update umum

---

## Testing & Verifikasi (Kritis — sering dilanggar sebelumnya)

- **Setiap klaim "berhasil" WAJIB disertai bukti mentah**: HTTP status code + response body PERSIS, bukan ringkasan naratif ("semua tes lulus" tanpa data tidak diterima)
- Kalau ada perbandingan (contoh: dua token harus berbeda), **paste kedua nilai lengkapnya**, jangan hanya bilang "berhasil beda"
- Kalau ada test yang hasilnya TIDAK sesuai ekspektasi, laporkan apa adanya — jangan diperhalus atau disembunyikan di balik ringkasan positif
- Test transaction rollback wajib mengecek TIDAK ada data parsial tersisa di database, bukan cuma cek response error muncul

---

## Modul Booking & Jamaah — Catatan Skema Kritis

Detail lengkap field & formula ada di `analisis-modul-booking-jamaah.md` — WAJIB baca sebelum sentuh modul ini. Poin paling penting:

- **2 konsep "Add-On" BEDA, jangan tertukar**: `add_ons`+`schedule_add_ons` (master paket, TANPA harga) vs `booking_addons` (per transaksi booking, ADA `nominal`). Kalau prompt sebut "add-on" tanpa konteks jelas, TANYA dulu.
- `bookings.harga_dasar` = snapshot harga kamar saat dibuat (bukan referensi live ke `schedules`)
- `total_harga = GREATEST(0, harga_dasar + SUM(booking_addons.nominal) - diskon)`
- `seat_sisa` berkurang HANYA saat booking pertama kali masuk status `dp` (row-level lock wajib), dikembalikan kalau `batal` setelah sempat terkunci
- "Turut Serta"/rombongan sengaja TIDAK dibangun — jangan tambahkan kecuali diminta ulang eksplisit

---

## Safety Guardrails (Kritis)

- **Migrasi database** (`ALTER TABLE`, `DROP COLUMN`, dst): boleh dieksekusi kalau sudah dijelaskan di prompt secara eksplisit, TAPI kalau menyimpang dari yang diminta (skema tambahan yang tidak diminta, dll), STOP dan laporkan dulu sebelum eksekusi
- **DILARANG ubah skema database manual di luar file `migrations/`** — semua perubahan (termasuk saat debugging) WAJIB jadi file migrasi resmi bernomor urut. Pelanggaran ini pernah terjadi (`payments.bukti_url`), berisiko environment baru gagal karena kolom tidak pernah tercipta.
- **Jangan hapus/drop tabel yang sudah berisi data** tanpa konfirmasi eksplisit terpisah, meski diminta "buat ulang dari awal" — kecuali BAGIAN 0 prompt secara eksplisit menyebut tabel itu aman dihapus
- **Jangan commit atau tampilkan isi `.env`** ke chat/log dengan kredensial asli (JWT_SECRET, password DB) — kalau perlu menyebutnya, gunakan placeholder
- **Jangan install dependency baru** di luar yang disebutkan eksplisit di prompt tanpa menyebutkan alasannya dulu
- Tidak ada langkah "deploy ke production" di project ini saat ini — kalau suatu saat ada, WAJIB approval eksplisit terpisah, jangan diasumsikan dari konteks

---

## Code Quality

- Fokus scope ketat sesuai prompt — jangan implementasikan modul/fitur di luar yang diminta meski terasa "related" (pola prompt di project ini selalu eksplisit menyebut "Jangan sentuh modul X")
- Ikuti pola/struktur file modul yang sudah ada persis (lihat modul `hotel` atau `itinerary` sebagai referensi) saat membuat modul baru — jangan improvisasi struktur berbeda tanpa alasan
- Command Windows/PowerShell (bukan bash) untuk semua contoh perintah terminal — environment dev di Windows (Laragon)

---

## Git Conventions

- Belum ada konvensi commit message yang disepakati secara resmi — gunakan format bebas tapi deskriptif untuk saat ini, akan diformalkan saat tim bertambah

---

## Frontend — Design System

**WAJIB baca `.agents/rules/design-system.md` dan `design-system.md` sebelum menyentuh kode UI apapun.** Ringkasan aturan paling penting: dilarang keras nilai Tailwind arbitrary/hardcode warna, selalu pakai token semantik (`primary-*`, `neutral-*`, dst) dan komponen dari `src/components/ui/`.

---

## Frontend — Responsivitas & Tipografi (WAJIB, berlaku ke semua layout baru)

- **Setiap layout UI baru WAJIB responsive** — berfungsi dan enak dilihat di desktop maupun mobile, bukan cuma di-test/dibangun untuk 1 lebar layar (default dev sejauh ini condong ke desktop, ini harus mulai diperbaiki mulai dari fitur berikutnya)
- Pendekatan: **mobile-first** — style dasar untuk mobile, lalu tambahkan breakpoint Tailwind (`sm:`, `md:`, `lg:`, `xl:`) untuk memperbesar/reorganisasi layout di layar lebih lebar. Contoh: layout 2 kolom (sidebar kanan ala ScheduleFormPage) di desktop harus otomatis jadi 1 kolom bertumpuk di mobile (`flex-col md:flex-row` atau setara), bukan tetap 2 kolom sempit yang tidak terbaca
- Sidebar dashboard: WAJIB collapsible/hidden-by-default di mobile (hamburger menu atau drawer), TIDAK boleh selalu memakan lebar layar di layar sempit
- Tabel data: di mobile, pertimbangkan scroll horizontal (`overflow-x-auto` pada wrapper) daripada memaksa semua kolom muat, ATAU susun ulang jadi card list per baris — pilih salah satu pendekatan yang konsisten dipakai lintas halaman, bukan tiap halaman beda pendekatan
- **Tipografi WAJIB scaling mengikuti lebar layar** — bukan ukuran font tetap satu nilai di semua breakpoint. Pakai varian responsive Tailwind (contoh: `text-xl md:text-2xl lg:text-3xl` untuk judul), bukan satu `text-2xl` statis yang sama persis di HP dan desktop
- Test manual sebelum lapor "selesai": resize browser ke lebar ±375px (mobile) DAN ±1440px (desktop) minimal, screenshot keduanya sebagai bukti verifikasi — bukan cuma screenshot desktop seperti kebiasaan sebelumnya di project ini

---

## Communication

- Ringkas dan langsung ke inti — hindari penjelasan konsep dasar yang tidak diminta
- Kalau ragu dengan maksud instruksi (ambigu), tanya dulu daripada menebak dan implementasi salah arah
- Kalau menemukan potensi bug/celah di luar scope prompt saat ini, laporkan tapi jangan langsung diperbaiki tanpa diminta — biar dikonfirmasi dulu prioritasnya